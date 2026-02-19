# lzctl Demo

## Overview

This demo walks through a full landing zone lifecycle:

1. **Doctor** — Verify prerequisites (az CLI, Terraform, Go, Azure auth)
2. **Init** — Scaffold a full Terraform repository from a YAML manifest
3. **Validate** — Schema + cross-field + `terraform validate`
4. **Plan** — Multi-layer Terraform plan in CAF order
5. **Apply** — Deploy all layers with auto-approve
6. **Status** — Check deployment status with live Azure queries
7. **Workload** — Add a workload landing zone

## Prerequisites

- `az` CLI authenticated (`az login`)
- Terraform ≥ 1.5
- `lzctl` binary in `$PATH`

## Running the Demo

```bash
# Make the script executable
chmod +x demo.sh

# Run it
./demo.sh
```

> **Note**: The demo uses placeholder tenant/subscription IDs.
> Replace them with real values for an actual deployment.

## What happens

| Step | Command | Description |
|------|---------|-------------|
| 1 | `lzctl doctor` | Checks az, terraform, git, go versions and Azure auth |
| 2 | `lzctl init` | Generates `lzctl.yaml` + Terraform files for 5 CAF layers |
| 3 | `cat lzctl.yaml` | Shows the generated manifest |
| 4 | `lzctl validate --strict` | Full validation pass |
| 5 | `lzctl plan` | Plans management-groups → identity → management → governance → connectivity |
| 6 | `lzctl apply --auto-approve` | Deploys all layers in order |
| 7 | `lzctl status --live` | Queries Azure for real deployment status |
| 8 | `lzctl workload add` | Adds a new workload subscription |

## After the demo

```bash
lzctl audit         # Score against CAF (6 disciplines)
lzctl drift         # Detect manual Azure Portal changes
lzctl upgrade       # Update AVM module versions
lzctl state health  # Verify state backend security
```
