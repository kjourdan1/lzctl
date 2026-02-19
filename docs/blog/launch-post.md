# Introducing lzctl — Azure Landing Zones, Finally as Code

> Landing Zone Factory CLI for Azure Platform Engineering

## The Problem

You've read the Cloud Adoption Framework. You've looked at the Azure Landing Zone Accelerator. You know you need management groups, policies, hub-spoke networking, identity, and logging. But turning that knowledge into a maintainable, version-controlled, CI/CD-deployed infrastructure? That's where teams get stuck.

Every Azure tenant ends up with a bespoke Terraform setup. Policies are clicked through the Portal. Nobody knows what's actually deployed vs. what's in code. Changes go straight to production without review.

## What is lzctl?

**lzctl** is a stateless CLI that orchestrates Azure Landing Zones following the Cloud Adoption Framework. Think of it as **"kubectl for Azure Landing Zones"**: you declare what you want in a YAML file, and lzctl generates, validates, deploys, audits, and maintains it.

### Greenfield: From Zero to Landing Zone in 5 Minutes

```bash
lzctl doctor                    # Check prerequisites
lzctl init                      # Interactive wizard → full Terraform repo
lzctl validate --strict         # Schema + cross-field + terraform validate
lzctl plan                      # Multi-layer plan in CAF order
lzctl apply --auto-approve      # Deploy all layers

git add . && git commit -m "Initial landing zone" && git push
```

That's it. You now have:
- 5 Terraform layers (management-groups → identity → management → governance → connectivity)
- Azure Verified Modules with pinned versions
- CI/CD pipelines (GitHub Actions or Azure DevOps)
- State backend with versioning and soft delete
- A `lzctl.yaml` manifest as your single source of truth

### Brownfield: Adopt What You Have

```bash
lzctl audit --json --output audit-report.json    # Scan your tenant
lzctl import --from audit-report.json            # Generate import blocks
terraform plan                                    # Verify zero-diff
```

Score: 72/100 — 2 warnings, 0 failures. Now you know exactly where your gaps are.

### Day-2: Keep It Running

```bash
lzctl drift                     # Detect manual changes
lzctl upgrade --apply           # Update AVM module versions
lzctl state health              # Verify state backend security
lzctl workload add --name app   # Add a new landing zone
lzctl policy test               # Test policies before enforcing
```

## Key Features

### Multi-Layer Orchestration

lzctl deploys Terraform layers in CAF dependency order:

```
management-groups → identity → management → governance → connectivity
```

Each layer has its own state file. If connectivity fails, management-groups is unaffected.

### Compliance Audit

`audit` scores your environment against the Cloud Adoption Framework across 6 disciplines: Resource Organisation, Security, Governance, Identity, Management, and Connectivity.

### Drift Detection

`drift` runs `terraform plan -detailed-exitcode` on every layer and creates GitHub Issues or Azure DevOps work items when drift is found.

### Module Upgrade

`upgrade` queries the Terraform Registry for newer AVM module versions and can auto-update your `.tf` files while preserving constraint operators.

### State Lifecycle Management

State is treated as a first-class asset:
- **Blob versioning** — full history of every state mutation
- **Soft delete** — 30-day recovery window
- **Pre-apply snapshots** — automatic in CI pipelines
- **Health checks** — TLS, encryption, versioning verification
- **Lease management** — force-unlock stuck state files

### Policy-as-Code

Full lifecycle: scaffold → test (audit mode) → verify → remediate → deploy (enforce).

## What You Get

| Layer | Purpose | AVM Module |
|-------|---------|------------|
| management-groups | CAF hierarchy (Standard or Lite) | `avm-ptn-alz` |
| identity | Managed identity + OIDC federation | custom |
| management | Log Analytics + Defender for Cloud | AVM management |
| governance | CAF policy assignments | AVM policy |
| connectivity | Hub-Spoke or vWAN networking | AVM network |

## Getting Started

```bash
# Install
git clone https://github.com/kjourdan1/lzctl.git
cd lzctl && go build -o bin/lzctl .

# Or download from releases
# https://github.com/kjourdan1/lzctl/releases
```

## Links

- [README](https://github.com/kjourdan1/lzctl/blob/main/README.md)
- [CLI Reference](https://github.com/kjourdan1/lzctl/blob/main/docs/cli-reference.md)
- [Per-command docs](https://github.com/kjourdan1/lzctl/blob/main/docs/commands/)
- [Contributing](https://github.com/kjourdan1/lzctl/blob/main/CONTRIBUTING.md)

## License

Apache 2.0 — because enterprise-grade infrastructure tooling should be open.

---

*Built by platform engineers, for platform engineers.*
