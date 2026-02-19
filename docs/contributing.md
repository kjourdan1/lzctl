# Contributing to lzctl

## Development Setup

```bash
# Prérequis
go version    # >= 1.24
terraform -v  # >= 1.5
git --version # >= 2.30

# Clone et build
git clone https://github.com/kjourdan1/lzctl.git
cd lzctl
go mod tidy
go build -o bin/lzctl .
```

## Workflow de développement

1. Créer une branche : `git checkout -b feature/<name>`
2. Développer
3. Lancer les vérifications :
   ```bash
   make fmt      # Formater le code
   make vet      # Analyse statique
   make test     # Lancer tous les tests
   make build    # Vérifier la compilation
   ```
4. Valider les manifestes : `lzctl validate --strict`
5. Soumettre une Pull Request

## Organisation du code

| Répertoire | Rôle |
|------------|------|
| `cmd/` | Commandes Cobra CLI (wrappers fins appelant `internal/`) |
| `internal/` | Logique métier (config, applier, audit, drift, state, policy, etc.) |
| `schemas/` | Schéma JSON embarqué pour validation de `lzctl.yaml` |
| `templates/` | Templates Go pour la génération Terraform, pipelines, manifestes |
| `profiles/` | Catalogue de profils CAF (`catalog.yaml`) |
| `policies/` | Artefacts Policy-as-Code (définitions, initiatives, assignments) |
| `docs/` | Documentation (commandes, architecture, opérations) |

## Tests

```bash
# Tous les tests
go test ./...

# Un package spécifique
go test ./internal/config/...

# Un test spécifique
go test ./internal/config/ -run TestCrossValidate

# Avec sortie détaillée
go test -v ./internal/audit/...

# Avec race detection
go test -race ./...
```

Voir [TESTING.md](../TESTING.md) pour le guide complet des tests.

## Conventions

- **Commands** (`cmd/`): Fins — parser les flags, appeler `internal/`, afficher le résultat
- **Internal packages**: Toute la logique métier. Pas de dépendances Cobra.
- **Manifest**: `apiVersion: lzctl/v1`, `kind: LandingZone`
- **Erreurs**: `fmt.Errorf("context: %w", err)` — wrapper, jamais avaler
- **Output**: Utiliser `internal/output` pour les messages formatés (Info, Success, Warning, Error)

## Ajouter une nouvelle commande

1. Créer `cmd/<command>.go` avec une commande Cobra
2. Créer `internal/<package>/<package>.go` avec la logique
3. Connecter dans `cmd/root.go` (`rootCmd.AddCommand`)
4. Ajouter des tests dans `cmd/cmd_test.go` et `internal/<package>/<package>_test.go`
5. Mettre à jour la CLI Reference dans `docs/cli-reference.md`

## Ajouter une règle d'audit

1. Créer une fonction dans `internal/audit/` suivant le pattern `check<Rule>()`
2. Ajouter à la liste dans `compliance_engine.go`
3. Tester avec des fixtures dans `audit_test.go`
4. Documenter dans `docs/commands/audit.md`
