# lzctl status

Display an overview of the landing zone project status.

## Synopsis

```bash
lzctl status [flags]
```

## Description

Reads `lzctl.yaml` and displays:
- Project metadata (name, tenant, region)
- Enabled platform layers and their state
- Configured landing zones
- Git information (branch, last commit)

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--live` | `false` | Query Azure to check actual state |

## Output

```
Project: contoso-platform
Tenant:  00000000-0000-0000-0000-000000000000
Region:  westeurope

Platform Layers:
  LAYER                STATUS
  management-groups    ✅ active
  identity             ✅ active
  management           ✅ active
  governance           ✅ active
  connectivity         ✅ active

Landing Zones: 2
  NAME       ARCHETYPE   CONNECTED
  app-prod   corp        yes
  sandbox    sandbox     no

Git: main (abc1234) — 2026-02-19
```

## Examples

```bash
# Local status
lzctl status

# Live status (requires Azure)
lzctl status --live

# JSON
lzctl status --json
```

## See Also

- [drift](drift.md) — detect changes
- [doctor](doctor.md) — check the environment
