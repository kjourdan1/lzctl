# lzctl

> **Landing Zone Factory CLI** — Orchestrateur stateless pour Azure Landing Zones, aligné Cloud Adoption Framework.

[![Go](https://img.shields.io/badge/Go-1.24-blue)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache_2.0-green)](LICENSE)

## Pourquoi lzctl ?

Les équipes plateforme qui déploient des Azure Landing Zones font face à un problème récurrent : chaque tenant a un setup Terraform ad-hoc, les policies sont appliquées manuellement via le portail, et personne ne sait ce qui est réellement déployé vs ce qui est en code.

**lzctl résout cela** en ajoutant la couche d'orchestration manquante au-dessus du chemin recommandé par Microsoft (Azure Verified Modules). Il transforme la gestion des landing zones en produit : versionné, testé et déployé via PR — exactement comme du code applicatif, mais pour l'infrastructure plateforme.

## Fonctionnalités

| Catégorie | Fonctionnalités |
|-----------|----------------|
| **Scaffolding** | Wizard interactif, génération Terraform layerisée, bootstrap state backend |
| **Validation** | Schéma JSON, cross-validation (CIDR overlaps, UUID, state backend) |
| **Orchestration** | Plan/Apply multi-couche en ordre CAF, rollback automatisé |
| **Conformité** | Audit CAF 6 disciplines, score de conformité, Policy-as-Code lifecycle |
| **Day-2 Ops** | Drift detection, module upgrade, state lifecycle management |
| **Brownfield** | Import de ressources existantes via `terraform import` natif |
| **Workloads** | Ajout/adoption/suppression de landing zones (subscriptions) |

## Architecture

```
lzctl.yaml                    ← Source de vérité (déclaratif)
    │
    ├── platform/
    │   ├── management-groups/  (1. Resource Organisation)
    │   ├── identity/           (2. Identity & Access)
    │   ├── management/         (3. Management & Monitoring)
    │   ├── governance/         (4. Azure Policies)
    │   └── connectivity/       (5. Hub-Spoke ou vWAN)
    │
    ├── landing-zones/          (Workload subscriptions)
    ├── pipelines/              (CI/CD — GitHub Actions ou Azure DevOps)
    └── backend.hcl             (State backend partagé)
```

Chaque couche utilise des **Azure Verified Modules (AVM)** avec des versions pinées et un **state Terraform séparé** dans un Azure Storage Account partagé.

## Quick Start

### Prérequis

| Outil | Version | Usage |
|-------|---------|-------|
| Go | >= 1.24 | Compilation |
| Terraform | >= 1.5 | Déploiement IaC |
| Azure CLI | >= 2.50 | Authentification + opérations Azure |
| Git | >= 2.30 | Versioning |
| GitHub CLI | optionnel | Intégration GitHub |

### Installation

```bash
# Depuis les sources
git clone https://github.com/kjourdan1/lzctl.git
cd lzctl
go build -o bin/lzctl .

# Vérification
./bin/lzctl version
./bin/lzctl doctor
```

### Déploiement d'une Landing Zone

```bash
# 1. Vérifier les prérequis
lzctl doctor

# 2. Initialiser le projet (wizard interactif + bootstrap state backend)
lzctl init

# 3. Valider la configuration
lzctl validate

# 4. Prévisualiser les changements
lzctl plan

# 5. Déployer (layers en ordre CAF)
lzctl apply --auto-approve

# 6. Vérifier le statut
lzctl status
```

### Ajouter une Landing Zone

```bash
# Interactif
lzctl workload add --name app-prod --archetype corp --address-space 10.2.0.0/24

# Adopter une subscription existante
lzctl workload adopt --name legacy-app --subscription 00000000-0000-0000-0000-000000000000

# Lister
lzctl workload list
```

## Commandes

| Commande | Description |
|----------|-------------|
| `lzctl init` | Initialiser un nouveau projet landing zone |
| `lzctl validate` | Valider le manifeste et la configuration Terraform |
| `lzctl plan` | Terraform plan multi-couche en ordre CAF |
| `lzctl apply` | Terraform apply multi-couche en ordre CAF |
| `lzctl drift` | Détecter le drift d'infrastructure |
| `lzctl status` | Aperçu de l'état du projet |
| `lzctl rollback` | Rollback des couches en ordre inverse |
| `lzctl audit` | Audit de conformité CAF du tenant Azure |
| `lzctl assess` | Évaluation de la maturité du projet |
| `lzctl select` | Parcourir le catalogue de couches CAF |
| `lzctl upgrade` | Vérifier/appliquer les mises à jour AVM |
| `lzctl doctor` | Vérifier les prérequis et l'environnement |
| `lzctl import` | Générer des blocs d'import Terraform |
| `lzctl schema` | Exporter/valider le schéma JSON |
| `lzctl docs` | Générer la documentation du projet |
| `lzctl state list` | Lister les fichiers d'état Terraform |
| `lzctl state snapshot` | Créer un snapshot des fichiers d'état |
| `lzctl state health` | Vérifier la posture de sécurité du state backend |
| `lzctl state unlock` | Forcer le déblocage d'un lease stuck |
| `lzctl policy create` | Scaffolder une définition de policy |
| `lzctl policy test` | Déployer en mode audit (DoNotEnforce) |
| `lzctl policy verify` | Générer un rapport de conformité |
| `lzctl policy remediate` | Créer des tâches de remédiation |
| `lzctl policy deploy` | Passer en enforcement Default |
| `lzctl policy status` | Voir l'état du workflow policy |
| `lzctl policy diff` | Comparer local vs déployé |
| `lzctl workload add` | Ajouter une landing zone |
| `lzctl workload adopt` | Adopter une subscription existante |
| `lzctl workload list` | Lister les landing zones |
| `lzctl workload remove` | Supprimer une landing zone |
| `lzctl version` | Afficher la version |

Voir la [référence CLI complète](docs/cli-reference.md) pour les détails de chaque commande.

## Flags globaux

| Flag | Court | Défaut | Description |
|------|-------|--------|-------------|
| `--config` | | `lzctl.yaml` | Chemin du fichier de configuration |
| `--repo-root` | | `.` | Chemin racine du repository |
| `--verbose` | `-v` | `0` | Niveau de verbosité (-v, -vv, -vvv) |
| `--dry-run` | | `false` | Simuler sans modifier Azure |
| `--json` | | `false` | Sortie JSON (machine-readable) |

## Configuration — lzctl.yaml

```yaml
apiVersion: lzctl/v1
kind: LandingZone

metadata:
  name: contoso-platform
  tenant: 00000000-0000-0000-0000-000000000000
  primaryRegion: westeurope

spec:
  platform:
    managementGroups:
      model: caf-standard    # caf-standard | caf-lite
    connectivity:
      type: hub-spoke        # hub-spoke | vwan | none
      hub:
        region: westeurope
        addressSpace: 10.0.0.0/16
        firewall:
          enabled: true
          sku: Standard
    identity:
      type: workload-identity-federation
    management:
      logAnalytics:
        retentionDays: 90

  stateBackend:
    resourceGroup: rg-contoso-tfstate-weu
    storageAccount: stcontosotfstate
    container: tfstate
    subscription: 00000000-0000-0000-0000-000000000000
    versioning: true          # Blob versioning (audit trail + rollback)
    softDelete: true          # Protection contre la suppression accidentelle
    softDeleteDays: 30

  landingZones:
    - name: app-prod
      subscription: 00000000-0000-0000-0000-000000000000
      archetype: corp
      addressSpace: 10.1.0.0/24
      connected: true

  cicd:
    platform: github-actions  # github-actions | azure-devops
    branchPolicy:
      mainBranch: main
      requirePR: true
```

## Principes de conception

| Principe | Description |
|----------|-------------|
| **Stateless CLI** | lzctl ne stocke aucun état local ; tout vit dans `lzctl.yaml` + Git |
| **Terraform natif** | Le code généré fonctionne avec `terraform plan/apply` standard — aucune dépendance runtime à lzctl |
| **Azure Verified Modules** | Modules AVM maintenus par Microsoft, versions pinées |
| **GitOps** | Les PR déclenchent `plan`, les merges déclenchent `apply` |
| **State as First-Class** | Versioning, soft delete, health checks, snapshots automatiques |
| **Séparation des couches** | Un state file par couche CAF — blast radius réduit |

## Documentation

| Document | Description |
|----------|-------------|
| [CLI Reference](docs/cli-reference.md) | Référence complète des commandes et flags |
| [Architecture](docs/architecture.md) | Décisions architecturales (ADRs) |
| [Contributing](CONTRIBUTING.md) | Guide de contribution |
| [State Management](docs/operations/state-management.md) | Guide opérationnel de gestion du state |
| [CI Headless](docs/operations/ci-headless.md) | Exécuter init/validate/plan/apply sans interaction |
| [Rollback](docs/operations/rollback.md) | Procédures de rollback |
| [Drift Response](docs/operations/drift-response.md) | Réponse au drift d'infrastructure |
| [Testing](TESTING.md) | Guide de tests |

## Licence

[Apache License 2.0](LICENSE)
