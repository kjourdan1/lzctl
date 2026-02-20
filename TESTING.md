# Testing Guide

## Prerequisites

| Tool | Version | Usage |
|------|---------|-------|
| Go | >= 1.24 | Build and tests |
| Terraform | >= 1.5 | Integration tests |
| Azure CLI | >= 2.50 | Azure integration tests |

## Build

```bash
# Build the binary
go build -o bin/lzctl .

# Verify
./bin/lzctl version
```

## Unit Tests

```bash
# All tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Verbose output
go test -v ./...

# Specific package
go test -v ./internal/config/...
go test -v ./internal/audit/...
go test -v ./internal/state/...
go test -v ./internal/doctor/...

# Specific test
go test -v ./internal/config/ -run TestCrossValidate

# With race detection
go test -race ./...
```

## Tests by Package

| Package | Test File | What It Covers |
|---------|-----------|----------------|
| `internal/config` | `crossvalidator_test.go`, `loader_test.go` | Schema validation, cross-checks CIDR/UUID |
| `internal/audit` | `audit_test.go`, `renderers_test.go` | CAF compliance rules, MD/JSON renderers |
| `internal/doctor` | `checks_test.go` | terraform, az, git, state backend checks |
| `internal/state` | `state_test.go` | List, snapshot, health check, manager |
| `internal/template` | `engine_test.go`, `pipeline_updater_test.go` | Template rendering |
| `internal/upgrade` | `registry_test.go`, `updater_test.go` | AVM module upgrades |
| `internal/importer` | `hcl_generator_test.go` | Import block generation |
| `internal/policy` | `policy_test.go` | Policy-as-Code lifecycle |
| `cmd/` | `cmd_test.go`, `rollback_test.go`, `local_ops_test.go` | CLI commands |

## Integration Tests

```bash
# Integration tests (require an Azure environment)
go test -v ./test/integration/...

# Template rendering tests
go test -v ./test/integration/ -run TestTemplateRender

# Brownfield tests
go test -v ./test/integration/ -run TestBrownfield
```

## Local Smoke Test

```bash
# Build
go build -o bin/lzctl .

# Doctor
./bin/lzctl doctor

# Init dry-run
./bin/lzctl init --dry-run

# Validate
./bin/lzctl validate --strict

# Schema export
./bin/lzctl schema export

# State health (requires Azure)
./bin/lzctl state health
```

## Makefile

```bash
make build              # Build the binary
make test               # Unit tests
make test-verbose       # Tests with details
make fmt                # Format code
make vet                # Static analysis
make lint               # Linter (golangci-lint)
make test-coverage      # Test coverage report
make test-coverage-check # Coverage gate (threshold: 45%)
make clean              # Clean build artefacts
```
