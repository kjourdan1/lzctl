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

`--from-file` permet de fournir un input déclaratif transitoire (`lzctl-init-input.yaml`) converti en `lzctl.yaml` complet lors de l'init.

En mode non-interactif, `init` peut aussi être piloté par flags ou variables d'environnement (`LZCTL_*`) avec priorité : **flag > env > défaut**.

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--tenant-id` | auto-détecté | Azure AD tenant ID |
| `--subscription-id` | auto-détecté | Azure Subscription ID |
| `--from-file` | | Input one-shot à convertir en `lzctl.yaml` |
| `--project-name` | `landing-zone` | Nom du projet |
| `--mg-model` | `caf-standard` | Modèle MG (`caf-standard` \| `caf-lite`) |
| `--connectivity` | `hub-spoke` | Modèle connectivité (`hub-spoke` \| `vwan` \| `none`) |
| `--identity` | `workload-identity-federation` | Modèle identité (`workload-identity-federation` \| `sp-federated` \| `sp-secret`) |
| `--primary-region` | `westeurope` | Région primaire |
| `--secondary-region` | vide | Région secondaire optionnelle |
| `--cicd-platform` | `github-actions` | Plateforme CI/CD (`github-actions` \| `azure-devops`) |
| `--state-strategy` | `create-new` | Stratégie backend (`create-new` \| `existing` \| `terraform-cloud`) |
| `--force` | `false` | Écraser les fichiers existants |
| `--no-bootstrap` | `false` | Ne pas provisionner le state backend |
| `--ci` | `false` | Mode strict non-interactif (échec si paramètre requis absent) |
| `--config` | global | Charger depuis un fichier (mode non-interactif) |

### Variables d'environnement supportées

- `LZCTL_TENANT_ID`
- `LZCTL_SUBSCRIPTION_ID`
- `LZCTL_FROM_FILE`
- `LZCTL_PROJECT_NAME`
- `LZCTL_MG_MODEL`
- `LZCTL_CONNECTIVITY`
- `LZCTL_IDENTITY`
- `LZCTL_PRIMARY_REGION`
- `LZCTL_SECONDARY_REGION`
- `LZCTL_CICD_PLATFORM`
- `LZCTL_STATE_STRATEGY`

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

# Input one-shot converti en lzctl.yaml
lzctl init --from-file docs/examples/pipeline-init/lzctl-init-input.yaml

# Non-interactif 100% flags
lzctl init \
    --tenant-id 00000000-0000-0000-0000-000000000001 \
    --project-name contoso-platform \
    --mg-model caf-lite \
    --connectivity none \
    --cicd-platform github-actions

# Non-interactif via variables d'environnement
LZCTL_TENANT_ID=00000000-0000-0000-0000-000000000001 \
LZCTL_MG_MODEL=caf-standard \
LZCTL_CONNECTIVITY=hub-spoke \
lzctl init --repo-root ./lz-repo

# Mode CI strict (aucun prompt)
CI=true lzctl init --ci --tenant-id 00000000-0000-0000-0000-000000000001

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
- [CI headless](../operations/ci-headless.md) — exécuter init/validate/plan en pipeline non-interactif
