# lzctl upgrade

Update Terraform module versions to the latest available versions.

## Synopsis

```bash
lzctl upgrade [flags]
```

## Description

Scans all `.tf` files in the project, identifies version pins (`version = "..."`) in `module {}` blocks, and checks available versions on the [Terraform Registry](https://registry.terraform.io/).

The scan:
1. Recursively walks `.tf` files (ignores `.terraform/`, `.git/`, `node_modules/`)
2. Extracts `source` + `version` from each module block
3. Queries the Registry API (`/v1/modules/.../versions`)
4. Compares the local version with the latest stable version
5. Displays available upgrades (or applies them with `--apply`)

### Constraint Operators Preserved

The updater preserves the constraint operator:

| Before | After (`--apply`) |
|--------|-------------------|
| `version = "1.2.0"` | `version = "1.3.0"` |
| `version = "~> 1.2.0"` | `version = "~> 1.3.0"` |
| `version = ">= 1.2.0"` | `version = ">= 1.3.0"` |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--apply` | `false` | Apply updates to `.tf` files |
| `--module` | | Filter by module name (exact match) |
| `--dry-run` | `false` | Show changes without modifying (same as omitting `--apply`) |
| `--json` | `false` | Structured JSON output |

## Examples

```bash
# List available upgrades
lzctl upgrade

# Specific module
lzctl upgrade --module resource-org

# Apply updates
lzctl upgrade --apply

# JSON output
lzctl upgrade --json

# Pipeline: check + apply
lzctl upgrade --json -o upgrades.json
cat upgrades.json | jq '.upgrades | length'
lzctl upgrade --apply
```

## Text Output

```
🔍 Scanning .tf files for module version pins...

Found 6 module pins across 4 files.

Module                          Current   Latest    Status
──────────────────────────────  ────────  ────────  ──────
Azure/avm-res-network-vnet      0.4.0     0.5.2    ⬆ upgrade
Azure/avm-res-keyvault-vault    0.9.1     0.9.1    ✅ up-to-date
Azure/avm-res-network-nsg       1.0.0     1.1.0    ⬆ upgrade
Azure/avm-ptn-alz               0.10.0    0.11.0   ⬆ upgrade

3 upgrades available, 1 up-to-date.
Run 'lzctl upgrade --apply' to update.
```

## JSON Output (`--json`)

```json
{
  "scanned_files": 4,
  "total_pins": 6,
  "upgrades": [
    {
      "module": "Azure/avm-res-network-vnet/azurerm",
      "file": "modules/connectivity-hubspoke/main.tf",
      "line": 12,
      "current": "0.4.0",
      "latest": "0.5.2",
      "constraint": "~>"
    },
    {
      "module": "Azure/avm-ptn-alz/azurerm",
      "file": "modules/resource-org/main.tf",
      "line": 5,
      "current": "0.10.0",
      "latest": "0.11.0",
      "constraint": ""
    }
  ],
  "up_to_date": [
    {
      "module": "Azure/avm-res-keyvault-vault/azurerm",
      "current": "0.9.1"
    }
  ]
}
```

## Compatibility

The upgrade scanner supports modules published on:
- **Terraform Registry** (`registry.terraform.io`) — full support
- **Private modules** — not currently supported (skipped with warning)

## See Also

- [`validate`](validate.md) — validate after upgrade
- [`plan`](plan.md) — check changes related to new versions
- [`doctor`](doctor.md) — verify Terraform prerequisites
