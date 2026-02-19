# Design Area : Management & Monitoring

Couche `management` — troisième couche dans l'ordre CAF.

## Ce que lzctl déploie

- Log Analytics workspace
- Automation Account (optionnel)
- Defender for Cloud (plans configurables)
- Diagnostic settings

## Configuration — lzctl.yaml

```yaml
spec:
  platform:
    management:
      logAnalytics:
        retentionDays: 90        # 30-730
        solutions:               # Solutions optionnelles
          - SecurityInsights
          - VMInsights
      automationAccount: true
      defenderForCloud:
        enabled: true
        plans:
          - Servers
          - AppServices
          - KeyVaults
          - Storage
```

## Module AVM

- Template : `templates/platform/management/`
- State key : `platform-management.tfstate`

## Brownfield

Pour les tenants avec un Log Analytics existant :
1. `lzctl audit` vérifie la présence de diagnostic settings
2. `lzctl import` peut importer le workspace existant
3. La rétention et les solutions peuvent être ajustées dans `lzctl.yaml`
