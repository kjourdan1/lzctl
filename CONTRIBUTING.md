# Contributing

Thank you for your interest in contributing to **lzctl**!

## Quick Start

```bash
git clone https://github.com/kjourdan1/lzctl.git
cd lzctl
go mod tidy
go build -o bin/lzctl .
go test ./...
```

## Workflow

1. Fork the repository
2. Create a branch: `git checkout -b feature/<name>`
3. Develop with tests
4. Verify: `make fmt && make vet && make test`
5. Submit a Pull Request

## Standards

- Code formatted with `gofmt`
- Unit tests for all logic in `internal/`
- CLI commands (`cmd/`) = thin wrappers — logic lives in `internal/`
- Wrapped errors: `fmt.Errorf("context: %w", err)`
- No credentials in code or templates

## Project Structure

```
cmd/              Cobra CLI commands
internal/         Business logic
  audit/          CAF compliance audit
  azauth/         Azure authentication
  azure/          az CLI wrapper
  bootstrap/      State backend bootstrap
  config/         Parsing and validation of lzctl.yaml
  doctor/         Prerequisite checks
  exitcode/       Standardised exit codes
  importer/       Terraform import block generation
  oidcsetup/      OIDC federated credential setup
  output/         Output formatting (colours, JSON)
  planverify/     Plan verification
  policy/         Policy-as-Code lifecycle
  state/          State file lifecycle management
  template/       Go template engine
  upgrade/        AVM module version checker and updater
  wizard/         Interactive wizard (survey)
schemas/          JSON schema for lzctl.yaml
templates/        Terraform, pipeline, and manifest templates
profiles/         CAF profile catalogue
docs/             Documentation
```

## Tests

```bash
go test ./...                    # All tests
go test -v ./internal/config/... # Single package with details
go test -race ./...              # With race detection
make test-coverage-check         # Local coverage gate (current threshold: 45%)
```

### Coverage Policy

- Current threshold (CI): **45%** total coverage (`go test -coverprofile=coverage.out ./...`)
- Next milestones: **60%** then PRD target **80%** after E6/E7 story completion
- PRs must stay above the current CI threshold

## License

By contributing, you agree that your contributions will be licensed under [Apache 2.0](LICENSE).
