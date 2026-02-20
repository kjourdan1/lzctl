# lzctl drift

Detect infrastructure drift between the Terraform state and Azure resources.

## Synopsis

```bash
lzctl drift [flags]
```

## Description

Runs `terraform plan -detailed-exitcode` on each platform layer and analyses detected changes:
- **Addition** — resource created outside Terraform
- **Modification** — resource modified manually
- **Deletion** — resource deleted manually

The scan runs layer by layer in CAF order:
`management-groups` → `identity` → `management` → `governance` → `connectivity`

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--layer` | all | Specific layer to check |

## Output

```
═══ Drift scan ═══

  management-groups   ✅ No drift
  identity            ✅ No drift
  management          ✅ No drift
  governance          ⚠️  2 changes detected
  connectivity        ⚠️  1 change detected

Summary: 2 layers with drift (3 total changes)
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No drift |
| 2 | Drift detected |

## CI/CD Integration

Generated pipelines include a scheduled drift detection workflow (weekly). When drift is detected, a GitHub issue or Azure DevOps work item is automatically created.

## Examples

```bash
# Scan all layers
lzctl drift

# Scan a single layer
lzctl drift --layer connectivity

# JSON output
lzctl drift --json
```

## See Also

- [plan](plan.md) — see planned changes
- [Drift Response](../operations/drift-response.md) — drift response procedure
