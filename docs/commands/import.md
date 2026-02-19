# lzctl import

Génère des blocs d'import Terraform et la configuration HCL pour des ressources Azure existantes.

## Synopsis

```bash
lzctl import [flags]
```

## Description

Permet l'adoption progressive d'Infrastructure as Code en générant :
- Des blocs `import {}` (syntax Terraform 1.5+ native)
- La configuration HCL correspondante avec des modules AVM si applicable
- Un plan de migration organisé par couche

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--from` | | Chemin vers le rapport d'audit JSON |
| `--subscription` | | Subscription ID pour la découverte |
| `--resource-group` | | Resource group pour la découverte |
| `--include` | | Types de ressources à inclure (séparés par virgule) |
| `--exclude` | | Types de ressources à exclure (séparés par virgule) |
| `--layer` | auto | Couche cible pour les fichiers générés |

## Exemples

```bash
# Import depuis un rapport d'audit
lzctl import --from audit-report.json

# Import depuis un resource group
lzctl import --resource-group rg-core --layer connectivity

# Dry-run
lzctl import --subscription 00000000-... --dry-run

# Filtrer par type
lzctl import --from audit-report.json --include Microsoft.Network/virtualNetworks
```

## Fichiers générés

```
imports/
├── connectivity/
│   ├── import.tf        # Blocs import {}
│   └── resources.tf     # Configuration HCL
└── general/
    ├── import.tf
    └── resources.tf
```

## Workflow

```
lzctl audit --json --output audit-report.json
    → lzctl import --from audit-report.json --dry-run
    → lzctl import --from audit-report.json
    → terraform plan (vérifier zero-diff)
    → git add → commit → push
```

## Voir aussi

- [audit](audit.md) — générer le rapport d'entrée
- [workload adopt](../cli-reference.md) — adopter une subscription complète
