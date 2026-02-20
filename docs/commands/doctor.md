# lzctl doctor

Check prerequisites and environment health.

## Synopsis

```bash
lzctl doctor
```

## Description

Checks the presence and version of tools required by lzctl:

### Checked Tools

| Tool | Min Version | Required |
|------|-------------|----------|
| Terraform | >= 1.5 | ✅ |
| Azure CLI | >= 2.50 | ✅ |
| Git | >= 2.30 | ✅ |
| GitHub CLI | any | ❌ optional |

### Azure Checks

| Check | Description |
|-------|-------------|
| Azure session | `az account show` returns a valid session |
| Management group access | Read access to the root management group |
| Resource providers | `Microsoft.Management`, `Microsoft.Authorization`, `Microsoft.Network`, `Microsoft.ManagedIdentity` registered |

### State Backend Check

| Check | Description |
|-------|-------------|
| Storage account accessible | Storage account tagged `purpose=terraform-state` is accessible |

## Output

```
═══ lzctl doctor ═══

Tools:
  ✅  terraform   v1.9.0
  ✅  az          v2.65.0
  ✅  git         v2.45.0
  ⚠️  gh          not found (optional)

Auth:
  ✅  Azure session active (tenant: contoso.onmicrosoft.com)
  ✅  Management group access verified

Azure:
  ✅  Microsoft.Management registered
  ✅  Microsoft.Authorization registered
  ✅  Microsoft.Network registered
  ✅  Microsoft.ManagedIdentity registered

State Backend:
  ✅  Storage account accessible
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All critical checks pass |
| 1 | One or more critical checks fail |

## See Also

- [init](init.md) — run doctor before init
- [state health](../operations/state-management.md) — detailed state backend verification
