# lzctl

> **Landing Zone Factory CLI** — Stateless orchestrator for Azure Landing Zones, aligned with the Cloud Adoption Framework.

[![Go](https://img.shields.io/badge/Go-1.24-blue)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache_2.0-green)](LICENSE)

## Why lzctl?

Platform teams deploying Azure Landing Zones face a recurring problem: every tenant ends up with ad-hoc Terraform setups, policies applied manually through the portal, and no reliable picture of what is actually deployed versus what is in code.

**lzctl adds the missing orchestration layer** on top of Microsoft's recommended path (Azure Verified Modules). It turns landing zone management into a product: versioned, tested, and deployed via PR — just like application code, but for platform infrastructure.

## Features

| Category | Capabilities |
|----------|-------------|
| **Scaffolding** | Interactive wizard, layered Terraform generation, state backend bootstrap |
| **Blueprints** | Secure-by-default workload blueprints: `paas-secure`, `aks-platform`, `aca-platform`, `avd-secure` |
| **ArgoCD** | First-class GitOps: extension or Helm mode, ApplicationSet, WIF federated credential for source-controller |
| **Validation** | JSON Schema, cross-validation (CIDR overlaps, UUID format, state backend) |
| **Orchestration** | Multi-layer plan/apply in CAF dependency order, automated rollback |
| **Compliance** | CAF 6-discipline audit, compliance scoring, Policy-as-Code lifecycle |
| **Day-2 Ops** | Drift detection, AVM module upgrades, state lifecycle management |
| **Brownfield** | Import existing Azure resources via native `terraform import`; AVM module stubs auto-generated |
| **Workloads** | Add / adopt / remove landing zone subscriptions; CI pipeline matrix auto-updated |

## Architecture

```
lzctl.yaml                          ← Single source of truth (declarative)
    │
    ├── platform/
    │   ├── management-groups/        (1. Resource Organisation)
    │   ├── identity/                 (2. Identity & Access)
    │   ├── management/               (3. Management & Monitoring)
    │   ├── governance/               (4. Azure Policies)
    │   └── connectivity/             (5. Hub-Spoke or vWAN)
    │
    ├── landing-zones/
    │   ├── <zone>/                   (Workload subscription — AVM lz-vending)
    │   └── <zone>/blueprint/         (Optional secure blueprint layer)
    │
    ├── pipelines/                    (CI/CD — GitHub Actions or Azure DevOps)
    └── backend.hcl                   (Shared state backend)
```

Each layer uses **Azure Verified Modules (AVM)** with pinned versions and a **separate Terraform state file**, minimising blast radius.

## Quick Start

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | >= 1.24 | Build |
| Terraform | >= 1.5 | IaC deployment |
| Azure CLI | >= 2.50 | Authentication + Azure operations |
| Git | >= 2.30 | Versioning |
| GitHub CLI | optional | GitHub integration |

### Installation

```bash
# From source
git clone https://github.com/kjourdan1/lzctl.git
cd lzctl
go build -o bin/lzctl .

# Verify
./bin/lzctl version
./bin/lzctl doctor
```

### Deploy a Landing Zone (greenfield)

```bash
# 1. Check prerequisites
lzctl doctor

# 2. Initialise the project (interactive wizard + state backend bootstrap)
lzctl init

# 3. Validate configuration
lzctl validate

# 4. Preview changes
lzctl plan

# 5. Deploy (CAF-ordered layers)
lzctl apply --auto-approve

# 6. Check status
lzctl status
```

### Add a Workload Landing Zone

```bash
# Interactive
lzctl workload add --name app-prod --archetype corp --address-space 10.2.0.0/24

# Adopt an existing subscription (brownfield)
lzctl workload adopt --name legacy-app --subscription <sub-id>

# List
lzctl workload list
```

### Attach a Secure Blueprint

Blueprints generate an opinionated, secure-by-default Terraform layer under
`landing-zones/<name>/blueprint/` and automatically update the CI pipeline matrix.

```bash
# PaaS blueprint — App Service + Key Vault + Private Endpoints
lzctl add-blueprint --landing-zone app-prod --type paas-secure

# Override defaults inline
lzctl add-blueprint --landing-zone app-prod --type paas-secure \
  --set appService.sku=P2v3 \
  --set apim.enabled=true

# AKS platform blueprint with ArgoCD GitOps
lzctl add-blueprint --landing-zone platform --type aks-platform \
  --set argocd.enabled=true \
  --set argocd.mode=helm \
  --set argocd.repoUrl=https://github.com/myorg/gitops
```

**Available blueprint types:**

| Type | Description | Key secure defaults |
|------|-------------|-------------------|
| `paas-secure` | App Service + APIM + Key Vault, all behind Private Endpoints | Public network access disabled, private DNS zones |
| `aks-platform` | Private AKS + ACR + Key Vault + optional ArgoCD | Private cluster, OIDC issuer, Azure Policy add-on, Defender |
| `aca-platform` | Container Apps environment + Key Vault + Private Endpoints | VNet injection, private DNS, no public ingress |
| `avd-secure` | Azure Virtual Desktop — session hosts + FSLogix + Private DNS | Private endpoints, managed identity, Entra ID join |

### Brownfield Import with AVM Stubs

```bash
# Discover and import from an existing resource group
lzctl import --resource-group rg-legacy --layer connectivity

# Import using an audit report, targeting a blueprint layer
lzctl import --from audit-report.json --layer landing-zones/app-prod/blueprint
```

When a resource type has a matching AVM module, lzctl generates an AVM stub
instead of a raw `resource` block:

```hcl
# Auto-generated by lzctl import
module "key_vault" {
  source  = "Azure/avm-res-keyvault-vault/azurerm"
  version = "~> 0.9"

  name                = "kv-app-prod-weu"
  resource_group_name = var.resource_group_name
  location            = var.location
  tenant_id           = var.tenant_id
}
```

## Commands

| Command | Description |
|---------|-------------|
| `lzctl init` | Initialise a new landing zone project |
| `lzctl validate` | Validate the manifest and Terraform configuration |
| `lzctl plan` | Multi-layer `terraform plan` in CAF dependency order |
| `lzctl apply` | Multi-layer `terraform apply` in CAF dependency order |
| `lzctl add-blueprint` | Attach a secure blueprint to a landing zone |
| `lzctl drift` | Detect infrastructure drift |
| `lzctl status` | Project state overview |
| `lzctl rollback` | Rollback layers in reverse CAF order |
| `lzctl audit` | CAF compliance audit of the Azure tenant |
| `lzctl assess` | Project maturity assessment |
| `lzctl select` | Browse the CAF layer catalogue |
| `lzctl upgrade` | Check / apply AVM module updates |
| `lzctl import` | Generate Terraform import blocks (with AVM stubs) |
| `lzctl doctor` | Check prerequisites and environment |
| `lzctl schema` | Export / validate the JSON schema |
| `lzctl docs` | Generate project documentation |
| `lzctl history` | Show deployment history |
| `lzctl state list` | List Terraform state files |
| `lzctl state snapshot` | Snapshot state files |
| `lzctl state health` | Check state backend security posture |
| `lzctl state unlock` | Force-unlock a stuck lease |
| `lzctl policy create` | Scaffold a policy definition |
| `lzctl policy test` | Deploy in DoNotEnforce (audit) mode |
| `lzctl policy verify` | Generate a compliance report |
| `lzctl policy remediate` | Create remediation tasks |
| `lzctl policy deploy` | Switch to Default enforcement |
| `lzctl policy status` | Show policy workflow state |
| `lzctl policy diff` | Compare local vs. deployed policy |
| `lzctl workload add` | Add a landing zone |
| `lzctl workload adopt` | Adopt an existing subscription |
| `lzctl workload list` | List landing zones |
| `lzctl workload remove` | Remove a landing zone |
| `lzctl version` | Show version |

See the [full CLI reference](docs/cli-reference.md) for flags and examples.

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | | `lzctl.yaml` | Config file path |
| `--repo-root` | | `.` | Repository root path |
| `--verbose` | `-v` | `0` | Verbosity level (0–3) |
| `--dry-run` | | `false` | Simulate without modifying Azure |
| `--json` | | `false` | Machine-readable JSON output |
| `--ci` | | `false` | Non-interactive mode (auto-detected via `CI=true`) |

## Configuration — `lzctl.yaml`

```yaml
apiVersion: lzctl/v1
kind: LandingZone

metadata:
  name: contoso-platform
  tenant: aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee
  primaryRegion: westeurope

spec:
  platform:
    managementGroups:
      model: caf-standard        # caf-standard | caf-lite
    connectivity:
      type: hub-spoke            # hub-spoke | vwan | none
      hub:
        region: westeurope
        addressSpace: 10.0.0.0/16
        firewall:
          enabled: true
          sku: Standard
    identity:
      type: workload-identity-federation
    management:
      logAnalytics:
        retentionDays: 90

  stateBackend:
    resourceGroup: rg-contoso-tfstate-weu
    storageAccount: stcontosotfstate
    container: tfstate
    subscription: aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee
    versioning: true             # Blob versioning (audit trail + rollback)
    softDelete: true             # Protection against accidental deletion
    softDeleteDays: 30

  landingZones:
    - name: app-prod
      subscription: bbbbbbbb-cccc-4ddd-8eee-ffffffffffff
      archetype: corp
      addressSpace: 10.1.0.0/24
      connected: true
      blueprint:                 # Optional secure blueprint
        type: paas-secure
        overrides:
          appService:
            sku: P2v3
            runtimeStack: "DOTNET|8.0"

    - name: aks-platform
      subscription: cccccccc-dddd-4eee-8fff-000000000001
      archetype: corp
      connected: true
      blueprint:
        type: aks-platform
        overrides:
          argocd:
            enabled: true
            mode: helm
            repoUrl: https://github.com/myorg/gitops
            targetRevision: main

  cicd:
    platform: github-actions     # github-actions | azure-devops
    branchPolicy:
      mainBranch: main
      requirePR: true
```

## Pipeline Matrix Auto-Update

Every time a blueprint is attached to a landing zone, lzctl automatically updates
`.lzctl/zone-matrix.json` and all CI/CD pipeline files to include the new
`landing-zones/<name>/blueprint` directory in the correct dependency order:

```
landing-zones/app-prod           → terraform apply
landing-zones/app-prod/blueprint  → terraform apply  (after parent zone)
```

Both GitHub Actions and Azure DevOps pipelines are updated in a single `add-blueprint` call.

## Design Principles

| Principle | Description |
|-----------|-------------|
| **Stateless CLI** | lzctl stores no local state; everything lives in `lzctl.yaml` + Git |
| **Native Terraform** | Generated code works with standard `terraform plan/apply` — no runtime dependency on lzctl |
| **Azure Verified Modules** | Microsoft-maintained AVM modules, pinned versions |
| **GitOps** | PRs trigger `plan`; merges trigger `apply` |
| **State as First-Class** | Versioning, soft delete, health checks, automatic snapshots |
| **Layer Isolation** | One state file per CAF layer — minimal blast radius |
| **Secure-by-default Blueprints** | Every blueprint enforces private networking, RBAC, and encryption; overrides are explicit opt-in |

## Documentation

| Document | Description |
|----------|-------------|
| [CLI Reference](docs/cli-reference.md) | Full command and flag reference |
| [Architecture](docs/architecture.md) | Architectural decision records (ADRs) |
| [Contributing](CONTRIBUTING.md) | Contribution guide |
| [State Management](docs/operations/state-management.md) | State lifecycle operations |
| [CI Headless](docs/operations/ci-headless.md) | Run init/validate/plan/apply without interaction |
| [Rollback](docs/operations/rollback.md) | Rollback procedures |
| [Drift Response](docs/operations/drift-response.md) | Responding to infrastructure drift |
| [Testing](TESTING.md) | Testing guide |

## License

[Apache License 2.0](LICENSE)
