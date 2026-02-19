# ─────────────────────────────────────────────────────────────
# lzctl — Azure Landing Zone Factory
# Build automation
# ─────────────────────────────────────────────────────────────

# Variables
BINARY_NAME  := lzctl
MODULE       := github.com/kjourdan1/lzctl
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT       ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE   ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO           := go
GOFLAGS      := -trimpath
LDFLAGS      := -s -w \
	-X '$(MODULE)/cmd.Version=$(VERSION)' \
	-X '$(MODULE)/cmd.Commit=$(COMMIT)' \
	-X '$(MODULE)/cmd.BuildDate=$(BUILD_DATE)'
BIN_DIR      := bin
PLATFORMS    := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test lint fmt vet tidy install cross-compile help

# ── Default ──────────────────────────────────────────────────

all: tidy fmt vet lint test build ## Run full build pipeline

# ── Build ────────────────────────────────────────────────────

build: ## Build binary for current platform
	@echo "==> Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "==> Built: $(BIN_DIR)/$(BINARY_NAME)"

build-debug: ## Build with debug symbols
	@echo "==> Building $(BINARY_NAME) (debug)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build -gcflags="all=-N -l" -o $(BIN_DIR)/$(BINARY_NAME) .

install: build ## Install binary to $GOPATH/bin
	@echo "==> Installing $(BINARY_NAME)..."
	@cp $(BIN_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# ── Cross Compile ────────────────────────────────────────────

cross-compile: ## Build for all platforms
	@echo "==> Cross-compiling $(BINARY_NAME) $(VERSION)..."
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		output="$(BIN_DIR)/$(BINARY_NAME)-$$os-$$arch$$ext"; \
		echo "  Building $$output..."; \
		GOOS=$$os GOARCH=$$arch $(GO) build $(GOFLAGS) \
			-ldflags "$(LDFLAGS)" -o $$output . || exit 1; \
	done
	@echo "==> Cross-compilation complete"

# ── Test ─────────────────────────────────────────────────────

# Detect whether -race is supported (requires CGO on Windows)
RACE_FLAG := $(shell $(GO) env CGO_ENABLED 2>/dev/null)
ifeq ($(RACE_FLAG),1)
  RACE := -race
else
  RACE :=
endif

test: ## Run tests
	@echo "==> Running tests..."
	$(GO) test $(RACE) -cover ./...

test-verbose: ## Run tests with verbose output
	@echo "==> Running tests (verbose)..."
	$(GO) test $(RACE) -cover -v ./...

test-coverage: ## Run tests with coverage report
	@echo "==> Running tests with coverage..."
	@mkdir -p $(BIN_DIR)
	$(GO) test $(RACE) -coverprofile=$(BIN_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BIN_DIR)/coverage.out -o $(BIN_DIR)/coverage.html
	@echo "==> Coverage report: $(BIN_DIR)/coverage.html"

# ── Code Quality ─────────────────────────────────────────────

lint: ## Run golangci-lint
	@echo "==> Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "  golangci-lint not installed, skipping"; \
	fi

fmt: ## Format code
	@echo "==> Formatting code..."
	$(GO) fmt ./...

vet: ## Run go vet
	@echo "==> Running vet..."
	$(GO) vet ./...

# ── Dependencies ─────────────────────────────────────────────

tidy: ## Tidy dependencies
	@echo "==> Tidying dependencies..."
	$(GO) mod tidy

deps: ## Download dependencies
	@echo "==> Downloading dependencies..."
	$(GO) mod download

verify: ## Verify dependencies
	@echo "==> Verifying dependencies..."
	$(GO) mod verify

# ── Clean ────────────────────────────────────────────────────

clean: ## Clean build artifacts
	@echo "==> Cleaning..."
	@rm -rf $(BIN_DIR)
	@echo "==> Clean complete"

# ── Docker ───────────────────────────────────────────────────

docker-build: ## Build Docker image
	@echo "==> Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		.

# ── Validation ───────────────────────────────────────────────

validate-terraform: ## Validate all Terraform modules
	@echo "==> Validating Terraform modules..."
	@for dir in modules/*/; do \
		echo "  Validating $$dir..."; \
		cd $$dir && terraform init -backend=false -input=false >/dev/null 2>&1 && \
		terraform validate && cd ../..; \
	done

validate-yaml: ## Validate YAML files
	@echo "==> Validating YAML files..."
	@find . -name "*.yaml" -o -name "*.yml" | while read f; do \
		python3 -c "import yaml; yaml.safe_load(open('$$f'))" 2>/dev/null || \
		echo "  WARN: $$f may have issues"; \
	done

validate-json: ## Validate JSON files
	@echo "==> Validating JSON files..."
	@find . -name "*.json" | while read f; do \
		python3 -c "import json; json.load(open('$$f'))" 2>/dev/null || \
		echo "  WARN: $$f may have issues"; \
	done

# ── Help ─────────────────────────────────────────────────────

help: ## Show this help
	@echo "lzctl — Azure Landing Zone Factory"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
