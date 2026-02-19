# Design Area : Workload Vending

Gestion des landing zones applicatives (subscriptions).

## Ce que lzctl déploie

Pour chaque landing zone :
- Dossier Terraform sous `landing-zones/<name>/`
- VNet spoke avec peering au hub (si `connected: true`)
- NSG par défaut
- Configuration dans `lzctl.yaml` (`spec.landingZones[]`)

## Configuration — lzctl.yaml

```yaml
spec:
  landingZones:
    - name: app-prod
      subscription: 00000000-0000-0000-0000-000000000000
      archetype: corp          # corp | online | sandbox
      addressSpace: 10.1.0.0/24
      connected: true          # Peering au hub
      tags:
        environment: production
        team: platform

    - name: sandbox-dev
      subscription: 11111111-1111-1111-1111-111111111111
      archetype: sandbox
      addressSpace: 10.2.0.0/24
      connected: false
```

## Archetypes

| Archetype | Description | Policies |
|-----------|-------------|----------|
| `corp` | Workload connecté au réseau corporate | Network + governance policies |
| `online` | Workload exposé sur internet | Security policies (WAF, DDoS) |
| `sandbox` | Environnement d'expérimentation | Policies minimales, pas de peering |

## Commandes

```bash
# Ajouter une landing zone
lzctl workload add --name app-prod --archetype corp --address-space 10.1.0.0/24

# Adopter une subscription existante
lzctl workload adopt --name legacy-app --subscription 00000000-... --archetype corp

# Lister les landing zones
lzctl workload list

# Supprimer une landing zone
lzctl workload remove app-prod
```

## Structure générée

```
landing-zones/
└── app-prod/
    ├── main.tf              # Module AVM avec spoke VNet + peering
    ├── variables.tf
    └── app-prod.auto.tfvars
```

## Validation

`lzctl validate` vérifie :
- Pas de chevauchement d'adresses avec le hub ou d'autres spokes
- Format UUID valide pour les subscription IDs
- Le hub de connectivité existe quand `connected: true`
