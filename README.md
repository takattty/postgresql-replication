# PostgreSQL Streaming Replication Demo

[![CI](https://github.com/takattty/postgresql-replication/actions/workflows/ci.yml/badge.svg)](https://github.com/takattty/postgresql-replication/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/takattty/postgresql-replication/branch/main/graph/badge.svg)](https://codecov.io/gh/takattty/postgresql-replication)
[![Go Report Card](https://goreportcard.com/badge/github.com/takattty/postgresql-replication)](https://goreportcard.com/report/github.com/takattty/postgresql-replication)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

This repository contains a Docker Compose setup for learning how PostgreSQL streaming replication works. The environment runs a primary server, a standby server initialised by `pg_basebackup` and a pgAdmin instance for administration.  It also includes a small Go application that demonstrates read/write splitting and provides basic tests.

## Directory overview

- `docker-compose.yml` – orchestrates the containers
- `primary/` and `standby/` – PostgreSQL configuration files
- `scripts/` – setup scripts executed when the containers initialise
- `app/` – Go sample application and tests
- `*.md` – additional documentation

## Prerequisites

- Docker and Docker Compose
- Go (for running the demo application)

## Quick start

1. Copy the environment template and edit the passwords:
   ```bash
   cp .env.example .env
   # open .env and set secure values
   ```
2. Start the containers:
   ```bash
   docker compose up -d
   ```
3. Check container status and logs:
   ```bash
   docker compose ps
   docker compose logs postgres-primary
   docker compose logs postgres-standby
   ```
4. Confirm replication:
   ```bash
   docker exec -it postgres-primary psql -U postgres -d testdb -c "SELECT * FROM pg_stat_replication;"
   docker exec -it postgres-standby psql -U postgres -d testdb -c "SELECT * FROM pg_stat_wal_receiver;"
   ```
5. Stop the environment:
   ```bash
   docker compose down
   ```

pgAdmin will be available on <http://localhost:8080> (credentials are defined in `.env`).

## Go demo application

Inside `app/` you will find a simple program that writes data to the primary server using `docker exec` and reads from the standby using a direct connection. The code also includes a comprehensive test suite.

### 🐳 Docker Development (Recommended)

**Complete development environment with PostgreSQL replication:**

```bash
# Start complete development environment
make docker-dev          # Starts Go + PostgreSQL replication

# Run comprehensive tests (6 test cases)
make docker-test         # ✅ All replication tests

# Code quality checks
make docker-lint         # golangci-lint static analysis
make docker-security     # gosec security scanning
make docker-ci           # Complete CI pipeline

# Interactive development
make docker-shell        # Open shell in Go container
```

### 🔧 Local Development

```bash
# Setup development environment
make setup

# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific applications
make run-connection-check
make run-simple-demo
make run-replication-demo

# See all available commands (40+)
make help
```

### Manual commands

```bash
cd app
go mod tidy
go test -v
```

### 📊 Test Results

The comprehensive test suite validates:
- **Database Connectivity**: Primary/Standby connection health
- **Replication Functionality**: Data synchronization verification  
- **Performance Metrics**: Read/write operation timing
- **Data Consistency**: Multi-write synchronization validation
- **Lag Monitoring**: Replication delay measurement

## Security notes

This project is for demonstration purposes. Default passwords are provided only for convenience – **always** change them in your `.env` file. In production you should further restrict network access and consider enabling SSL/TLS as outlined in `SECURITY.md`.

## Development

### Code Quality

This project includes automated code quality checks:

- **Linting**: `golangci-lint` with comprehensive rules
- **Security**: `gosec` for security vulnerability scanning
- **Testing**: Comprehensive test suite with coverage reporting
- **Formatting**: `go fmt` for consistent code style

### CI/CD

The project uses GitHub Actions for:
- **Automated Quality Checks**: golangci-lint static analysis
- **Security Scanning**: gosec vulnerability detection  
- **Build Verification**: Multi-command compilation testing
- **Docker Integration**: Container-based development workflow

**Local CI Simulation:**
```bash
make docker-ci    # Run complete CI pipeline locally
```

### Project Structure

```
.
├── .github/workflows/    # GitHub Actions CI/CD
├── app/                  # Go application
│   ├── cmd/             # Command-line applications
│   │   ├── connection_check/  # Database connectivity tool
│   │   ├── simple_demo/       # Basic replication demo
│   │   └── replication_demo/  # Comprehensive demo
│   ├── bin/             # Compiled binaries
│   ├── .golangci.yml    # Linter configuration (25+ rules)
│   └── replication_test.go    # Test suite (6 test cases)
├── docker-compose.yml    # Container orchestration
├── Makefile             # Development automation (40+ commands)
├── codecov.yml          # Coverage configuration
├── primary/             # PostgreSQL primary config
├── standby/             # PostgreSQL standby config
├── scripts/             # Setup scripts
└── *.md                 # Documentation
```

### 🚀 Quick Start

```bash
# 1. Clone and setup
git clone <repository>
cd postgresql-replication

# 2. Start Docker development environment
make docker-dev

# 3. Run all tests
make docker-test

# 4. Check code quality  
make docker-ci
```

## Further reading

More detailed explanations of the experiment, troubleshooting steps and development notes can be found in the other Markdown files in this repository.
