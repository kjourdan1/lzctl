# Go Test Backlog — lzctl

> Tests to run when Go is available in the environment.
> Generated 2026-02-19.

---

## Test Commands

```bash
# All unit tests
go test ./...

# Unit tests per package
go test ./internal/importer/...
go test ./internal/wizard/...
go test ./internal/audit/...
go test ./internal/doctor/...
go test ./internal/config/...
go test ./internal/template/...
go test ./internal/policy/...
go test ./internal/upgrade/...
go test ./cmd/...

# Integration tests
go test ./test/integration/...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...

# Build check (compile without running)
go build ./...

# Lint
golangci-lint run
```

---

## Tests by Sprint

### Sprint 9 — Import Command + Brownfield Integration

| Test File | Package | What Is Tested | Priority |
|-----------|---------|----------------|----------|
| `internal/importer/hcl_generator_test.go` | importer | Supported/unsupported import blocks, resource blocks (VNet, RG), grouping by layer, local names | **High** |
| `internal/wizard/import_wizard_test.go` | wizard | 3 wizard modes (subscription, audit-report, resource-group), multi-select | **High** |
| `cmd/import_test.go` | cmd | Import from audit report, include/exclude filters, mode/target dir resolution | **High** |
| `test/integration/brownfield_test.go` | integration | E2E flow audit → JSON report → import → generated file validation | **High** |

### Sprints 1–8 — Existing Tests to Validate

| Test File | Package | Priority |
|-----------|---------|----------|
| `internal/importer/discovery_test.go` | importer | High |
| `internal/importer/resource_mapping_test.go` | importer | High |
| `internal/wizard/init_wizard_test.go` | wizard | High (modified: added `MultiSelect` to mock) |
| `internal/config/loader_test.go` | config | High |
| `internal/config/validator_test.go` | config | High |
| `internal/config/crossvalidator_test.go` | config | High |
| `internal/doctor/checks_test.go` | doctor | Medium |
| `internal/template/engine_test.go` | template | High |
| `internal/template/helpers_test.go` | template | Medium |
| `internal/audit/renderers_test.go` | audit | Medium |
| `internal/policy/policy_test.go` | policy | Medium |
| `internal/output/output_test.go` | output | Low |
| `cmd/cmd_test.go` | cmd | High |
| `cmd/rollback_test.go` | cmd | Medium |
| `cmd/local_ops_test.go` | cmd | Medium |
| `test/integration/template_render_test.go` | integration | High |

### Sprint 10 — Day-2 Ops

| Test File | Package | Story | What Is Tested | Priority |
|-----------|---------|-------|----------------|----------|
| `internal/upgrade/registry_test.go` | upgrade | E4-S3 | Semver comparison, ModuleRef.String() | **High** |
| `internal/upgrade/updater_test.go` | upgrade | E4-S3 | .tf file scanning, version extraction, pin update, .terraform ignore | **High** |
| `internal/config/saver_test.go` | config | E4-S1 | Round-trip Save/Load, AddLandingZone, duplicate detection | **High** |
| `internal/workload/workload_test.go` | workload | E4-S1 | Workload add, duplicate detection, CIDR overlap, validation | **High** |
| `internal/template/pipeline_updater_test.go` | template | E4-S5 | Zone matrix JSON, pipeline re-rendering, dry-run | **High** |
| `cmd/workload_add_test.go` | cmd | E4-S1 | workload add/remove, dry-run JSON | **Medium** |

### Files Created/Modified in Sprint 10

| File | Type | Description |
|------|------|-------------|
| `cmd/workload_add.go` | **New** | `lzctl workload add` command (wizard + flags) |
| `cmd/upgrade.go` | **New** | `lzctl upgrade` command (registry check + apply) |
| `cmd/status.go` | **Modified** | Added --live, --json, project metadata, git info |
| `cmd/drift.go` | **Modified** | Added --json support |
| `internal/config/saver.go` | **New** | Save() and AddLandingZone() |
| `internal/upgrade/registry.go` | **New** | Terraform Registry API client |
| `internal/upgrade/updater.go` | **New** | .tf scanner + version pin updater |
| `internal/workload/workload.go` | **New** | Workload add/adopt/list/remove logic |
| `internal/template/pipeline_updater.go` | **New** | PipelineUpdater, zone-matrix.json |
| `internal/template/engine.go` | **Modified** | Added RenderZone(), RenderPipelines(), renderTemplate() |
| `templates/pipelines/github/drift.yml.tmpl` | **Modified** | Added auto GitHub issue + LZ scanning |
| `templates/pipelines/azuredevops/drift.yml.tmpl` | **Modified** | Added ADO work item + LZ scanning |

---

## Build Checks

```bash
# Cross-platform build (GoReleaser)
goreleaser build --snapshot --clean

# Or manually
GOOS=linux GOARCH=amd64 go build -o lzctl-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o lzctl-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o lzctl-windows-amd64.exe
```

---

## Validation Criteria

- [ ] `go build ./...` — compiles without errors
- [ ] `go test ./...` — all tests pass
- [ ] `go test -cover ./...` — coverage > 80%
- [ ] `golangci-lint run` — no warnings
- [ ] `go vet ./...` — no errors
