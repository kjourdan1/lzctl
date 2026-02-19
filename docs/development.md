# Development Guide

Guide technique pour les contributeurs et mainteneurs.

## Architecture du code

```
cmd/                    Commandes Cobra CLI
internal/               Logique métier
  applier/              Orchestration terraform apply
  assess/               Évaluation de la maturité
  audit/                Audit de conformité CAF
  azauth/               Authentification Azure
  azure/                Wrapper az CLI
  bootstrap/            Bootstrap du state backend
  config/               Parsing, validation, cross-checks de lzctl.yaml
  doctor/               Vérification des prérequis
  drift/                Détection de drift
  exitcode/             Codes de sortie standardisés
  importer/             Génération de blocs import Terraform
  manifest/             Lecture/écriture du manifeste
  output/               Formatage de sortie (Info, Success, Warning, Error)
  planner/              Orchestration terraform plan
  planverify/           Vérification des plans
  policy/               Policy-as-Code lifecycle
  profiles/             Catalogue de profils CAF
  rollback/             Rollback multi-couche
  scaffold/             Génération de structure projet
  schema/               Validation JSON Schema
  state/                Gestion du lifecycle des state files
  template/             Moteur de templates Go
  tfutil/               Utilitaires Terraform
  upgrade/              Vérification et mise à jour des modules AVM
  validate/             Orchestration des validations
  wizard/               Wizard interactif (survey)
  workload/             Gestion des landing zones
schemas/                Schéma JSON embarqué (embed.FS)
templates/              Templates Go (embed.FS)
  manifest/             Templates lzctl.yaml
  shared/               Templates partagés (backend, providers, gitignore)
  platform/             Templates Terraform par couche
  pipelines/            Templates CI/CD (GitHub Actions, Azure DevOps)
profiles/               Catalogue de profils CAF (catalog.yaml)
policies/               Artefacts Policy-as-Code
```

## Conventions

### Commandes (cmd/)

Les fichiers dans `cmd/` sont des **wrappers fins** :
1. Parser les flags
2. Charger la configuration
3. Appeler la logique dans `internal/`
4. Formater et afficher le résultat

```go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Description courte",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := config.Load(cfgFile, repoRoot)
        if err != nil {
            return err
        }
        return mypackage.Run(cfg)
    },
}
```

### Packages internes

- Pas de dépendance sur Cobra
- Interfaces pour la testabilité
- Erreurs wrappées : `fmt.Errorf("context: %w", err)`

### Output

Utiliser `internal/output` pour les messages :

```go
output.Info("Processing layer: %s", layer)
output.Success("Deployment complete")
output.Warning("Versioning is disabled")
output.Error("Failed to apply: %v", err)
```

### Templates

Les templates Go sont dans `templates/` et embarqués via `embed.FS` :

```
templates/
├── manifest/
│   └── lzctl.yaml.tmpl
├── shared/
│   ├── backend.tf.tmpl
│   ├── backend.hcl.tmpl
│   ├── providers.tf.tmpl
│   └── gitignore.tmpl
├── platform/
│   ├── management-groups/
│   ├── identity/
│   ├── management/
│   ├── governance/
│   └── connectivity/
└── pipelines/
    ├── github/
    └── azuredevops/
```

## Debug

```bash
# Verbose output
lzctl plan -vvv

# Dry-run
lzctl apply --dry-run
```

## Ajouter une nouvelle commande

1. Créer `cmd/<command>.go` avec une commande Cobra
2. Créer `internal/<package>/<package>.go` avec la logique
3. Wire dans `cmd/root.go` : `rootCmd.AddCommand(myCmd)`
4. Ajouter tests dans `cmd/cmd_test.go` et `internal/<package>/<package>_test.go`
5. Documenter dans `docs/commands/<command>.md`
6. Mettre à jour `docs/cli-reference.md`

## Ajouter une règle d'audit

1. Créer la fonction dans `internal/audit/compliance_engine.go`
2. Suivre le pattern existant `check<Rule>()`
3. Tester avec des fixtures dans `audit_test.go`
4. Documenter la règle et la remédiation

## Ajouter un check doctor

1. Ajouter dans `internal/doctor/checks.go`
2. Catégoriser : `tools`, `auth`, `azure`, `state`
3. Retourner un `CheckResult` avec Name, Status, Details, Fix
4. Tester dans `checks_test.go`

## Ajouter un champ à lzctl.yaml

1. Ajouter le champ dans `internal/config/schema.go`
2. Ajouter la valeur par défaut dans `internal/config/defaults.go`
3. Mettre à jour `schemas/lzctl-v1.schema.json`
4. Ajouter la validation cross-field si nécessaire dans `internal/config/crossvalidator.go`
5. Mettre à jour le template dans `templates/manifest/lzctl.yaml.tmpl`
6. Documenter dans le README
