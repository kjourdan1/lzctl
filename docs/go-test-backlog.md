# Go Test Backlog — lzctl

> Tests à exécuter quand Go sera disponible sur l'environnement.
> Généré le 2026-02-19.

---

## Commandes de test

```bash
# Tous les tests unitaires
go test ./...

# Tests unitaires par package
go test ./internal/importer/...
go test ./internal/wizard/...
go test ./internal/audit/...
go test ./internal/doctor/...
go test ./internal/config/...
go test ./internal/template/...
go test ./internal/drift/...
go test ./internal/rollback/...
go test ./internal/policy/...
go test ./internal/upgrade/...
go test ./cmd/...

# Tests d'intégration
go test ./test/integration/...

# Avec couverture
go test -cover ./...

# Avec verbose
go test -v ./...

# Build check (compilation sans exécution)
go build ./...

# Lint
golangci-lint run
```

---

## Tests par Sprint

### Sprint 9 — Import Command + Brownfield Integration

| Fichier de test | Package | Ce qui est testé | Priorité |
|----------------|---------|-----------------|----------|
| `internal/importer/hcl_generator_test.go` | importer | Import blocks supportés/non-supportés, resource blocks (VNet, RG), groupement par layer, local names | **Haute** |
| `internal/wizard/import_wizard_test.go` | wizard | 3 modes wizard (subscription, audit-report, resource-group), sélection multi-select | **Haute** |
| `cmd/import_test.go` | cmd | Import depuis audit report, filtres include/exclude, résolution mode/target dir | **Haute** |
| `test/integration/brownfield_test.go` | integration | Flow E2E audit → report JSON → import → validation fichiers générés | **Haute** |

### Sprints 1–8 — Tests existants à valider

| Fichier de test | Package | Priorité |
|----------------|---------|----------|
| `internal/importer/discovery_test.go` | importer | Haute |
| `internal/importer/resource_mapping_test.go` | importer | Haute |
| `internal/wizard/init_wizard_test.go` | wizard | Haute (modifié : ajout `MultiSelect` au mock) |
| `internal/config/loader_test.go` | config | Haute |
| `internal/config/validator_test.go` | config | Haute |
| `internal/config/crossvalidator_test.go` | config | Haute |
| `internal/doctor/checks_test.go` | doctor | Moyenne |
| `internal/template/engine_test.go` | template | Haute |
| `internal/template/helpers_test.go` | template | Moyenne |
| `internal/audit/renderers_test.go` | audit | Moyenne |
| `internal/drift/drift_test.go` | drift | Moyenne |
| `internal/rollback/rollback_test.go` | rollback | Moyenne |
| `internal/policy/policy_test.go` | policy | Moyenne |
| `internal/output/output_test.go` | output | Basse |
| `cmd/cmd_test.go` | cmd | Haute |
| `cmd/rollback_test.go` | cmd | Moyenne |
| `cmd/local_ops_test.go` | cmd | Moyenne |
| `test/integration/template_render_test.go` | integration | Haute |

### Sprint 10 — Day-2 Ops

| Fichier de test | Package | Story | Ce qui est testé | Priorité |
|----------------|---------|-------|-----------------|----------|
| `internal/upgrade/registry_test.go` | upgrade | E4-S3 | Comparaison de versions semver, ModuleRef.String() | **Haute** |
| `internal/upgrade/updater_test.go` | upgrade | E4-S3 | Scan de fichiers .tf, extraction de version, mise à jour de pins, ignore .terraform | **Haute** |
| `internal/config/saver_test.go` | config | E4-S1 | Round-trip Save/Load, AddLandingZone, détection de doublons | **Haute** |
| `internal/workload/workload_test.go` | workload | E4-S1 | Workload add, détection doublons, CIDR overlap, validation | **Haute** |
| `internal/template/pipeline_updater_test.go` | template | E4-S5 | Zone matrix JSON, pipeline re-rendering, dry-run | **Haute** |
| `cmd/workload_add_test.go` | cmd | E4-S1 | workload add/remove, dry-run JSON | **Moyenne** |

### Fichiers créés/modifiés Sprint 10

| Fichier | Type | Description |
|---------|------|-------------|
| `cmd/workload_add.go` | **Nouveau** | Commande `lzctl workload add` (wizard + flags) |
| `cmd/upgrade.go` | **Nouveau** | Commande `lzctl upgrade` (registry check + apply) |
| `cmd/status.go` | **Modifié** | Ajout --live, --json, métadonnées projet, git info |
| `cmd/drift.go` | **Modifié** | Ajout support --json |
| `internal/config/saver.go` | **Nouveau** | Save() et AddLandingZone() |
| `internal/upgrade/registry.go` | **Nouveau** | Client Terraform Registry API |
| `internal/upgrade/updater.go` | **Nouveau** | Scanner .tf + mise à jour version pins |
| `internal/workload/workload.go` | **Nouveau** | Logique workload add/adopt/list/remove |
| `internal/template/pipeline_updater.go` | **Nouveau** | PipelineUpdater, zone-matrix.json |
| `internal/template/engine.go` | **Modifié** | Ajout RenderZone(), RenderPipelines(), renderTemplate() |
| `templates/pipelines/github/drift.yml.tmpl` | **Modifié** | Ajout issue GitHub auto + scan des LZ |
| `templates/pipelines/azuredevops/drift.yml.tmpl` | **Modifié** | Ajout work item ADO + scan des LZ |

---

## Vérifications de build

```bash
# Compilation multi-plateforme (GoReleaser)
goreleaser build --snapshot --clean

# Ou manuellement
GOOS=linux GOARCH=amd64 go build -o lzctl-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o lzctl-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o lzctl-windows-amd64.exe
```

---

## Critères de validation

- [ ] `go build ./...` — compilation sans erreur
- [ ] `go test ./...` — tous les tests passent
- [ ] `go test -cover ./...` — couverture > 80%
- [ ] `golangci-lint run` — aucun warning
- [ ] `go vet ./...` — aucune erreur
