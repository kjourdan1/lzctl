# lzctl audit

CAF compliance audit of an Azure tenant.

## Synopsis

```bash
lzctl audit [flags]
```

## Description

Scans Azure tenant resources and evaluates compliance against the 6 CAF disciplines:
- **Resource Organisation** — management groups, subscriptions
- **Identity & Access** — RBAC, privileged roles
- **Management** — Log Analytics, diagnostics
- **Governance** — policy assignments, compliance
- **Connectivity** — VNets, peering, DNS
- **Security** — Defender for Cloud, encryption

Produces a report with an overall score and detailed findings.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--scope` | tenant root | Management group scope |
| `--output` | stdout | Output file path |

## Output

```
═══ CAF Compliance Audit ═══

Score: 72/100

  Resource Organisation    85/100  ✅
  Identity & Access        60/100  ⚠️
  Management               80/100  ✅
  Governance               70/100  ⚠️
  Connectivity             75/100  ✅
  Security                 65/100  ⚠️

12 findings (3 critical, 5 high, 4 medium)
```

## Examples

```bash
# Full tenant audit
lzctl audit

# Audit a specific scope
lzctl audit --scope mg-platform

# Save to a file
lzctl audit --output audit-report.md

# JSON
lzctl audit --json --output audit-report.json
```

## See Also

- [import](import.md) — import resources from the report
- [assess](assess.md) — maturity assessment
