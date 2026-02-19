# lzctl validate

Validation multi-couche du manifeste et de la configuration Terraform.

## Synopsis

```bash
lzctl validate [flags]
```

## Description

Exécute trois niveaux de validation :

1. **Schema JSON** — Valide `lzctl.yaml` contre le schéma embarqué
2. **Cross-validation** — Vérifie les règles inter-champs :
   - Format UUID (tenant, subscription, state backend)
   - Chevauchements CIDR (hub vs spokes)
   - Longueur du nom du storage account (3-24 caractères)
   - State versioning et soft delete activés
3. **Terraform validate** — Exécute `terraform validate` sur chaque couche

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--strict` | `false` | Traiter les warnings comme des erreurs |

## Sortie

```
✅ Schema validation passed
✅ Cross-field validation: 0 errors, 0 warnings
✅ Terraform validate: management-groups — ok
✅ Terraform validate: identity — ok
✅ Terraform validate: connectivity — ok
```

## Exit codes

| Code | Signification |
|------|---------------|
| 0 | Tout valide |
| 1 | Erreurs de validation |

En mode `--strict`, les warnings déclenchent aussi un exit code non-zero.

## Exemples

```bash
# Validation standard
lzctl validate

# Strict (CI)
lzctl validate --strict

# JSON output
lzctl validate --json
```

## Voir aussi

- [schema](schema.md) — exporter le schéma JSON
- [init](init.md) — valider après initialisation
