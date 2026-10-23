package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"financial-engine/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

var requestCounter atomic.Uint64

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestCounter.Add(1)
		requestID := time.Now().UTC().Format("20060102T150405") +
			"-" + formatCounter(id)

		w.Header().Set("X-Request-ID", requestID)

		start := time.Now()
		next.ServeHTTP(w, r)

		slog.Info(
			"http request completed",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func formatCounter(n uint64) string {
	const digits = "0123456789"

	if n == 0 {
		return "0"
	}

	var buffer [20]byte
	i := len(buffer)

	for n > 0 {
		i--
		buffer[i] = digits[n%10]
		n /= 10
	}

	return string(buffer[i:])
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "healthy",
		Service: "financial-engine",
	})
}

func readinessHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			slog.Error("database readiness check failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not_ready",
				"reason": "database unavailable",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":   "ready",
			"service":  "financial-engine",
			"database": "connected",
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("FINANCIAL_ENGINE_PORT")
	if port == "" {
		port = "8081"
	}

	// Connect to PostgreSQL before accepting requests.
	startupCtx, cancelStartup := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	pool, err := database.OpenPostgres(startupCtx)
	cancelStartup()

	if err != nil {
		slog.Error("PostgreSQL initialization failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("PostgreSQL connection pool initialized")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readinessHandler(pool))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           requestIDMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("financial engine starting", "port", port)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("financial engine failed", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdownSignal:
		slog.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("graceful shutdown failed", "error", err)

			if closeErr := server.Close(); closeErr != nil {
				slog.Error("forced server close failed", "error", closeErr)
			}
		}

		if err := <-serverErrors; err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped with error", "error", err)
		}
	}

	slog.Info("financial engine stopped")
}
