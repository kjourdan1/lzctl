# lzctl init

Initialise a new landing zone project.

## Synopsis

```bash
lzctl init [flags]
```

## Description

Runs an interactive wizard that collects:
1. Project name
2. Azure Tenant ID
3. CI/CD platform (GitHub Actions / Azure DevOps)
4. Management group model (CAF Standard / CAF Lite)
5. Connectivity model (Hub & Spoke / vWAN / None)
6. Primary region (and optionally secondary)
7. State backend configuration

Then generates:
- `lzctl.yaml` — declarative manifest (source of truth)
- `platform/` — Terraform layers (management-groups, identity, management, governance, connectivity)
- `landing-zones/` — workload stubs
- `pipelines/` — CI/CD files adapted to the chosen platform
- `backend.hcl` — state backend configuration
- `.gitignore`, `README.md`

If `--config` is provided, loads configuration from the YAML file and skips the wizard.

`--from-file` allows providing a transient declarative input (`lzctl-init-input.yaml`) converted to a full `lzctl.yaml` during init.

In non-interactive mode, `init` can also be driven by flags or environment variables (`LZCTL_*`) with priority: **flag > env > default**.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--tenant-id` | auto-detected | Azure AD tenant ID |
| `--subscription-id` | auto-detected | Azure Subscription ID |
| `--from-file` | | One-shot input to convert to `lzctl.yaml` |
| `--project-name` | `landing-zone` | Project name |
| `--mg-model` | `caf-standard` | MG model (`caf-standard` \| `caf-lite`) |
| `--connectivity` | `hub-spoke` | Connectivity model (`hub-spoke` \| `vwan` \| `none`) |
| `--identity` | `workload-identity-federation` | Identity model (`workload-identity-federation` \| `sp-federated` \| `sp-secret`) |
| `--primary-region` | `westeurope` | Primary region |
| `--secondary-region` | empty | Optional secondary region |
| `--cicd-platform` | `github-actions` | CI/CD platform (`github-actions` \| `azure-devops`) |
| `--state-strategy` | `create-new` | Backend strategy (`create-new` \| `existing` \| `terraform-cloud`) |
| `--force` | `false` | Overwrite existing files |
| `--no-bootstrap` | `false` | Skip state backend provisioning |
| `--ci` | `false` | Strict non-interactive mode (fails if required parameter is missing) |
| `--config` | global | Load from a file (non-interactive mode) |

### Supported Environment Variables

- `LZCTL_TENANT_ID`
- `LZCTL_SUBSCRIPTION_ID`
- `LZCTL_FROM_FILE`
- `LZCTL_PROJECT_NAME`
- `LZCTL_MG_MODEL`
- `LZCTL_CONNECTIVITY`
- `LZCTL_IDENTITY`
- `LZCTL_PRIMARY_REGION`
- `LZCTL_SECONDARY_REGION`
- `LZCTL_CICD_PLATFORM`
- `LZCTL_STATE_STRATEGY`

## State Backend Bootstrap

By default, `lzctl init` automatically provisions:
- A resource group (`rg-<project>-tfstate-<region>`)
- A storage account (with versioning, soft delete, encryption)
- A blob container (`tfstate`)
- A managed identity with the required permissions
- OIDC federated credentials for CI/CD

The bootstrap uses `az` CLI directly (not Terraform — avoids the chicken-and-egg problem).

Pass `--no-bootstrap` to skip this step if the backend already exists.

## Examples

```bash
# Interactive wizard (recommended)
lzctl init

# Non-interactive from a file
lzctl init --config lzctl.yaml

# One-shot input converted to lzctl.yaml
lzctl init --from-file docs/examples/pipeline-init/lzctl-init-input.yaml

# Fully non-interactive with flags
lzctl init \
    --tenant-id 00000000-0000-0000-0000-000000000001 \
    --project-name contoso-platform \
    --mg-model caf-lite \
    --connectivity none \
    --cicd-platform github-actions

# Non-interactive via environment variables
LZCTL_TENANT_ID=00000000-0000-0000-0000-000000000001 \
LZCTL_MG_MODEL=caf-standard \
LZCTL_CONNECTIVITY=hub-spoke \
lzctl init --repo-root ./lz-repo

# Strict CI mode (no prompts)
CI=true lzctl init --ci --tenant-id 00000000-0000-0000-0000-000000000001

# Dry-run (preview without writing)
lzctl init --dry-run

# Force overwrite
lzctl init --force
```

## Generated Structure

```
.
├── lzctl.yaml
├── backend.hcl
├── .gitignore
├── README.md
├── platform/
│   ├── management-groups/
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── terraform.tfvars
│   ├── identity/
│   ├── management/
│   ├── governance/
│   └── connectivity/
├── landing-zones/
└── pipelines/
    └── .github/workflows/  (or .azuredevops/)
```

## See Also

- [validate](validate.md) — validate after init
- [doctor](doctor.md) — check prerequisites before init
- [CI headless](../operations/ci-headless.md) — run init/validate/plan in a non-interactive pipeline
