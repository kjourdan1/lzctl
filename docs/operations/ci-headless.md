# CI Headless (GitOps)

Guide d'exécution de lzctl sans interaction terminale (pipeline CI).

## Objectif

Exécuter le flux suivant sans prompt:

1. `init --ci`
2. `validate`
3. `plan`
4. `apply --auto-approve` (selon politique)

## Pré-requis

- Auth Azure non-interactive (OIDC recommandé)
- Variables CI standards (`CI=true`)
- Input déclaratif one-shot (`lzctl-init-input.yaml`) ou flags/env `LZCTL_*`

## Variables utiles

- `LZCTL_TENANT_ID`
- `LZCTL_SUBSCRIPTION_ID`
- `LZCTL_PROJECT_NAME`
- `LZCTL_MG_MODEL`
- `LZCTL_CONNECTIVITY`
- `LZCTL_IDENTITY`
- `LZCTL_PRIMARY_REGION`
- `LZCTL_SECONDARY_REGION`
- `LZCTL_CICD_PLATFORM`
- `LZCTL_STATE_STRATEGY`

## Exemple recommandé

```bash
export CI=true

lzctl init \
  --ci \
  --from-file docs/examples/pipeline-init/lzctl-init-input.yaml \
  --repo-root .

lzctl validate --repo-root .
lzctl plan --repo-root .
```

## Règles mode CI

- `init` en CI exige `--tenant-id` (ou `LZCTL_TENANT_ID`) sauf si `--from-file` est fourni.
- `apply` en CI exige `--auto-approve` (hors `--dry-run`).
- `rollback` en CI exige `--auto-approve` (hors `--dry-run`).
- `import` en CI interdit le wizard: fournir `--from`, `--subscription` ou `--resource-group`.

## Troubleshooting

- Erreur `--ci mode requires --tenant-id`:
  - Ajouter `--tenant-id` ou `LZCTL_TENANT_ID`, ou utiliser `--from-file`.
- Erreur `--ci mode requires --auto-approve for apply`:
  - Ajouter `--auto-approve` ou passer en `--dry-run`.
- Erreur import source en CI:
  - Ajouter `--from audit-report.json` ou `--subscription`.

## Exemples pipeline

- GitHub Actions: [github-actions-onboarding.yml](../examples/pipeline-init/github-actions-onboarding.yml)
- Azure DevOps: [azure-devops-onboarding.yml](../examples/pipeline-init/azure-devops-onboarding.yml)
- Input one-shot: [lzctl-init-input.yaml](../examples/pipeline-init/lzctl-init-input.yaml)
