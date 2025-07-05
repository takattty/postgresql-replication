# PostgreSQL Replication Demo Makefile

# Go設定
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt

# Docker設定
DOCKER_COMPOSE=docker compose

# アプリケーションディレクトリ
APP_DIR=app
BIN_DIR=$(APP_DIR)/bin

# バイナリ名
BINARY_CONNECTION_CHECK=connection_check
BINARY_SIMPLE_DEMO=simple_demo
BINARY_REPLICATION_DEMO=replication_demo

# デフォルトターゲット
.PHONY: all
all: clean deps lint test build

# 依存関係のインストール
.PHONY: deps
deps:
	@echo "📦 Installing dependencies..."
	cd $(APP_DIR) && $(GOMOD) download
	cd $(APP_DIR) && $(GOMOD) verify

# コードフォーマット
.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	cd $(APP_DIR) && $(GOFMT) ./...

# リンター実行
.PHONY: lint
lint:
	@echo "🔍 Running linters..."
	cd $(APP_DIR) && $(GOVET) ./...
	cd $(APP_DIR) && golangci-lint run

# テスト実行
.PHONY: test
test:
	@echo "🧪 Running tests..."
	cd $(APP_DIR) && $(GOTEST) -v -race ./...

# テストカバレッジ
.PHONY: test-coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	cd $(APP_DIR) && $(GOTEST) -v -race -coverprofile=coverage.out ./...
	cd $(APP_DIR) && $(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: $(APP_DIR)/coverage.html"

# ビルド
.PHONY: build
build: build-connection-check build-simple-demo build-replication-demo

.PHONY: build-connection-check
build-connection-check:
	@echo "🔨 Building connection_check..."
	cd $(APP_DIR) && $(GOBUILD) -o $(BIN_DIR)/$(BINARY_CONNECTION_CHECK) ./cmd/connection_check

.PHONY: build-simple-demo
build-simple-demo:
	@echo "🔨 Building simple_demo..."
	cd $(APP_DIR) && $(GOBUILD) -o $(BIN_DIR)/$(BINARY_SIMPLE_DEMO) ./cmd/simple_demo

.PHONY: build-replication-demo
build-replication-demo:
	@echo "🔨 Building replication_demo..."
	cd $(APP_DIR) && $(GOBUILD) -o $(BIN_DIR)/$(BINARY_REPLICATION_DEMO) ./cmd/replication_demo

# クリーンアップ
.PHONY: clean
clean:
	@echo "🧹 Cleaning up..."
	cd $(APP_DIR) && $(GOCLEAN)
	cd $(APP_DIR) && rm -rf $(BIN_DIR)
	cd $(APP_DIR) && rm -f coverage.out coverage.html

# Docker環境管理
.PHONY: docker-up
docker-up:
	@echo "🚀 Starting Docker environment..."
	$(DOCKER_COMPOSE) up -d

.PHONY: docker-down
docker-down:
	@echo "🛑 Stopping Docker environment..."
	$(DOCKER_COMPOSE) down

.PHONY: docker-restart
docker-restart: docker-down docker-up

.PHONY: docker-logs
docker-logs:
	@echo "📜 Showing Docker logs..."
	$(DOCKER_COMPOSE) logs -f

.PHONY: docker-status
docker-status:
	@echo "📋 Docker container status..."
	$(DOCKER_COMPOSE) ps

# PostgreSQL接続確認
.PHONY: db-status
db-status:
	@echo "🔍 Checking database status..."
	$(DOCKER_COMPOSE) exec postgres-primary psql -U postgres -d testdb -c "SELECT 'Primary OK' as status;"
	$(DOCKER_COMPOSE) exec postgres-standby psql -U postgres -d testdb -c "SELECT 'Standby OK' as status;"

# レプリケーション状態確認
.PHONY: replication-status
replication-status:
	@echo "📊 Checking replication status..."
	$(DOCKER_COMPOSE) exec postgres-primary psql -U postgres -d testdb -c "SELECT * FROM pg_stat_replication;"
	$(DOCKER_COMPOSE) exec postgres-standby psql -U postgres -d testdb -c "SELECT * FROM pg_stat_wal_receiver;"

# デモアプリケーション実行
.PHONY: run-connection-check
run-connection-check: build-connection-check
	@echo "🔗 Running connection check..."
	cd $(APP_DIR) && ./$(BIN_DIR)/$(BINARY_CONNECTION_CHECK)

.PHONY: run-simple-demo
run-simple-demo: build-simple-demo
	@echo "🚀 Running simple demo..."
	cd $(APP_DIR) && ./$(BIN_DIR)/$(BINARY_SIMPLE_DEMO)

.PHONY: run-replication-demo
run-replication-demo: build-replication-demo
	@echo "🔄 Running replication demo..."
	cd $(APP_DIR) && ./$(BIN_DIR)/$(BINARY_REPLICATION_DEMO)

# セキュリティチェック
.PHONY: security
security:
	@echo "🔐 Running security checks..."
	cd $(APP_DIR) && gosec ./...

# 開発環境セットアップ
.PHONY: setup
setup:
	@echo "🛠️  Setting up development environment..."
	@if [ ! -f .env ]; then \
		echo "📝 Creating .env file from template..."; \
		cp .env.example .env; \
		echo "⚠️  Please edit .env file with your actual values"; \
	fi
	@make deps
	@make docker-up
	@echo "✅ Development environment setup complete!"

# デモ実行フロー
.PHONY: demo
demo: setup test run-connection-check run-simple-demo run-replication-demo
	@echo "🎉 Demo execution complete!"

# CI/CDで使用するターゲット
.PHONY: ci
ci: deps fmt lint test build

# 開発者向けヘルプ
.PHONY: help
help:
	@echo "📚 Available commands:"
	@echo ""
	@echo "🔧 Build & Test:"
	@echo "  make deps                 - Install dependencies"
	@echo "  make fmt                  - Format code"
	@echo "  make lint                 - Run linters"
	@echo "  make test                 - Run tests"
	@echo "  make test-coverage        - Run tests with coverage"
	@echo "  make build                - Build all binaries"
	@echo "  make clean                - Clean build artifacts"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-up           - Start Docker environment"
	@echo "  make docker-down         - Stop Docker environment"
	@echo "  make docker-restart      - Restart Docker environment"
	@echo "  make docker-logs         - Show Docker logs"
	@echo "  make docker-status       - Show container status"
	@echo ""
	@echo "🔍 Database:"
	@echo "  make db-status           - Check database connectivity"
	@echo "  make replication-status  - Check replication status"
	@echo ""
	@echo "🚀 Run Applications:"
	@echo "  make run-connection-check  - Run connection check tool"
	@echo "  make run-simple-demo       - Run simple demo"
	@echo "  make run-replication-demo  - Run replication demo"
	@echo ""
	@echo "🛠️  Development:"
	@echo "  make setup               - Setup development environment"
	@echo "  make demo                - Run full demo flow"
	@echo "  make security            - Run security checks"
	@echo "  make ci                  - Run CI pipeline locally"
	@echo "  make help                - Show this help message"