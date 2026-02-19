# Greenfield Standard — Enterprise Landing Zone

Configuration complète pour un déploiement entreprise conforme au Cloud Adoption Framework.

## Caractéristiques

| Aspect | Choix |
|--------|-------|
| Profil MG | CAF Standard (full hierarchy) |
| Connectivité | Hub-Spoke avec Azure Firewall Premium |
| Identité | Workload Identity Federation (OIDC) |
| CI/CD | GitHub Actions |
| Monitoring | Log Analytics 365j + Defender for Cloud |
| DNS | Private DNS Resolver + forwarders |
| VPN | VpnGw2 |
| Landing Zones | 3 (2 corp + 1 sandbox) |
| Policies | 8 assignments (6 built-in + 2 custom) |

## Architecture

```
Tenant Root Group
├── contoso (root)
│   ├── Platform
│   │   ├── Connectivity    → Hub VNet (10.0.0.0/16) + Firewall + VPN
│   │   ├── Identity        → WIF / Federated credentials
│   │   └── Management      → Log Analytics + Defender + Automation
│   ├── Landing Zones
│   │   ├── Corp
│   │   │   ├── app-team-alpha (10.1.0.0/24) — peered
│   │   │   └── app-team-beta  (10.1.1.0/24) — peered
│   │   ├── Online
│   │   └── Sandbox
│   │       └── sandbox-dev (10.200.0.0/24) — isolated
│   └── Decommissioned
```

## Utilisation

```bash
# 1. Copier la config
cp docs/examples/greenfield-standard/lzctl.yaml ./lzctl.yaml

# 2. Modifier les valeurs spécifiques
#    - metadata.tenant → votre tenant ID
#    - stateBackend.* → votre storage account
#    - landingZones[*].subscription → vos subscription IDs

# 3. Vérifier les prérequis
lzctl doctor

# 4. Valider la configuration
lzctl validate

# 5. Scaffolding
lzctl init --from lzctl.yaml

# 6. Planifier
lzctl plan --tenant contoso

# 7. Déployer (canary first)
lzctl apply --tenant contoso --ring canary
lzctl apply --tenant contoso --ring prod
```

## Customisation

### Ajouter une landing zone

```bash
lzctl workload add --name app-team-gamma \
  --archetype corp \
  --subscription "11111111-..." \
  --address-space "10.1.2.0/24" \
  --connected \
  --tag environment=production \
  --tag costCenter=CC-9999
```

### Changer pour vWAN

Remplacer :

```yaml
connectivity:
  type: hub-spoke
  hub: ...
```

Par :

```yaml
connectivity:
  type: vwan
  hub:
    region: westeurope
    addressSpace: "10.0.0.0/16"
```

### Ajouter un ExpressRoute

```yaml
expressRouteGateway:
  enabled: true
  sku: ErGw1AZ
```

## Fichiers générés

Après `lzctl init`, la structure résultante :

```
.
├── lzctl.yaml
├── modules/
│   ├── resource-org/
│   ├── connectivity-hubspoke/
│   ├── governance/
│   ├── identity-access/
│   ├── management-logs/
│   ├── security/
│   └── policy-as-code/
├── landing-zones/
│   ├── app-team-alpha/
│   ├── app-team-beta/
│   └── sandbox-dev/
├── pipelines/
│   └── github/
│       ├── deploy.yml
│       ├── pr-validation.yml
│       └── drift.yml
└── tenants/
    └── contoso/
```
