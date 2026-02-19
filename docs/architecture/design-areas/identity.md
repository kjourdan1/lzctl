# Design Area : Identity & Access

Couche `identity` — deuxième couche dans l'ordre CAF.

## Ce que lzctl déploie

- Managed identity pour CI/CD (Workload Identity Federation)
- RBAC role assignments pour la managed identity
- Federated credentials pour GitHub Actions ou Azure DevOps

## Configuration — lzctl.yaml

```yaml
spec:
  platform:
    identity:
      type: workload-identity-federation  # wif | sp-federated | sp-secret
      clientId: ""     # Rempli après bootstrap
      principalId: ""  # Rempli après bootstrap
```

## Types d'identité

| Type | Description | Recommandé |
|------|-------------|------------|
| `workload-identity-federation` | OIDC, pas de secret | ✅ Oui |
| `sp-federated` | Service Principal + federated credential | Acceptable |
| `sp-secret` | Service Principal + secret | ⚠️ Non recommandé |

## Module AVM

- Template : `templates/platform/identity/`
- State key : `platform-identity.tfstate`

## Security

- Les credentials ne sont jamais stockés dans le code
- Le bootstrap configure automatiquement les federated credentials
- Les secrets CI/CD sont stockés dans GitHub Secrets ou Azure DevOps Variable Groups
