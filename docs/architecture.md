# Architecture Document — lzctl

> Version: 1.0
> Date: 2026-02-18
> Input: [PRD.md](./PRD.md)
> Author: Killian Jourdan

---

## Table of Contents

1. [Tech Stack Selection](#1-tech-stack-selection)
2. [System Design](#2-system-design)
3. [Project Structure](#3-project-structure)
4. [Component Architecture](#4-component-architecture)
5. [Key Interfaces & Data Models](#5-key-interfaces--data-models)
6. [Template Engine Design](#6-template-engine-design)
7. [Brownfield Engine Design](#7-brownfield-engine-design)
8. [CI/CD Pipeline Templates](#8-cicd-pipeline-templates)
9. [Coding Standards](#9-coding-standards)
10. [Architecture Decision Records](#10-architecture-decision-records)
11. [Risk Assessment](#11-risk-assessment)

---

## 1. Tech Stack Selection

### 1.1 CLI Core

| Component | Choice | Version | Justification |
|-----------|--------|---------|---------------|
| **Language** | Go | 1.24+ | Single binary, cross-platform, Cobra ecosystem, standard in infra tooling (kubectl, terraform, gh) |
| **CLI Framework** | [cobra](https://github.com/spf13/cobra) | v1.8+ | Industry standard for Go CLIs; built-in help, completions, subcommands |
| **Config Parsing** | [viper](https://github.com/spf13/viper) | v1.18+ | YAML/JSON/TOML config, env var binding, pairs naturally with Cobra |
| **Interactive Prompts** | [survey/v2](https://github.com/go-survey/survey) or [huh](https://github.com/charmbracelet/huh) | latest | Charmbracelet `huh` preferred for modern TUI; fallback to survey for simplicity |
| **Terminal Output** | [lipgloss](https://github.com/charmbracelet/lipgloss) + [log](https://github.com/charmbracelet/log) | latest | Styled output, respects `NO_COLOR`, consistent UX |
| **JSON Schema** | [gojsonschema](https://github.com/xeipuuv/gojsonschema) | v1 | Validate `lzctl.yaml` against embedded JSON Schema |
| **Template Engine** | Go `text/template` + custom helpers | stdlib | No external dependency; sufficient for HCL/YAML generation |
| **Embedded Files** | Go `embed` | stdlib | Templates and schemas compiled into the binary |
| **Testing** | `testing` + [testify](https://github.com/stretchr/testify) | latest | Standard assertions + mocking |

### 1.2 Build & Release

| Component | Choice | Justification |
|-----------|--------|---------------|
| **Build** | [GoReleaser](https://goreleaser.com/) | Cross-compilation, checksums, Homebrew tap, changelog generation |
| **CI/CD** | GitHub Actions | Dogfooding + standard for OSS Go projects |
| **Linting** | [golangci-lint](https://golangci-lint.run/) | Aggregated linter with sensible defaults |
| **Versioning** | Semantic Versioning (semver) | Conventional commits → automatic version bump |

### 1.3 External Runtime Dependencies (user must install)

| Dependency | Minimum Version | Used By |
|-----------|----------------|---------|
| `terraform` | >= 1.5.0 (recommended 1.9+) | Plan, apply, validate, import blocks |
| `az` CLI | >= 2.50.0 | Authentication, bootstrap, audit |
| `git` | >= 2.30.0 | Checked by doctor (user manages git workflow) |

---

## 2. System Design

### 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         lzctl CLI Binary                        │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │  Cobra   │  │  Config   │  │ Template │  │   Brownfield  │  │
│  │  Router  │→ │  Loader   │→ │  Engine  │  │    Engine     │  │
│  │          │  │ (Viper)   │  │          │  │               │  │
│  └──────────┘  └──────────┘  └──────────┘  └───────────────┘  │
│       │              │             │               │            │
│       │         ┌────▼────┐  ┌────▼────┐    ┌─────▼──────┐    │
│       │         │  Schema │  │   File  │    │   Azure    │    │
│       │         │Validator│  │ Writer  │    │  Scanner   │    │
│       │         └─────────┘  └─────────┘    └────────────┘    │
│       │              │             │               │            │
│  ┌────▼────────────────────────────────────────────────────┐   │
│  │                    Executor Layer                        │   │
│  │  ┌──────────┐  ┌──────────────┐  ┌──────────────────┐  │   │
│  │  │ Terraform│  │  az CLI      │  │   Git (read-only)│  │   │
│  │  │ Runner   │  │  Runner      │  │   Inspector      │  │   │
│  │  └──────────┘  └──────────────┘  └──────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Embedded Assets (go:embed)                  │   │
│  │  ┌───────────┐  ┌──────────┐  ┌─────────────────────┐  │   │
│  │  │ Templates │  │  JSON    │  │  Policy Definitions  │  │   │
│  │  │ (HCL/YAML)│  │  Schema  │  │  (CAF defaults)     │  │   │
│  │  └───────────┘  └──────────┘  └─────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
   ┌───────────┐      ┌────────────┐      ┌──────────────┐
   │ Generated  │      │   Azure    │      │  Terraform   │
   │ Repo Files │      │   Tenant   │      │    State     │
   │ (on disk)  │      │  (via az)  │      │  (Storage)   │
   └───────────┘      └────────────┘      └──────────────┘
```

### 2.2 Component Interaction — `lzctl init` Flow

```
User
  │
  ▼
CobraRouter.Execute("init")
  │
  ├─ flags: --config? --dry-run? --force?
  │
  ▼
InitCommand.Run()
  │
  ├── [if --config] ConfigLoader.LoadFromFile(path)
  │
  ├── [else] InteractiveWizard.Run()
  │     │
  │     ├── Prompt: project name
  │     ├── Prompt: tenant ID
  │     ├── Prompt: CI/CD platform
  │     ├── Prompt: MG model
  │     ├── Prompt: connectivity model
  │     ├── Prompt: regions
  │     ├── Prompt: identity model
  │     └── Prompt: state backend
  │     │
  │     └── Returns: InitConfig struct
  │
  ├── SchemaValidator.Validate(config)
  │     └── [fail] → error + exit 1
  │
  ├── [if bootstrap requested] BootstrapRunner.Run(config)
  │     │
  │     ├── az group create ...
  │     ├── az storage account create ...
  │     ├── az storage container create ...
  │     ├── az identity create ...
  │     ├── az role assignment create ...
  │     └── az identity federated-credential create ...
  │
  ├── TemplateEngine.Render(config)
  │     │
  │     ├── RenderManifest()         → lzctl.yaml
  │     ├── RenderLayer("mgmt-groups", config) → platform/management-groups/*.tf
  │     ├── RenderLayer("identity", config)    → platform/identity/*.tf
  │     ├── RenderLayer("management", config)  → platform/management/*.tf
  │     ├── RenderLayer("governance", config)  → platform/governance/*.tf
  │     ├── RenderLayer("connectivity", config)→ platform/connectivity/*.tf
  │     ├── RenderPipelines(config)  → .github/ or .azuredevops/
  │     ├── RenderBackendConfig()    → backend.tf / backend.hcl
  │     ├── RenderReadme(config)     → README.md
  │     └── RenderGitignore()        → .gitignore
  │
  ├── [if --dry-run] → print file list + exit 0
  │
  ├── FileWriter.WriteAll(renderedFiles, targetDir)
  │
  └── Print success summary + next steps
```

### 2.3 Component Interaction — `lzctl audit` Flow

```
User
  │
  ▼
CobraRouter.Execute("audit")
  │
  ├─ flags: --scope? --json? --output?
  │
  ▼
AuditCommand.Run()
  │
  ├── AzureScanner.Scan(scope)
  │     │
  │     ├── ScanManagementGroups()   → []ManagementGroup
  │     ├── ScanSubscriptions()      → []Subscription
  │     ├── ScanPolicyAssignments()  → []PolicyAssignment
  │     ├── ScanRBAC()               → []RoleAssignment
  │     ├── ScanNetworking()         → []VNet, []Peering
  │     ├── ScanDiagnostics()        → []DiagSetting
  │     └── ScanDefender()           → DefenderStatus
  │     │
  │     └── Returns: TenantSnapshot struct
  │
  ├── ComplianceEngine.Evaluate(snapshot)
  │     │
  │     ├── CheckManagementGroupHierarchy(snapshot.MGs)
  │     ├── CheckPolicyCompliance(snapshot.Policies)
  │     ├── CheckRBACHygiene(snapshot.RBAC)
  │     ├── CheckConnectivityPatterns(snapshot.VNets)
  │     ├── CheckLoggingAndMonitoring(snapshot.Diagnostics)
  │     └── CheckSecurityPosture(snapshot.Defender)
  │     │
  │     └── Returns: AuditReport struct (findings + score)
  │
  ├── [if --json] → JSONRenderer.Render(report) → stdout or file
  │
  ├── [else] → MarkdownRenderer.Render(report) → stdout or file
  │
  └── Print summary (score, critical findings count)
```

---

## 3. Project Structure

```
lzctl/
├── .github/
│   └── workflows/
│       ├── ci.yml                    # lint + test + build on PR
│       ├── release.yml               # GoReleaser on tag push
│       └── integration-test.yml      # weekly, deploys to test tenant
│
├── cmd/                              # Cobra command definitions
│   ├── root.go                       # Root command, global flags
│   ├── version.go                    # lzctl version
│   ├── doctor.go                     # lzctl doctor
│   ├── init.go                       # lzctl init
│   ├── validate.go                   # lzctl validate
│   ├── plan.go                       # lzctl plan
│   ├── apply.go                      # lzctl apply
│   ├── workload_add.go                # lzctl workload add
│   ├── workload_adopt.go              # lzctl workload adopt
│   ├── workload_list.go               # lzctl workload list
│   ├── workload_remove.go             # lzctl workload remove
│   ├── audit.go                      # lzctl audit
│   ├── import.go                     # lzctl import
│   ├── drift.go                      # lzctl drift
│   ├── upgrade.go                    # lzctl upgrade
│   └── status.go                     # lzctl status
│
├── internal/                         # Private packages (not importable)
│   ├── config/
│   │   ├── schema.go                 # Go structs matching lzctl.yaml
│   │   ├── loader.go                 # Load + parse lzctl.yaml
│   │   ├── validator.go              # JSON Schema validation
│   │   └── defaults.go               # Default values for optional fields
│   │
│   ├── wizard/
│   │   ├── init_wizard.go            # Interactive init prompts
│   │   └── import_wizard.go          # Interactive import selection
│   │
│   ├── template/
│   │   ├── engine.go                 # Template rendering orchestrator
│   │   ├── helpers.go                # Template helper functions (cidr, naming, etc.)
│   │   └── writer.go                 # File writer (dry-run aware)
│   │
│   ├── terraform/
│   │   ├── runner.go                 # Execute terraform commands
│   │   ├── plan_parser.go            # Parse terraform plan output
│   │   ├── layer_order.go            # Layer dependency ordering
│   │   └── state.go                  # State backend routing
│   │
│   ├── azure/
│   │   ├── cli.go                    # az CLI command executor
│   │   ├── bootstrap.go              # State backend bootstrap logic
│   │   ├── scanner.go                # Tenant scanning for audit
│   │   ├── management_groups.go      # MG hierarchy scanner
│   │   ├── policies.go               # Policy assignment scanner
│   │   ├── networking.go             # VNet/peering scanner
│   │   ├── rbac.go                   # RBAC scanner
│   │   ├── diagnostics.go            # Diagnostic settings scanner
│   │   └── defender.go               # Defender for Cloud scanner
│   │
│   ├── audit/
│   │   ├── compliance.go             # CAF compliance rules engine
│   │   ├── rules/                    # Individual compliance rules
│   │   │   ├── management_groups.go
│   │   │   ├── policies.go
│   │   │   ├── rbac.go
│   │   │   ├── connectivity.go
│   │   │   ├── logging.go
│   │   │   └── security.go
│   │   ├── scoring.go                # CAF alignment score calculation
│   │   ├── report.go                 # AuditReport data model
│   │   ├── markdown_renderer.go      # Markdown report output
│   │   └── json_renderer.go          # JSON report output
│   │
│   ├── importer/
│   │   ├── discovery.go              # Discover importable resources
│   │   ├── hcl_generator.go          # Generate HCL for existing resources
│   │   ├── import_block.go           # Generate terraform import blocks
│   │   └── resource_mapping.go       # Azure resource type → AVM module mapping
│   │
│   ├── doctor/
│   │   ├── checks.go                 # All prerequisite checks
│   │   └── reporter.go               # Check results formatter
│   │
│   ├── drift/
│   │   ├── detector.go               # Drift detection logic
│   │   └── reporter.go               # Drift report formatter
│   │
│   ├── upgrade/
│   │   ├── registry.go               # Query Terraform registry for versions
│   │   ├── updater.go                # Update module refs in HCL files
│   │   └── changelog.go              # Version diff summary
│   │
│   └── output/
│       ├── logger.go                 # Structured logging (charmbracelet/log)
│       ├── spinner.go                # Progress indicators
│       └── json.go                   # JSON output formatter
│
├── templates/                        # go:embed source — all templates
│   ├── manifest/
│   │   └── lzctl.yaml.tmpl          # lzctl.yaml template
│   │
│   ├── platform/
│   │   ├── management-groups/
│   │   │   ├── caf-standard/
│   │   │   │   ├── main.tf.tmpl
│   │   │   │   ├── variables.tf.tmpl
│   │   │   │   └── terraform.tfvars.tmpl
│   │   │   └── caf-lite/
│   │   │       ├── main.tf.tmpl
│   │   │       ├── variables.tf.tmpl
│   │   │       └── terraform.tfvars.tmpl
│   │   │
│   │   ├── connectivity/
│   │   │   ├── hub-spoke-fw/
│   │   │   │   ├── main.tf.tmpl
│   │   │   │   ├── variables.tf.tmpl
│   │   │   │   └── terraform.tfvars.tmpl
│   │   │   ├── hub-spoke-nva/
│   │   │   │   └── ...
│   │   │   └── vwan/
│   │   │       └── ...
│   │   │
│   │   ├── identity/
│   │   │   ├── main.tf.tmpl
│   │   │   └── variables.tf.tmpl
│   │   │
│   │   ├── management/
│   │   │   ├── main.tf.tmpl
│   │   │   ├── variables.tf.tmpl
│   │   │   └── terraform.tfvars.tmpl
│   │   │
│   │   └── governance/
│   │       ├── main.tf.tmpl
│   │       ├── variables.tf.tmpl
│   │       ├── terraform.tfvars.tmpl
│   │       └── policies/
│   │           ├── caf-default.tf.tmpl
│   │           └── ...
│   │
│   ├── landing-zones/
│   │   ├── corp/
│   │   │   ├── main.tf.tmpl
│   │   │   ├── variables.tf.tmpl
│   │   │   └── terraform.tfvars.tmpl
│   │   ├── online/
│   │   │   └── ...
│   │   └── sandbox/
│   │       └── ...
│   │
│   ├── pipelines/
│   │   ├── github/
│   │   │   ├── validate.yml.tmpl
│   │   │   ├── deploy.yml.tmpl
│   │   │   └── drift.yml.tmpl
│   │   └── azuredevops/
│   │       ├── validate.yml.tmpl
│   │       ├── deploy.yml.tmpl
│   │       └── drift.yml.tmpl
│   │
│   ├── shared/
│   │   ├── backend.tf.tmpl
│   │   ├── backend.hcl.tmpl
│   │   ├── providers.tf.tmpl
│   │   ├── gitignore.tmpl
│   │   └── readme.md.tmpl
│   │
│   └── audit/
│       └── report.md.tmpl           # Markdown audit report template
│
├── schemas/                          # go:embed source — JSON schemas
│   └── lzctl-v1.schema.json         # JSON Schema for lzctl.yaml
│
├── docs/
│   ├── commands/                     # Per-command reference docs
│   ├── architecture/                 # This document + ADRs
│   └── examples/                     # Example configs and repos
│
├── test/
│   ├── fixtures/                     # Test fixtures (sample configs, Azure responses)
│   │   ├── configs/
│   │   │   ├── standard-hub-spoke.yaml
│   │   │   ├── lite-no-connectivity.yaml
│   │   │   └── brownfield-audit.json
│   │   └── azure/
│   │       ├── management-groups.json
│   │       ├── subscriptions.json
│   │       └── policies.json
│   ├── integration/                  # Integration tests (require Azure)
│   │   ├── init_test.go
│   │   ├── audit_test.go
│   │   └── validate_test.go
│   └── e2e/                          # End-to-end tests (full deploy)
│       └── deploy_test.go
│
├── main.go                           # Entry point
├── go.mod
├── go.sum
├── .goreleaser.yml                   # GoReleaser config
├── .golangci.yml                     # Linter config
├── Makefile                          # Dev convenience targets
├── LICENSE                           # Apache 2.0
├── README.md
├── CONTRIBUTING.md
└── CHANGELOG.md
```

---

## 4. Component Architecture

### 4.1 Component Responsibilities

| Component | Package | Responsibility | Dependencies |
|-----------|---------|---------------|-------------|
| **Cobra Router** | `cmd/` | Parse CLI args, route to command handlers, global flags | cobra, viper |
| **Config Loader** | `internal/config/` | Load, parse, validate, and provide defaults for `lzctl.yaml` | viper, gojsonschema |
| **Interactive Wizard** | `internal/wizard/` | Collect user input via TUI prompts, produce config structs | huh or survey |
| **Template Engine** | `internal/template/` | Render Go templates to HCL/YAML/Markdown files | go `text/template`, embed |
| **File Writer** | `internal/template/` | Write rendered files to disk, handle dry-run mode | stdlib |
| **Terraform Runner** | `internal/terraform/` | Execute terraform CLI commands, capture output | os/exec |
| **Azure CLI Runner** | `internal/azure/` | Execute az CLI commands, parse JSON output | os/exec |
| **Bootstrap Runner** | `internal/azure/` | Create state backend resources via az CLI | Azure CLI Runner |
| **Azure Scanner** | `internal/azure/` | Collect tenant inventory for audit | Azure CLI Runner |
| **Compliance Engine** | `internal/audit/` | Evaluate tenant snapshot against CAF rules | Azure Scanner output |
| **Import Engine** | `internal/importer/` | Discover resources, generate HCL + import blocks | Azure CLI Runner |
| **Doctor** | `internal/doctor/` | Check prerequisites, versions, permissions | os/exec |
| **Drift Detector** | `internal/drift/` | Parse terraform plan for changes | Terraform Runner |
| **Upgrade Checker** | `internal/upgrade/` | Query registry, update module versions | net/http |
| **Output** | `internal/output/` | Logging, spinners, JSON formatting | charmbracelet/log, lipgloss |

### 4.2 Dependency Graph

```
cmd/* (command handlers)
  │
  ├── internal/config      (all commands that read lzctl.yaml)
  ├── internal/wizard      (init, import)
  ├── internal/template    (init, workload add)
  ├── internal/terraform   (plan, apply, validate, drift)
  ├── internal/azure       (init bootstrap, audit, import)
  ├── internal/audit       (audit)
  ├── internal/importer    (import)
  ├── internal/doctor      (doctor)
  ├── internal/drift       (drift)
  ├── internal/upgrade     (upgrade)
  └── internal/output      (all commands)

internal/audit
  └── internal/azure (scanner)

internal/importer
  └── internal/azure (CLI runner)
  └── internal/template (HCL generation)

internal/terraform
  └── (no internal deps — wraps CLI only)

internal/azure
  └── (no internal deps — wraps CLI only)
```

**Design rule:** No circular dependencies. `internal/azure` and `internal/terraform` are leaf packages that wrap external CLIs. Higher-level packages compose them.

---

## 5. Key Interfaces & Data Models

### 5.1 Core Config Structs (Go)

```go
// internal/config/schema.go

package config

// LZConfig is the root struct matching lzctl.yaml
type LZConfig struct {
    APIVersion string   `yaml:"apiVersion" json:"apiVersion"` // "lzctl/v1"
    Kind       string   `yaml:"kind" json:"kind"`             // "LandingZone"
    Metadata   Metadata `yaml:"metadata" json:"metadata"`
    Spec       Spec     `yaml:"spec" json:"spec"`
}

type Metadata struct {
    Name            string `yaml:"name" json:"name"`
    Tenant          string `yaml:"tenant" json:"tenant"`
    PrimaryRegion   string `yaml:"primaryRegion" json:"primaryRegion"`
    SecondaryRegion string `yaml:"secondaryRegion,omitempty" json:"secondaryRegion,omitempty"`
}

type Spec struct {
    Platform     Platform     `yaml:"platform" json:"platform"`
    Governance   Governance   `yaml:"governance" json:"governance"`
    Naming       Naming       `yaml:"naming" json:"naming"`
    StateBackend StateBackend `yaml:"stateBackend" json:"stateBackend"`
    LandingZones []LandingZone `yaml:"landingZones" json:"landingZones"`
    CICD         CICD         `yaml:"cicd" json:"cicd"`
}

type Platform struct {
    ManagementGroups ManagementGroupsConfig `yaml:"managementGroups" json:"managementGroups"`
    Connectivity     ConnectivityConfig     `yaml:"connectivity" json:"connectivity"`
    Identity         IdentityConfig         `yaml:"identity" json:"identity"`
    Management       ManagementConfig       `yaml:"management" json:"management"`
}

type ManagementGroupsConfig struct {
    Model    string   `yaml:"model" json:"model"`       // "caf-standard" | "caf-lite"
    Disabled []string `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

type ConnectivityConfig struct {
    Type string    `yaml:"type" json:"type"` // "hub-spoke" | "vwan" | "none"
    Hub  *HubConfig `yaml:"hub,omitempty" json:"hub,omitempty"`
}

type HubConfig struct {
    Region       string          `yaml:"region" json:"region"`
    AddressSpace string          `yaml:"addressSpace" json:"addressSpace"`
    Firewall     FirewallConfig  `yaml:"firewall" json:"firewall"`
    DNS          DNSConfig       `yaml:"dns" json:"dns"`
    VPNGateway   GatewayConfig   `yaml:"vpnGateway" json:"vpnGateway"`
    ERGateway    GatewayConfig   `yaml:"expressRouteGateway" json:"expressRouteGateway"`
}

type FirewallConfig struct {
    Enabled     bool   `yaml:"enabled" json:"enabled"`
    SKU         string `yaml:"sku,omitempty" json:"sku,omitempty"`
    ThreatIntel string `yaml:"threatIntel,omitempty" json:"threatIntel,omitempty"`
}

type DNSConfig struct {
    PrivateResolver bool     `yaml:"privateResolver" json:"privateResolver"`
    Forwarders      []string `yaml:"forwarders,omitempty" json:"forwarders,omitempty"`
}

type GatewayConfig struct {
    Enabled bool   `yaml:"enabled" json:"enabled"`
    SKU     string `yaml:"sku,omitempty" json:"sku,omitempty"`
}

type IdentityConfig struct {
    Type        string `yaml:"type" json:"type"` // "workload-identity-federation" | "sp-federated" | "sp-secret"
    ClientID    string `yaml:"clientId,omitempty" json:"clientId,omitempty"`
    PrincipalID string `yaml:"principalId,omitempty" json:"principalId,omitempty"`
}

type ManagementConfig struct {
    LogAnalytics      LogAnalyticsConfig `yaml:"logAnalytics" json:"logAnalytics"`
    AutomationAccount bool               `yaml:"automationAccount" json:"automationAccount"`
    Defender          DefenderConfig     `yaml:"defenderForCloud" json:"defenderForCloud"`
}

type LogAnalyticsConfig struct {
    RetentionDays int      `yaml:"retentionDays" json:"retentionDays"`
    Solutions     []string `yaml:"solutions,omitempty" json:"solutions,omitempty"`
}

type DefenderConfig struct {
    Enabled bool     `yaml:"enabled" json:"enabled"`
    Plans   []string `yaml:"plans" json:"plans"`
}

type Governance struct {
    Policies PolicyConfig `yaml:"policies" json:"policies"`
}

type PolicyConfig struct {
    Assignments []string `yaml:"assignments" json:"assignments"`
    Custom      []string `yaml:"custom,omitempty" json:"custom,omitempty"`
}

type Naming struct {
    Convention string            `yaml:"convention" json:"convention"` // "caf"
    Overrides  map[string]string `yaml:"overrides,omitempty" json:"overrides,omitempty"`
}

type StateBackend struct {
    ResourceGroup  string `yaml:"resourceGroup" json:"resourceGroup"`
    StorageAccount string `yaml:"storageAccount" json:"storageAccount"`
    Container      string `yaml:"container" json:"container"`
    Subscription   string `yaml:"subscription" json:"subscription"`
}

type LandingZone struct {
    Name         string            `yaml:"name" json:"name"`
    Subscription string            `yaml:"subscription" json:"subscription"`
    Archetype    string            `yaml:"archetype" json:"archetype"` // "corp" | "online" | "sandbox"
    AddressSpace string            `yaml:"addressSpace" json:"addressSpace"`
    Connected    bool              `yaml:"connected" json:"connected"`
    Tags         map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

type CICD struct {
    Platform     string       `yaml:"platform" json:"platform"` // "github-actions" | "azure-devops"
    Repository   string       `yaml:"repository,omitempty" json:"repository,omitempty"`
    BranchPolicy BranchPolicy `yaml:"branchPolicy" json:"branchPolicy"`
}

type BranchPolicy struct {
    MainBranch string `yaml:"mainBranch" json:"mainBranch"` // default: "main"
    RequirePR  bool   `yaml:"requirePR" json:"requirePR"`   // default: true
}
```

### 5.2 Audit Data Models

```go
// internal/audit/report.go

package audit

type TenantSnapshot struct {
    TenantID         string
    ManagementGroups []ManagementGroup
    Subscriptions    []Subscription
    PolicyAssignments []PolicyAssignment
    RoleAssignments  []RoleAssignment
    VNets            []VirtualNetwork
    Peerings         []VNetPeering
    DiagSettings     []DiagnosticSetting
    DefenderStatus   DefenderStatus
    ScannedAt        time.Time
}

type AuditReport struct {
    TenantID  string           `json:"tenantId"`
    ScannedAt time.Time        `json:"scannedAt"`
    Score     AuditScore       `json:"score"`
    Findings  []AuditFinding   `json:"findings"`
    Summary   AuditSummary     `json:"summary"`
}

type AuditScore struct {
    Overall    int `json:"overall"`    // 0-100
    Governance int `json:"governance"`
    Identity   int `json:"identity"`
    Management int `json:"management"`
    Connectivity int `json:"connectivity"`
    Security   int `json:"security"`
}

type AuditFinding struct {
    ID           string         `json:"id"`           // e.g., "GOV-001"
    Discipline   string         `json:"discipline"`   // governance|identity|management|connectivity|security
    Severity     string         `json:"severity"`     // critical|high|medium|low
    Title        string         `json:"title"`
    CurrentState string         `json:"currentState"`
    ExpectedState string        `json:"expectedState"`
    Remediation  string         `json:"remediation"`
    AutoFixable  bool           `json:"autoFixable"`  // can lzctl import fix this?
    Resources    []ResourceRef  `json:"resources,omitempty"`
}

type ResourceRef struct {
    ResourceID   string `json:"resourceId"`
    ResourceType string `json:"resourceType"`
    Name         string `json:"name"`
}

type AuditSummary struct {
    TotalFindings  int `json:"totalFindings"`
    Critical       int `json:"critical"`
    High           int `json:"high"`
    Medium         int `json:"medium"`
    Low            int `json:"low"`
    AutoFixable    int `json:"autoFixable"`
}
```

### 5.3 Core Interfaces

```go
// internal/terraform/runner.go

// TerraformRunner abstracts terraform CLI execution
type TerraformRunner interface {
    Init(workDir string, backendConfig string) error
    Validate(workDir string) error
    Plan(workDir string, opts PlanOptions) (*PlanResult, error)
    Apply(workDir string, opts ApplyOptions) error
    Version() (string, error)
}

type PlanOptions struct {
    Out         string // save plan to file
    Parallelism int
    Targets     []string
}

type PlanResult struct {
    HasChanges bool
    Add        int
    Change     int
    Destroy    int
    RawOutput  string
}

// internal/azure/cli.go

// AzureCLI abstracts az CLI execution
type AzureCLI interface {
    Run(args ...string) ([]byte, error)      // raw execution
    RunJSON(args ...string) (any, error)     // parse JSON output
    GetCurrentTenant() (string, error)
    GetCurrentSubscription() (string, error)
    IsLoggedIn() bool
}

// internal/template/engine.go

// TemplateEngine renders templates to files
type TemplateEngine interface {
    RenderManifest(cfg *config.LZConfig) ([]RenderedFile, error)
    RenderLayer(layer string, cfg *config.LZConfig) ([]RenderedFile, error)
    RenderPipelines(cfg *config.LZConfig) ([]RenderedFile, error)
    RenderAll(cfg *config.LZConfig) ([]RenderedFile, error)
}

type RenderedFile struct {
    Path    string // relative path from repo root
    Content []byte
}

// internal/audit/compliance.go

// ComplianceRule evaluates one aspect of CAF compliance
type ComplianceRule interface {
    ID() string
    Discipline() string
    Evaluate(snapshot *TenantSnapshot) []AuditFinding
}
```

---

## 6. Template Engine Design

### 6.1 Strategy

Templates use Go's `text/template` (not `html/template` — we generate HCL, not HTML). Templates are embedded in the binary via `go:embed`.

Each template has access to the full `LZConfig` struct plus a set of custom helper functions.

### 6.2 Template Helpers

```go
// internal/template/helpers.go

var funcMap = template.FuncMap{
    // Naming
    "cafName":       cafResourceName,     // cafName "rg" "platform" "weu" → "rg-platform-weu"
    "slugify":       slugify,             // "My Project" → "my-project"
    "storageAccName": storageAccountName, // truncate + sanitize to 24 chars

    // Networking
    "cidrSubnet":    cidrSubnet,          // cidrSubnet "10.0.0.0/16" 24 0 → "10.0.0.0/24"
    "cidrHost":      cidrHost,            // cidrHost "10.0.0.0/24" 4 → "10.0.0.4"

    // Conditionals
    "enabled":       func(b bool) string { if b { return "true" }; return "false" },
    "when":          func(cond bool, val string) string { if cond { return val }; return "" },

    // Formatting
    "indent":        indent,              // indent 4 "block" → "    block" (each line)
    "quote":         func(s string) string { return `"` + s + `"` },
    "join":          strings.Join,
    "toJSON":        toJSON,
    "toYAML":        toYAML,

    // Azure
    "regionShort":   regionShortCode,     // "westeurope" → "weu", "northeurope" → "neu"
    "regionDisplay": regionDisplayName,   // "westeurope" → "West Europe"
}
```

### 6.3 Template Example — Management Groups (CAF Standard)

```hcl
{{/* templates/platform/management-groups/caf-standard/main.tf.tmpl */}}

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }

  backend "azurerm" {}
}

provider "azurerm" {
  features {}
  tenant_id = "{{ .Metadata.Tenant }}"
}

module "management_groups" {
  source  = "Azure/avm-ptn-alz/azurerm"
  version = "0.11.0"

  architecture_definition_name = "alz"
  location                     = "{{ .Metadata.PrimaryRegion }}"

  {{- if .Spec.Platform.ManagementGroups.Disabled }}
  management_group_configuration = {
    {{- range .Spec.Platform.ManagementGroups.Disabled }}
    {{ . }} = {
      enabled = false
    }
    {{- end }}
  }
  {{- end }}
}
```

### 6.4 Generated Output Guarantees

- All generated `.tf` files pass `terraform fmt`
- All generated `.tf` files pass `terraform validate` (given correct provider config)
- All generated YAML passes schema validation
- No lzctl-specific markers or magic comments — pure standard Terraform
- Comments explain each section's purpose for readability

---

## 7. Brownfield Engine Design

### 7.1 Azure Scanner

The scanner uses `az` CLI (not Azure SDK directly) to maintain the same dependency model as the rest of lzctl. Each scan function runs one or more `az` commands and parses JSON output.

```go
// internal/azure/scanner.go

type Scanner struct {
    cli    AzureCLI
    scope  string  // management group ID or "" for tenant root
}

// Scan methods execute az CLI commands:
//   az account management-group list
//   az account list
//   az policy assignment list --scope <mg>
//   az role assignment list --scope <mg> --include-inherited
//   az network vnet list --subscription <sub>
//   az monitor diagnostic-settings list --resource <id>
//   az security pricing list --subscription <sub>
```

**Performance optimization:** Run subscription-scoped queries in parallel (bounded concurrency pool, max 5 concurrent `az` invocations) to keep audit under 5 minutes for 100 subscriptions.

### 7.2 Compliance Rules

Each rule is a standalone struct implementing `ComplianceRule`. Rules are registered in a global registry and evaluated by the compliance engine.

```go
// Example rule: Check management group hierarchy matches CAF

type MGHierarchyRule struct{}

func (r *MGHierarchyRule) ID() string         { return "GOV-001" }
func (r *MGHierarchyRule) Discipline() string  { return "governance" }

func (r *MGHierarchyRule) Evaluate(s *TenantSnapshot) []AuditFinding {
    // Compare s.ManagementGroups against expected CAF hierarchy
    // Return findings for missing/extra/misplaced MGs
}
```

**Initial rule set (MVP):**

| ID | Discipline | Rule |
|----|-----------|------|
| GOV-001 | Governance | MG hierarchy matches CAF model |
| GOV-002 | Governance | Subscriptions placed in correct MGs (not root) |
| GOV-003 | Governance | CAF default policies assigned at root |
| GOV-004 | Governance | No subscriptions in Tenant Root Group |
| IDT-001 | Identity | No Owner assignments at high scopes (use PIM) |
| IDT-002 | Identity | Service principals have federated credentials |
| MGT-001 | Management | Log Analytics workspace exists |
| MGT-002 | Management | Diagnostic settings configured on subscriptions |
| MGT-003 | Management | Defender for Cloud enabled (at least Servers) |
| NET-001 | Connectivity | Hub VNet exists (if hub-spoke expected) |
| NET-002 | Connectivity | Peering between hub and spokes is established |
| NET-003 | Connectivity | No overlapping address spaces |
| SEC-001 | Security | Storage accounts enforce TLS 1.2+ |
| SEC-002 | Security | Key Vaults have soft delete enabled |

### 7.3 Import Engine

The import engine maps Azure resource types to Terraform resource types and generates both `import` blocks and HCL configuration.

```go
// internal/importer/resource_mapping.go

var resourceMapping = map[string]ResourceMapping{
    "Microsoft.Resources/resourceGroups": {
        TerraformType: "azurerm_resource_group",
        AVMModule:     "",  // no AVM module for RGs, use native resource
    },
    "Microsoft.Network/virtualNetworks": {
        TerraformType: "azurerm_virtual_network",
        AVMModule:     "Azure/avm-res-network-virtualnetwork/azurerm",
    },
    "Microsoft.Network/networkSecurityGroups": {
        TerraformType: "azurerm_network_security_group",
        AVMModule:     "Azure/avm-res-network-networksecuritygroup/azurerm",
    },
    // ... more mappings
}
```

**Generated import block example:**

```hcl
import {
  to = azurerm_resource_group.imported["rg-networking-weu"]
  id = "/subscriptions/xxx/resourceGroups/rg-networking-weu"
}

resource "azurerm_resource_group" "imported" {
  for_each = var.imported_resource_groups

  name     = each.value.name
  location = each.value.location
  tags     = each.value.tags
}
```

---

## 8. CI/CD Pipeline Templates

### 8.1 GitHub Actions — Validate on PR

```yaml
# templates/pipelines/github/validate.yml.tmpl

name: "Landing Zone — Validate"
on:
  pull_request:
    branches: [{{ .Spec.CICD.BranchPolicy.MainBranch }}]
    paths:
      - "platform/**"
      - "landing-zones/**"
      - "lzctl.yaml"

permissions:
  id-token: write
  contents: read
  pull-requests: write

env:
  ARM_TENANT_ID: "{{ .Metadata.Tenant }}"
  ARM_SUBSCRIPTION_ID: "{{ .Spec.StateBackend.Subscription }}"
  ARM_CLIENT_ID: "{{ `${{ secrets.AZURE_CLIENT_ID }}` }}"

jobs:
  validate-and-plan:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: true
      matrix:
        layer:
          - platform/management-groups
          - platform/identity
          - platform/management
          - platform/governance
          - platform/connectivity
{{- range .Spec.LandingZones }}
          - landing-zones/{{ .Name }}
{{- end }}
      max-parallel: 1

    steps:
      - uses: actions/checkout@v4

      - uses: azure/login@v2
        with:
          client-id: {{ `${{ secrets.AZURE_CLIENT_ID }}` }}
          tenant-id: {{ .Metadata.Tenant }}
          subscription-id: {{ .Spec.StateBackend.Subscription }}

      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "~> 1.9"

      - name: Terraform Init
        working-directory: {{ `${{ matrix.layer }}` }}
        run: terraform init -backend-config=../../backend.hcl

      - name: Terraform Validate
        working-directory: {{ `${{ matrix.layer }}` }}
        run: terraform validate

      - name: Terraform Plan
        id: plan
        working-directory: {{ `${{ matrix.layer }}` }}
        run: |
          terraform plan -no-color -input=false \
            -out=tfplan 2>&1 | tee plan-output.txt

      - name: Comment PR with Plan
        uses: actions/github-script@v7
        if: github.event_name == 'pull_request'
        with:
          script: |
            const fs = require('fs');
            const plan = fs.readFileSync(
              `{{ "${{ matrix.layer }}" }}/plan-output.txt`, 'utf8'
            );
            const body = `### Terraform Plan — \`{{ "${{ matrix.layer }}" }}\`
            \`\`\`
            ${plan.substring(0, 60000)}
            \`\`\``;
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: body
            });
```

### 8.2 GitHub Actions — Deploy on Merge

```yaml
# templates/pipelines/github/deploy.yml.tmpl

name: "Landing Zone — Deploy"
on:
  push:
    branches: [{{ .Spec.CICD.BranchPolicy.MainBranch }}]
    paths:
      - "platform/**"
      - "landing-zones/**"
      - "lzctl.yaml"

permissions:
  id-token: write
  contents: read

env:
  ARM_TENANT_ID: "{{ .Metadata.Tenant }}"
  ARM_SUBSCRIPTION_ID: "{{ .Spec.StateBackend.Subscription }}"
  ARM_CLIENT_ID: "{{ `${{ secrets.AZURE_CLIENT_ID }}` }}"

jobs:
  deploy:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: true
      matrix:
        layer:
          - platform/management-groups
          - platform/identity
          - platform/management
          - platform/governance
          - platform/connectivity
{{- range .Spec.LandingZones }}
          - landing-zones/{{ .Name }}
{{- end }}
      max-parallel: 1

    steps:
      - uses: actions/checkout@v4

      - uses: azure/login@v2
        with:
          client-id: {{ `${{ secrets.AZURE_CLIENT_ID }}` }}
          tenant-id: {{ .Metadata.Tenant }}
          subscription-id: {{ .Spec.StateBackend.Subscription }}

      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "~> 1.9"

      - name: Terraform Init
        working-directory: {{ `${{ matrix.layer }}` }}
        run: terraform init -backend-config=../../backend.hcl

      - name: Terraform Apply
        working-directory: {{ `${{ matrix.layer }}` }}
        run: terraform apply -auto-approve -input=false
```

### 8.3 Azure DevOps — Equivalent Structure

Same logic, adapted to ADO YAML pipelines (`trigger`, `pool`, `stages/jobs/steps`, `AzureCLI@2` task, variable groups for secrets). Template structure mirrors GitHub Actions but uses ADO-specific syntax.

---

## 9. Coding Standards

### 9.1 Go Conventions

| Area | Standard |
|------|----------|
| **Formatting** | `gofmt` (enforced by CI) |
| **Linting** | golangci-lint with: `govet`, `errcheck`, `staticcheck`, `gosimple`, `unused`, `misspell` |
| **Naming** | Standard Go conventions; packages are lowercase single-word; avoid stutter (`config.Config` → `config.LZConfig`) |
| **Errors** | Wrap with `fmt.Errorf("doing X: %w", err)` for context; never swallow errors |
| **Testing** | Table-driven tests; testify assertions; mocks for `AzureCLI` and `TerraformRunner` interfaces |
| **Logging** | Use `internal/output/logger.go` everywhere; never `fmt.Println` in library code |
| **Documentation** | All exported functions and types have GoDoc comments |

### 9.2 Git Conventions

| Area | Standard |
|------|----------|
| **Commits** | Conventional Commits (`feat:`, `fix:`, `docs:`, `chore:`, `test:`, `refactor:`) |
| **Branching** | `main` (release), `feat/*`, `fix/*`, `docs/*` |
| **PRs** | Require at least 1 review; CI must pass; squash merge |
| **Releases** | Tag-based (`v0.1.0`); GoReleaser handles build + publish |

### 9.3 Generated Code Conventions

| Area | Standard |
|------|----------|
| **Terraform formatting** | All generated `.tf` files pass `terraform fmt` |
| **Module versioning** | Exact pin (e.g., `version = "0.11.0"`, not `~>`) |
| **Naming** | CAF convention by default: `<resource-type>-<workload>-<environment>-<region>-<instance>` |
| **Comments** | Each generated file has a header comment: `# Generated by lzctl v{version} — safe to edit` |
| **No lock-in** | Zero lzctl-specific markers; repo works with plain `terraform` commands |

---

## 10. Architecture Decision Records

### ADR-001: Go as CLI Language

**Status:** Accepted

**Context:** Need a language for the CLI that produces a single binary, is cross-platform, and is standard in the infrastructure tooling ecosystem.

**Decision:** Go with Cobra framework.

**Alternatives considered:**
- **Python:** Faster to prototype but distribution is painful (virtualenvs, pip, OS packaging). Not standard for infra CLIs.
- **Rust:** Excellent performance but slower development speed, steeper learning curve, smaller team pool.
- **TypeScript (Node):** Good ecosystem but requires Node runtime, not standard for infra CLIs.

**Consequences:** Team must know Go (or learn). Go's template engine is limited but sufficient for HCL generation.

---

### ADR-002: az CLI over Azure SDK

**Status:** Accepted

**Context:** Need to interact with Azure for bootstrap, audit, and import. Options are Azure SDK for Go or wrapping `az` CLI.

**Decision:** Wrap `az` CLI via `os/exec`, parse JSON output.

**Rationale:**
- `az` CLI is already a prerequisite (checked by `lzctl doctor`)
- No additional binary size bloat from Azure SDK
- `az` handles authentication, token refresh, and retries
- Simpler to maintain — CLI output is stable, SDK requires version tracking
- Users can debug by running the same `az` commands manually

**Trade-offs:**
- Slower than direct SDK calls (process spawn overhead)
- Parsing JSON output is fragile if Azure changes output format (low risk, JSON output is stable)
- Cannot easily handle streaming/pagination of very large datasets (acceptable for our scale)

**Consequences:** All Azure interaction goes through `internal/azure/cli.go`. If perf becomes an issue, we can swap to SDK for hot paths without changing the interface.

---

### ADR-003: `text/template` over Embedded HCL Generation

**Status:** Accepted

**Context:** Need to generate Terraform HCL files from configuration. Options: Go templates, HCL write library (`hclwrite`), or string concatenation.

**Decision:** Go `text/template` with embedded `.tf.tmpl` files.

**Rationale:**
- Templates are readable — a Terraform engineer can review them without knowing Go
- Easy to add new templates or modify existing ones
- `go:embed` makes distribution seamless (no external template files)
- `hclwrite` is lower-level and harder to maintain for complex files

**Trade-offs:**
- Go templates have limited logic (no complex conditionals) — mitigated by helper functions
- No HCL syntax validation at template level — mitigated by running `terraform fmt` + `validate` on output

---

### ADR-004: State Backend Bootstrap via az CLI

**Status:** Accepted

**Context:** Terraform needs a state backend before it can run, but the backend itself is infra that needs to be created. This is the chicken-and-egg problem.

**Decision:** Bootstrap state backend using direct `az` CLI commands (not Terraform).

**Rationale:**
- Clean separation: `az` CLI creates the minimum infra (RG, storage, identity), then Terraform manages everything else
- No Terraform state for the bootstrap itself (no meta-state problem)
- Simple, auditable, reversible (user can delete the RG to undo)

**Consequences:** Bootstrap logic in `internal/azure/bootstrap.go` must handle idempotency (re-running doesn't fail if resources exist).

---

### ADR-005: Layers with Separate State Files

**Status:** Accepted

**Context:** A full CAF landing zone has many resources with different lifecycles and blast radii. Options: single state, multiple state files in one backend, separate backends.

**Decision:** Multiple state files in a single Azure Storage Account, keyed by layer path.

**Rationale:**
- Blast radius isolation: breaking connectivity doesn't affect management groups
- Independent `terraform apply` per layer
- Shared backend simplifies permission management
- State key convention: `<layer-path>.tfstate` (e.g., `platform-management-groups.tfstate`)

**Trade-offs:**
- Cross-layer references require `terraform_remote_state` data sources or convention-based naming
- More complex pipeline (must deploy in order)

**Consequences:** Layer dependency order is defined in `internal/terraform/layer_order.go` and enforced by both `lzctl plan/apply` and generated CI/CD pipelines.

---

### ADR-006: No Runtime Dependency on lzctl in CI

**Status:** Accepted

**Context:** Generated CI/CD pipelines could either invoke `lzctl plan/apply` or call Terraform directly.

**Decision:** Generated pipelines call `terraform` directly. No `lzctl` needed in CI.

**Rationale:**
- Zero lock-in: if lzctl disappears, the repo still works
- Simpler CI (no need to install lzctl binary in runners)
- Easier to debug pipeline issues (standard Terraform commands)
- `lzctl plan/apply` are convenience wrappers for local use only

**Consequences:** Pipeline templates must contain all orchestration logic (layer ordering, backend config routing) in YAML, not delegated to lzctl.

---

### ADR-007: Compliance Rules as Pluggable Go Structs

**Status:** Accepted

**Context:** The audit system needs extensible rules. Options: hardcoded rules, external rule files (YAML/Rego/OPA), Go interface-based rules.

**Decision:** Go structs implementing `ComplianceRule` interface, registered in a global registry.

**Rationale:**
- Type-safe, testable, refactorable
- No need for an external policy engine (OPA would be overkill for MVP)
- Each rule is self-contained with ID, discipline, and evaluation logic
- Easy to add rules: create a struct, register it

**Trade-offs:**
- Adding rules requires Go code (not user-extensible via config)
- Acceptable for MVP — custom rule extensibility is a phase 2+ feature

---

### ADR-008: Apache 2.0 License

**Status:** Accepted

**Context:** Need an OSS license that maximizes adoption while providing reasonable protection.

**Decision:** Apache License 2.0.

**Rationale:**
- Industry standard for infrastructure tools (Terraform, Kubernetes, AVM)
- Patent protection clause reassures enterprise adopters
- Permissive enough for commercial use by clients
- Compatible with most other OSS licenses

---

## 11. Risk Assessment

### 11.1 Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| AVM module API changes between versions | Medium | Medium | Pin exact versions; `lzctl upgrade` with semver awareness; integration tests against latest AVM |
| `az` CLI JSON output format changes | Low | High | Pin expected output shapes in tests; abstract parsing in `internal/azure/` |
| Go template complexity grows unwieldy | Medium | Medium | Keep templates simple; move complex logic to Go helpers; consider `hclwrite` for specific cases |
| Terraform provider breaking changes (azurerm v4+) | Low | High | Pin provider version in generated code; test against provider upgrades |
| State file corruption during multi-layer deploy | Low | High | State versioning + soft delete on storage; clear error handling with rollback guidance |

### 11.2 Architecture Bottlenecks

| Bottleneck | Scenario | Resolution |
|-----------|----------|-----------|
| Sequential layer deployment | 5 layers × 2-5 min each = 10-25 min deploy | Acceptable for landing zone changes (infrequent); parallelism for independent layers in phase 2 |
| Audit scanning speed | 100 subscriptions × N queries each | Bounded concurrency pool (5 parallel az invocations); cache subscription list |
| Template rendering | Large number of landing zones | Negligible — Go templates render in milliseconds |
| CLI binary size | Embedded templates + schema | Expected < 30 MB; acceptable |

### 11.3 Alternative Approaches Considered

| Component | Chosen | Alternative | Why Not |
|-----------|--------|------------|---------|
| CLI framework | Cobra | urfave/cli, Kong | Cobra is the Go CLI standard; best docs and community |
| Config format | YAML | TOML, JSON, HCL | YAML is most readable for this use case; supports comments |
| Template engine | text/template | Jinja (Python), hclwrite, Mustache | text/template is stdlib, sufficient, no external deps |
| Azure interaction | az CLI wrapper | Azure SDK for Go | SDK adds complexity, binary size; az CLI is already required |
| Policy engine | Go structs | OPA/Rego, Azure Policy (audit only) | OPA is overkill for MVP; Go structs are simpler and faster |
| State backend | Azure Storage | Terraform Cloud, S3-compatible | Azure-native is the right default for an Azure tool |

---

### ADR-010: State Life Management as First-Class Concern

**Status:** Accepted

**Context:** Terraform state files are the single source of truth for deployed infrastructure. Loss, corruption, or concurrent writes to state can cause deployment failures, resource leaks, or security incidents. Most IaC projects treat state as an afterthought.

**Decision:** Treat Terraform state as a critical asset with dedicated lifecycle management:

1. **Centralized remote state** — Azure Storage with shared backend (`backend.hcl`)
2. **Blob lease locking** — Prevents concurrent writes (Azure-native, no extra service needed)
3. **Blob versioning + soft delete** — Enabled by default in config; validated by `lzctl validate`
4. **Pre-apply snapshots** — Automated in generated CI pipelines; available via `lzctl state snapshot`
5. **Health checks** — `lzctl state health` validates encryption, versioning, soft delete, TLS
6. **Doctor integration** — `lzctl doctor` checks state backend accessibility
7. **Azure AD auth** — `use_azuread_auth = true` in backend config (no storage access keys)

**Rationale:**
- State loss is catastrophic — versioning and soft delete provide defense in depth
- Concurrent pipeline runs are a real risk — blob lease locking is zero-config on Azure
- Pre-apply snapshots make rollback simple and auditable
- Health checks enforce the security posture continuously, not just at setup time
- Documentation + tabletop drills ensure the team knows the recovery procedures

**Consequences:**
- `internal/state/` package provides lifecycle operations (list, snapshot, health, unlock)
- `cmd/state_*.go` exposes these as `lzctl state` subcommands
- `config.StateBackend` includes `versioning`, `softDelete`, `softDeleteDays` fields (defaults: true, true, 30)
- `crossvalidator.go` warns when versioning or soft delete is disabled
- Generated CI pipelines include a pre-apply snapshot step
- Backend templates include comments explaining locking and encryption
- Full operational guide at `docs/operations/state-management.md`

---

*Ce document d'architecture fournit le contexte technique nécessaire à la création d'histoires d'implémentation détaillées avec chemins de fichiers, critères d'acceptation et dépendances.*
