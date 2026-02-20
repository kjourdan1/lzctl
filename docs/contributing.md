# Contributing to lzctl

## Development Setup

```bash
# Prerequisites
go version    # >= 1.24
terraform -v  # >= 1.5
git --version # >= 2.30

# Clone and build
git clone https://github.com/kjourdan1/lzctl.git
cd lzctl
go mod tidy
go build -o bin/lzctl .
```

## Development Workflow

1. Create a branch: `git checkout -b feature/<name>`
2. Develop
3. Run checks:
   ```bash
   make fmt      # Format code
   make vet      # Static analysis
   make test     # Run all tests
   make build    # Verify compilation
   ```
4. Validate manifests: `lzctl validate --strict`
5. Submit a Pull Request

## Code Organisation

| Directory | Role |
|-----------|------|
| `cmd/` | Cobra CLI commands (thin wrappers calling `internal/`) |
| `internal/` | Business logic (config, audit, state, policy, template, etc.) |
| `schemas/` | Embedded JSON schema for `lzctl.yaml` validation |
| `templates/` | Go templates for Terraform, pipeline, and manifest generation |
| `profiles/` | CAF profile catalogue (`catalog.yaml`) |
| `docs/` | Documentation (commands, architecture, operations) |

## Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/config/...

# Specific test
go test ./internal/config/ -run TestCrossValidate

# Verbose output
go test -v ./internal/audit/...

# With race detection
go test -race ./...
```

See [TESTING.md](../TESTING.md) for the full testing guide.

## Conventions

- **Commands** (`cmd/`): Thin — parse flags, call `internal/`, display the result
- **Internal packages**: All business logic. No Cobra dependencies.
- **Manifest**: `apiVersion: lzctl/v1`, `kind: LandingZone`
- **Errors**: `fmt.Errorf("context: %w", err)` — wrap, never swallow
- **Output**: Use `internal/output` for formatted messages (Info, Success, Warning, Error)

## Adding a New Command

1. Create `cmd/<command>.go` with a Cobra command
2. Create `internal/<package>/<package>.go` with the logic
3. Wire in `cmd/root.go` (`rootCmd.AddCommand`)
4. Add tests in `cmd/cmd_test.go` and `internal/<package>/<package>_test.go`
5. Update the CLI Reference in `docs/cli-reference.md`

## Adding an Audit Rule

1. Create a function in `internal/audit/` following the `check<Rule>()` pattern
2. Add to the list in `compliance_engine.go`
3. Test with fixtures in `audit_test.go`
4. Document in `docs/commands/audit.md`
