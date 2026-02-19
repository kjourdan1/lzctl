# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **CLI Foundation** — Go CLI with Cobra/Viper, global flags (`--config`, `--repo-root`, `--verbose`, `--dry-run`, `--json`)
- **`lzctl doctor`** — Environment preflight checks (az, terraform, git, Azure auth, state backend)
- **`lzctl init`** — Interactive wizard generating a complete Terraform repository with CAF layers
- **`lzctl validate`** — Multi-layer validation (JSON Schema, cross-field, `terraform validate`)
- **`lzctl plan`** — Orchestrated multi-layer Terraform plan in CAF dependency order
- **`lzctl apply`** — Orchestrated multi-layer Terraform apply with state snapshots
- **`lzctl status`** — Landing zone status overview with `--live` Azure queries
- **`lzctl drift`** — Drift detection via `terraform plan -detailed-exitcode` across all layers
- **`lzctl rollback`** — Rollback to previous state snapshot
- **`lzctl audit`** — CAF compliance scoring across 6 disciplines (Markdown + JSON output)
- **`lzctl assess`** — Brownfield environment assessment
- **`lzctl import`** — Generate Terraform import blocks from audit reports
- **`lzctl select`** — Interactive resource selection for import
- **`lzctl schema`** — Display or validate the `lzctl.yaml` JSON Schema
- **`lzctl docs`** — Open documentation in browser
- **`lzctl version`** — Display CLI version information
- **`lzctl upgrade`** — AVM module version checker and updater

#### State Lifecycle Management

- **`lzctl state list`** — List state snapshots with metadata
- **`lzctl state snapshot`** — Create manual state snapshots
- **`lzctl state health`** — Verify state backend security (TLS, encryption, versioning)
- **`lzctl state unlock`** — Force-unlock stuck state leases

#### Policy-as-Code

- **`lzctl policy create`** — Scaffold policy definitions and assignments
- **`lzctl policy test`** — Test policies in audit mode before enforcing
- **`lzctl policy verify`** — Verify policy compliance on target scope
- **`lzctl policy remediate`** — Trigger remediation tasks for non-compliant resources
- **`lzctl policy deploy`** — Deploy policy assignments via Terraform
- **`lzctl policy status`** — View policy compliance status
- **`lzctl policy diff`** — Compare local vs. deployed policy state

#### Workload Management

- **`lzctl workload add`** — Add a new workload landing zone
- **`lzctl workload adopt`** — Adopt an existing Azure subscription as a workload
- **`lzctl workload list`** — List managed workload landing zones
- **`lzctl workload remove`** — Remove a workload landing zone

### Architecture

- **Manifest format** — `apiVersion: lzctl/v1`, `kind: LandingZone`
- **5 CAF layers** — management-groups → identity → management → governance → connectivity
- **Azure Verified Modules (AVM)** — Pinned versions with `~>` constraint operators
- **CI/CD templates** — GitHub Actions and Azure DevOps pipelines
- **Profile system** — Extensible profiles catalog (`profiles/catalog.yaml`)
- **Tenant configuration** — Multi-tenant support via `tenants/` directory
- **Embedded schemas and templates** — Compiled into binary via Go `embed`

[Unreleased]: https://github.com/kjourdan1/lzctl/commits/main
