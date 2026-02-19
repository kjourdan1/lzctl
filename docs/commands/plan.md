# lzctl plan

Exécute `terraform plan` sur les couches plateforme en ordre de dépendance CAF.

## Synopsis

```bash
lzctl plan [flags]
```

## Description

Orchestre `terraform plan` sur chaque couche dans l'ordre :
1. `management-groups` — Resource Organisation
2. `identity` — Identity & Access
3. `management` — Management & Monitoring
4. `governance` — Azure Policies
5. `connectivity` — Hub-Spoke ou vWAN

Chaque couche utilise son propre state file dans le backend partagé.

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--layer` | toutes | Couche spécifique à planifier |
| `--target` | | Alias pour `--layer` |
| `--out` | | Enregistrer le résumé dans un fichier |

## Exemples

```bash
# Plan sur toutes les couches
lzctl plan

# Plan sur une couche spécifique
lzctl plan --layer connectivity

# Enregistrer le résumé
lzctl plan --out plan-output.txt

# Sortie JSON
lzctl plan --json
```

## Sortie

```
═══ Planning: management-groups ═══
  No changes. Infrastructure is up-to-date.

═══ Planning: identity ═══
  No changes. Infrastructure is up-to-date.

═══ Planning: connectivity ═══
  Plan: 3 to add, 0 to change, 0 to destroy.
```

## Voir aussi

- [apply](apply.md) — appliquer les changements
- [drift](drift.md) — détecter le drift
