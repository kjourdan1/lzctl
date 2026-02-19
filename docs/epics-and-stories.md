# Epics & Stories — lzctl

> Version: 1.0
> Date: 2026-02-18
> Input: [PRD.md](./PRD.md), [architecture.md](./architecture.md)
> Author: Killian Jourdan

---

## Sprint Planning Overview

### Delivery Phases

| Phase | Epics | Est. Duration | Goal |
|-------|-------|--------------|------|
| **Phase 1** | E1 (Foundation) + E2 (Templates) | 6-8 weeks | `lzctl init` + `validate` + `plan/apply` — deployable greenfield repo |
| **Phase 2** | E3 (Brownfield) | 4-6 weeks | `lzctl audit` + `lzctl import` |
| **Phase 3** | E4 (Day-2 Ops) | 4-6 weeks | `workload add`, `drift`, `upgrade`, `status` |
| **Phase 4** | E5 (Community) | 2-3 weeks | Docs, examples, launch |

### Story Point Scale

| Points | Meaning | Typical Duration |
|--------|---------|-----------------|
| 1 | Trivial — boilerplate, copy-paste, config | < 2 hours |
| 2 | Small — single file, clear scope | 2-4 hours |
| 3 | Medium — multiple files, some logic | 4-8 hours |
| 5 | Large — complex logic, multiple packages | 1-2 days |
| 8 | XL — cross-cutting, integration-heavy | 2-3 days |
| 13 | Epic-level — should probably be split further | 3-5 days |

### Dependency Notation

`→ E1-S3` means "depends on story E1-S3 being complete"

---

## Epic 1 — CLI Foundation & Scaffolding

> **Goal:** Working CLI binary with `doctor`, `init`, `validate`, `plan`, `apply` commands.
> **Phase:** 1
> **Total points:** 67

---

### E1-S1: Go Project Scaffolding

**Points:** 3
**Dependencies:** None (first story)
**Priority:** Must

**Description:**
Initialize the Go module, Cobra root command, CI pipeline, and GoReleaser configuration. This is the skeleton that everything else builds on.

**Files to create:**
```
main.go
go.mod
go.sum
cmd/root.go
cmd/version.go
internal/output/logger.go
.github/workflows/ci.yml
.github/workflows/release.yml
.goreleaser.yml
.golangci.yml
Makefile
LICENSE                         (Apache 2.0)
README.md                      (minimal, will be expanded in E5)
CONTRIBUTING.md
CHANGELOG.md
.gitignore
```

**Acceptance Criteria:**
- [ ] `go build ./...` compiles successfully
- [ ] `go test ./...` passes (even if only 1 trivial test)
- [ ] `golangci-lint run` passes
- [ ] `lzctl` prints help text with available commands
- [ ] `lzctl version` prints `lzctl v0.1.0-dev (go1.22, <os>/<arch>)`
- [ ] GitHub Actions CI runs on PR: lint + test + build
- [ ] GoReleaser config builds for linux/macOS/windows × amd64/arm64
- [ ] Makefile has targets: `build`, `test`, `lint`, `install`

**Implementation notes:**
- Use `cmd/root.go` with `cobra.Command` as the root
- Embed version via `-ldflags "-X main.version=..."` in GoReleaser
- Logger uses `charmbracelet/log` with `NO_COLOR` env var support

---

### E1-S2: Output & UX Utilities

**Points:** 2
**Dependencies:** → E1-S1
**Priority:** Must

**Description:**
Shared output utilities used by all commands: styled logging, spinners, JSON output mode, error formatting.

**Files to create:**
```
internal/output/logger.go      (extend from S1)
internal/output/spinner.go
internal/output/json.go
internal/output/errors.go
internal/output/colors.go
```

**Acceptance Criteria:**
- [ ] `logger.Info("message")` prints styled output with emoji prefix
- [ ] `logger.Error("message")` prints red styled output
- [ ] Spinner starts/stops cleanly, handles interrupt (Ctrl+C)
- [ ] `--json` flag on root command sets global JSON output mode
- [ ] When JSON mode is active, all output goes through `output.JSON()` as structured data
- [ ] `NO_COLOR=1` disables all colors and emoji
- [ ] `--verbose` / `-v` flag enables debug-level logging
- [ ] Error formatting includes suggested fix when available

**Implementation notes:**
- Use `lipgloss` for styling, `charmbracelet/log` for structured logging
- Spinner uses a goroutine; `Stop()` must be safe to call multiple times
- JSON output struct: `{"status": "ok|error", "data": {...}, "error": "..."}`

---

### E1-S3: Doctor Command

**Points:** 5
**Dependencies:** → E1-S2
**Priority:** Must

**Description:**
Implement `lzctl doctor` that verifies all prerequisites.

**Files to create:**
```
cmd/doctor.go
internal/doctor/checks.go
internal/doctor/reporter.go
internal/doctor/checks_test.go
```

**Acceptance Criteria:**
- [ ] Checks `terraform` binary exists and version >= 1.5.0
- [ ] Checks `az` CLI exists and version >= 2.50.0
- [ ] Checks `git` exists and version >= 2.30.0
- [ ] Checks optional: `gh` CLI presence (info-level, not blocking)
- [ ] Checks `az account show` returns valid session; displays tenant ID, sub ID, user
- [ ] If not logged in, suggests `az login --tenant <id>`
- [ ] Checks Azure permissions: can list management groups (proxy for sufficient access)
- [ ] Checks resource providers registered: `Microsoft.Management`, `Microsoft.Authorization`, `Microsoft.Network`, `Microsoft.ManagedIdentity`
- [ ] Each check shows ✅ (pass), ❌ (fail), or ⚠️ (warning) with actionable fix
- [ ] Summary: "N issues found" or "All checks passed"
- [ ] Exit code 0 if all critical checks pass, 1 otherwise
- [ ] `--json` flag outputs structured check results
- [ ] Unit tests with mocked command execution for each check

**Implementation notes:**
- Use `os/exec` to run version commands; parse semver from output
- Permission check: `az account management-group list --top 1` — if it succeeds, user has read access
- Provider check: `az provider show -n Microsoft.Management --query registrationState -o tsv`

---

### E1-S4: Config Schema & Loader

**Points:** 5
**Dependencies:** → E1-S1
**Priority:** Must

**Description:**
Define the Go structs for `lzctl.yaml`, the JSON Schema, and the config loader/validator.

**Files to create:**
```
internal/config/schema.go          (Go structs — from architecture doc)
internal/config/loader.go          (load + parse YAML)
internal/config/validator.go       (JSON Schema validation)
internal/config/defaults.go        (default values for optional fields)
schemas/lzctl-v1.schema.json       (JSON Schema definition)
internal/config/loader_test.go
internal/config/validator_test.go
test/fixtures/configs/standard-hub-spoke.yaml
test/fixtures/configs/lite-no-connectivity.yaml
test/fixtures/configs/invalid-overlap.yaml
```

**Acceptance Criteria:**
- [ ] Go structs match the full schema from architecture doc section 5.1
- [ ] `config.Load("lzctl.yaml")` returns populated `LZConfig` struct
- [ ] Missing optional fields get default values (e.g., `mainBranch` → `"main"`, `retentionDays` → `90`)
- [ ] `config.Validate(cfg)` validates against JSON Schema
- [ ] Validation catches: missing required fields, invalid enum values, wrong types
- [ ] JSON Schema is embedded via `go:embed`
- [ ] Test fixtures cover: valid standard config, valid lite config, invalid config (missing tenant)
- [ ] Round-trip test: load → marshal → unmarshal → compare

**Implementation notes:**
- Use `gojsonschema` for validation
- Use `yaml.v3` for parsing (supports comments preservation for future edit features)
- Defaults are applied after parsing, before validation

---

### E1-S5: Interactive Wizard Framework

**Points:** 5
**Dependencies:** → E1-S2
**Priority:** Must

**Description:**
Build the interactive wizard for `lzctl init` using Charmbracelet `huh` (or `survey/v2` as fallback). Reusable framework for all wizard-based commands.

**Files to create:**
```
internal/wizard/wizard.go           (shared wizard utilities)
internal/wizard/init_wizard.go      (init-specific prompts)
internal/wizard/init_wizard_test.go
```

**Acceptance Criteria:**
- [ ] `InitWizard.Run()` collects all init parameters and returns an `InitConfig` struct
- [ ] Prompts in order: project name → tenant ID → CI/CD platform → MG model → connectivity model → primary region → secondary region → identity model → state backend strategy → bootstrap confirmation
- [ ] Connectivity sub-prompts (firewall SKU, VPN gateway, etc.) only appear if connectivity != "none"
- [ ] DNS sub-prompts only appear if connectivity is hub-spoke
- [ ] Tenant ID validates as UUID format before proceeding
- [ ] Region selection offers common Azure regions with autocomplete
- [ ] Each prompt has sensible default values
- [ ] Wizard can be cancelled at any point with Ctrl+C (clean exit, no partial state)
- [ ] `InitConfig` struct is convertible to `config.LZConfig` for downstream use
- [ ] Non-interactive mode (`--config`) skips wizard entirely and loads from file

**Implementation notes:**
- Charmbracelet `huh` provides form groups with validation
- Conditional prompts: use `huh.NewForm().WithAccessible(true)` for a11y
- Test with mocked stdin or by testing the config construction logic separately

---

### E1-S6: Template Engine Core

**Points:** 8
**Dependencies:** → E1-S4
**Priority:** Must

**Description:**
Build the template rendering engine that takes an `LZConfig` and produces a list of `RenderedFile` objects. Includes all helper functions.

**Files to create:**
```
internal/template/engine.go
internal/template/helpers.go
internal/template/writer.go
internal/template/engine_test.go
internal/template/helpers_test.go
templates/manifest/lzctl.yaml.tmpl
templates/shared/backend.tf.tmpl
templates/shared/backend.hcl.tmpl
templates/shared/providers.tf.tmpl
templates/shared/gitignore.tmpl
templates/shared/readme.md.tmpl
```

**Acceptance Criteria:**
- [ ] `engine.RenderAll(cfg)` returns `[]RenderedFile` with correct paths and content
- [ ] All templates are embedded via `go:embed` directive on `templates/` directory
- [ ] Template helpers work correctly:
  - `cafName "rg" "platform" "weu"` → `"rg-platform-weu"`
  - `regionShort "westeurope"` → `"weu"`
  - `cidrSubnet "10.0.0.0/16" 24 0` → `"10.0.0.0/24"`
  - `slugify "My Project"` → `"my-project"`
  - `storageAccName "contoso-platform-tfstate"` → `"contosoplattfstate"` (≤ 24 chars)
- [ ] `writer.WriteAll(files, targetDir)` writes files to disk, creating directories as needed
- [ ] `writer.WriteAll` with `dryRun=true` returns file list without writing
- [ ] All rendered `.tf` content is valid HCL syntax (tested by running `terraform fmt -check`)
- [ ] Rendered `lzctl.yaml` round-trips through `config.Load()` without error
- [ ] Header comment on all generated files: `# Generated by lzctl vX.Y.Z — safe to edit`

**Implementation notes:**
- Use `template.Must(template.New("").Funcs(funcMap).ParseFS(embeddedFS, pattern))` to parse all templates
- Writer creates parent directories with `os.MkdirAll`
- Test by rendering with fixture configs and validating output

---

### E1-S7: Platform Layer Templates — Management Groups

**Points:** 5
**Dependencies:** → E1-S6
**Priority:** Must

**Description:**
Create Terraform templates for the management groups layer (both CAF Standard and CAF Lite models).

**Files to create:**
```
templates/platform/management-groups/caf-standard/main.tf.tmpl
templates/platform/management-groups/caf-standard/variables.tf.tmpl
templates/platform/management-groups/caf-standard/terraform.tfvars.tmpl
templates/platform/management-groups/caf-lite/main.tf.tmpl
templates/platform/management-groups/caf-lite/variables.tf.tmpl
templates/platform/management-groups/caf-lite/terraform.tfvars.tmpl
test/fixtures/golden/caf-standard-mgmt-groups/       (golden file tests)
test/fixtures/golden/caf-lite-mgmt-groups/
```

**Acceptance Criteria:**
- [ ] CAF Standard template produces hierarchy: Tenant Root MG → Intermediate (project name) → Platform, Landing Zones, Decommissioned, Sandbox → Corp, Online (under LZ)
- [ ] CAF Lite template produces: Tenant Root MG → Intermediate → Platform, Landing Zones, Sandbox
- [ ] `disabled` list in config correctly omits specified MG nodes
- [ ] Uses `Azure/avm-ptn-alz/azurerm` module with pinned version
- [ ] Generated HCL passes `terraform validate` (with mock provider)
- [ ] Golden file tests: render template with fixture config, compare output to expected files

**Implementation notes:**
- AVM ALZ pattern module handles the hierarchy creation; template configures it
- Test with `terraform validate` requires a valid provider config; use `terraform init -backend=false`

---

### E1-S8: Platform Layer Templates — Connectivity

**Points:** 8
**Dependencies:** → E1-S6
**Priority:** Must

**Description:**
Create Terraform templates for the connectivity layer (Hub & Spoke with Firewall, Hub & Spoke with NVA, Virtual WAN, None).

**Files to create:**
```
templates/platform/connectivity/hub-spoke-fw/main.tf.tmpl
templates/platform/connectivity/hub-spoke-fw/variables.tf.tmpl
templates/platform/connectivity/hub-spoke-fw/terraform.tfvars.tmpl
templates/platform/connectivity/hub-spoke-nva/main.tf.tmpl
templates/platform/connectivity/hub-spoke-nva/variables.tf.tmpl
templates/platform/connectivity/hub-spoke-nva/terraform.tfvars.tmpl
templates/platform/connectivity/vwan/main.tf.tmpl
templates/platform/connectivity/vwan/variables.tf.tmpl
templates/platform/connectivity/vwan/terraform.tfvars.tmpl
test/fixtures/golden/hub-spoke-fw/
```

**Acceptance Criteria:**
- [ ] Hub & Spoke (Firewall) template creates: hub VNet, Azure Firewall (Standard or Premium), route table, subnets (AzureFirewallSubnet, GatewaySubnet if VPN enabled, AzureBastionSubnet optional)
- [ ] Hub & Spoke (NVA) template creates: hub VNet, placeholder for NVA, route table pointing to NVA IP
- [ ] vWAN template creates: Virtual WAN, Virtual Hub, Firewall in hub
- [ ] If `connectivity.type == "none"`, no connectivity files are generated
- [ ] VPN Gateway created only if `vpnGateway.enabled == true`
- [ ] ExpressRoute Gateway created only if `expressRouteGateway.enabled == true`
- [ ] DNS Private Resolver created only if `dns.privateResolver == true`
- [ ] Uses AVM modules: `avm-res-network-virtualnetwork`, `avm-res-network-azurefirewall`, `avm-res-network-virtualwan` (as available)
- [ ] All generated HCL passes `terraform fmt -check`
- [ ] Address space correctly applied from config

**Implementation notes:**
- Hub & Spoke is the most complex template; NVA and vWAN can be simpler for MVP
- vWAN template may use `avm-ptn-virtualwan` if the AVM pattern module is mature; otherwise native resources

---

### E1-S9: Platform Layer Templates — Management & Governance & Identity

**Points:** 5
**Dependencies:** → E1-S6
**Priority:** Must

**Description:**
Create Terraform templates for the remaining platform layers: management (Log Analytics, Defender), governance (policy assignments), and identity (CI/CD managed identity).

**Files to create:**
```
templates/platform/management/main.tf.tmpl
templates/platform/management/variables.tf.tmpl
templates/platform/management/terraform.tfvars.tmpl
templates/platform/governance/main.tf.tmpl
templates/platform/governance/variables.tf.tmpl
templates/platform/governance/terraform.tfvars.tmpl
templates/platform/governance/policies/caf-default.tf.tmpl
templates/platform/identity/main.tf.tmpl
templates/platform/identity/variables.tf.tmpl
templates/platform/identity/terraform.tfvars.tmpl
```

**Acceptance Criteria:**
- [ ] Management layer creates: Log Analytics workspace with configured retention, Automation Account (if enabled), Defender for Cloud plans
- [ ] Governance layer assigns: CAF default policy sets at the intermediate root MG, custom policy paths if configured
- [ ] Identity layer creates: User Assigned Managed Identity, federated credential for CI/CD platform
- [ ] Defender plans are configurable: only enable plans listed in config
- [ ] Policy assignments use AVM policy modules where available
- [ ] All generated HCL passes `terraform fmt -check`

---

### E1-S10: CI/CD Pipeline Templates — GitHub Actions

**Points:** 5
**Dependencies:** → E1-S6
**Priority:** Must

**Description:**
Create GitHub Actions workflow templates for validate-on-PR, deploy-on-merge, and scheduled drift detection.

**Files to create:**
```
templates/pipelines/github/validate.yml.tmpl
templates/pipelines/github/deploy.yml.tmpl
templates/pipelines/github/drift.yml.tmpl
```

**Acceptance Criteria:**
- [ ] Validate workflow triggers on PR to main (configurable branch)
- [ ] Validate workflow runs `terraform init` + `validate` + `plan` for each layer in order
- [ ] Validate workflow posts plan output as PR comment
- [ ] Deploy workflow triggers on push to main
- [ ] Deploy workflow runs `terraform init` + `apply -auto-approve` for each layer in order
- [ ] Both workflows use Workload Identity Federation (default) — `permissions: id-token: write`
- [ ] Layer matrix is dynamically populated from `lzctl.yaml` landing zones
- [ ] `max-parallel: 1` ensures sequential deployment
- [ ] Drift workflow runs on cron schedule (weekly default)
- [ ] Drift workflow creates GitHub Issue if drift detected
- [ ] All generated YAML is valid GitHub Actions syntax
- [ ] SP + Secret variant uses `${{ secrets.AZURE_CLIENT_SECRET }}` instead of WIF

**Implementation notes:**
- Template escaping: Go template delimiters conflict with GitHub Actions `${{ }}` — use backtick-quoted strings or alternate delimiters
- See architecture doc section 8.1 for the escaping pattern

---

### E1-S11: CI/CD Pipeline Templates — Azure DevOps

**Points:** 5
**Dependencies:** → E1-S6
**Priority:** Must

**Description:**
Create Azure DevOps pipeline templates equivalent to GitHub Actions.

**Files to create:**
```
templates/pipelines/azuredevops/validate.yml.tmpl
templates/pipelines/azuredevops/deploy.yml.tmpl
templates/pipelines/azuredevops/drift.yml.tmpl
```

**Acceptance Criteria:**
- [ ] Validate pipeline triggers on PR to main branch
- [ ] Uses `AzureCLI@2` task for authentication
- [ ] Uses pipeline stages with dependencies for layer ordering
- [ ] Plan output is published as pipeline artifact and posted as PR comment (via ADO REST API or extension)
- [ ] Deploy pipeline triggers on merge to main
- [ ] Deploy pipeline runs `terraform apply` per layer in dependency order
- [ ] Service connection uses Workload Identity Federation (default)
- [ ] Variable group reference for secrets when using SP + Secret
- [ ] Drift pipeline uses scheduled trigger (cron)
- [ ] All generated YAML is valid Azure DevOps pipeline syntax

**Implementation notes:**
- ADO uses `trigger`/`pr` instead of `on`, `pool` instead of `runs-on`, `steps` with `task` instead of `uses`
- WIF in ADO: use `AzureCLI@2` with `addSpnToEnvironment: true` and `useWorkloadIdentityFederation: true`

---

### E1-S12: Bootstrap Runner

**Points:** 5
**Dependencies:** → E1-S3 (doctor), E1-S4 (config)
**Priority:** Must

**Description:**
Implement the state backend bootstrap logic that creates Azure resources via `az` CLI.

**Files to create:**
```
internal/azure/cli.go
internal/azure/bootstrap.go
internal/azure/bootstrap_test.go
```

**Acceptance Criteria:**
- [ ] `AzureCLI` interface implemented with `Run()` and `RunJSON()` methods
- [ ] `Bootstrap(cfg)` creates in order:
  1. Resource group: `rg-<project>-tfstate-<regionShort>`
  2. Storage account: sanitized name ≤ 24 chars, TLS 1.2, versioning, soft-delete
  3. Blob container: `tfstate`
  4. User Assigned Managed Identity: `id-<project>-deploy`
  5. Role assignment: Owner at root MG scope
  6. Federated credential: configured for GitHub Actions or ADO (based on CICD config)
- [ ] Each step is idempotent (re-running doesn't fail if resource exists)
- [ ] Each step prints progress with spinner: `✅ Resource Group: rg-contoso-tfstate-weu`
- [ ] If a step fails, prints clear error and stops (no partial rollback — resources are cheap to clean up)
- [ ] Returns populated `StateBackend` and `IdentityConfig` to update the config
- [ ] Unit tests mock `AzureCLI` interface

**Implementation notes:**
- Storage account name: lowercase alphanumeric only, ≤ 24 chars. Use `storageAccName` helper.
- Federated credential for GitHub: issuer `https://token.actions.githubusercontent.com`, subject `repo:<org>/<repo>:ref:refs/heads/main`
- Federated credential for ADO: issuer `https://vstoken.dev.azure.com/<org-id>`, subject from service connection

---

### E1-S13: Init Command — Wire Everything Together

**Points:** 8
**Dependencies:** → E1-S5, E1-S6, E1-S7, E1-S8, E1-S9, E1-S10, E1-S11, E1-S12
**Priority:** Must

**Description:**
Wire the init command: wizard → config → template engine → bootstrap → file writer. This is the integration story.

**Files to create:**
```
cmd/init.go
cmd/init_test.go
```

**Acceptance Criteria:**
- [ ] `lzctl init` runs the wizard, generates all files, prints summary
- [ ] `lzctl init --config <file>` skips wizard, uses config file
- [ ] `lzctl init --dry-run` prints file list without writing
- [ ] `lzctl init` in existing lzctl project (has `lzctl.yaml`) warns and exits unless `--force`
- [ ] Bootstrap runs only if user confirms in wizard
- [ ] If bootstrap is skipped, `lzctl.yaml` has placeholder values for state backend
- [ ] File summary shows all generated files grouped by category
- [ ] Next steps printed: git init, add, commit, push
- [ ] Integration test: run init with fixture config → verify all expected files exist → verify `terraform validate` passes on each layer
- [ ] End-to-end test (manual/CI): init → push to GitHub → pipeline runs successfully

---

### E1-S14: Validate Command

**Points:** 5
**Dependencies:** → E1-S4, E1-S6
**Priority:** Must

**Description:**
Implement `lzctl validate` with multi-layer validation.

**Files to create:**
```
cmd/validate.go
internal/config/crossvalidator.go
internal/config/crossvalidator_test.go
```

**Acceptance Criteria:**
- [ ] Validates `lzctl.yaml` against JSON Schema (FR-4.1)
- [ ] Checks IP address space overlaps across all VNets and landing zones (FR-4.2)
- [ ] Verifies policy references resolve (FR-4.3)
- [ ] Verifies cross-layer references (FR-4.4)
- [ ] Runs `terraform validate` on each layer directory (FR-4.5)
- [ ] Checks subscription IDs are valid UUID format (FR-4.6)
- [ ] Warns on small address spaces (FR-4.7)
- [ ] Output grouped by severity: error, warning, info (FR-4.8)
- [ ] `--json` flag (FR-4.9)
- [ ] Exit code 0 only if zero errors (FR-4.10)
- [ ] No interactive prompts (FR-4.11)
- [ ] Tests with fixture configs: valid (passes), overlapping IPs (error), invalid policy ref (error), small CIDR (warning)

---

### E1-S15: Plan & Apply Commands

**Points:** 5
**Dependencies:** → E1-S4, E1-S12
**Priority:** Must

**Description:**
Implement `lzctl plan` and `lzctl apply` as orchestration wrappers.

**Files to create:**
```
cmd/plan.go
cmd/apply.go
internal/terraform/runner.go
internal/terraform/layer_order.go
internal/terraform/plan_parser.go
internal/terraform/runner_test.go
internal/terraform/layer_order_test.go
```

**Acceptance Criteria:**
- [ ] `lzctl plan` runs terraform plan on all layers in dependency order (FR-5.1)
- [ ] `lzctl plan <layer>` runs plan on a specific layer only (FR-5.2)
- [ ] `lzctl apply` runs terraform apply on all layers with auto-approve (FR-5.3)
- [ ] `lzctl apply <layer>` applies a specific layer only (FR-5.4)
- [ ] State key per layer: `platform-management-groups.tfstate`, etc. (FR-5.5)
- [ ] `--out <file>` saves plan output (FR-5.6)
- [ ] If any layer fails, execution stops with clear error (FR-5.7)
- [ ] `--parallelism` flag forwarded to terraform (FR-5.8)
- [ ] Layer dependency order: management-groups → identity → management → governance → connectivity → landing-zones/*
- [ ] Layer order is unit tested

---

**Epic 1 Total: 15 stories, 79 points**

---

## Epic 2 — Terraform Templates & Archetypes

> **Goal:** Landing zone archetype templates (Corp, Online, Sandbox) and naming integration.
> **Phase:** 1 (runs in parallel with late E1 stories)
> **Total points:** 18

---

### E2-S1: Landing Zone Archetype — Corp

**Points:** 5
**Dependencies:** → E1-S6
**Priority:** Must

**Description:**
Template for "Corp" archetype: internal applications, peered to hub, NSG defaults, resource group.

**Files to create:**
```
templates/landing-zones/corp/main.tf.tmpl
templates/landing-zones/corp/variables.tf.tmpl
templates/landing-zones/corp/terraform.tfvars.tmpl
test/fixtures/golden/lz-corp/
```

**Acceptance Criteria:**
- [ ] Creates: resource group, VNet with configurable address space, default subnets (snet-default, snet-private-endpoints)
- [ ] Creates VNet peering to hub (both directions) if `connected == true`
- [ ] Creates default NSG with deny-all-inbound + allow-vnet + allow-lb rules
- [ ] Creates route table with default route to Azure Firewall (if hub has firewall)
- [ ] Uses AVM modules: `avm-res-network-virtualnetwork`, `avm-res-network-networksecuritygroup`
- [ ] Tags applied from config
- [ ] Terraform backend key: `landing-zones-<name>.tfstate`
- [ ] Golden file test passes

---

### E2-S2: Landing Zone Archetype — Online

**Points:** 3
**Dependencies:** → E2-S1
**Priority:** Must

**Description:**
Template for "Online" archetype: internet-facing applications, peered to hub, but with different NSG rules (allow HTTPS inbound).

**Files to create:**
```
templates/landing-zones/online/main.tf.tmpl
templates/landing-zones/online/variables.tf.tmpl
templates/landing-zones/online/terraform.tfvars.tmpl
```

**Acceptance Criteria:**
- [ ] Same base as Corp but with: NSG allows HTTPS (443) inbound from Internet
- [ ] Optional: Application Gateway subnet pre-provisioned
- [ ] Peering to hub if connected
- [ ] Golden file test passes

---

### E2-S3: Landing Zone Archetype — Sandbox

**Points:** 2
**Dependencies:** → E2-S1
**Priority:** Should

**Description:**
Template for "Sandbox" archetype: isolated, no hub connectivity, relaxed policies.

**Files to create:**
```
templates/landing-zones/sandbox/main.tf.tmpl
templates/landing-zones/sandbox/variables.tf.tmpl
templates/landing-zones/sandbox/terraform.tfvars.tmpl
```

**Acceptance Criteria:**
- [ ] Creates: resource group, VNet (isolated — no peering)
- [ ] No route table (no forced tunneling)
- [ ] Minimal NSG (allow-all-outbound, deny-all-inbound from Internet)
- [ ] No hub dependency
- [ ] Golden file test passes

---

### E2-S4: CAF Naming Module Integration

**Points:** 3
**Dependencies:** → E1-S6
**Priority:** Must

**Description:**
Integrate CAF naming convention into all templates. Naming follows the pattern: `<resource-type>-<workload>-<environment>-<region>-<instance>`.

**Files to create:**
```
internal/template/naming.go
internal/template/naming_test.go
```

**Acceptance Criteria:**
- [ ] `cafName(resourceType, workload, region)` generates correct names
- [ ] Resource type abbreviations follow Microsoft CAF docs (rg, vnet, snet, nsg, fw, pip, rt, kv, st, id, log)
- [ ] Region short codes: westeurope→weu, northeurope→neu, eastus→eus, eastus2→eus2, etc. (complete list for common regions)
- [ ] Storage account names: lowercase alphanumeric ≤ 24 chars with deterministic truncation
- [ ] Overrides from `spec.naming.overrides` applied correctly
- [ ] All templates use `cafName` helper instead of hardcoded names
- [ ] Unit tests for edge cases: long names, special characters, truncation

---

### E2-S5: Template Integration Tests

**Points:** 5
**Dependencies:** → E1-S7, E1-S8, E1-S9, E2-S1, E2-S2, E2-S3
**Priority:** Must

**Description:**
Comprehensive integration tests that render all templates with fixture configs and validate the output.

**Files to create:**
```
test/integration/template_test.go
test/fixtures/golden/full-standard/       (complete rendered repo for CAF Standard + Hub-Spoke-FW)
test/fixtures/golden/full-lite/           (complete rendered repo for CAF Lite + No connectivity)
```

**Acceptance Criteria:**
- [ ] Golden file test: render with standard config → compare all files to golden directory
- [ ] Golden file test: render with lite config → compare all files to golden directory
- [ ] All rendered `.tf` files pass `terraform fmt -check`
- [ ] All rendered `.tf` files pass `terraform validate` (with `-backend=false` and mock providers)
- [ ] All rendered YAML files are valid syntax
- [ ] Rendered `lzctl.yaml` round-trips through config loader
- [ ] Test flag `-update` regenerates golden files when templates change

---

**Epic 2 Total: 5 stories, 18 points**

---

## Epic 3 — Brownfield Capabilities

> **Goal:** `lzctl audit` + `lzctl import` — assess and onboard existing Azure estates.
> **Phase:** 2
> **Total points:** 47

---

### E3-S1: Azure Tenant Scanner

**Points:** 8
**Dependencies:** → E1-S12 (azure/cli.go)
**Priority:** Must

**Description:**
Implement the Azure scanner that collects tenant inventory for the audit command.

**Files to create:**
```
internal/azure/scanner.go
internal/azure/management_groups.go
internal/azure/policies.go
internal/azure/networking.go
internal/azure/rbac.go
internal/azure/diagnostics.go
internal/azure/defender.go
internal/azure/scanner_test.go
test/fixtures/azure/management-groups.json
test/fixtures/azure/subscriptions.json
test/fixtures/azure/policies.json
test/fixtures/azure/vnet-list.json
test/fixtures/azure/role-assignments.json
```

**Acceptance Criteria:**
- [ ] `Scanner.Scan(scope)` returns populated `TenantSnapshot` struct
- [ ] Scans: management groups hierarchy, subscriptions + placement, policy assignments at each MG scope, RBAC role assignments (Owner, Contributor, UAA) at MG and sub scope, VNets and peerings per subscription, diagnostic settings presence, Defender for Cloud status per subscription
- [ ] Subscription-scoped queries run in parallel (bounded concurrency, max 5)
- [ ] Progress reported via spinner: "Scanning subscriptions... (15/47)"
- [ ] Scope filter (`--scope <mg-id>`) limits scanning to a subtree
- [ ] Completes in < 5 minutes for 100 subscriptions (tested with mock data)
- [ ] Unit tests with mocked `az` CLI responses (from test fixtures)
- [ ] Handles gracefully: empty subscriptions, inaccessible subscriptions (skip with warning), rate limiting (retry with backoff)

---

### E3-S2: Compliance Rules Engine

**Points:** 8
**Dependencies:** → E3-S1
**Priority:** Must

**Description:**
Implement the compliance rules engine with the initial 14 CAF rules from architecture doc.

**Files to create:**
```
internal/audit/compliance.go
internal/audit/rules/management_groups.go    (GOV-001, GOV-002, GOV-004)
internal/audit/rules/policies.go             (GOV-003)
internal/audit/rules/rbac.go                 (IDT-001, IDT-002)
internal/audit/rules/logging.go              (MGT-001, MGT-002)
internal/audit/rules/security.go             (MGT-003, SEC-001, SEC-002)
internal/audit/rules/connectivity.go         (NET-001, NET-002, NET-003)
internal/audit/scoring.go
internal/audit/rules/management_groups_test.go
internal/audit/rules/connectivity_test.go
internal/audit/scoring_test.go
```

**Acceptance Criteria:**
- [ ] `ComplianceEngine` loads all rules from a registry
- [ ] `Evaluate(snapshot)` runs all rules and returns `AuditReport`
- [ ] Each rule implements `ComplianceRule` interface (ID, Discipline, Evaluate)
- [ ] All 14 MVP rules implemented:
  - GOV-001: MG hierarchy matches CAF
  - GOV-002: Subscriptions in correct MGs
  - GOV-003: CAF default policies assigned
  - GOV-004: No subs in Tenant Root Group
  - IDT-001: No persistent Owner at high scopes
  - IDT-002: SPs use federated credentials
  - MGT-001: Log Analytics workspace exists
  - MGT-002: Diagnostic settings on subscriptions
  - MGT-003: Defender for Cloud enabled
  - NET-001: Hub VNet exists
  - NET-002: Hub-spoke peering established
  - NET-003: No overlapping address spaces
  - SEC-001: Storage accounts enforce TLS 1.2+
  - SEC-002: Key Vaults have soft delete
- [ ] Scoring: each rule weighted by severity; overall 0-100 score; per-discipline scores
- [ ] `AutoFixable` flag set correctly (e.g., GOV-001 is fixable by `lzctl init`, SEC-001 is not)
- [ ] Unit tests for each rule with mock snapshots (pass case + fail case)

---

### E3-S3: Audit Command & Report Generation

**Points:** 5
**Dependencies:** → E3-S1, E3-S2
**Priority:** Must

**Description:**
Wire the audit command: scanner → compliance engine → report renderer.

**Files to create:**
```
cmd/audit.go
internal/audit/report.go
internal/audit/markdown_renderer.go
internal/audit/json_renderer.go
templates/audit/report.md.tmpl
cmd/audit_test.go
```

**Acceptance Criteria:**
- [ ] `lzctl audit` scans tenant and prints Markdown report to stdout
- [ ] `--output <path>` writes report to file
- [ ] `--json` outputs JSON format
- [ ] `--scope <mg-id>` limits scan scope
- [ ] Markdown report includes: executive summary (score + critical count), per-discipline sections, each finding with severity/current/expected/remediation, summary table
- [ ] JSON report matches `AuditReport` struct exactly
- [ ] Report template uses Go templates (embedded)
- [ ] Summary line printed: "CAF Alignment Score: 45/100 — 3 critical, 7 high, 12 medium findings"
- [ ] Exit code 0 (audit always succeeds; findings are informational)
- [ ] Integration test with mock scanner data

---

### E3-S4: Resource Discovery for Import

**Points:** 5
**Dependencies:** → E3-S1
**Priority:** Must

**Description:**
Discover importable resources from an existing tenant and map them to Terraform types.

**Files to create:**
```
internal/importer/discovery.go
internal/importer/resource_mapping.go
internal/importer/discovery_test.go
internal/importer/resource_mapping_test.go
```

**Acceptance Criteria:**
- [ ] `Discover(scope)` returns `[]ImportableResource` with Azure resource ID, type, name, and mapped Terraform type
- [ ] Resource mapping covers MVP types: resource groups, VNets, subnets, NSGs, route tables, key vaults, storage accounts, managed identities, policy assignments
- [ ] Resources not in mapping are flagged as "unsupported — manual import required"
- [ ] Filtering: `--subscription`, `--resource-group`, `--include <type>`, `--exclude <type>`
- [ ] Discovery uses `az resource list` per subscription
- [ ] Unit tests with mock resource lists

---

### E3-S5: HCL & Import Block Generator

**Points:** 8
**Dependencies:** → E3-S4, E1-S6
**Priority:** Must

**Description:**
Generate Terraform `import` blocks and corresponding HCL resource configuration for discovered resources.

**Files to create:**
```
internal/importer/hcl_generator.go
internal/importer/import_block.go
internal/importer/hcl_generator_test.go
internal/importer/import_block_test.go
```

**Acceptance Criteria:**
- [ ] For each importable resource, generates:
  - `import { to = <terraform_address> id = "<azure_resource_id>" }` (Terraform 1.5+ syntax)
  - Corresponding `resource` or `module` block with attributes populated from Azure API
- [ ] AVM modules used where a mapping exists; native `azurerm_*` resources for simpler types
- [ ] Generated HCL is syntactically valid (`terraform fmt` passes)
- [ ] Unsupported resources generate `# TODO: manual import required for <type> <name>` comments
- [ ] Import blocks grouped by layer (management-groups, connectivity, etc.) when possible
- [ ] Unit tests: generate HCL for a VNet → verify output matches expected template

---

### E3-S6: Import Command

**Points:** 8
**Dependencies:** → E3-S4, E3-S5
**Priority:** Must

**Description:**
Wire the import command: discovery → selection → generation → file writing.

**Files to create:**
```
cmd/import.go
internal/wizard/import_wizard.go
cmd/import_test.go
```

**Acceptance Criteria:**
- [ ] `lzctl import --from audit-report.json` reads audit report and imports auto-fixable resources
- [ ] `lzctl import --subscription <id>` discovers and imports from a specific subscription
- [ ] `lzctl import --resource-group <name>` imports from a specific RG
- [ ] Interactive mode: checklist of discovered resources, user selects which to import
- [ ] `--include <types>` / `--exclude <types>` for non-interactive filtering
- [ ] `--dry-run` shows what would be generated without writing files
- [ ] Generated files placed in `imports/` directory (or `--layer <layer>` to target specific layer)
- [ ] After generation, prints: "Next step: run `terraform plan` to verify zero-diff"
- [ ] Warning if imported resources conflict with existing TF-managed resources
- [ ] Integration test with mock discovery data

---

### E3-S7: Brownfield Integration Test

**Points:** 5
**Dependencies:** → E3-S3, E3-S6
**Priority:** Must

**Description:**
End-to-end integration test for the brownfield workflow: audit → import.

**Files to create:**
```
test/integration/brownfield_test.go
test/fixtures/azure/full-tenant-snapshot.json
```

**Acceptance Criteria:**
- [ ] Test creates a mock tenant snapshot with known gaps
- [ ] `audit` produces expected findings and score
- [ ] `import --from audit-report.json` generates valid import blocks
- [ ] Generated import blocks pass `terraform validate`
- [ ] Full flow: audit → import → validate passes without error

---

**Epic 3 Total: 7 stories, 47 points**

---

## Epic 4 — Day-2 Operations

> **Goal:** `workload add`, `drift`, `upgrade`, `status` for ongoing management.
> **Phase:** 3
> **Total points:** 25

---

### E4-S1: Add Zone Command

**Points:** 5
**Dependencies:** → E1-S5 (wizard), E1-S6 (template), E2-S1 (archetypes)
**Priority:** Must

**Description:**
Implement `lzctl workload add` interactive command.

**Files to create:**
```
cmd/workload_add.go
cmd/workload_helpers.go
internal/workload/workload.go
```

**Acceptance Criteria:**
- [ ] Interactive wizard collects: zone name, archetype (corp/online/sandbox), subscription ID, address space, hub connectivity
- [ ] `--config <file>` for non-interactive use
- [ ] Generates `landing-zones/<name>/` directory with main.tf, variables.tf, tfvars
- [ ] Updates `lzctl.yaml` with new entry in `spec.landingZones[]`
- [ ] Auto-runs `lzctl validate` after generation
- [ ] Blocks if IP overlap detected (unless `--force`)
- [ ] Updates CI/CD pipeline layer matrix (adds new landing zone to deploy pipeline)
- [ ] Prints next steps: commit, push, open PR

---

### E4-S2: Drift Detection Command

**Points:** 5
**Dependencies:** → E1-S15 (terraform runner)
**Priority:** Should

**Description:**
Implement `lzctl drift` that detects configuration drift.

**Files to create:**
```
cmd/drift.go
internal/drift/detector.go
internal/drift/reporter.go
internal/drift/detector_test.go
```

**Acceptance Criteria:**
- [ ] Runs `terraform plan` on each layer and parses for changes
- [ ] Summary per layer: ✅ no drift, ⚠️ N changes detected
- [ ] Classifies changes: add (created outside TF), change (modified outside TF), destroy (deleted outside TF)
- [ ] `--layer <layer>` checks specific layer only
- [ ] `--json` for CI integration
- [ ] Exit code non-zero if drift detected
- [ ] Unit tests with mock terraform plan output

---

### E4-S3: Upgrade Command

**Points:** 5
**Dependencies:** → E1-S4 (config)
**Priority:** Should

**Description:**
Implement `lzctl upgrade` to check and update AVM module versions.

**Files to create:**
```
cmd/upgrade.go
internal/upgrade/registry.go
internal/upgrade/updater.go
internal/upgrade/changelog.go
internal/upgrade/registry_test.go
internal/upgrade/updater_test.go
```

**Acceptance Criteria:**
- [ ] Queries Terraform registry API for latest versions of all AVM modules in the repo
- [ ] Displays table: module, current version, latest version, bump type (major/minor/patch)
- [ ] `--apply` updates version references in `.tf` files
- [ ] Major bumps blocked unless `--allow-major`
- [ ] `--dry-run` shows changes without applying
- [ ] After update, suggests running `lzctl validate` and `lzctl plan`
- [ ] Handles network errors gracefully (registry unreachable)
- [ ] Unit tests with mock registry responses

**Implementation notes:**
- Terraform registry API: `GET https://registry.terraform.io/v1/modules/<namespace>/<name>/<provider>/versions`
- Parse module references from `.tf` files with regex: `source = "Azure/<module>/azurerm"` + `version = "<semver>"`

---

### E4-S4: Status Command

**Points:** 3
**Dependencies:** → E1-S4 (config)
**Priority:** Could

**Description:**
Implement `lzctl status` for quick landing zone overview.

**Files to create:**
```
cmd/status.go
cmd/status_test.go
```

**Acceptance Criteria:**
- [ ] Displays: project name, tenant ID, primary region, MG model, connectivity type, number of layers, number of landing zones (with names), CI/CD platform, last git commit (from `git log`)
- [ ] `--live` queries Azure to verify resources exist
- [ ] `--json` for structured output
- [ ] Reads from `lzctl.yaml` and local git — no Azure calls by default
- [ ] Graceful handling when not in a git repo or no commits yet

---

### E4-S5: Pipeline Matrix Auto-Update

**Points:** 3
**Dependencies:** → E4-S1
**Priority:** Should

**Description:**
When `workload add` creates a new landing zone, the CI/CD pipeline matrix must be updated to include the new layer.

**Files to create:**
```
internal/template/pipeline_updater.go
internal/template/pipeline_updater_test.go
```

**Acceptance Criteria:**
- [ ] After `workload add`, the deploy and validate pipeline files are re-rendered with updated layer matrix
- [ ] For GitHub Actions: new entry in `matrix.layer` array
- [ ] For ADO: new stage in pipeline
- [ ] Existing pipeline customizations outside the matrix block are preserved
- [ ] If pipeline file has been manually modified beyond the matrix, warn user to update manually
- [ ] Unit test: add zone → verify pipeline YAML contains new layer

**Implementation notes:**
- Simplest approach: re-render the entire pipeline file from template (safe if user hasn't customized it)
- Detect customization: compare pipeline file against what template would generate; if different, warn

---

### E4-S6: Drift Pipeline Template Enhancement

**Points:** 2
**Dependencies:** → E1-S10, E4-S2
**Priority:** Should

**Description:**
Enhance the drift detection pipeline template to create issues/alerts when drift is found.

**Files to create/modify:**
```
templates/pipelines/github/drift.yml.tmpl     (enhance)
templates/pipelines/azuredevops/drift.yml.tmpl (enhance)
```

**Acceptance Criteria:**
- [ ] GitHub: drift detected → creates GitHub Issue with drift details, assigns label `drift-detected`
- [ ] ADO: drift detected → creates ADO Work Item or sends notification
- [ ] Cron schedule configurable in `lzctl.yaml` (default: weekly Sunday night)
- [ ] Pipeline passes if no drift, fails if drift detected (visible in CI dashboard)

---

**Epic 4 Total: 6 stories, 23 points**

---

## Epic 5 — Documentation & Community Launch

> **Goal:** Professional documentation, examples, and community launch.
> **Phase:** 4
> **Total points:** 18

---

### E5-S1: README & Quickstart

**Points:** 3
**Dependencies:** → E1-S13 (working init)
**Priority:** Must

**Description:**
Comprehensive README with install, quickstart, feature overview, and architecture diagram.

**Files to create:**
```
README.md                    (full rewrite)
docs/architecture-diagram.png (or mermaid in README)
```

**Acceptance Criteria:**
- [ ] Badges: CI status, latest release, license, Go version
- [ ] One-liner description + logo (or ASCII art)
- [ ] Install section: Homebrew, binary download, from source
- [ ] 5-minute quickstart: install → doctor → init → push → deployed
- [ ] Feature overview with command table
- [ ] Architecture diagram (mermaid or PNG)
- [ ] Comparison table vs. alternatives
- [ ] Contributing link
- [ ] License section

---

### E5-S2: Per-Command Documentation

**Points:** 3
**Dependencies:** → E1-S13, E3-S6, E4-S1
**Priority:** Must

**Description:**
Reference documentation for each command.

**Files to create:**
```
docs/commands/doctor.md
docs/commands/init.md
docs/commands/validate.md
docs/commands/plan.md
docs/commands/apply.md
docs/commands/workload.md
docs/commands/audit.md
docs/commands/import.md
docs/commands/drift.md
docs/commands/upgrade.md
docs/commands/status.md
docs/commands/README.md       (index)
```

**Acceptance Criteria:**
- [ ] Each doc includes: synopsis, description, flags/options, examples, related commands
- [ ] Examples are copy-pasteable
- [ ] Cross-references between related commands (e.g., audit → import)

---

### E5-S3: Example Configurations

**Points:** 3
**Dependencies:** → E1-S13, E3-S6
**Priority:** Should

**Description:**
Ready-to-use example configurations for common scenarios.

**Files to create:**
```
docs/examples/greenfield-standard/lzctl-config.yaml
docs/examples/greenfield-standard/README.md
docs/examples/greenfield-lite/lzctl-config.yaml
docs/examples/greenfield-lite/README.md
docs/examples/brownfield/README.md
```

**Acceptance Criteria:**
- [ ] Standard example: CAF Standard + Hub-Spoke-FW + GitHub Actions + WIF
- [ ] Lite example: CAF Lite + No connectivity + Azure DevOps
- [ ] Brownfield example: walkthrough of audit → import workflow
- [ ] Each example has a README explaining the scenario and how to use it
- [ ] Configs are valid and pass `lzctl validate`

---

### E5-S4: Contributing Guide & Developer Setup

**Points:** 2
**Dependencies:** → E1-S1
**Priority:** Must

**Description:**
Guide for contributors: dev setup, architecture overview, PR process, coding standards.

**Files to create:**
```
CONTRIBUTING.md              (full rewrite)
docs/development.md
```

**Acceptance Criteria:**
- [ ] Prerequisites: Go 1.22+, golangci-lint, terraform (for tests)
- [ ] Clone → make build → make test → make lint workflow
- [ ] Architecture overview pointing to architecture.md
- [ ] Coding standards summary
- [ ] How to add a new command, a new template, a new compliance rule
- [ ] PR process: conventional commits, CI must pass, 1 review required

---

### E5-S5: Demo Recording

**Points:** 3
**Dependencies:** → E1-S13
**Priority:** Should

**Description:**
Terminal recording (asciinema or GIF) showing the full greenfield workflow.

**Files to create:**
```
docs/demo/demo.sh              (scripted demo)
docs/demo/README.md
```

**Acceptance Criteria:**
- [ ] Recording shows: install → doctor → init (with wizard) → file listing → push → pipeline success
- [ ] Under 3 minutes
- [ ] Embedded in README (GIF or asciinema link)
- [ ] Optional: second recording for brownfield workflow (audit → import)

---

### E5-S6: Launch Content

**Points:** 4
**Dependencies:** → E5-S1, E5-S5
**Priority:** Should

**Description:**
Blog post and LinkedIn article for the public launch.

**Files to create:**
```
docs/blog/launch-post.md
```

**Acceptance Criteria:**
- [ ] Blog post: problem statement, solution overview, demo GIF, call to action (star + try it)
- [ ] LinkedIn version: shorter, more personal, link to GitHub
- [ ] Technical enough to be credible, accessible enough for non-experts
- [ ] Includes comparison to existing tools (why lzctl is different)
- [ ] Draft reviewed before publishing

---

**Epic 5 Total: 6 stories, 18 points**

---

## Summary

### All Epics

| Epic | Stories | Points | Phase |
|------|---------|--------|-------|
| E1 — CLI Foundation & Scaffolding | 15 | 79 | 1 |
| E2 — Templates & Archetypes | 5 | 18 | 1 |
| E3 — Brownfield Capabilities | 7 | 47 | 2 |
| E4 — Day-2 Operations | 6 | 23 | 3 |
| E5 — Documentation & Community | 6 | 18 | 4 |
| **Total** | **39** | **185** | |

### Critical Path

```
E1-S1 (scaffolding)
  → E1-S2 (output utils)
    → E1-S3 (doctor)
  → E1-S4 (config schema)
    → E1-S5 (wizard)
    → E1-S6 (template engine)
      → E1-S7 (MG templates)
      → E1-S8 (connectivity templates)
      → E1-S9 (mgmt/gov/id templates)
      → E1-S10 (GitHub pipelines)
      → E1-S11 (ADO pipelines)
  → E1-S12 (bootstrap)
    → E1-S13 ★ (init command — integration)
      → E1-S14 (validate)
      → E1-S15 (plan/apply)
        → Phase 1 DONE ✅

E3-S1 (scanner) → E3-S2 (rules) → E3-S3 (audit command)
E3-S1 → E3-S4 (discovery) → E3-S5 (HCL gen) → E3-S6 (import command)
  → E3-S7 (integration test)
    → Phase 2 DONE ✅

E4-S1 (workload add) → E4-S5 (pipeline update)
E4-S2 (drift) → E4-S6 (drift pipeline)
E4-S3 (upgrade)
E4-S4 (status)
  → Phase 3 DONE ✅

E5-S1 through E5-S6 (parallel, most depend only on working CLI)
  → Phase 4 DONE ✅ → LAUNCH 🚀
```

### Parallelization Opportunities

| Parallel Track A | Parallel Track B | Notes |
|-----------------|-----------------|-------|
| E1-S3 (doctor) | E1-S4 (config schema) | Both depend only on E1-S1/S2 |
| E1-S7 (MG templates) | E1-S8 (connectivity templates) | Both depend only on E1-S6 |
| E1-S10 (GitHub pipelines) | E1-S11 (ADO pipelines) | Independent CI/CD platforms |
| E2-S1/S2/S3 (archetypes) | E1-S14/S15 (validate/plan) | Different concerns |
| E3-S3 (audit cmd) | E3-S5 (HCL gen) | Different brownfield flows |
| E4-S1 (workload add) | E4-S2 (drift) | Independent day-2 features |
| E5-* (all docs) | Any E4 story | Docs can be written in parallel |

### Suggested Sprint Plan (2-week sprints)

| Sprint | Stories | Points | Milestone |
|--------|---------|--------|-----------|
| Sprint 1 | E1-S1, E1-S2, E1-S3, E1-S4 | 15 | CLI skeleton + doctor + config |
| Sprint 2 | E1-S5, E1-S6, E2-S4 | 16 | Wizard + template engine + naming |
| Sprint 3 | E1-S7, E1-S8, E1-S9 | 18 | All platform layer templates |
| Sprint 4 | E1-S10, E1-S11, E1-S12 | 15 | Pipelines + bootstrap |
| Sprint 5 | E1-S13, E1-S14, E1-S15 | 18 | ★ Init command + validate + plan/apply → Phase 1 MVP |
| Sprint 6 | E2-S1, E2-S2, E2-S3, E2-S5 | 15 | Landing zone archetypes + integration tests |
| Sprint 7 | E3-S1, E3-S2 | 16 | Azure scanner + compliance rules |
| Sprint 8 | E3-S3, E3-S4, E3-S5 | 18 | Audit command + import prep |
| Sprint 9 | E3-S6, E3-S7 | 13 | Import command + brownfield integration → Phase 2 MVP |
| Sprint 10 | E4-S1, E4-S2, E4-S3, E4-S4, E4-S5, E4-S6 | 23 | All day-2 ops → Phase 3 |
| Sprint 11 | E5-S1, E5-S2, E5-S3, E5-S4, E5-S5, E5-S6 | 18 | Docs + launch → Phase 4 🚀 |

---

*All stories are ready for implementation. Each story has clear file paths, acceptance criteria, and dependencies. Start with Sprint 1 (E1-S1) and follow the critical path.*

---

## Epic 6 — Quality & Reliability Hardening

> **Goal:** Stabiliser le pipeline CI/CD, fiabiliser GoReleaser, augmenter la couverture de tests sur les commandes métier et imposer des seuils de qualité.
> **Phase:** Transverse (peut être traité en parallèle de tout autre Epic)
> **Total points:** 38

---

### E6-S1 : Épingler golangci-lint et corriger le workflow CI

**Points:** 2
**Dependencies:** → E1-S1
**Priority:** Must

**Description:**
Le workflow CI utilise `version: latest` pour `golangci-lint-action`, ce qui casse silencieusement à chaque nouvelle release de linter (nouvelles règles non anticipées). Il faut épingler la version et ajouter un fichier de configuration partagée.

**Fichiers à créer/modifier :**
```
.github/workflows/ci.yml        (épingler golangci-lint à v1.64.x)
.golangci.yml                   (créer — règles communes, timeouts)
```

**Critères d'acceptation :**
- [ ] `golangci-lint-action` épinglee à `v6` avec `version: v1.64.x` (semver majeur stable)
- [ ] `.golangci.yml` actif avec au minimum : `gofmt`, `govet`, `errcheck`, `staticcheck`, `unused`, `exhaustive`
- [ ] Timeout lint global : 5 minutes
- [ ] `make lint` local donne le même résultat que CI
- [ ] PR avec violation lint rejeted (exit code != 0)
- [ ] Aucune fausse positive sur le code actuel (`go vet ./...` + lint passent)

**Notes d'implémentation :**
- `golangci/golangci-lint-action@v6` + `version: v1.64.5` (ou la dernière patch)
- `.golangci.yml` : `linters-settings.govet.enable-all: false` pour ne pas sur-activer

---

### E6-S2 : Valider `.goreleaser.yml` en CI (goreleaser check)

**Points:** 2
**Dependencies:** → E1-S1
**Priority:** Must

**Description:**
Le fichier `.goreleaser.yml` n'est jamais validé avant le push d'un tag. Un erreur de syntaxe bloque le release job. Ajouter une étape `goreleaser check` dans le CI pour valider la config à chaque PR.

**Fichiers à modifier :**
```
.github/workflows/ci.yml        (ajouter job goreleaser-check)
.goreleaser.yml                 (corriger {{ .ModulePath }} → hardcoded)
```

**Critères d'acceptation :**
- [ ] Nouveau job `goreleaser-check` dans `ci.yml`, déclenché sur PR + push main
- [ ] Le job exécute `goreleaser check --clean` (lint config sans build)
- [ ] `.goreleaser.yml` : `{{ .ModulePath }}` remplacé par `github.com/kjourdan1/lzctl` hardcodé (la variable n'existe pas en v2)
- [ ] Le job passe sur le code actuel
- [ ] En cas d'erreur `.goreleaser.yml`, le CI échoue visiblement avant le tag

**Notes d'implémentation :**
- `goreleaser/goreleaser-action@v6` + `args: check --clean`
- Ne nécessite pas `GITHUB_TOKEN`

---

### E6-S3 : Couverture de tests — commandes `plan`, `apply`, `validate`

**Points:** 8
**Dependencies:** → E1-S14, E1-S15
**Priority:** Must

**Description:**
Les commandes `plan`, `apply` et `validate` n'ont que des tests `--help`. Dans l'architecture de lzctl, **les pipelines générés appellent Terraform directement** — lzctl n'est pas le runtime d'exécution Terraform. Les tests doivent donc valider ce que lzctl *produit* (le repo Terraform généré, les pipelines, le routage des backends) plutôt que de mocker l'exécution Terraform elle-même.

**Fichiers à créer :**
```
cmd/plan_test.go
cmd/apply_test.go
cmd/validate_test.go
internal/planverify/planverify_test.go   (si pas déjà couvert)
```

**Critères d'acceptation :**

*`validate` — vérification de la config et du repo généré :*
- [ ] `lzctl validate` sur un repo valide → exit 0, affiche "Validation passed"
- [ ] `lzctl validate` sans `lzctl.yaml` → exit 1, message d'erreur lisible
- [ ] `lzctl validate --json` → sortie JSON `{valid: true, errors: []}`
- [ ] `lzctl validate` avec config invalide (champ requis manquant) → exit 1, liste les erreurs
- [ ] `lzctl validate` vérifie que chaque layer déclarée dans `lzctl.yaml` a son répertoire `platform/<layer>/` avec au minimum `main.tf` et `backend.hcl`
- [ ] `lzctl validate` vérifie que les clés de backend (`key = "<layer>.tfstate"`) sont uniques par layer

*`plan` — validation du repo généré, pas de l'exécution Terraform :*
- [ ] `lzctl plan --dry-run` → affiche l'ordre d'exécution des layers (CAF dependency order) sans appeler terraform
- [ ] `lzctl plan --layer connectivity` → cible uniquement connectivity ; le message de sortie liste uniquement cette layer
- [ ] `lzctl plan` sur un repo généré valide → fichiers `platform/<layer>/` trouvés dans le bon ordre (management-groups → identity → management → governance → connectivity)
- [ ] `lzctl plan` sans repo initialisé → exit 1, message "run lzctl init first"

*`apply` — validation de la séquence générée et des pipelines :*
- [ ] `lzctl apply --dry-run` → affiche la séquence d'exécution par layer avec `max-parallel: 1` respecté (single layer at a time)
- [ ] `lzctl apply` sans `--auto-approve` → demande confirmation ; input "no" → annulation propre (exit 0, message "Apply cancelled")
- [ ] `lzctl apply --layer management-groups` → sortie indique uniquement la layer management-groups
- [ ] Fichier pipeline généré par `lzctl init` contient bien `max-parallel: 1` pour le job de déploiement
- [ ] Chaque layer du pipeline généré référence le bon `backend.hcl` (ex: `connectivity.tfstate`, pas `management-groups.tfstate`)

**Notes d'implémentation :**
- Tous ces tests s'exécutent dans un `t.TempDir()` avec un repo généré par `lzctl init --tenant-id <uuid> --dry-run=false`
- Aucun mock Terraform nécessaire : les assertions portent sur les *fichiers générés*, pas sur la sortie d'une commande terraform

---

### E6-S4 : Couverture de tests — `drift`, `rollback`, `history`

**Points:** 8
**Dependencies:** → E4-S2, rollback existant
**Priority:** Should

**Description:**
Les commandes `drift`, `rollback` et `history` ont peu ou pas de tests sur leur logique métier. Conformément à l'architecture (les pipelines générés appellent Terraform directement), les tests valident la *logique de parsing et de génération de rapport* de lzctl à partir d'une sortie Terraform fixturée — pas d'exécution Terraform réelle. Les tests d'intégration Azure réels (état live) restent hors PR (nightly/manual, cf. E6-S9).

**Fichiers à créer/compléter :**
```
cmd/drift_test.go                        (compléter au-delà du --help)
cmd/rollback_test.go                     (compléter les cas d'erreur)
cmd/history_test.go
internal/drift/detector_test.go
test/fixtures/terraform/plan-no-changes.json   (fixture)
test/fixtures/terraform/plan-with-drift.json   (fixture : 2 add, 1 change)
```

**Critères d'acceptation :**

*`drift` — parsing de la sortie terraform plan fixturée :*
- [ ] Fixture `plan-no-changes.json` (exit 0) → rapport "No drift detected across N layers", exit 0
- [ ] Fixture `plan-with-drift.json` (exit 2) → rapport liste les ressources modifiées, exit non nul si `--fail-on-drift`
- [ ] `--json` → JSON structuré `{layers: [{name, status, changes: [{action, resource}]}]}`
- [ ] `--layer <name>` → le rapport ne porte que sur la layer nommée
- [ ] Pipeline drift généré par `lzctl init` contient bien `lzctl drift --json` (pas `terraform plan` brut) pour la détection centralisée

*`rollback` — logique d'identification du snapshot précédent :*
- [ ] Rollback réussi vers snapshot précédent → exit 0, affiche la liste des opérations à effectuer
- [ ] Rollback sur layer inexistante → erreur lisible, exit 1
- [ ] `--dry-run` → liste les fichiers state qui seraient restaurés, sans modification
- [ ] Absence de snapshot → message d'erreur clair "No previous state found for layer <name>"
- [ ] Les fichiers de snapshot utilisent la convention de nommage `<layer>-<timestamp>.tfstate.bak` (cohérence avec template généré)

*`history` — lecture du log d'audit local :*
- [ ] Log d'audit vide (`~/.lzctl/audit.log` absent ou vide) → message "No audit history found"
- [ ] Log non vide → affiche les N dernières entrées : commande, timestamp, exit code
- [ ] `--json` → sortie JSON de la liste
- [ ] `--limit N` → limite l'affichage aux N entrées les plus récentes

**Notes d'implémentation :**
- Les fixtures JSON (`plan-no-changes.json`, `plan-with-drift.json`) reproduisent le format exact de `terraform show -json` sur un plan existant
- Aucun appel réseau ni terraform réel en test unitaire — les tests d'intégration Azure live sont planifiés dans E6-S9 (nightly)

---

### E6-S5 : Enforcement de couverture en CI

**Points:** 3
**Dependencies:** → E6-S3, E6-S4
**Priority:** Should

**Description:**
Ajouter un seuil minimum de couverture dans le CI pour éviter les régressions silencieuses. Le PRD cible **80 %** de couverture du code Go ; la mise en place se fait en deux paliers pour ne pas bloquer le CI sur le code existant.

**Fichiers à modifier :**
```
.github/workflows/ci.yml        (step coverage gate)
Makefile                        (target test-coverage-check)
CONTRIBUTING.md                 (section testing expectations)
```

**Critères d'acceptation :**
- [ ] CI calcule la couverture globale via `go test -coverprofile=coverage.out ./...`
- [ ] **Palier 1 (Sprint 14) : seuil à 60 %** — valeur réaliste pour l'état du code après E6-S3/S4
- [ ] **Palier 2 (Sprint 17, cible PRD) : seuil à 80 %** — atteint après complétion de E6 et E7
- [ ] Le seuil actuel est documenté dans un commentaire dans `ci.yml` avec la date de révision prévue
- [ ] Si couverture < seuil → le job `test` échoue avec message lisible : `"Coverage X.X% is below threshold Y%"`
- [ ] Le rapport de couverture est uploadé comme artefact CI (déjà présent, à conserver)
- [ ] `make test-coverage-check` reproduit la vérification localement avec le même seuil
- [ ] Documentation dans `CONTRIBUTING.md` : tableau des seuils par palier + commandes pour mesurer

**Notes d'implémentation :**
- Utiliser `go tool cover -func=coverage.out` et `awk` pour extraire le pourcentage total
- Alternative : `github.com/vladopajic/go-test-coverage` action (supporte les seuils par package)
- Exclure les packages sans logique testable (`schemas/embed.go`, `templates/embed.go`) avec `coverpkg` ou `exclude` pattern

---

### E6-S6 : Tests pour le check `state-backend` du `doctor`

**Points:** 5
**Dependencies:** → E1-S3
**Priority:** Should

**Description:**
`checkStateBackend()` a été ajouté à `AllChecks()` mais n'a pas de tests unitaires dédiés. Couvrir tous ses chemins de retour : warn (az non connecté), warn (aucun compte taggé), pass (compte trouvé et accessible), warn (compte trouvé mais inaccessible).

**Fichiers à modifier :**
```
internal/doctor/checks_test.go    (ajouter les cas state-backend)
```

**Critères d'acceptation :**
- [ ] `TestCheckStateBackend_AzNotConnected` → `StatusWarn`, message "Could not query"
- [ ] `TestCheckStateBackend_NoTaggedAccount` → `StatusWarn`, message "No storage account tagged"
- [ ] `TestCheckStateBackend_FoundAndAccessible` → `StatusPass`, message contient le nom du compte
- [ ] `TestCheckStateBackend_FoundButInaccessible` → `StatusWarn`, message contient le nom du compte
- [ ] Les mocks `az storage account list` et `az storage account show` sont correctement chaînés
- [ ] Tous les tests passent avec `go test ./internal/doctor/...`

---

### E6-S7 : Smoke test CLI multi-plateforme en CI

**Points:** 5
**Dependencies:** → E1-S1
**Priority:** Should

**Description:**
Le smoke test actuel (`./bin/lzctl version`) tourne uniquement sur `ubuntu-latest`. Le build matriciel (ubuntu, macos, windows) ne valide pas le binaire après compilation. Ajouter un smoke test par OS dans la matrice.

**Fichiers à modifier :**
```
.github/workflows/ci.yml
```

**Critères d'acceptation :**
- [ ] Job `build` étendu ou remplacé par un job `build-and-smoke` matriciel sur [ ubuntu-latest, macos-latest, windows-latest ]
- [ ] Chaque OS compile le binaire et exécute : `lzctl version`, `lzctl --help`, `lzctl doctor --help`
- [ ] Sur Windows : le binaire s'appelle `lzctl.exe`, le chemin est adapté
- [ ] En cas d'échec sur un OS, le job matriciel signale lequel a échoué
- [ ] Artefacts binaires uploadés par OS (pour inspection manuelle)

**Notes d'implémentation :**
- `go build -o bin/lzctl${{ matrix.ext }} .` avec `matrix.ext` = `""` or `".exe"`
- Utiliser `shell: bash` y compris sur Windows (Git Bash disponible dans les runners GitHub)

---

### E6-S8 : Tests unitaires — `localName`, `inferLayer`, et `GenerateAll` du `HCLGenerator`

**Points:** 3
**Dependencies:** → E3-S5
**Priority:** Must

**Description:**
Suite aux bugs corrigés (`localName` n'ignorait pas les espaces, `GenerateAll` ne créait pas de sous-dossier `general/`), ajouter des cas limites explicites dans les tests pour prévenir les régressions.

**Fichiers à modifier :**
```
internal/importer/hcl_generator_test.go
```

**Critères d'acceptation :**
- [ ] `TestHCLGenerator_LocalName` couvre : tirets, underscores, espaces, caractères spéciaux, nom vide, nom tout-majuscule
- [ ] `TestHCLGenerator_GenerateAll_GroupsByLayer` vérifie que **tous** les layers (y compris `general`) génèrent des chemins `<dir>/<layer>/import.tf` et `<dir>/<layer>/resources.tf`
- [ ] `TestHCLGenerator_GenerateAll_OnlyUnsupported` : si tous les ressources sont `Supported: false`, les fichiers générés contiennent uniquement des `# TODO:` mais sont quand même créés
- [ ] `TestHCLGenerator_GenerateAll_MultipleLayersNoCollision` : ressources sur plusieurs layers → chemins distincts, pas de collision

---

### E6-S9 : Tests d'intégration Azure live (nightly / manuel)

**Points:** 3
**Dependencies:** → E3-S1, E6-S4
**Priority:** Should

**Description:**
Les tests unitaires et d'intégration en PR s'exécutent sans accès Azure réel (mocks). Les tests qui nécessitent un tenant Azure de test (vérification de drift live, validation d'état réel) doivent s'exécuter dans un workflow séparé, déclenché manuellement ou en nightly — conformément à l'architecture (workflow d'intégration hebdomadaire sur tenant de test mentionné dans `architecture.md`).

**Fichiers à créer :**
```
.github/workflows/integration-azure.yml    (workflow nightly/manual)
test/integration/azure_live_test.go        (build tag: //go:build integration)
docs/development.md                        (section "Running Azure integration tests")
```

**Critères d'acceptation :**
- [ ] Nouveau workflow `integration-azure.yml` avec `on: [workflow_dispatch, schedule: cron '0 2 * * 1']` (lundi 2h)
- [ ] Tests marqués avec build tag `//go:build integration` → exclus des `go test ./...` classiques
- [ ] Le workflow requiert des secrets : `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_SUBSCRIPTION_ID` (OIDC)
- [ ] Au moins 2 tests live : `TestLiveDoctor_AzureSession` et `TestLiveDrift_NoChanges` (sur tenant de test vide)
- [ ] Le workflow CI PR n'inclut **jamais** les tests `integration` (flag `-tags=integration` absent)
- [ ] `CONTRIBUTING.md` et `docs/development.md` documentent clairement la séparation nightly vs PR

---

**Epic 6 Total : 9 stories, 41 points**

---

## Epic 7 — Mode Non-interactif & GitOps Headless

> **Goal:** Permettre à `lzctl init` (et aux futures commandes) de fonctionner entièrement sans TTY — depuis un pipeline CI, un script ou une action GitHub — en passant des flags ou des variables d'environnement.
> **Phase:** Transverse (priorité haute pour les équipes plateforme qui automatisent l'onboarding)
> **Total points:** 28

---

### E7-S1 : Compléter les flags non-interactifs de `lzctl init`

**Points:** 5
**Dependencies:** → E1-S5 (wizard), E1-S13 (init command)
**Priority:** Must

**Description:**
La commande `lzctl init --tenant-id <uuid>` évite maintenant le wizard, mais utilise des valeurs CAF standard pour tous les autres paramètres. Les équipes plateforme ont besoin de contrôler chaque dimension sans interaction.

**Fichiers à modifier :**
```
cmd/init.go
cmd/cmd_test.go
```

**Nouveaux flags à ajouter :**

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--subscription-id` | string | `` | Azure subscription ID de management |
| `--project-name` | string | `"landing-zone"` | Nom du projet |
| `--mg-model` | string | `"caf-standard"` | `caf-standard` \| `caf-lite` |
| `--connectivity` | string | `"hub-spoke"` | `hub-spoke` \| `vwan` \| `none` |
| `--identity` | string | `"workload-identity-federation"` | Modèle d'identité |
| `--primary-region` | string | `"westeurope"` | Région principale Azure |
| `--secondary-region` | string | `` | Région secondaire (optionnel) |
| `--cicd-platform` | string | `"github-actions"` | `github-actions` \| `azure-devops` |
| `--state-strategy` | string | `"create-new"` | `create-new` \| `existing` \| `terraform-cloud` |

**Critères d'acceptation :**
- [ ] Tous les flags ci-dessus sont enregistrés sur `initCmd`
- [ ] Quand `--tenant-id` est fourni, le branch non-interactif de `runInit` utilise toutes les valeurs des flags (priorité : flag > défaut wizard)
- [ ] `lzctl init --tenant-id <uuid> --project-name myproj --mg-model caf-lite --connectivity none` → génère un projet sans aucun prompt
- [ ] `lzctl init --tenant-id <uuid> --dry-run` → affiche les fichiers qui seraient générés sans les écrire
- [ ] L'aide (`--help`) documente clairement les flags et leurs valeurs possibles avec `enum` dans le texte
- [ ] Tests : un test par combinaison significative (caf-lite + none, caf-standard + hub-spoke, caf-standard + vwan)
- [ ] Validation : si la valeur d'un enum est invalide (ex: `--connectivity foobar`) → erreur claire avant démarrage

---

### E7-S2 : Support des variables d'environnement `LZCTL_*`

**Points:** 3
**Dependencies:** → E7-S1
**Priority:** Must

**Description:**
Les pipelines CI stockent les secrets et paramètres dans des variables d'environnement. Viper est déjà configuré avec `AutomaticEnv` mais les variables ne couvrent pas les flags de `init`. Mapper chaque flag init à une variable d'environnement.

**Fichiers à modifier :**
```
cmd/init.go          (BindPFlag + BindEnv pour chaque flag)
cmd/cmd_test.go      (tester les env vars)
docs/commands/init.md
```

**Variables d'environnement :**

| Variable | Flag correspondant |
|----------|-------------------|
| `LZCTL_TENANT_ID` | `--tenant-id` |
| `LZCTL_SUBSCRIPTION_ID` | `--subscription-id` |
| `LZCTL_PROJECT_NAME` | `--project-name` |
| `LZCTL_MG_MODEL` | `--mg-model` |
| `LZCTL_CONNECTIVITY` | `--connectivity` |
| `LZCTL_IDENTITY` | `--identity` |
| `LZCTL_PRIMARY_REGION` | `--primary-region` |
| `LZCTL_CICD_PLATFORM` | `--cicd-platform` |

**Critères d'acceptation :**
- [ ] `LZCTL_TENANT_ID=<uuid> lzctl init` fonctionne sans `--tenant-id` (priorité : flag > env > défaut)
- [ ] `LZCTL_CONNECTIVITY=none LZCTL_MG_MODEL=caf-lite lzctl init --tenant-id <uuid>` → config lite sans connectivity
- [ ] Tests : `t.Setenv("LZCTL_TENANT_ID", uuid)` + `executeCommand("init")` → no error
- [ ] Documentation dans `docs/commands/init.md` : tableau flags / env vars
- [ ] L'aide `--help` mentionne le préfixe `LZCTL_` pour les surcharges

---

### E7-S3 : Mode `--ci` (non-TTY strict)

**Points:** 5
**Dependencies:** → E7-S1, E7-S2
**Priority:** Should

**Description:**
Quand lzctl est exécuté dans un pipeline (pas de TTY), tout prompt interactif doit échouer proprement avec un message explicite plutôt que de bloquer ou produire une erreur cryptique (`EOF`). Ajouter un flag global `--ci` qui active ce mode.

**Fichiers à modifier :**
```
cmd/root.go              (ajouter flag global --ci)
cmd/init.go              (appliquer la garde --ci)
internal/wizard/wizard.go   (mode strict non-TTY)
cmd/cmd_test.go
```

**Critères d'acceptation :**
- [ ] `lzctl --ci init` sans `--tenant-id` → erreur explicite : `"--ci mode requires --tenant-id (or LZCTL_TENANT_ID)"`
- [ ] En mode CI, toute tentative de prompt retourne immédiatement une erreur (pas de blocage sur stdin)
- [ ] Détection automatique du mode CI si `CI=true` (variable standard GitHub Actions / GitLab / etc.)
- [ ] `lzctl init --ci --tenant-id <uuid>` (ou `CI=true lzctl init --tenant-id <uuid>`) → fonctionne sans prompt
- [ ] Toutes les commandes avec wizard (init, workload add, import) respectent le mode CI
- [ ] Exit code 1 avec message lisible si mode CI et paramètre obligatoire manquant
- [ ] Tests : `t.Setenv("CI", "true")` vérifie que le wizard ne démarre pas

---

### E7-S4 : Input déclaratif pipeline → génération de `lzctl.yaml`

**Points:** 5
**Dependencies:** → E7-S1, E7-S2
**Priority:** Could

**Description:**
`lzctl.yaml` est et reste le **seul manifeste déclaratif** de l'état du projet (source of truth, défini dans le PRD). Pour les équipes qui onboardent en masse depuis un pipeline CI, permettre de passer un fichier d'entrée simplifié (`lzctl-init-input.yaml`) qui est **converti en `lzctl.yaml`** lors de l'init, puis supprimé ou archivé — il ne coexiste pas avec `lzctl.yaml`.

Ce fichier d'entrée est un *input transitoire* (one-shot), pas un second manifeste à maintenir.

**Fichiers à créer :**
```
docs/examples/pipeline-init/lzctl-init-input.yaml   (exemple d'input CI)
cmd/init.go                                          (support --from-file)
internal/config/init_input.go                        (struct + loader + "converter vers LZConfig")
internal/config/init_input_test.go
```

**Format `lzctl-init-input.yaml` (input CI, one-shot) :**
```yaml
# Input transitoire pour lzctl init --from-file
# Ce fichier est converti en lzctl.yaml et n'est plus nécessaire ensuite.
tenantId: "00000000-0000-0000-0000-000000000001"
projectName: "contoso-platform"
mgModel: "caf-standard"
connectivity: "hub-spoke"
primaryRegion: "westeurope"
cicdPlatform: "github-actions"
stateStrategy: "create-new"
landingZones:
  - name: "corp-prod"
    archetype: "corp"
    subscriptionId: "sub-001"
    addressSpace: "10.10.0.0/16"
  - name: "online-dev"
    archetype: "online"
    subscriptionId: "sub-002"
    addressSpace: "10.20.0.0/16"
```

**Critères d'acceptation :**
- [ ] `lzctl init --from-file lzctl-init-input.yaml` convertit l'input en `lzctl.yaml` complet (avec toutes les sections CAF) puis exécute l'init normalement
- [ ] Le fichier d'entrée **ne remplace pas** `lzctl.yaml` dans le repo : après `lzctl init --from-file`, seul `lzctl.yaml` est présent comme source of truth
- [ ] Si `lzctl.yaml` existe déjà dans le repo cible → erreur explicite "lzctl.yaml already exists, use --force to overwrite" (pas de merge silencieux)
- [ ] Validation de l'input avant conversion (champs requis, enums valides, pas d'overlap d'adresses IP)
- [ ] `--dry-run` affiche le `lzctl.yaml` qui serait généré sans l'écrire
- [ ] Tests : fixture `lzctl-init-input.yaml` avec 2 landing zones → vérifier que `lzctl.yaml` généré est valide et contient les 2 zones

**Notes d'implémentation :**
- `InitInput.ToLZConfig()` → retourne un `*config.LZConfig` complet avec les défauts CAF appliqués
- Mentionner dans la doc que `lzctl-init-input.yaml` peut être committé dans un repo de bootstrap séparé (pas dans le repo landing zone cible)

---

### E7-S5 : Documentation — guide d'utilisation en pipeline CI

**Points:** 3
**Dependencies:** → E7-S1, E7-S2, E7-S3
**Priority:** Should

**Description:**
Documenter de bout en bout comment utiliser `lzctl` depuis un pipeline GitHub Actions et Azure DevOps sans interaction manuelle.

**Fichiers à créer :**
```
docs/operations/ci-headless.md
docs/examples/pipeline-init/github-actions-onboarding.yml
docs/examples/pipeline-init/azure-devops-onboarding.yml
```

**Critères d'acceptation :**
- [ ] Guide couvre : prérequis (OIDC ou SP), variables secrets à configurer, étapes `init → validate → plan → apply`
- [ ] Exemple complet GitHub Actions : job qui crée un nouveau projet landing zone via `lzctl init --ci`
- [ ] Exemple complet Azure DevOps : pipeline YAML équivalent
- [ ] Section troubleshooting : erreurs courantes en CI (no TTY, permissions, az login)
- [ ] Lien depuis `README.md` et depuis `docs/commands/init.md`

---

### E7-S6 : Test d'intégration headless end-to-end

**Points:** 5
**Dependencies:** → E7-S1, E7-S2, E7-S3
**Priority:** Should

**Description:**
Test d'intégration qui simule un pipeline CI complet sans TTY : init non-interactif → validate → plan (dry-run) → vérification du repo généré.

**Fichiers à créer :**
```
test/integration/headless_test.go
```

**Critères d'acceptation :**
- [ ] Test `TestHeadlessInit_FullWorkflow` : `lzctl init --tenant-id <uuid> --ci` dans un tmpDir → repo généré valide
- [ ] Validation du repo : `lzctl validate` dans le tmpDir → exit 0
- [ ] Vérification des fichiers générés : présence de `platform/`, `landing-zones/`, `pipelines/`, `lzctl.yaml`
- [ ] Test exécuté sans aucun prompt (stdin fermé) → pas de blocage ni de panic
- [ ] `t.Setenv("CI", "true")` confirme que le mode CI est activé automatiquement
- [ ] Test passe sur ubuntu, macos et windows (matrice CI)

---

**Epic 7 Total : 6 stories, 26 points**

---

## Mise à jour du Sprint Planning

### Nouvelles phases

| Phase | Epics | Durée est. | Objectif |
|-------|-------|------------|----------|
| **Phase 5** | E6 (Quality) | 4-5 semaines | Zéro test en échec, couverture ≥ 60 % (palier 1) → 80 % (palier 2, cible PRD), CI robuste |
| **Phase 6** | E7 (GitOps) | 3-4 semaines | `lzctl init` headless, pipeline-ready |

### Sprints suggérés

| Sprint | Stories | Points | Milestone |
|--------|---------|--------|-----------|
| Sprint 12 | E6-S1, E6-S2, E6-S7, E6-S8 | 12 | CI hardening + GoReleaser validé + smoke multi-OS (tôt pour éviter régressions CRLF/chemins) |
| Sprint 13 | E6-S3, E6-S6 | 13 | Tests output généré (repo TF, pipelines, backend routing) + state-backend |
| Sprint 14 | E6-S4, E6-S5, E6-S9 | 17 | Tests drift/rollback fixtures + seuil couverture 60 % + workflow nightly Azure |
| Sprint 15 | E7-S1, E7-S2 | 8 | Flags non-interactifs complets + env vars |
| Sprint 16 | E7-S3, E7-S5 | 8 | Mode --ci + guide documentation CI |
| Sprint 17 | E7-S4, E7-S6 | 10 | Input déclaratif CI → lzctl.yaml + test E2E headless |
| Sprint 18 | E6-S5 (palier 2) | — | Relèvement seuil couverture à 80 % (cible PRD) |

> **Principe tests Azure live :** les tests nécessitant un tenant Azure réel (drift live, session check) sont **hors PR** — ils s'exécutent dans le workflow `integration-azure.yml` déclenché manuellement ou nightly (E6-S9). Les PR ne contiennent que des tests unitaires et des tests d'intégration sans appel réseau Azure.

### Opportunités de parallélisation

| Track A | Track B | Note |
|---------|---------|------|
| E6-S1 (lint pin) | E6-S2 (goreleaser check) | Indépendants, même fichier CI — merger en une PR |
| E6-S7 (smoke multi-OS) | E6-S8 (HCLGenerator edge cases) | Indépendants |
| E6-S3 (plan/apply/validate output) | E6-S4 (drift/rollback output) | Packages différents, même approche fixtures |
| E7-S1 (flags init) | E7-S2 (env vars) | E7-S2 dépend de E7-S1 mais faiblement |
| E7-S4 (input déclaratif) | E7-S5 (doc CI) | Indépendants |


