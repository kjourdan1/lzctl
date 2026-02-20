# CI Headless (GitOps)

Guide for running lzctl without terminal interaction (CI pipeline).

## Objective

Execute the following flow without any prompt:

1. `init --ci`
2. `validate`
3. `plan`
4. `apply --auto-approve` (depending on policy)

## Prerequisites

- Non-interactive Azure auth (OIDC recommended)
- Standard CI variables (`CI=true`)
- Declarative one-shot input (`lzctl-init-input.yaml`) or flags/env `LZCTL_*`

## Useful Variables

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

## Recommended Example

```bash
export CI=true

lzctl init \
  --ci \
  --from-file docs/examples/pipeline-init/lzctl-init-input.yaml \
  --repo-root .

lzctl validate --repo-root .
lzctl plan --repo-root .
```

## CI Mode Rules

- `init` in CI requires `--tenant-id` (or `LZCTL_TENANT_ID`) unless `--from-file` is provided.
- `apply` in CI requires `--auto-approve` (except with `--dry-run`).
- `rollback` in CI requires `--auto-approve` (except with `--dry-run`).
- `import` in CI forbids the wizard: provide `--from`, `--subscription`, or `--resource-group`.

## Troubleshooting

- Error `--ci mode requires --tenant-id`:
  - Add `--tenant-id` or `LZCTL_TENANT_ID`, or use `--from-file`.
- Error `--ci mode requires --auto-approve for apply`:
  - Add `--auto-approve` or switch to `--dry-run`.
- Error import source in CI:
  - Add `--from audit-report.json` or `--subscription`.

## Pipeline Examples

- GitHub Actions: [github-actions-onboarding.yml](../examples/pipeline-init/github-actions-onboarding.yml)
- Azure DevOps: [azure-devops-onboarding.yml](../examples/pipeline-init/azure-devops-onboarding.yml)
- One-shot input: [lzctl-init-input.yaml](../examples/pipeline-init/lzctl-init-input.yaml)
