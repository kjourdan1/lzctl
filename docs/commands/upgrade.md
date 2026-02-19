# lzctl upgrade

Met à jour les versions des modules Terraform vers les dernières versions disponibles.

## Synopsis

```bash
lzctl upgrade [flags]
```

## Description

Scanne tous les fichiers `.tf` du projet, identifie les pins de version (`version = "..."`) dans les blocs `module {}`, et vérifie les versions disponibles sur le [Terraform Registry](https://registry.terraform.io/).

Le scan :
1. Parcourt récursivement les `.tf` (ignore `.terraform/`, `.git/`, `node_modules/`)
2. Extrait `source` + `version` de chaque bloc module
3. Interroge l'API Registry (`/v1/modules/.../versions`)
4. Compare la version locale avec la dernière version stable
5. Affiche les upgrades disponibles (ou les applique avec `--apply`)

### Opérateurs de contrainte préservés

L'updater conserve l'opérateur de contrainte :

| Avant | Après (`--apply`) |
|-------|-------------------|
| `version = "1.2.0"` | `version = "1.3.0"` |
| `version = "~> 1.2.0"` | `version = "~> 1.3.0"` |
| `version = ">= 1.2.0"` | `version = ">= 1.3.0"` |

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--apply` | `false` | Appliquer les mises à jour aux fichiers `.tf` |
| `--module` | | Filtrer par nom de module (exact match) |
| `--dry-run` | `false` | Afficher les changements sans modifier (identique à l'absence de `--apply`) |
| `--json` | `false` | Sortie JSON structurée |

## Exemples

```bash
# Lister les upgrades disponibles
lzctl upgrade

# Module spécifique
lzctl upgrade --module resource-org

# Appliquer les mises à jour
lzctl upgrade --apply

# Sortie JSON
lzctl upgrade --json

# Pipeline : check + apply
lzctl upgrade --json -o upgrades.json
cat upgrades.json | jq '.upgrades | length'
lzctl upgrade --apply
```

## Sortie texte

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

## Sortie JSON (`--json`)

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

## Compatibilité

L'upgrade scanner supporte les modules publiés sur :
- **Terraform Registry** (`registry.terraform.io`) — support complet
- **Modules privés** — non supporté actuellement (skippés avec warning)

## Voir aussi

- [`validate`](validate.md) — valider après upgrade
- [`plan`](plan.md) — vérifier les changements liés aux nouvelles versions
- [`doctor`](doctor.md) — vérifier les prérequis Terraform
