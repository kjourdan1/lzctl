# lzctl audit

Audit de conformité CAF d'un tenant Azure.

## Synopsis

```bash
lzctl audit [flags]
```

## Description

Scanne les ressources Azure du tenant et évalue la conformité par rapport aux 6 disciplines CAF :
- **Resource Organisation** — management groups, subscriptions
- **Identity & Access** — RBAC, privileged roles
- **Management** — Log Analytics, diagnostics
- **Governance** — policy assignments, compliance
- **Connectivity** — VNets, peering, DNS
- **Security** — Defender for Cloud, encryption

Produit un rapport avec un score global et des findings détaillés.

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--scope` | tenant root | Management group scope |
| `--output` | stdout | Chemin du fichier de sortie |

## Sortie

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

## Exemples

```bash
# Audit du tenant complet
lzctl audit

# Audit d'un scope spécifique
lzctl audit --scope mg-platform

# Enregistrer dans un fichier
lzctl audit --output audit-report.md

# JSON
lzctl audit --json --output audit-report.json
```

## Voir aussi

- [import](import.md) — importer des ressources depuis le rapport
- [assess](assess.md) — évaluation de la maturité
