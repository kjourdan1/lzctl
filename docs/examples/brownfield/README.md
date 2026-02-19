# Brownfield — Import d'infrastructure existante

Walkthrough pour importer une infrastructure Azure existante dans lzctl.

## Scénario

Vous avez déjà :
- Des management groups créés manuellement
- Un hub VNet avec firewall
- Quelques subscriptions avec des VNets peerés
- Des policies assignées au niveau tenant

Vous souhaitez les gérer avec lzctl sans tout recréer.

## Workflow

```
┌─────────┐    ┌─────────┐    ┌──────────┐    ┌──────────┐    ┌─────────┐
│  doctor  │───▶│  init   │───▶│  audit   │───▶│  import  │───▶│  plan   │
└─────────┘    └─────────┘    └──────────┘    └──────────┘    └─────────┘
```

## Étape 1 : Prérequis

```bash
# Vérifier les outils nécessaires
lzctl doctor

# Résultat attendu :
# ✅ terraform >= 1.5.0
# ✅ az cli >= 2.50.0
# ✅ az logged in
# ✅ git initialized
```

## Étape 2 : Initialiser le projet

```bash
# Créer un lzctl.yaml correspondant à l'existant
lzctl init

# Le wizard pose les questions :
# ? Project name: my-existing-alz
# ? Tenant: mycompany.onmicrosoft.com
# ? Primary region: westeurope
# ? MG model: caf-standard
# ? Connectivity: hub-spoke
# ? Hub address space: 10.0.0.0/16  (votre hub existant)
# ? Firewall: yes, Premium
# ...
```

## Étape 3 : Auditer l'environnement

```bash
# Lancer un audit complet
lzctl audit --tenant mycompany --format json -o audit-report.json

# Lire le résumé
lzctl audit --tenant mycompany --format markdown

# Résultat type :
# CAF Compliance Audit Report
# ═══════════════════════════
# Overall Score: 62/100
# 
# ❌ Resource Organization:
#   - Management group hierarchy incomplete (missing Decommissioned)
#   - 2 subscriptions not placed in correct MG
#
# ⚠️  Connectivity:
#   - Hub VNet found but no DNS forwarder configured
#   - 1 VNet not peered to hub
#
# ✅ Governance:
#   - Tagging policy active
#   - Allowed locations enforced
```

## Étape 4 : Importer les ressources

```bash
# Dry-run d'abord — voir ce qui serait importé
lzctl import --from audit-report.json --dry-run

# Résultat :
# Would generate imports for 15 resources:
#   connectivity/import.tf — 6 resources (VNet, Subnets, Firewall, ...)
#   governance/import.tf — 4 resources (Policy Assignments)
#   general/import.tf — 5 resources (Resource Groups)

# Lancer l'import
lzctl import --from audit-report.json

# Résultat :
# ✅ Generated imports/connectivity/import.tf (6 resources)
# ✅ Generated imports/governance/import.tf (4 resources)
# ✅ Generated imports/general/import.tf (5 resources)
#
# Next steps:
#   1. Review generated files in imports/
#   2. Run 'lzctl validate' to check
#   3. Run 'lzctl plan --tenant mycompany' to preview
```

## Étape 5 : Importer par subscription (alternatif)

```bash
# Scanner une subscription spécifique
lzctl import \
  --subscription 11111111-2222-3333-4444-555555555555 \
  --resource-group rg-networking \
  --include virtualNetwork,networkSecurityGroup

# Résultat :
# ✅ Generated imports/connectivity/import.tf (3 resources)
```

## Étape 6 : Valider et planifier

```bash
# Valider la configuration + imports
lzctl validate

# Planifier — Terraform détecte les imports
lzctl plan --tenant mycompany

# Plan output :
# Plan: 0 to add, 0 to change, 0 to destroy.
# 15 resources to import.
```

## Étape 7 : Appliquer les imports

```bash
# Appliquer en mode canary (import seulement)
lzctl apply --tenant mycompany --ring canary

# Vérifier l'état
lzctl status --tenant mycompany --live

# Une fois validé, passer en prod
lzctl apply --tenant mycompany --ring prod
```

## Étape 8 : Vérifier l'absence de drift

```bash
# Après import, vérifier que l'état est clean
lzctl drift --tenant mycompany

# Résultat attendu :
# 0/8 modules drifted | 0 resources affected
```

## Conseils

### Ressources non importables

Certaines ressources nécessitent une gestion manuelle :
- **Ressources dans d'autres tenants** — cross-tenant non supporté
- **Ressources sans resource ID** — ex: RBAC custom roles (utilisez `az role definition list`)
- **Ressources dépréciées** — ex: Classic resources

### Gestion des conflits

Si l'import détecte un conflit avec une ressource déjà dans le state :

```bash
# Lister les ressources dans l'état
terraform state list -state=modules/connectivity/terraform.tfstate

# Retirer une ressource du state si nécessaire
terraform state rm azurerm_virtual_network.old_vnet
```

### Ordre d'import recommandé

1. **Resource Groups** — base pour tout le reste
2. **Networking** — VNets, Subnets, NSGs, Route Tables
3. **Identity** — Managed Identities, RBAC
4. **Governance** — Policies, Initiatives
5. **Landing Zones** — Subscription-level resources

## Voir aussi

- [docs/commands/audit.md](../commands/audit.md) — détails de l'audit
- [docs/commands/import.md](../commands/import.md) — référence de la commande import
- [docs/operations/rollback.md](../operations/rollback.md) — rollback si nécessaire
