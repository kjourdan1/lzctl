# lzctl drift

Détecte le drift d'infrastructure entre l'état Terraform et les ressources Azure.

## Synopsis

```bash
lzctl drift [flags]
```

## Description

Exécute `terraform plan -detailed-exitcode` sur chaque couche plateforme et analyse les changements détectés :
- **Ajout** — ressource créée en dehors de Terraform
- **Modification** — ressource modifiée manuellement
- **Suppression** — ressource supprimée manuellement

Le scan s'effectue couche par couche dans l'ordre CAF :
`management-groups` → `identity` → `management` → `governance` → `connectivity`

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--layer` | toutes | Couche spécifique à vérifier |

## Sortie

```
═══ Drift scan ═══

  management-groups   ✅ No drift
  identity            ✅ No drift
  management          ✅ No drift
  governance          ⚠️  2 changes detected
  connectivity        ⚠️  1 change detected

Summary: 2 layers with drift (3 total changes)
```

## Exit codes

| Code | Signification |
|------|---------------|
| 0 | Aucun drift |
| 2 | Drift détecté |

## Intégration CI/CD

Les pipelines générés incluent un workflow de drift detection planifié (hebdomadaire). Quand du drift est détecté, une issue GitHub ou un work item Azure DevOps est automatiquement créé.

## Exemples

```bash
# Scan toutes les couches
lzctl drift

# Scan une seule couche
lzctl drift --layer connectivity

# Sortie JSON
lzctl drift --json
```

## Voir aussi

- [plan](plan.md) — voir les changements planifiés
- [Drift Response](../operations/drift-response.md) — procédure de réponse au drift
