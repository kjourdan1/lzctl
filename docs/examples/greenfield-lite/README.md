# Greenfield Lite — Configuration minimale

Configuration légère pour sandbox, PoC ou environnements de développement.

## Caractéristiques

| Aspect | Choix |
|--------|-------|
| Profil MG | CAF Lite (hiérarchie simplifiée) |
| Connectivité | Aucune (pas de hub) |
| Identité | Service Principal + secret |
| CI/CD | Azure DevOps |
| Monitoring | Log Analytics 30j, Defender off |
| Landing Zones | 1 sandbox |
| Policies | 2 assignments minimales |

## Architecture

```
Tenant Root Group
├── mycompany (root)
│   ├── Platform
│   │   └── Management → Log Analytics (30j retention)
│   └── Landing Zones
│       └── Sandbox
│           └── poc-workload (10.100.0.0/24) — isolated
```

## Utilisation

```bash
# 1. Copier la config
cp docs/examples/greenfield-lite/lzctl.yaml ./lzctl.yaml

# 2. Personnaliser
#    - metadata.tenant → votre tenant
#    - stateBackend.* → votre storage account
#    - landingZones[0].subscription → votre subscription

# 3. Scaffolding
lzctl init --from lzctl.yaml

# 4. Déployer
lzctl plan --tenant mycompany
lzctl apply --tenant mycompany
```

## Pourquoi Lite ?

- **Plus rapide** — Moins de modules à déployer
- **Moins de prérequis** — Pas besoin de firewall, VPN, DNS
- **Coût réduit** — Pas de ressources de connectivité
- **Idéal pour** — PoC, formation, dev/test isolé

## Évoluer vers Standard

Pour migrer vers une configuration complète :

```bash
# 1. Modifier le profil MG
#    managementGroups.model: caf-lite → caf-standard

# 2. Ajouter la connectivité
#    connectivity.type: none → hub-spoke

# 3. Re-scaffolder
lzctl init --from lzctl.yaml

# 4. Planifier les changements
lzctl plan --tenant mycompany
```

## Différences avec Standard

| Fonctionnalité | Lite | Standard |
|---------------|------|----------|
| Hiérarchie MG | Simplifiée | Complète CAF |
| Hub Network | ❌ | ✅ Hub + Firewall |
| VPN / ExpressRoute | ❌ | ✅ |
| DNS Resolver | ❌ | ✅ |
| Defender for Cloud | ❌ | ✅ |
| Log Analytics retention | 30j | 365j |
| OIDC / WIF | ❌ | ✅ |
| PR obligatoire | ❌ | ✅ |
