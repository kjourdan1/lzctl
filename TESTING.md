# Testing Guide

## Prérequis

| Outil | Version | Usage |
|-------|---------|-------|
| Go | >= 1.24 | Compilation et tests |
| Terraform | >= 1.5 | Tests d'intégration |
| Azure CLI | >= 2.50 | Tests d'intégration Azure |

## Build

```bash
# Build le binaire
go build -o bin/lzctl .

# Vérifier
./bin/lzctl version
```

## Tests unitaires

```bash
# Tous les tests
go test ./...

# Avec couverture
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Avec sortie détaillée
go test -v ./...

# Un package spécifique
go test -v ./internal/config/...
go test -v ./internal/audit/...
go test -v ./internal/state/...
go test -v ./internal/doctor/...

# Un test spécifique
go test -v ./internal/config/ -run TestCrossValidate

# Avec race detection
go test -race ./...
```

## Tests par package

| Package | Fichier de test | Ce qu'il couvre |
|---------|----------------|----------------|
| `internal/config` | `crossvalidator_test.go`, `loader_test.go` | Validation schema, cross-checks CIDR/UUID |
| `internal/audit` | `audit_test.go`, `renderers_test.go` | Règles de conformité CAF, renderers MD/JSON |
| `internal/doctor` | `checks_test.go` | Vérification terraform, az, git, state backend |
| `internal/state` | `state_test.go` | List, snapshot, health check, manager |
| `internal/drift` | `drift_test.go` | Détection de drift |
| `internal/applier` | `applier_test.go` | Orchestration plan/apply |
| `internal/rollback` | `rollback_test.go` | Rollback multi-couche |
| `internal/validate` | `validate_test.go` | Validation schema JSON |
| `internal/template` | `engine_test.go`, `pipeline_updater_test.go` | Rendu templates |
| `internal/upgrade` | `registry_test.go`, `updater_test.go` | Upgrade de modules AVM |
| `internal/importer` | `hcl_generator_test.go` | Génération import blocks |
| `cmd/` | `cmd_test.go`, `rollback_test.go`, `local_ops_test.go` | Commandes CLI |

## Tests d'intégration

```bash
# Tests d'intégration (nécessitent un environnement Azure)
go test -v ./test/integration/...

# Tests de rendu de templates
go test -v ./test/integration/ -run TestTemplateRender

# Tests brownfield
go test -v ./test/integration/ -run TestBrownfield
```

## Smoke test local

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

# State health (nécessite Azure)
./bin/lzctl state health
```

## Makefile

```bash
make build        # Build le binaire
make test         # Tests unitaires
make test-verbose # Tests avec détails
make fmt          # Formater le code
make vet          # Analyse statique
make lint         # Linter (golangci-lint)
make cover        # Couverture de tests
make clean        # Nettoyer les artefacts
```
