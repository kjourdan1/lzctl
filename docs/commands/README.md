# lzctl Command Reference

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | | `lzctl.yaml` | Config file path |
| `--repo-root` | | `.` | Repository root path |
| `--verbose` | `-v` | `0` | Verbosity (-v, -vv, -vvv) |
| `--dry-run` | | `false` | Simulate without modifying Azure |
| `--json` | | `false` | Machine-readable JSON output |
| `--ci` | | `false` | Non-interactive mode (auto-detected via `CI=true`) |

## Commands

### Scaffolding & Configuration

| Command | Description | Doc |
|---------|-------------|-----|
| [init](init.md) | Initialise a landing zone project | ✅ |
| [validate](validate.md) | Validate `lzctl.yaml` and Terraform configuration | ✅ |
| [select](select.md) | Browse the CAF layer catalogue | — |
| [schema](schema.md) | Export / validate the JSON schema | — |
| [docs](docs.md) | Generate project documentation | — |

### Terraform Operations

| Command | Description | Doc |
|---------|-------------|-----|
| [plan](plan.md) | Multi-layer plan in CAF dependency order | ✅ |
| [apply](apply.md) | Multi-layer apply in CAF dependency order | ✅ |
| [drift](drift.md) | Detect infrastructure drift | ✅ |
| [rollback](rollback.md) | Rollback layers in reverse CAF order | — |

### Blueprints

| Command | Description | Doc |
|---------|-------------|-----|
| [add-blueprint](add-blueprint.md) | Attach a secure blueprint to a landing zone | ✅ |

Blueprint types generate a Terraform layer under `landing-zones/<name>/blueprint/`
and automatically update the CI/CD pipeline matrix.

| Type | Description |
|------|-------------|
| `paas-secure` | App Service + APIM + Key Vault, all behind Private Endpoints |
| `aks-platform` | Private AKS + ACR + Key Vault + optional ArgoCD GitOps |
| `aca-platform` | Container Apps environment + Key Vault + Private Endpoints |
| `avd-secure` | Azure Virtual Desktop session hosts + FSLogix + Private DNS |

### Day-2

| Command | Description | Doc |
|---------|-------------|-----|
| [status](status.md) | Project state overview | ✅ |
| [upgrade](upgrade.md) | Check / apply AVM module updates | ✅ |
| [audit](audit.md) | CAF compliance audit | ✅ |
| [assess](assess.md) | Maturity assessment | — |
| [import](import.md) | Generate Terraform import blocks (with AVM stubs) | ✅ |
| [doctor](doctor.md) | Check prerequisites | ✅ |

### State Management

| Command | Description |
|---------|-------------|
| `lzctl state list` | List Terraform state files |
| `lzctl state snapshot` | Snapshot state files |
| `lzctl state health` | Check state backend security posture |
| `lzctl state unlock` | Force-unlock a stuck lease |

See [State Management Guide](../operations/state-management.md).

### Policy-as-Code

| Command | Description |
|---------|-------------|
| `lzctl policy create` | Scaffold a policy definition |
| `lzctl policy test` | Deploy in DoNotEnforce (audit) mode |
| `lzctl policy verify` | Generate a compliance report |
| `lzctl policy remediate` | Create remediation tasks |
| `lzctl policy deploy` | Switch to Default enforcement |
| `lzctl policy status` | Show policy workflow state |
| `lzctl policy diff` | Compare local vs. deployed policy |

### Workload Management

| Command | Description |
|---------|-------------|
| `lzctl workload add` | Add a landing zone |
| `lzctl workload adopt` | Adopt an existing subscription (brownfield) |
| `lzctl workload list` | List landing zones |
| `lzctl workload remove` | Remove a landing zone |
