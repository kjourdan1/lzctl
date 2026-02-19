# lzctl init

Initialise un nouveau projet landing zone.

## Synopsis

```bash
lzctl init [flags]
```

## Description

Lance un wizard interactif qui collecte :
1. Nom du projet
2. Tenant ID Azure
3. Plateforme CI/CD (GitHub Actions / Azure DevOps)
4. Modèle de management groups (CAF Standard / CAF Lite)
5. Modèle de connectivité (Hub & Spoke / vWAN / None)
6. Région primaire (et optionnellement secondaire)
7. Configuration du state backend

Génère ensuite :
- `lzctl.yaml` — manifeste déclaratif (source de vérité)
- `platform/` — couches Terraform (management-groups, identity, management, governance, connectivity)
- `landing-zones/` — stubs pour les workloads
- `pipelines/` — fichiers CI/CD adaptés à la plateforme choisie
- `backend.hcl` — configuration du state backend
- `.gitignore`, `README.md`

Si `--config` est fourni, charge la configuration depuis le fichier YAML et skip le wizard.

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--tenant-id` | auto-détecté | Azure AD tenant ID |
| `--subscription-id` | auto-détecté | Azure Subscription ID |
| `--force` | `false` | Écraser les fichiers existants |
| `--no-bootstrap` | `false` | Ne pas provisionner le state backend |
| `--config` | global | Charger depuis un fichier (mode non-interactif) |

## Bootstrap du state backend

Par défaut, `lzctl init` provisionne automatiquement :
- Un resource group (`rg-<project>-tfstate-<region>`)
- Un storage account (avec versioning, soft delete, encryption)
- Un blob container (`tfstate`)
- Un managed identity avec les droits nécessaires
- Des federated credentials OIDC pour CI/CD

Le bootstrap utilise `az` CLI directement (pas Terraform — évite le problème chicken-and-egg).

Passer `--no-bootstrap` pour sauter cette étape si le backend existe déjà.

## Exemples

```bash
# Wizard interactif (recommandé)
lzctl init

# Non-interactif depuis un fichier
lzctl init --config lzctl.yaml

# Dry-run (prévisualiser sans écrire)
lzctl init --dry-run

# Forcer la réécriture
lzctl init --force
```

## Structure générée

```
.
├── lzctl.yaml
├── backend.hcl
├── .gitignore
├── README.md
├── platform/
│   ├── management-groups/
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── terraform.tfvars
│   ├── identity/
│   ├── management/
│   ├── governance/
│   └── connectivity/
├── landing-zones/
└── pipelines/
    └── .github/workflows/  (ou .azuredevops/)
```

## Voir aussi

- [validate](validate.md) — valider après init
- [doctor](doctor.md) — vérifier les prérequis avant init
