# lzctl Documentation

> Landing Zone Factory CLI — stateless orchestrator for Azure Landing Zones, aligned with the Cloud Adoption Framework.

## Getting Started

| Step | Command | Description |
|------|---------|-------------|
| 1 | `lzctl doctor` | Check prerequisites |
| 2 | `lzctl init` | Initialise the project (interactive wizard) |
| 3 | `lzctl validate` | Validate the manifest and configuration |
| 4 | `lzctl plan` | Preview changes |
| 5 | `lzctl apply` | Deploy layers in CAF order |
| 6 | `lzctl status` | Check deployment status |

See the [README](../README.md) for the full quick start.

## Reference

- [CLI Reference](cli-reference.md) — all commands, flags, and environment variables
- [Per-Command Docs](commands/) — detailed documentation per command

## Architecture

- [Architecture Decision Records (ADRs)](architecture/decisions/)
  - [ADR-001: Go as CLI Language](architecture/decisions/001-go-as-cli-language.md)
  - [ADR-002: YAML Manifests as Source of Truth](architecture/decisions/002-yaml-manifests-as-source-of-truth.md)
  - [ADR-003: Terraform with AVM](architecture/decisions/003-terraform-with-avm.md)
  - [ADR-004: Bootstrap via az CLI](architecture/decisions/004-bootstrap-az-cli.md)
  - [ADR-005: Stateless CLI with GitOps](architecture/decisions/005-stateless-cli-gitops.md)

### CAF Design Areas

| Design Area | Layer | Page |
|-------------|-------|------|
| Resource Organisation | `management-groups` | [resource-org.md](architecture/design-areas/resource-org.md) |
| Identity & Access | `identity` | [identity.md](architecture/design-areas/identity.md) |
| Management & Monitoring | `management` | [management.md](architecture/design-areas/management.md) |
| Governance | `governance` | [governance.md](architecture/design-areas/governance.md) |
| Network Topology | `connectivity` | [network.md](architecture/design-areas/network.md) |
| Security | `security` | [security.md](architecture/design-areas/security.md) |
| Workload Vending | `landing-zones/` | [workload-vending.md](architecture/design-areas/workload-vending.md) |

## Operations

- [State Management](operations/state-management.md) — Terraform state file lifecycle management
- [Rollback](operations/rollback.md) — standard and emergency rollback procedures
- [Drift Response](operations/drift-response.md) — infrastructure drift handling
- [Policy Incident](operations/policy-incident.md) — policy incident management

## Development

- [Contributing](contributing.md) — setup, conventions, and development workflow
- [Testing](../TESTING.md) — unit and integration testing guide

## Upgrade

- [Upgrade Guide](upgrade/upgrade-guide.md) — version compatibility, migrations
