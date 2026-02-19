# Upgrade Guide

## Overview

`lzctl upgrade` keeps your landing zone up to date by:

1. Scanning Terraform files for Azure Verified Modules (AVM) references
2. Querying the Terraform Registry for newer versions
3. Updating version constraints in-place
4. Validating the result with `terraform validate`

## Usage

```bash
# Check for available upgrades (dry-run by default)
lzctl upgrade

# Apply upgrades automatically
lzctl upgrade --apply

# Target a specific layer
lzctl upgrade --layer governance
```

## What Gets Upgraded

| Component | Source | Example |
|-----------|--------|---------|
| AVM modules | Terraform Registry | `avm-ptn-alz` 0.9.0 → 0.10.0 |
| Provider versions | Terraform Registry | `azurerm` 3.x → 4.x |
| lzctl itself | GitHub Releases | `v0.1.0` → `v0.2.0` |

## Upgrade Workflow

### 1. Preview Changes

```bash
lzctl upgrade
```

Output:

```
Module                         Current   Available   Layer
avm-ptn-alz                    0.9.0     0.10.0      management-groups
avm-res-network-virtualnetwork 0.4.0     0.5.1       connectivity

2 upgrades available. Run 'lzctl upgrade --apply' to update.
```

### 2. Apply Upgrades

```bash
lzctl upgrade --apply
```

This updates the `source` and `version` constraints in your `.tf` files while preserving constraint operators (`~>`, `>=`, etc.).

### 3. Validate

```bash
lzctl validate --strict
lzctl plan
```

### 4. Commit & Deploy

```bash
git add .
git commit -m "chore: upgrade AVM modules"
git push
```

CI/CD pipelines will run `plan` and `apply` automatically.

## Breaking Changes

When a major version bump is detected:

1. `lzctl upgrade` shows a **warning** with a link to the module changelog
2. The module is **NOT** auto-upgraded unless `--apply` is used
3. Review the changelog before applying

### Common Breaking Changes

| Module | Migration Notes |
|--------|----------------|
| `avm-ptn-alz` | Major versions may rename variables or change output shapes |
| `azurerm` provider | Provider 4.x requires explicit `subscription_id` in provider blocks |

## Rollback

If an upgrade causes issues:

```bash
# Revert to previous AVM versions
git checkout HEAD~1 -- modules/

# Or use state snapshot
lzctl state list
lzctl rollback --to <snapshot-id>
```

## Configuration

The `lzctl.yaml` manifest controls upgrade behavior:

```yaml
apiVersion: lzctl/v1
kind: LandingZone
metadata:
  name: my-landing-zone
spec:
  terraform:
    version: ">= 1.5"
  modules:
    pinStrategy: minor    # minor | patch | exact
```

| Pin Strategy | Behavior |
|-------------|----------|
| `minor` | Allow `~>` minor upgrades (default) |
| `patch` | Only patch upgrades |
| `exact` | Exact version pins, manual upgrade only |

## CI/CD Integration

Add an upgrade check to your pipeline:

```yaml
# GitHub Actions
- name: Check for upgrades
  run: |
    lzctl upgrade --json > upgrades.json
    if [ -s upgrades.json ]; then
      echo "::warning::AVM module upgrades available"
    fi
```

```yaml
# Azure DevOps
- script: |
    lzctl upgrade --json > upgrades.json
  displayName: 'Check AVM upgrades'
```
