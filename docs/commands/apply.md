# lzctl apply

Exécute `terraform apply` sur les couches plateforme en ordre de dépendance CAF.

## Synopsis

```bash
lzctl apply [flags]
```

## Description

Orchestre `terraform apply` sur chaque couche dans l'ordre CAF :
1. `management-groups`
2. `identity`
3. `management`
4. `governance`
5. `connectivity`

Si une couche échoue, l'exécution s'arrête et un message clair indique la couche et l'erreur.

Avant chaque apply, un snapshot automatique des state files est créé dans le CI (via le pipeline généré).

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--layer` | toutes | Couche spécifique à appliquer |
| `--target` | | Alias pour `--layer` |
| `--auto-approve` | `false` | Passer la confirmation (CI uniquement) |
| `--ci` | `false` | Mode strict non-interactif (global) |

En mode CI (`--ci` ou `CI=true`), `lzctl apply` exige `--auto-approve` (sauf `--dry-run`).

## Exemples

```bash
# Apply interactif sur toutes les couches
lzctl apply

# Apply sur une seule couche sans confirmation
lzctl apply --layer connectivity --auto-approve

# CI headless
CI=true lzctl apply --layer connectivity --auto-approve

# Dry-run
lzctl apply --dry-run
```

## Voir aussi

- [plan](plan.md) — prévisualiser avant apply
- [rollback](../operations/rollback.md) — annuler un apply
- [state snapshot](../operations/state-management.md) — sauvegarder avant apply
