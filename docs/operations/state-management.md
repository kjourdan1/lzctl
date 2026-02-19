# State Life Management

> **Terraform state is a critical asset. Protect it, version it, and treat changes to it like code changes.**

lzctl embeds state lifecycle management as a first-class concern. This guide covers the philosophy, architecture, and operational procedures for managing Terraform state across your Azure landing zone.

---

## Philosophy

State is the **single source of truth** for your infrastructure. If it's lost, corrupted, or tampered with, your IaC workflow breaks. lzctl enforces these principles:

| Principle | Implementation |
|-----------|---------------|
| **Centralize state** | Azure Storage Account with shared backend (`backend.hcl`) |
| **Protect with locking** | Azure blob lease locking prevents concurrent writes |
| **Version everything** | Blob versioning preserves every state mutation for rollback |
| **Soft delete** | 30-day retention protects against accidental deletion |
| **Encrypt at rest** | Azure default AES-256 + optional infrastructure encryption |
| **Encrypt in transit** | HTTPS-only + TLS 1.2 minimum enforced |
| **Automate safety** | CI pipelines snapshot state before every `terraform apply` |
| **Audit access** | Storage analytics + Azure Monitor for state access logging |

---

## Architecture

### Layer isolation (ADR-005)

Each platform layer and landing zone has its own state file within a **shared storage account**:

```
Azure Storage Account: st<project>tfstate<region>
└── Container: tfstate
    ├── platform-management-groups.tfstate
    ├── platform-identity.tfstate
    ├── platform-management.tfstate
    ├── platform-governance.tfstate
    ├── platform-connectivity.tfstate
    ├── landing-zones-app1.tfstate
    └── landing-zones-app2.tfstate
```

**Why?** Blast radius isolation — a bad apply on connectivity doesn't corrupt management groups state.

### Locking

Azure Storage uses **blob leases** for state locking (equivalent to AWS DynamoDB locking):

- When `terraform apply` starts, it acquires a blob lease
- The lease prevents other operations from writing to the same state
- When apply completes (or fails), the lease is released
- If a pipeline crashes, use `lzctl state unlock` to force-break the lease

### State key convention

State files are keyed by layer path, replacing `/` with `-`:

| Layer directory | State key |
|----------------|-----------|
| `platform/management-groups/` | `platform-management-groups.tfstate` |
| `platform/connectivity/` | `platform-connectivity.tfstate` |
| `landing-zones/app1/` | `landing-zones-app1.tfstate` |

---

## Configuration

In `lzctl.yaml`, the `stateBackend` section configures the remote state:

```yaml
spec:
  stateBackend:
    resourceGroup: rg-myproject-tfstate-westeurope
    storageAccount: stmyprojecttfstate
    container: tfstate
    subscription: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    versioning: true       # blob versioning for state history (default: true)
    softDelete: true       # soft delete protection (default: true)
    softDeleteDays: 30     # retention for soft-deleted blobs (default: 30)
```

`versioning` and `softDelete` are **enabled by default** — lzctl validates this in `lzctl validate`.

---

## CLI Commands

### `lzctl state list`

Enumerate all state files, their sizes, last modification, and lock status:

```
$ lzctl state list
LAYER               KEY                                SIZE       LAST MODIFIED             LOCK
management-groups   platform-management-groups.tfstate  2048 B     2026-02-19T10:30:00Z      🔓
identity            platform-identity.tfstate           1234 B     2026-02-19T10:31:00Z      🔓
connectivity        platform-connectivity.tfstate       8192 B     2026-02-19T10:32:00Z      🔒
```

### `lzctl state snapshot`

Create point-in-time backups before mutations:

```bash
# Snapshot all state files
lzctl state snapshot --all --tag "pre-sprint-5-deploy"

# Snapshot a specific layer
lzctl state snapshot --layer connectivity --tag "before-firewall-change"
```

### `lzctl state health`

Validate the backend security posture:

```
$ lzctl state health
State Backend Health — stmyprojecttfstate/tfstate
─────────────────────────────────────────
  ✅  HTTPS-only traffic is enforced
  ✅  Minimum TLS 1.2 enforced
  ✅  Blob versioning is enabled — state history is preserved for rollback
  ✅  Blob soft delete is enabled — protection against accidental deletion
  ⚠️  Infrastructure encryption is not enabled
       💡 Enable infrastructure encryption when creating the storage account

✅ State backend is healthy
```

### `lzctl state unlock`

Force-release a stuck blob lease (after a failed pipeline):

```bash
lzctl state unlock --key platform-connectivity.tfstate
```

---

## CI/CD Integration

### Pre-apply snapshots

Generated CI/CD pipelines (GitHub Actions and Azure DevOps) include an automatic **state snapshot step** before every `terraform apply`:

```yaml
# From generated .github/workflows/deploy.yml
- name: Snapshot state before apply
  run: |
    for key in $(az storage blob list ...); do
      az storage blob snapshot --name "$key" ...
    done
```

This ensures every deploy has a rollback point.

### Pipeline flow

```
PR created
  ↓
terraform validate + plan (posted as PR comment)
  ↓
Merge to main
  ↓
Pre-apply state snapshot ← safety net
  ↓
terraform apply (in layer dependency order)
  ↓
Post-apply: drift detection scheduled (weekly)
```

---

## Operational Procedures

### Rollback a state change

1. List versions of the affected state file:
   ```bash
   az storage blob list \
     --account-name stmyprojecttfstate \
     --container tfstate \
     --prefix platform-connectivity.tfstate \
     --include v \
     --auth-mode login
   ```

2. Copy the desired version to the current blob:
   ```bash
   az storage blob copy start \
     --account-name stmyprojecttfstate \
     --destination-container tfstate \
     --destination-blob platform-connectivity.tfstate \
     --source-uri "https://stmyprojecttfstate.blob.core.windows.net/tfstate/platform-connectivity.tfstate?versionid=<version-id>" \
     --auth-mode login
   ```

3. Run `terraform plan` to verify the restored state matches expectations.

### State migration

When moving state to a new storage account:

1. **Test in sandbox first** — never migrate production state without a rehearsal
2. Create a PR with the backend config change
3. Run `terraform init -migrate-state` in each layer (in dependency order)
4. Verify with `terraform plan` (should show zero diff)
5. Keep the old storage account for 30 days as a safety net

### Tabletop drill

Schedule quarterly team drills covering:

- [ ] Corrupt state recovery (restore from blob version)
- [ ] Stuck lock recovery (`lzctl state unlock`)
- [ ] State backend migration (to new storage account)
- [ ] Accidental state deletion (recover from soft delete)
- [ ] Cross-layer dependency failure (partial rollback)

---

## Doctor integration

`lzctl doctor` includes a state backend check that:

1. Searches for storage accounts tagged `purpose=terraform-state`
2. Verifies the account is accessible
3. Recommends running `lzctl state health` for detailed validation

---

## Validation integration

`lzctl validate` checks state backend configuration:

- ✅ `stateBackend.subscription` is a valid UUID
- ✅ `stateBackend.storageAccount` name is 3-24 characters
- ⚠️ Warning if `versioning: false` (state history disabled)
- ⚠️ Warning if `softDelete: false` (deletion protection disabled)

---

## Security checklist

| Control | Status |
|---------|--------|
| Remote state (not local) | ✅ Enforced by `backend.hcl` |
| Blob lease locking | ✅ Native to azurerm backend |
| Blob versioning | ✅ Default: true |
| Soft delete (30 days) | ✅ Default: true |
| HTTPS-only (TLS 1.2) | ✅ Validated by `state health` |
| Encryption at rest (AES-256) | ✅ Azure default |
| Azure AD auth (no access keys) | ✅ `use_azuread_auth = true` |
| Pipeline-only writes | ✅ No manual `terraform apply` in CI |
| Pre-apply snapshots | ✅ In generated CI pipelines |
| Access audit trail | ✅ Azure Storage analytics |
