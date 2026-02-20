# Product Brief — lzctl

> Version: 1.0 | Date: 2026-02-16

## Summary

**lzctl** is an open-source CLI tool (Go, single binary) that bootstraps and maintains Azure Landing Zones aligned with the Cloud Adoption Framework (CAF).

It generates production-ready Terraform repositories using Azure Verified Modules (AVM), connected to CI/CD pipelines (GitHub Actions or Azure DevOps), following a GitOps workflow where PRs trigger plans and merges trigger deployments.

## Problem

Azure platform teams face:
- **Ad-hoc setup** — each tenant has a unique, non-standardized Terraform configuration
- **Manual policies** — applied via the portal, without versioning or review
- **No visibility** — nobody knows what is actually deployed vs in code
- **No platform CI/CD** — changes go directly to production
- **Risky brownfield** — existing environments are too risky to terraformize

## Solution

lzctl adds the **missing orchestration layer** on top of the Microsoft-recommended path:

1. **Scaffolding** — an interactive wizard generates a complete Terraform repository
2. **Validation** — JSON schema, CIDR/UUID cross-validation, terraform validate
3. **Orchestration** — multi-layer plan/apply in CAF dependency order
4. **Day-2 Ops** — drift detection, module upgrade, policy lifecycle, state management
5. **Brownfield** — CAF audit, progressive import of existing resources

## Architecture

```
lzctl (Go binary)
    ↓ generates
lzctl.yaml → Terraform (AVM) → Azure Landing Zone
    ↓ orchestrates
CI/CD (GitHub Actions / Azure DevOps)
```

### CAF Layers (deployment order)

1. `management-groups` — Resource Organisation
2. `identity` — Identity & Access
3. `management` — Management & Monitoring
4. `governance` — Azure Policies
5. `connectivity` — Hub-Spoke or vWAN

### Principles

| Principle | Description |
|-----------|-------------|
| Stateless | No local state — everything in lzctl.yaml + Git + Terraform state |
| Native Terraform | Generated code works without lzctl |
| GitOps | PR = review + plan, merge = apply |
| State as First-Class | Versioning, soft delete, health checks |

## Target Audience

- Azure platform teams (Cloud Engineers, Platform Engineers)
- Consultants deploying landing zones for their clients
- Organizations adopting the Cloud Adoption Framework

## Differentiation

| Feature | lzctl | ALZ Terraform Module | Manual Terraform |
|---------|-------|---------------------|------------------|
| Interactive scaffolding | ✅ | ❌ | ❌ |
| Cross-field validation | ✅ | ❌ | ❌ |
| Drift detection | ✅ | ❌ | Manual |
| Module upgrade | ✅ | ❌ | Manual |
| Policy-as-Code lifecycle | ✅ | ❌ | ❌ |
| State lifecycle management | ✅ | ❌ | ❌ |
| Brownfield import | ✅ | ❌ | Manual |
| CI/CD generation | ✅ | ❌ | Manual |

## Technical Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24 |
| CLI framework | Cobra + Viper |
| IaC | Terraform >= 1.5 |
| Modules | Azure Verified Modules (AVM) |
| Auth | Azure CLI + Workload Identity Federation |
| State | Azure Storage (versioning, soft delete, blob lease locking) |
| CI/CD | GitHub Actions or Azure DevOps |
| Templates | Go text/template (embed.FS) |
| Validation | JSON Schema (embedded) |

## License

Apache License 2.0
