# lzctl plan

Run `terraform plan` across platform layers in CAF dependency order.

## Synopsis

```bash
lzctl plan [flags]
```

## Description

Orchestrates `terraform plan` on each layer in order:
1. `management-groups` — Resource Organisation
2. `identity` — Identity & Access
3. `management` — Management & Monitoring
4. `governance` — Azure Policies
5. `connectivity` — Hub-Spoke or vWAN

Each layer uses its own state file in the shared backend.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--layer` | all | Specific layer to plan |
| `--target` | | Alias for `--layer` |
| `--out` | | Save the summary to a file |

## Examples

```bash
# Plan all layers
lzctl plan

# Plan a specific layer
lzctl plan --layer connectivity

# Save the summary
lzctl plan --out plan-output.txt

# JSON output
lzctl plan --json
```

## Output

```
═══ Planning: management-groups ═══
  No changes. Infrastructure is up-to-date.

═══ Planning: identity ═══
  No changes. Infrastructure is up-to-date.

═══ Planning: connectivity ═══
  Plan: 3 to add, 0 to change, 0 to destroy.
```

## See Also

- [apply](apply.md) — apply changes
- [drift](drift.md) — detect drift
