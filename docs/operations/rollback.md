# Rollback

Rollback procedures for platform layers.

## Principle

Rollback is performed in **reverse CAF order**:
1. `connectivity` (first — downstream dependencies)
2. `governance`
3. `management`
4. `identity`
5. `management-groups` (last — foundation)

Each layer has its own state file, which limits the blast radius.

## Rollback via lzctl

### Full Rollback

```bash
# Preview
lzctl rollback --dry-run

# Execute (with confirmation)
lzctl rollback

# Without confirmation (CI)
lzctl rollback --auto-approve
```

### Rollback a Specific Layer

```bash
lzctl rollback --layer connectivity
```

## Rollback via State Snapshot

If an apply has corrupted the state, restore from a snapshot:

### 1. List Available Snapshots

```bash
lzctl state list
```

### 2. Identify the Snapshot to Restore

```bash
# Via Azure CLI
az storage blob list \
  --account-name <storage-account> \
  --container-name tfstate \
  --include s \
  --query "[?name=='platform-connectivity.tfstate'].{name:name, snapshot:snapshot, lastModified:properties.lastModified}" \
  --output table
```

### 3. Restore the Snapshot

```bash
az storage blob copy start \
  --account-name <storage-account> \
  --destination-container tfstate \
  --destination-blob platform-connectivity.tfstate \
  --source-uri "https://<storage-account>.blob.core.windows.net/tfstate/platform-connectivity.tfstate?snapshot=<snapshot-id>" \
  --auth-mode login
```

### 4. Verify

```bash
lzctl plan --layer connectivity
```

## Emergency Rollback

In case of a critical incident:

1. **Immediate snapshot**: `lzctl state snapshot --all --tag "pre-emergency"`
2. **Identify the layer**: `lzctl drift`
3. **Targeted rollback**: `lzctl rollback --layer <layer> --auto-approve`
4. **Verify**: `lzctl plan` (should show zero changes)
5. **Post-mortem**: document the incident and corrective actions

## Prevention

- Always run `lzctl plan` before `lzctl apply`
- Use CI/CD pipelines with review (PR) for changes
- Enable blob versioning and soft delete on the state backend
- Verify with `lzctl state health`
