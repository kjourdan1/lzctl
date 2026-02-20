# lzctl apply

Run `terraform apply` across platform layers in CAF dependency order.

## Synopsis

```bash
lzctl apply [flags]
```

## Description

Orchestrates `terraform apply` on each layer in CAF order:
1. `management-groups`
2. `identity`
3. `management`
4. `governance`
5. `connectivity`

If a layer fails, execution stops and a clear message indicates the layer and the error.

Before each apply, an automatic state file snapshot is created in CI (via the generated pipeline).

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--layer` | all | Specific layer to apply |
| `--target` | | Alias for `--layer` |
| `--auto-approve` | `false` | Skip confirmation (CI only) |
| `--ci` | `false` | Strict non-interactive mode (global) |

In CI mode (`--ci` or `CI=true`), `lzctl apply` requires `--auto-approve` (except with `--dry-run`).

## Examples

```bash
# Interactive apply on all layers
lzctl apply

# Apply a single layer without confirmation
lzctl apply --layer connectivity --auto-approve

# CI headless
CI=true lzctl apply --layer connectivity --auto-approve

# Dry-run
lzctl apply --dry-run
```

## See Also

- [plan](plan.md) — preview before apply
- [rollback](../operations/rollback.md) — undo an apply
- [state snapshot](../operations/state-management.md) — snapshot before apply
