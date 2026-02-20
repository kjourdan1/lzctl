# Product Requirements Document — lzctl

> Version: 1.0
> Date: 2026-02-18
> Author: Killian Jourdan

---

## 1. Product Overview

**lzctl** is an open-source CLI tool (Go, single binary) that bootstraps and maintains Azure Landing Zones aligned with the Cloud Adoption Framework (CAF). It generates production-ready Terraform repositories using Azure Verified Modules (AVM), wired to CI/CD pipelines (GitHub Actions or Azure DevOps), following a GitOps workflow where PRs trigger plans and merges trigger deployments.

It supports both greenfield (scaffold from scratch) and brownfield (audit existing Azure estates, generate gap analysis, and assist Terraform import of existing resources).

---

## 2. Resolved Open Questions

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| Q1 | Default MG hierarchy | CAF Standard (5-level) as default, CAF Lite (3-level) as option. Custom deferred to phase 2 | Standard covers most enterprise needs; Lite for small environments; keeps wizard simple |
| Q2 | Git operations in init | File generation only — no `git init`, no commit, no push | User controls their git workflow; avoids permission issues and surprises |
| Q3 | Naming opinionation | Flexible with CAF naming convention as default (e.g., `rg-<workload>-<env>-<region>-<instance>`) | Provides sensible defaults; user can override in `lzctl.yaml` |
| Q4 | Audit output format | Both Markdown (human) and JSON (machine-readable) | JSON enables integration with other tools, dashboards, CI gates |
| Q5 | CLI upgrade story | `lzctl upgrade` updates AVM module versions in generated code; no re-scaffold | Generated repo belongs to the user; lzctl doesn't own it after init |
| Q6 | License | Apache 2.0 | Industry standard (Terraform, K8s, AVM); patent protection for enterprise adoption |

---

## 3. Functional Requirements

### FR-1: Environment Preflight — `lzctl doctor`

**Description:** Verify that the user's local environment meets all prerequisites before any operation.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-1.1 | Check presence and minimum version of: `terraform` (>= 1.5), `az` CLI (>= 2.50), `git` (>= 2.30) |
| FR-1.2 | Check optional tools: `gh` CLI (for GitHub), `az devops` extension (for ADO) |
| FR-1.3 | Verify `az account show` returns a valid session; display tenant ID, subscription, and user |
| FR-1.4 | Check Azure permissions: user has Owner or User Access Administrator role on the root management group (or on the specified scope) |
| FR-1.5 | Check that required Azure resource providers are registered (`Microsoft.Management`, `Microsoft.Authorization`, `Microsoft.Network`, `Microsoft.ManagedIdentity`) |
| FR-1.6 | Output a clear pass/fail summary with actionable fix commands for each failure |
| FR-1.7 | Exit code 0 if all checks pass, non-zero if any critical check fails |
| FR-1.8 | Support `--json` flag for machine-readable output |

---

### FR-2: Greenfield Scaffolding — `lzctl init`

**Description:** Interactive wizard that generates a complete IaC repository ready for GitOps deployment.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-2.1 | Interactive wizard collects: project name, tenant ID, CI/CD platform (GitHub Actions / Azure DevOps), landing zone topology (CAF Standard / CAF Lite), connectivity model (Hub & Spoke with Firewall / Hub & Spoke with NVA / Virtual WAN / None), primary region, optional secondary region, identity model for CI/CD (Workload Identity Federation / SP + Federated Credential / SP + Secret), state backend strategy (create new / existing / Terraform Cloud) |
| FR-2.2 | Support `--config <file>` flag to skip the wizard and use a pre-filled YAML configuration (for CI/automation) |
| FR-2.3 | Generate `lzctl.yaml` manifest as the declarative source of truth for the entire landing zone |
| FR-2.4 | Generate layered Terraform structure with separate state per layer: `platform/management-groups/`, `platform/identity/`, `platform/management/`, `platform/governance/`, `platform/connectivity/` |
| FR-2.5 | Each layer uses AVM modules with pinned versions; generated HCL is standard, readable, and maintainable without lzctl |
| FR-2.6 | Generate CI/CD pipeline files based on chosen platform: `.github/workflows/` for GitHub Actions, `.azuredevops/pipelines/` for Azure DevOps |
| FR-2.7 | Generated pipelines implement: on PR → `terraform validate` + `terraform plan` (posted as PR comment); on merge to main → `terraform apply` per layer in dependency order |
| FR-2.8 | Optionally bootstrap the state backend (Azure Storage Account, resource group, managed identity, RBAC assignments, federated credential) using direct `az` CLI commands — not Terraform (avoids chicken-and-egg) |
| FR-2.9 | If bootstrapping, create the minimum infrastructure: resource group (`rg-<project>-tfstate-<region>`), storage account (with versioning and soft delete), blob container (`tfstate`), managed identity with Owner on root MG, federated credential for chosen CI/CD platform |
| FR-2.10 | Generate a `README.md` with onboarding instructions, architecture overview, and next steps |
| FR-2.11 | Generate a `.gitignore` covering Terraform state, `.terraform/`, `.lzctl/` local config |
| FR-2.12 | All generated files are written to the current directory; no git operations performed |
| FR-2.13 | Support `--dry-run` flag that shows what files would be generated without writing them |
| FR-2.14 | Idempotent: running `lzctl init` in an existing lzctl project warns and exits unless `--force` is passed |

---

### FR-3: Manifest Schema — `lzctl.yaml`

**Description:** Declarative YAML manifest that describes the entire landing zone configuration. Serves as the single source of truth that lzctl commands read from and write to.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-3.1 | Schema uses a versioned API format: `apiVersion: lzctl/v1`, `kind: LandingZone` |
| FR-3.2 | `metadata` section: `name`, `tenant`, `primaryRegion`, `secondaryRegion` (optional) |
| FR-3.3 | `spec.platform.managementGroups`: model (`caf-standard` / `caf-lite`), optional `disabled` list to remove specific MG nodes, optional `custom` overrides |
| FR-3.4 | `spec.platform.connectivity`: `type` (hub-spoke / vwan / none), hub config (region, address space, firewall settings, VPN/ER gateway, DNS), peering settings |
| FR-3.5 | `spec.platform.identity`: CI/CD identity type and references |
| FR-3.6 | `spec.platform.management`: Log Analytics config (retention, solutions), Automation Account, Defender for Cloud plans |
| FR-3.7 | `spec.governance.policies`: list of built-in CAF policy set assignments + paths to custom policy definitions |
| FR-3.8 | `spec.landingZones[]`: array of application landing zones, each with name, subscription ID, archetype (corp / online / sandbox), VNET address space, connectivity (peered / isolated) |
| FR-3.9 | `spec.naming`: naming convention template with CAF defaults, overridable per resource type |
| FR-3.10 | `spec.stateBackend`: backend configuration (storage account, container, resource group, subscription) |
| FR-3.11 | Schema is validated by `lzctl validate`; JSON Schema definition shipped with the CLI binary |
| FR-3.12 | Schema supports comments and is documented with examples in the generated file |

---

### FR-4: Validation — `lzctl validate`

**Description:** Multi-layer validation that goes beyond `terraform validate` to check cross-cutting concerns and manifest coherence.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-4.1 | Validate `lzctl.yaml` against the embedded JSON Schema (structure, types, required fields) |
| FR-4.2 | Check for IP address space overlaps across hub VNet, spoke VNets, and all landing zone VNets |
| FR-4.3 | Verify all policy references in the manifest resolve to existing built-in or custom policy definitions |
| FR-4.4 | Verify cross-layer references (e.g., landing zone referencing a connectivity hub that exists) |
| FR-4.5 | Run `terraform validate` on each layer directory |
| FR-4.6 | Check that subscription IDs in landing zones are valid GUIDs (format only, not Azure API call) |
| FR-4.7 | Warn if address spaces are too small for expected use (e.g., /28 for a hub VNet) |
| FR-4.8 | Output grouped by severity: error (blocks deployment), warning (should fix), info (suggestion) |
| FR-4.9 | Support `--json` flag for CI integration |
| FR-4.10 | Exit code 0 only if zero errors; warnings do not cause non-zero exit |
| FR-4.11 | Usable both locally and in CI pipelines (no interactive prompts) |

---

### FR-5: Plan & Apply Orchestration — `lzctl plan` / `lzctl apply`

**Description:** Orchestrate Terraform plan and apply across multiple layers in the correct dependency order, routing to the correct state backend.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-5.1 | `lzctl plan` runs `terraform plan` on each layer in dependency order: management-groups → identity → management → governance → connectivity → landing-zones/* |
| FR-5.2 | `lzctl plan <layer>` runs plan on a specific layer only |
| FR-5.3 | `lzctl apply` runs `terraform apply` on each layer in dependency order with auto-approve |
| FR-5.4 | `lzctl apply <layer>` applies a specific layer only |
| FR-5.5 | Each layer uses its own state file within the shared backend (keyed by layer path) |
| FR-5.6 | Plan output is captured and can be saved to a file (`--out <file>`) for CI artifact upload |
| FR-5.7 | If any layer fails during plan/apply, stop execution and report clearly which layer failed and why |
| FR-5.8 | Support `--parallelism` flag to control Terraform parallelism per layer |
| FR-5.9 | Support `--target <layer>` to plan/apply a single layer (alias for positional argument) |
| FR-5.10 | These commands are convenience wrappers — the generated CI/CD pipelines call Terraform directly (no runtime dependency on lzctl in CI) |

---

### FR-6: Add Landing Zone — `lzctl workload add`

**Description:** Interactive command to add a new application landing zone to an existing repo.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-6.1 | Interactive wizard collects: zone name, archetype (corp / online / sandbox), subscription ID, VNET address space, hub connectivity (yes/no) |
| FR-6.2 | Support `--config <file>` for non-interactive use |
| FR-6.3 | Generate new directory under `landing-zones/<zone-name>/` with `main.tf`, `variables.tf`, `<zone-name>.auto.tfvars` |
| FR-6.4 | Generated Terraform uses AVM modules for: resource group, VNet, VNet peering (if connected to hub), NSG defaults |
| FR-6.5 | Update `lzctl.yaml` to include the new landing zone in `spec.landingZones[]` |
| FR-6.6 | Run `lzctl validate` automatically after generation to catch conflicts (e.g., IP overlap) |
| FR-6.7 | Warn if address space overlaps with existing zones; block generation if overlap detected unless `--force` |

---

### FR-7: Brownfield Audit — `lzctl audit`

**Description:** Analyze an existing Azure tenant and produce a gap analysis comparing current state to CAF best practices.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-7.1 | Scan the current tenant (from `az` CLI session) and collect: management group hierarchy, subscription placement, policy assignments, RBAC role assignments (high-privilege), virtual networks and peering topology, diagnostic settings, Defender for Cloud status |
| FR-7.2 | Compare findings against CAF reference architecture and produce a gap analysis |
| FR-7.3 | Gap analysis organized by CAF discipline: governance, identity, management, connectivity, security |
| FR-7.4 | Each gap includes: current state, expected state (CAF recommendation), severity (critical / high / medium / low), remediation guidance (what lzctl can help with vs. manual action needed) |
| FR-7.5 | Output as Markdown report (`audit-report.md`) by default |
| FR-7.6 | Support `--json` flag for machine-readable output (`audit-report.json`) |
| FR-7.7 | Support `--scope <management-group-id>` to limit audit to a subtree |
| FR-7.8 | Support `--output <path>` to specify output location |
| FR-7.9 | Read-only operation — makes no changes to the Azure environment |
| FR-7.10 | Provide a summary score (e.g., "CAF alignment: 45/100") for quick assessment |
| FR-7.11 | Execution time < 5 minutes for tenants with up to 100 subscriptions |

---

### FR-8: Brownfield Import — `lzctl import`

**Description:** Generate Terraform import blocks and HCL configuration for existing Azure resources, enabling progressive adoption of IaC.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-8.1 | Accept input from the audit report (`lzctl import --from audit-report.json`) or from explicit resource specification (`lzctl import --resource-group <name>` / `--subscription <id>`) |
| FR-8.2 | For each resource to import, generate: a Terraform `import` block (Terraform 1.5+ native syntax) and the corresponding HCL resource configuration using AVM modules where applicable |
| FR-8.3 | Generated import code is placed in a dedicated `imports/` directory or appended to the appropriate layer |
| FR-8.4 | Support selective import: user chooses which resources/resource groups to import (interactive checklist or `--include`/`--exclude` flags) |
| FR-8.5 | Generate a plan preview (`lzctl import --dry-run`) showing what would be imported without executing |
| FR-8.6 | After import, user runs `terraform plan` to verify zero diff (state matches reality) |
| FR-8.7 | Handle common resource types: resource groups, VNets, subnets, NSGs, route tables, key vaults, storage accounts, managed identities, policy assignments |
| FR-8.8 | For unsupported or complex resources, generate a TODO comment with manual import instructions |
| FR-8.9 | Support `--layer <layer>` to place imported resources in a specific Terraform layer |
| FR-8.10 | Warn if imported resources conflict with existing Terraform-managed resources |

---

### FR-9: Drift Detection — `lzctl drift`

**Description:** Detect configuration drift between Terraform state and actual Azure resources.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-9.1 | Run `terraform plan` on each layer and parse output for changes |
| FR-9.2 | Summarize drift per layer: no drift / N changes detected |
| FR-9.3 | Classify drift: resource added outside Terraform, resource modified outside Terraform, resource deleted outside Terraform |
| FR-9.4 | Support `--json` flag for CI integration (e.g., scheduled pipeline that alerts on drift) |
| FR-9.5 | Support `--layer <layer>` to check a specific layer only |
| FR-9.6 | Generated CI/CD pipelines include a scheduled drift detection workflow (weekly by default) |
| FR-9.7 | Exit code non-zero if drift detected (useful for CI gates) |

---

### FR-10: Module Upgrade — `lzctl upgrade`

**Description:** Check for and apply updates to AVM module versions in the generated Terraform code.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-10.1 | Query the Terraform registry for latest versions of all AVM modules used in the repo |
| FR-10.2 | Display current vs. available versions with semver classification (major / minor / patch) |
| FR-10.3 | Support `--apply` to update module version references in HCL files |
| FR-10.4 | Major version bumps require explicit `--allow-major` flag (breaking changes) |
| FR-10.5 | After updating, suggest running `lzctl validate` and `lzctl plan` to check for regressions |
| FR-10.6 | Support `--dry-run` to preview changes without applying |
| FR-10.7 | Generate a changelog summary of what changed between versions (from module release notes if available) |

---

### FR-11: Status Overview — `lzctl status`

**Description:** Display a summary of the current landing zone state.

**Acceptance Criteria:**

| # | Criterion |
|---|-----------|
| FR-11.1 | Display: project name, tenant ID, number of layers, number of landing zones, last deployment info (from git log), drift status (if available from last drift check) |
| FR-11.2 | Read from `lzctl.yaml` and local git state — no Azure API calls unless `--live` flag is passed |
| FR-11.3 | With `--live` flag, query Azure to verify resources exist and match expected state |
| FR-11.4 | Support `--json` flag |

---

## 4. Non-Functional Requirements

### NFR-1: Performance

| # | Requirement |
|---|------------|
| NFR-1.1 | `lzctl init` completes file generation in < 10 seconds (excluding bootstrap) |
| NFR-1.2 | `lzctl validate` completes in < 30 seconds for a standard CAF deployment |
| NFR-1.3 | `lzctl audit` completes in < 5 minutes for tenants with up to 100 subscriptions |
| NFR-1.4 | `lzctl doctor` completes in < 5 seconds |
| NFR-1.5 | CLI binary size < 50 MB |

### NFR-2: Portability & Distribution

| # | Requirement |
|---|------------|
| NFR-2.1 | Single binary for Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64) |
| NFR-2.2 | Available via: GitHub Releases, Homebrew, winget/scoop, direct download |
| NFR-2.3 | No runtime dependencies beyond Terraform and az CLI |
| NFR-2.4 | CI/CD pipelines generated by lzctl have zero runtime dependency on lzctl (pure Terraform + az CLI) |

### NFR-3: Developer Experience

| # | Requirement |
|---|------------|
| NFR-3.1 | All commands support `--help` with examples |
| NFR-3.2 | Colored terminal output with emoji indicators (configurable, respects `NO_COLOR` env var) |
| NFR-3.3 | All destructive commands require confirmation or `--force` flag |
| NFR-3.4 | All commands that produce output support `--json` for scripting |
| NFR-3.5 | Meaningful error messages with suggested fixes (never raw stack traces) |
| NFR-3.6 | `--verbose` / `-v` flag for debug output |
| NFR-3.7 | Shell completion for bash, zsh, fish, PowerShell (via Cobra built-in) |
| NFR-3.8 | Consistent command structure: `lzctl <verb> [args] [flags]` |

### NFR-4: Security

| # | Requirement |
|---|------------|
| NFR-4.1 | Never store Azure credentials in generated files or lzctl config |
| NFR-4.2 | Generated CI/CD pipelines use Workload Identity Federation by default (no secrets) |
| NFR-4.3 | If SP + Secret is chosen, credentials are stored in GitHub Secrets / ADO Variable Groups only — never in code |
| NFR-4.4 | Generated Terraform code follows Azure security best practices: storage account with TLS 1.2+, private access, versioning |
| NFR-4.5 | Audit command is read-only — verify no write API calls |
| NFR-4.6 | CLI binary is signed and published with checksums |

### NFR-5: Maintainability & Code Quality

| # | Requirement |
|---|------------|
| NFR-5.1 | Generated Terraform code is formatted (`terraform fmt`) and linted |
| NFR-5.2 | Generated code includes comments explaining each section's purpose |
| NFR-5.3 | AVM module versions are pinned with exact versions (not ranges) |
| NFR-5.4 | All lzctl Go code has unit tests with > 80% coverage |
| NFR-5.5 | Integration tests that validate generated repos against Terraform validate |
| NFR-5.6 | CI pipeline on the lzctl repo itself: lint, test, build, release |

### NFR-6: Documentation

| # | Requirement |
|---|------------|
| NFR-6.1 | README.md with quickstart, features, install instructions, architecture overview |
| NFR-6.2 | `docs/` directory with per-command reference, examples, and architecture decision records |
| NFR-6.3 | Generated repos include their own README with onboarding guide |
| NFR-6.4 | CONTRIBUTING.md with dev setup, coding standards, PR process |
| NFR-6.5 | CHANGELOG.md following Keep a Changelog format |

---

## 5. CLI Command Summary

| Command | Category | Description | Phase |
|---------|----------|-------------|-------|
| `lzctl doctor` | Preflight | Check prerequisites and environment | 1 |
| `lzctl init` | Greenfield | Scaffold a complete landing zone repo | 1 |
| `lzctl validate` | Validation | Multi-layer validation of manifest and Terraform | 1 |
| `lzctl plan [layer]` | Operations | Orchestrated Terraform plan across layers | 1 |
| `lzctl apply [layer]` | Operations | Orchestrated Terraform apply across layers | 1 |
| `lzctl workload add` | Day-2 | Add a new application landing zone | 3 |
| `lzctl audit` | Brownfield | Gap analysis of existing Azure tenant | 2 |
| `lzctl import` | Brownfield | Generate Terraform import blocks for existing resources | 2 |
| `lzctl drift` | Day-2 | Detect configuration drift | 3 |
| `lzctl upgrade` | Day-2 | Update AVM module versions | 3 |
| `lzctl status` | Day-2 | Landing zone status overview | 3 |
| `lzctl version` | Meta | Display CLI version | 1 |

---

## 6. Epic Breakdown

### Epic 1 — CLI Foundation & Scaffolding (Phase 1)

| Story | Description | Priority |
|-------|-------------|----------|
| E1-S1 | Go project scaffolding with Cobra, CI/CD, release automation | Must |
| E1-S2 | `lzctl version` command | Must |
| E1-S3 | `lzctl doctor` — prerequisite checks | Must |
| E1-S4 | `lzctl.yaml` schema definition (JSON Schema + Go structs) | Must |
| E1-S5 | `lzctl init` — interactive wizard (survey library) | Must |
| E1-S6 | `lzctl init` — template engine for Terraform generation | Must |
| E1-S7 | `lzctl init` — GitHub Actions pipeline templates | Must |
| E1-S8 | `lzctl init` — Azure DevOps pipeline templates | Must |
| E1-S9 | `lzctl init` — state backend bootstrap (az CLI) | Must |
| E1-S10 | `lzctl init --config` non-interactive mode | Should |
| E1-S11 | `lzctl validate` — schema + cross-cutting checks | Must |
| E1-S12 | `lzctl plan` / `lzctl apply` — multi-layer orchestration | Must |

### Epic 2 — Terraform Templates & AVM Integration (Phase 1)

| Story | Description | Priority |
|-------|-------------|----------|
| E2-S1 | Management groups layer template (CAF Standard hierarchy) | Must |
| E2-S2 | Management groups layer template (CAF Lite hierarchy) | Must |
| E2-S3 | Connectivity layer — Hub & Spoke with Azure Firewall | Must |
| E2-S4 | Connectivity layer — Hub & Spoke with NVA placeholder | Should |
| E2-S5 | Connectivity layer — Virtual WAN | Should |
| E2-S6 | Management layer — Log Analytics, Automation, Defender | Must |
| E2-S7 | Governance layer — CAF default policy assignments | Must |
| E2-S8 | Identity layer — managed identities for CI/CD | Must |
| E2-S9 | Landing zone archetype template — Corp | Must |
| E2-S10 | Landing zone archetype template — Online | Must |
| E2-S11 | Landing zone archetype template — Sandbox | Should |
| E2-S12 | Naming convention module integration (CAF naming) | Must |

### Epic 3 — Brownfield Capabilities (Phase 2)

| Story | Description | Priority |
|-------|-------------|----------|
| E3-S1 | Azure tenant scanner — collect management groups, subs, policies, VNets | Must |
| E3-S2 | CAF compliance rules engine (expected vs. actual) | Must |
| E3-S3 | `lzctl audit` — Markdown report generator | Must |
| E3-S4 | `lzctl audit` — JSON output | Must |
| E3-S5 | `lzctl audit` — CAF alignment score calculation | Should |
| E3-S6 | `lzctl import` — resource discovery and HCL generation | Must |
| E3-S7 | `lzctl import` — Terraform import block generation | Must |
| E3-S8 | `lzctl import` — interactive resource selection | Should |
| E3-S9 | `lzctl import --dry-run` preview | Must |

### Epic 4 — Day-2 Operations (Phase 3)

| Story | Description | Priority |
|-------|-------------|----------|
| E4-S1 | `lzctl workload add` — interactive wizard + generation | Must |
| E4-S2 | `lzctl drift` — drift detection wrapper | Should |
| E4-S3 | `lzctl upgrade` — AVM module version checker | Should |
| E4-S4 | `lzctl status` — local status overview | Could |
| E4-S5 | `lzctl status --live` — Azure live status | Could |
| E4-S6 | Drift detection CI/CD pipeline template (scheduled) | Should |

### Epic 5 — Documentation & Community (Phase 4)

| Story | Description | Priority |
|-------|-------------|----------|
| E5-S1 | README.md with quickstart, architecture diagram, install | Must |
| E5-S2 | Per-command documentation (`docs/commands/`) | Must |
| E5-S3 | Example repos (greenfield standard, greenfield lite, brownfield) | Should |
| E5-S4 | CONTRIBUTING.md + developer setup guide | Must |
| E5-S5 | Demo recording (terminal GIF / asciinema) | Should |
| E5-S6 | Launch blog post / LinkedIn article | Should |

---

## 7. User Flows

### 7.1 Greenfield — First Deployment

```
User installs lzctl (brew/binary)
  → lzctl doctor (verify env)
  → lzctl init (wizard → generates repo + optionally bootstraps backend)
  → User reviews generated files
  → User: git init → git add → git commit → git push
  → User opens first PR (or pushes to main)
  → CI: terraform validate → terraform plan (on PR) or terraform apply (on merge)
  → Landing zone deployed ✅
```

### 7.2 Brownfield — Assess and Adopt

```
User installs lzctl
  → lzctl doctor
  → lzctl audit (scans existing tenant → produces gap report)
  → User reviews audit-report.md, prioritizes gaps
  → lzctl init (scaffolds repo for the target architecture)
  → lzctl import --from audit-report.json (generates import blocks for existing resources)
  → User reviews import blocks, runs terraform plan to verify zero-diff
  → User commits, pushes, merges → existing resources now under IaC
  → User iterates: lzctl workload add, apply governance policies, close remaining gaps
```

### 7.3 Day-2 — Add a Landing Zone

```
User: lzctl workload add (wizard → collects zone config)
  → Files generated in landing-zones/<name>/
  → lzctl.yaml updated
  → lzctl validate (automatic, checks IP overlaps etc.)
  → User commits → opens PR → CI shows plan → merge → deployed
```

### 7.4 Day-2 — Drift Detection (Automated)

```
Scheduled CI pipeline (weekly)
  → Runs: lzctl drift (or direct terraform plan per layer)
  → If drift detected: creates issue/alert with drift details
  → Team reviews and decides: reconcile in Terraform or revert manual change
```

---

## 8. Data Model — `lzctl.yaml` Full Schema

```yaml
apiVersion: lzctl/v1            # Schema version
kind: LandingZone               # Resource type

metadata:
  name: string                  # Project name (e.g., "contoso-platform")
  tenant: uuid                  # Azure AD Tenant ID
  primaryRegion: string         # e.g., "westeurope"
  secondaryRegion: string?      # Optional secondary region

spec:
  platform:
    managementGroups:
      model: caf-standard | caf-lite
      disabled: string[]?       # MG names to exclude (e.g., ["sandbox"])
    
    connectivity:
      type: hub-spoke | vwan | none
      hub:
        region: string
        addressSpace: cidr
        firewall:
          enabled: boolean
          sku: Standard | Premium
          threatIntel: Off | Alert | Deny
        dns:
          privateResolver: boolean
          forwarders: string[]?
        vpnGateway:
          enabled: boolean
          sku: string?          # VpnGw1, VpnGw2, etc.
        expressRouteGateway:
          enabled: boolean
          sku: string?
    
    identity:
      type: workload-identity-federation | sp-federated | sp-secret
      clientId: string?         # Populated after bootstrap
      principalId: string?
    
    management:
      logAnalytics:
        retentionDays: integer  # 30-730
        solutions: string[]?    # e.g., ["SecurityInsights", "VMInsights"]
      automationAccount: boolean
      defenderForCloud:
        enabled: boolean
        plans: string[]         # e.g., ["Servers", "AppServices", "KeyVaults"]
  
  governance:
    policies:
      assignments: string[]     # Built-in CAF policy set names
      custom: string[]?         # Paths to custom policy HCL files
    
  naming:
    convention: caf             # Only "caf" for MVP; extensible later
    overrides: map?             # Per-resource-type overrides
  
  stateBackend:
    resourceGroup: string
    storageAccount: string
    container: string
    subscription: uuid
  
  landingZones:
    - name: string
      subscription: uuid
      archetype: corp | online | sandbox
      addressSpace: cidr
      connected: boolean        # Peered to hub?
      tags: map?

  cicd:
    platform: github-actions | azure-devops
    repository: string?         # Repo URL (informational)
    branchPolicy:
      mainBranch: string        # Default: "main"
      requirePR: boolean        # Default: true
```

---

## 9. Acceptance Criteria — MVP Definition of Done

The MVP (Phase 1 + Phase 2) is considered complete when:

| # | Criterion |
|---|-----------|
| AC-1 | `lzctl doctor` correctly identifies missing prerequisites on a clean Ubuntu 24.04 and macOS 14 |
| AC-2 | `lzctl init` (greenfield, CAF Standard, Hub & Spoke, GitHub Actions) produces a repo that deploys successfully to an Azure tenant |
| AC-3 | `lzctl init` (greenfield, CAF Lite, None connectivity, Azure DevOps) produces a repo that deploys successfully |
| AC-4 | Generated GitHub Actions pipeline: PR triggers plan, merge triggers apply, both complete without errors |
| AC-5 | Generated Azure DevOps pipeline: same behavior as GitHub Actions |
| AC-6 | `lzctl validate` catches: invalid YAML schema, IP overlaps, dangling policy references |
| AC-7 | `lzctl audit` produces a readable, accurate gap analysis for a test tenant with known gaps |
| AC-8 | `lzctl import` generates valid import blocks that result in zero-diff `terraform plan` for: resource groups, VNets, NSGs |
| AC-9 | All commands support `--help`, `--json`, and `--verbose` flags |
| AC-10 | CLI binary available for Linux/macOS/Windows via GitHub Releases |
| AC-11 | README with install instructions, quickstart, and at least one example |
| AC-12 | Generated Terraform code is readable and maintainable without lzctl |

---

*This PRD provides the requirements for the technical architecture design, data models, and implementation plan.*
