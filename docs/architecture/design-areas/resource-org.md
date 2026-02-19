# Design Area : Resource Organisation

Couche `management-groups` — première couche déployée dans l'ordre CAF.

## Ce que lzctl déploie

- Hiérarchie de management groups selon le modèle choisi
- Assignation des subscriptions aux management groups

## Modèles disponibles

### CAF Standard (5 niveaux)

```
Tenant Root Group
└── <Organisation>
    ├── Platform
    │   ├── Management
    │   ├── Identity
    │   └── Connectivity
    ├── Landing Zones
    │   ├── Corp
    │   ├── Online
    │   └── Sandbox
    └── Decommissioned
```

### CAF Lite (3 niveaux)

```
Tenant Root Group
└── <Organisation>
    ├── Platform
    └── Workloads
```

## Configuration — lzctl.yaml

```yaml
spec:
  platform:
    managementGroups:
      model: caf-standard    # caf-standard | caf-lite
      disabled: []            # MG à exclure (ex: ["sandbox"])
```

## Module AVM

- Source : `Azure/avm-ptn-alz/azurerm`
- Template : `templates/platform/management-groups/`
- State key : `platform-management-groups.tfstate`

## Brownfield

Si des management groups existent déjà :
1. `lzctl audit` détecte la hiérarchie existante
2. `lzctl import` génère les blocs d'import
3. `terraform plan` vérifie zero-diff après import
