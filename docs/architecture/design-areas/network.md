# Design Area : Network Topology

Couche `connectivity` — cinquième couche dans l'ordre CAF.

## Ce que lzctl déploie

- Hub VNet (ou vWAN hub)
- Azure Firewall (optionnel, Standard ou Premium)
- VPN Gateway (optionnel)
- ExpressRoute Gateway (optionnel)
- Private DNS Resolver (optionnel)
- Peering entre hub et spokes (landing zones)

## Configuration — lzctl.yaml

```yaml
spec:
  platform:
    connectivity:
      type: hub-spoke          # hub-spoke | vwan | none
      hub:
        region: westeurope
        addressSpace: 10.0.0.0/16
        firewall:
          enabled: true
          sku: Standard         # Standard | Premium
          threatIntel: Alert    # Off | Alert | Deny
        dns:
          privateResolver: true
          forwarders: []
        vpnGateway:
          enabled: false
          sku: VpnGw1
        expressRouteGateway:
          enabled: false
          sku: Standard
```

## Modèles de connectivité

| Modèle | Description | Template |
|--------|-------------|----------|
| Hub & Spoke + Firewall | Hub VNet avec Azure Firewall | `hub-spoke-fw/` |
| Hub & Spoke + NVA | Hub VNet avec appliance réseau virtuelle | `hub-spoke-nva/` |
| Virtual WAN | Azure vWAN avec hub virtuel | `vwan/` |
| None | Pas de connectivité centralisée | — |

## Module AVM

- Template : `templates/platform/connectivity/<model>/`
- State key : `platform-connectivity.tfstate`

## Validation

`lzctl validate` vérifie :
- Pas de chevauchement CIDR entre hub et spokes
- Espaces d'adresses suffisamment grands
- Pas de conflit avec les landing zones existantes

## Brownfield

- `lzctl audit` détecte les VNets et peerings existants
- `lzctl import` génère les blocs d'import pour les ressources réseau
