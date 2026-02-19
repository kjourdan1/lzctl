# Design Area : Security

Couche transversale — intégrée dans les couches `management` et `governance`.

## Ce que lzctl configure

- Microsoft Defender for Cloud (plans par type de ressource)
- Encryption at rest (Azure Storage, Key Vault)
- TLS 1.2 minimum sur les services exposés
- Network security (NSG par défaut sur les spokes)
- RBAC least-privilege

## Configuration — lzctl.yaml

```yaml
spec:
  platform:
    management:
      defenderForCloud:
        enabled: true
        plans:
          - Servers
          - AppServices
          - KeyVaults
          - Storage
          - Databases
          - Containers
```

## State backend security

Le state backend est sécurisé avec :

| Contrôle | Détail |
|----------|--------|
| HTTPS only | Trafic chiffré en transit |
| TLS 1.2 | Version TLS minimum |
| AES-256 | Encryption at rest (Microsoft-managed keys) |
| Infrastructure encryption | Double encryption |
| Blob versioning | Audit trail et rollback |
| Soft delete | Protection contre la suppression accidentelle |
| Azure AD auth | `use_azuread_auth = true` — pas de storage access keys |

Vérifier avec : `lzctl state health`

## Audit

`lzctl audit` vérifie dans la discipline Security :
- Defender for Cloud activé et plans configurés
- Encryption des storage accounts
- TLS minimum version
- NSG sur les subnets
- RBAC privileged roles

## Voir aussi

- [State Management](../../operations/state-management.md)
- [Governance](governance.md) — policies de sécurité
