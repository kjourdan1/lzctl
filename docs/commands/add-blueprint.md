# lzctl add-blueprint

Attach a secure, opinionated blueprint to an existing landing zone.

## Synopsis

```bash
lzctl add-blueprint [flags]
```

## Description

`add-blueprint` generates a Terraform layer under `landing-zones/<name>/blueprint/`
that implements secure-by-default workload infrastructure. Every blueprint:

- Uses **Azure Verified Modules** with pinned versions
- Enforces **private networking** (Private Endpoints, no public IPs)
- Enforces **encryption at rest and in transit**
- Generates a separate `backend.hcl` keyed to the blueprint layer
- Automatically updates `.lzctl/zone-matrix.json` and all CI/CD pipeline files
  so the blueprint layer runs after its parent landing zone in parallel jobs

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--landing-zone` | required (interactive) | Target landing zone name (must exist in `lzctl.yaml`) |
| `--type` | required (interactive) | Blueprint type (see below) |
| `--set` | | Override in `path=value` format — repeatable |
| `--overwrite` | `false` | Replace an existing blueprint on the landing zone |

In CI mode (`--ci` or `CI=true`), `--landing-zone` and `--type` are required.

## Blueprint types

### `paas-secure`

App Service + optional APIM + Key Vault, all connected via Private Endpoints.

**Generated files:**
- `main.tf` — AVM modules for App Service Plan, Web App, Key Vault, Private Endpoints, Private DNS zones
- `variables.tf`
- `blueprint.auto.tfvars` — reflects any `--set` overrides
- `backend.hcl`

**Secure defaults:**
- `public_network_access_enabled = false`
- `https_only = true`
- `minimum_tls_version = "1.2"`
- Private DNS zones: `privatelink.azurewebsites.net`, `privatelink.vaultcore.azure.net`

**Overrides:**

| Path | Default | Description |
|------|---------|-------------|
| `appService.sku` | `P1v3` | App Service Plan SKU |
| `appService.runtimeStack` | `DOTNET\|8.0` | Runtime stack |
| `appService.scmPublicAccess` | `false` | Allow SCM public access |
| `apim.enabled` | `false` | Deploy API Management |
| `apim.sku` | `Developer` | APIM SKU (`Developer`, `Standard`, `Premium`) |
| `apim.capacity` | `1` | APIM capacity units |
| `keyVault.enablePurgeProtection` | `true` | Enable purge protection |
| `keyVault.softDeleteRetentionDays` | `90` | Soft-delete retention period |

---

### `aks-platform`

Private AKS cluster + Azure Container Registry + Key Vault + optional ArgoCD GitOps.

**Generated files (ArgoCD disabled):**
- `main.tf`, `variables.tf`, `blueprint.auto.tfvars`, `backend.hcl`

**Additional files when `argocd.enabled = true`:**
- `argocd/appset.yaml` — ArgoCD ApplicationSet with `selfHeal` and `ServerSideApply`
- `Makefile` — `argocd-login`, `argocd-sync-all`, `aks-credentials` helper targets

**Secure defaults:**
- `private_cluster_enabled = true`
- `oidc_issuer_enabled = true`
- `workload_identity_enabled = true`
- `azure_policy_enabled = true`
- ACR: `public_network_access_enabled = false`
- Key Vault: Private Endpoint attached

**Overrides:**

| Path | Default | Description |
|------|---------|-------------|
| `aks.kubernetesVersion` | `1.29` | Kubernetes version |
| `aks.nodeCount` | `3` | System node pool count |
| `aks.vmSize` | `Standard_D4s_v5` | System node VM size |
| `acr.sku` | `Premium` | ACR SKU (Premium required for Private Endpoints) |
| `defender.enabled` | `true` | Microsoft Defender for Containers |
| `argocd.enabled` | `false` | Deploy ArgoCD |
| `argocd.mode` | `extension` | `extension` (AKS add-on) or `helm` |
| `argocd.repoUrl` | | GitOps repository URL — **required** when ArgoCD enabled |
| `argocd.targetRevision` | `HEAD` | Git branch / tag / commit |
| `argocd.appPath` | `apps` | Path within repo scanned by ApplicationSet |
| `argocd.ssoEnabled` | `false` | Enable SSO for ArgoCD UI |
| `argocd.chartVersion` | `5.55.0` | ArgoCD Helm chart version (helm mode only) |

---

### `aca-platform`

Azure Container Apps environment + Key Vault + Private Endpoints.

**Secure defaults:**
- VNet injection (internal only)
- No public ingress
- Private DNS zone: `privatelink.azurecontainerapps.io`

---

### `avd-secure`

Azure Virtual Desktop — session hosts + FSLogix + Private DNS.

**Secure defaults:**
- Managed identity on session hosts
- Entra ID join (no hybrid domain join required)
- FSLogix profiles in Azure Files with Private Endpoint
- Private DNS zone: `privatelink.file.core.windows.net`

---

## Examples

```bash
# Interactive (prompts for landing zone and type)
lzctl add-blueprint

# CI / headless — PaaS blueprint
lzctl add-blueprint --ci \
  --landing-zone app-prod \
  --type paas-secure \
  --set appService.sku=P2v3 \
  --set apim.enabled=true

# AKS with ArgoCD in Helm mode
lzctl add-blueprint --ci \
  --landing-zone aks-infra \
  --type aks-platform \
  --set argocd.enabled=true \
  --set argocd.mode=helm \
  --set argocd.repoUrl=https://github.com/myorg/gitops \
  --set argocd.targetRevision=main \
  --set argocd.appPath=clusters/westeurope

# Dry-run (preview files without writing)
lzctl add-blueprint --dry-run \
  --landing-zone app-prod \
  --type paas-secure

# Replace an existing blueprint
lzctl add-blueprint --landing-zone app-prod --type paas-secure --overwrite
```

## lzctl.yaml representation

`add-blueprint` persists the blueprint definition in `lzctl.yaml`:

```yaml
landingZones:
  - name: app-prod
    subscription: bbbbbbbb-cccc-4ddd-8eee-ffffffffffff
    archetype: corp
    blueprint:
      type: paas-secure
      overrides:
        appService:
          sku: P2v3
        apim:
          enabled: true
```

## Pipeline matrix update

After attaching a blueprint, `add-blueprint` updates zone-matrix and pipeline files
so `landing-zones/app-prod/blueprint` runs immediately after `landing-zones/app-prod`:

```json
// .lzctl/zone-matrix.json
[
  {"name": "app-prod",           "dir": "landing-zones/app-prod",           "archetype": "corp"},
  {"name": "app-prod-blueprint", "dir": "landing-zones/app-prod/blueprint", "archetype": "blueprint", "blueprintType": "paas-secure"}
]
```

## See also

- [import](import.md) — import existing resources into a blueprint layer
- [workload add](../cli-reference.md) — add a landing zone first
- [CLI Reference — add-blueprint](../cli-reference.md#lzctl-add-blueprint) — full flag reference
