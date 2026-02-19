# Design Area : Governance

Couche `governance` — quatrième couche dans l'ordre CAF.

## Ce que lzctl déploie

- Azure Policy assignments (jeux de policies CAF)
- Policy definitions personnalisées
- Policy initiatives
- Policy exemptions (avec expiration obligatoire)

## Configuration — lzctl.yaml

```yaml
spec:
  governance:
    policies:
      assignments:        # Jeux de policies CAF built-in
        - caf-default
      custom: []          # Chemins vers des policies custom
```

## Policy-as-Code Lifecycle

lzctl propose un workflow complet pour les policies :

```
create → test → verify → deploy
```

| Étape | Commande | Description |
|-------|----------|-------------|
| Scaffold | `lzctl policy create` | Générer une définition/initiative/assignment |
| Test | `lzctl policy test` | Déployer en mode `DoNotEnforce` |
| Verify | `lzctl policy verify` | Générer un rapport de conformité |
| Remediate | `lzctl policy remediate` | Créer des tâches de remédiation |
| Deploy | `lzctl policy deploy` | Passer en enforcement `Default` |
| Status | `lzctl policy status` | Voir l'état du workflow |
| Diff | `lzctl policy diff` | Comparer local vs déployé |

## Module AVM

- Template : `templates/platform/governance/`
- State key : `platform-governance.tfstate`
- Artefacts : `policies/` (definitions, initiatives, assignments, exemptions)

## Brownfield

`lzctl audit` vérifie :
- Les policy assignments existantes
- La couverture de conformité
- Les gaps par rapport aux recommandations CAF
