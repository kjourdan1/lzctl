# Documentation des commandes lzctl

## Flags globaux

| Flag | Court | Défaut | Description |
|------|-------|--------|-------------|
| `--config` | | `lzctl.yaml` | Fichier de configuration |
| `--repo-root` | | `.` | Racine du repository |
| `--verbose` | `-v` | `0` | Verbosité (-v, -vv, -vvv) |
| `--dry-run` | | `false` | Simuler sans modifier Azure |
| `--json` | | `false` | Sortie JSON |

## Commandes

### Scaffolding & Configuration

| Commande | Description | Doc |
|----------|-------------|-----|
| [init](init.md) | Initialiser un projet landing zone | ✅ |
| [validate](validate.md) | Valider lzctl.yaml et la configuration Terraform | ✅ |
| [select](select.md) | Parcourir le catalogue CAF | — |
| [schema](schema.md) | Exporter/valider le schéma JSON | — |
| [docs](docs.md) | Générer la documentation du projet | — |

### Opérations Terraform

| Commande | Description | Doc |
|----------|-------------|-----|
| [plan](plan.md) | Plan multi-couche en ordre CAF | ✅ |
| [apply](apply.md) | Apply multi-couche en ordre CAF | ✅ |
| [drift](drift.md) | Détecter le drift d'infrastructure | ✅ |
| [rollback](rollback.md) | Rollback en ordre inverse CAF | — |

### Day-2

| Commande | Description | Doc |
|----------|-------------|-----|
| [status](status.md) | Aperçu de l'état du projet | ✅ |
| [upgrade](upgrade.md) | Vérifier/appliquer les mises à jour AVM | ✅ |
| [audit](audit.md) | Audit de conformité CAF | ✅ |
| [assess](assess.md) | Évaluation de la maturité | — |
| [import](import.md) | Générer des blocs d'import Terraform | ✅ |
| [doctor](doctor.md) | Vérifier les prérequis | ✅ |

### State Management

| Commande | Description |
|----------|-------------|
| `lzctl state list` | Lister les fichiers d'état |
| `lzctl state snapshot` | Créer un snapshot |
| `lzctl state health` | Vérifier la sécurité du backend |
| `lzctl state unlock` | Débloquer un lease |

Voir [State Management Guide](../operations/state-management.md).

### Policy-as-Code

| Commande | Description |
|----------|-------------|
| `lzctl policy create` | Scaffolder une définition |
| `lzctl policy test` | Déployer en audit mode |
| `lzctl policy verify` | Rapport de conformité |
| `lzctl policy remediate` | Créer des tâches de remédiation |
| `lzctl policy deploy` | Passer en enforcement |
| `lzctl policy status` | État du workflow |
| `lzctl policy diff` | Comparer local vs déployé |

### Workload Management

| Commande | Description |
|----------|-------------|
| `lzctl workload add` | Ajouter une landing zone |
| `lzctl workload adopt` | Adopter une subscription existante |
| `lzctl workload list` | Lister les landing zones |
| `lzctl workload remove` | Supprimer une landing zone |
