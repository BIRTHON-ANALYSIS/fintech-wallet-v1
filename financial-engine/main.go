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

	w.Header().Set("Content-Type", "application/json")

	response := HealthResponse{
		Status:  "healthy",
		Service: "financial-engine",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode health response", "error", err)
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("FINANCIAL_ENGINE_PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

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
