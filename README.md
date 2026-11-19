# Fintech Wallet

A production-oriented fintech wallet portfolio project built with **Django, Go, PostgreSQL, Redis, Vue.js and Python**.

The project explores the engineering principles behind modern digital wallet systems, including secure authentication, wallet management, financial transactions, double-entry bookkeeping, payment-provider integration, transaction verification, idempotency and fraud detection.

> **Project status:** Under active development
> **Project type:** Fintech engineering portfolio / demonstration platform
> **Target:** A secure, scalable and well-documented wallet application

---

## 1. Project Overview

Fintech Wallet is designed to demonstrate how a modern wallet platform can be architected using multiple technologies, with clear separation of responsibilities.

The system uses:

- **Django** for authentication, user management and application APIs.
- **Go** for financial transaction processing and wallet operations.
- **PostgreSQL** for persistent financial data.
- **Redis** for caching and background-processing support.
- **Vue.js** for the frontend application.
- **Python** for fraud detection and risk analysis.

The financial engine is designed to maintain accurate wallet balances, enforce transaction integrity and prevent duplicate financial operations.

This is a portfolio and demonstration project. It is not a licensed financial institution and does not hold real customer funds.

## 2. Technology Stack

| Technology            | Responsibility                                   |
| --------------------- | ------------------------------------------------ |
| Python / Django       | Backend application and authentication           |
| Django REST Framework | REST API development                             |
| Go (Golang)           | Financial engine and transaction processing      |
| PostgreSQL            | Relational database and financial persistence    |
| Redis                 | Caching and background-processing infrastructure |
| Vue.js 3              | Frontend application                             |
| Pinia                 | Frontend state management                        |
| Axios                 | HTTP communication                               |
| Python / Scikit-learn | Fraud detection and risk analysis                |
| Docker                | Application containerisation                     |
| Git / GitHub          | Version control and collaboration                |

## 3. System Architecture

The application follows a service-oriented architecture with a clear separation between application-level operations and financial processing.

```text
                 Vue.js Frontend
                        |
                        v
              Django REST Framework
                        |
             Authentication & Security
                        |
                        v
                Go Financial Engine
                        |
          +-------------+-------------+
          |             |             |
       Wallets       Transfers      Ledger
          |             |             |
          +-------------+-------------+
                        |
                        v
                    PostgreSQL
                        |
              Financial Persistence

       Payment Providers
       /       |        \
  Paystack    OPay    Moniepoint
       \       |        /
        Payment Integration
                 |
                 v
          Django / Go APIs

       Python Fraud Detection
                 |
                 v
           Risk Assessment
```

### Architectural principles

- Separation of application and financial responsibilities.
- PostgreSQL-backed financial persistence.
- Atomic database transactions.
- Row-level locking for concurrent financial operations.
- Idempotency for duplicate-request protection.
- Double-entry ledger principles.
- Secure internal service communication.
- Explicit validation of wallet ownership and transaction permissions.

## 4. Core Features and Development Status

### Authentication and User Management

- [ ] Custom Django user model.
- [ ] User registration.
- [ ] JWT authentication.
- [ ] Refresh-token support.
- [ ] Token blacklisting.
- [ ] Email OTP registration verification.
- [ ] OTP resend functionality.
- [ ] Additional account lockout and security controls.

### Go Financial Engine

- [ ] Financial domain models.
- [ ] Wallet domain model.
- [ ] Transaction domain model.
- [ ] Money representation using integer minor currency units.
- [ ] Ledger domain model.
- [ ] Idempotency domain model.
- [ ] Transfer service.
- [ ] Repository abstraction.
- [ ] In-memory repository implementation.
- [ ] PostgreSQL repository implementation.
- [ ] PostgreSQL transfer transaction operations.
- [ ] Database migration runner.
- [ ] Automated unit and integration test suite.
- [ ] Full concurrency and failure-recovery verification.

### Payment Integration

Planned provider integrations:

- Paystack.
- OPay.
- Moniepoint.

Planned payment functionality:

- Deposits.
- Payment initiation.
- Provider payment verification.
- Webhook processing.
- Duplicate payment protection.
- Withdrawals and payouts where supported.
- Refund tracking.
- Transaction reconciliation.

The payment architecture will use provider adapters to reduce dependence on any single payment provider.

Provider sandbox access, merchant approval and API availability will determine the extent of live end-to-end integration.

### KYC and Security

- Simulated identity verification workflow.
- OTP and transaction verification.
- Transaction PIN.
- Wallet security controls.
- Account activity monitoring.
- Transaction risk assessment.

### Frontend

Planned Vue.js application:

- Authentication pages.
- Wallet dashboard.
- Balance overview.
- Wallet transfers.
- Deposit and withdrawal interfaces.
- Transaction history.
- Payment-method management.
- KYC and security settings.
- Risk and administration dashboard.

### Fraud Detection

Planned Python fraud detection service:

- Transaction risk scoring.
- Rule-based fraud detection.
- Machine-learning-assisted anomaly detection.
- Suspicious transaction identification.
- Risk monitoring dashboard.

## 5. Financial Integrity and Security

Financial accuracy is a central design requirement.

### Integer-based money representation

Monetary values are represented using integer minor currency units rather than floating-point arithmetic.

For example:


This avoids floating-point rounding errors in financial calculations.

### Atomic Transfers

A wallet transfer must be processed as one database transaction.

The intended operation is:

1. Validate the transfer request.
2. Validate the idempotency key.
3. Lock the affected wallet rows.
4. Verify wallet status and currency.
5. Verify sufficient funds.
6. Debit the source wallet.
7. Credit the destination wallet.
8. Record the financial transaction.
9. Persist the corresponding ledger entries.
10. Commit the database transaction.

If any operation fails, the transaction must roll back.

### Idempotency

Idempotency prevents duplicate financial operations when clients retry requests or network failures occur.

The database must enforce uniqueness for the relevant idempotency key, with request consistency checks.

### Ledger Integrity

The financial ledger is designed around double-entry bookkeeping.

Every successful transfer must produce corresponding debit and credit entries of equal value.

## 6. Project Structure

The current project uses a Django application alongside a dedicated Go financial engine.

```text
fintech-wallet-v1/
│
├── apps/
│   └── wallets/
│
├── config/
│
├── financial-engine/
│   ├── cmd/
│   │   └── migrate/
│   │       └── main.go
│   │
│   ├── internal/
│   │   ├── database/
│   │   ├── domain/
│   │   ├── http/
│   │   ├── httpapi/
│   │   ├── migrations/
│   │   ├── ports/
│   │   ├── repository/
│   │   └── service/
│   │
│   ├── main.go
│   └── go.mod
│
├── .env.example
├── .gitignore
├── manage.py
└── README.md
```

The structure will evolve as the frontend, payment integrations and fraud detection components are introduced.

## 7. Local Development Setup

### Prerequisites

Install:

- Python.
- Go.
- PostgreSQL.
- Redis.
- Node.js and npm (for the Vue frontend when introduced).
- Git.

### Clone the repository

```bash
git clone https://github.com/BIRTHON-ANALYSIS/fintech-wallet-v1.git

cd fintech-wallet-v1
```

### Django setup

Create and activate a virtual environment:

```bash
python -m venv fin
```

Windows PowerShell:

```powershell
.\fin\Scripts\Activate.ps1
```

Install the project's Python dependencies:

```bash
pip install -r requirements.txt
```

Configure environment variables before running Django.

### Go financial engine setup

Navigate into the Go project:

```bash
cd financial-engine
```

Download dependencies:

```bash
go mod download
```

Build the financial engine:

```bash
go build ./...
```

### PostgreSQL configuration

Create a PostgreSQL database and a dedicated application user.

Configure the database connection through the environment variable:


go run ./cmd/migrate
```

### Running the applications

Django development server:

```bash
python manage.py runserver
```

The Go server is started using its configured application entry point and environment.

The exact service startup commands and API endpoints will be documented as the Django–Go integration is finalised.

## 8. Environment Variables

The application uses environment-based configuration.

Expected configuration includes:

```env
# Django
SECRET_KEY=
DEBUG=
ALLOWED_HOSTS=

# PostgreSQL
DATABASE_URL=

# Redis
REDIS_URL=

# Authentication
ACCESS_TOKEN_LIFETIME=
REFRESH_TOKEN_LIFETIME=

# Internal service communication
FINANCIAL_ENGINE_URL=
FINANCIAL_ENGINE_SECRET=

# Payment providers
PAYSTACK_SECRET_KEY=
OPAY_SECRET_KEY=
MONIEPOINT_SECRET_KEY=

# Fraud detection
FRAUD_SERVICE_URL=
```

These are configuration placeholders. Actual names and requirements will be aligned with the implementation of each service.

**Never commit `.env` files, private keys, API secrets or production credentials.**

## 9. Testing Strategy

Testing will cover both application behaviour and financial integrity.

Planned verification includes:

- Authentication and authorization tests.
- Transfer validation tests.
- Insufficient-balance tests.
- Idempotency tests.
- PostgreSQL transaction tests.
- Concurrent transfer tests.
- Database rollback tests.
- Payment webhook verification tests.
- Fraud detection tests.
- End-to-end integration tests.

PostgreSQL integration tests will be used to verify actual row-level locking, atomicity and persistence behaviour.

## 10. Deployment

The deployment phase will include:

- Docker containerisation.
- Production environment configuration.
- PostgreSQL deployment.
- Redis deployment.
- Secure service-to-service communication.
- Reverse proxy and HTTPS.
- Application logging.
- Health checks.
- Monitoring and backup procedures.

Deployment instructions and the live application link will be added when production deployment is complete.

## 11. Development Roadmap

| Phase | Scope                                          | Status                            |
| ----- | ---------------------------------------------- | --------------------------------- |
| 1     | Authentication and user management             | Implemented                       |
| 2     | Go financial engine and PostgreSQL persistence | Implemented; verification ongoing |
| 3     | Django–Go integration                          | Planned                           |
| 4     | Payment-provider integration                   | Planned                           |
| 5     | KYC and security workflows                     | Planned                           |
| 6     | Vue.js frontend                                | Planned                           |
| 7     | AI/ML fraud detection                          | Planned                           |
| 8     | Testing, Docker and production deployment      | Planned                           |

## 12. Project Goals

The objective is to demonstrate practical engineering skills in:

- Backend architecture.
- Python and Go development.
- Financial transaction processing.
- Database transaction management.
- API integration.
- Payment-provider integration.
- Security engineering.
- Distributed application communication.
- Automated testing.
- Containerisation and deployment.

The project is intended to serve as a technical portfolio demonstrating the design, implementation and deployment of a modern fintech wallet system.
---

**Repository:** [BIRTHON-ANALYSIS/fintech-wallet-v1](https://github.com/BIRTHON-ANALYSIS/fintech-wallet-v1)

**Development status:** Active development.
