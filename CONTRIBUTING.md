# Contributing

Merci de votre intérêt pour contribuer à **lzctl** !

## Quick Start

```bash
git clone https://github.com/kjourdan1/lzctl.git
cd lzctl
go mod tidy
go build -o bin/lzctl .
go test ./...
```

## Workflow

1. Fork le repository
2. Créer une branche : `git checkout -b feature/<name>`
3. Développer avec des tests
4. Vérifier : `make fmt && make vet && make test`
5. Soumettre une Pull Request

## Standards

- Code formaté avec `gofmt`
- Tests unitaires pour toute logique dans `internal/`
- Commandes CLI (`cmd/`) = wrappers fins — la logique vit dans `internal/`
- Erreurs wrappées : `fmt.Errorf("context: %w", err)`
- Pas de credentials dans le code ou les templates

## Structure du projet

```
cmd/              Commandes Cobra CLI
internal/         Logique métier
  applier/        Orchestration terraform apply
  audit/          Audit de conformité CAF
  config/         Parsing et validation de lzctl.yaml
  doctor/         Vérification des prérequis
  drift/          Détection de drift
  output/         Formatage de sortie (couleurs, JSON)
  policy/         Policy-as-Code lifecycle
  state/          Gestion du lifecycle des state files
  template/       Moteur de templates Go
  ...
schemas/          Schéma JSON pour lzctl.yaml
templates/        Templates Terraform, pipelines, manifestes
profiles/         Catalogue de profils CAF
policies/         Définitions de policies Azure
docs/             Documentation
```

## Tests

```bash
go test ./...                    # Tous les tests
go test -v ./internal/config/... # Un package avec détails
go test -race ./...              # Avec race detection
```

## Licence

En contribuant, vous acceptez que vos contributions soient sous licence [Apache 2.0](LICENSE).
