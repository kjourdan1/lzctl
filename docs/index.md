# lzctl Documentation

> Landing Zone Factory CLI — orchestrateur stateless pour Azure Landing Zones, aligné Cloud Adoption Framework.

## Pour commencer

| Étape | Commande | Description |
|-------|----------|-------------|
| 1 | `lzctl doctor` | Vérifier les prérequis |
| 2 | `lzctl init` | Initialiser le projet (wizard interactif) |
| 3 | `lzctl validate` | Valider le manifeste et la configuration |
| 4 | `lzctl plan` | Prévisualiser les changements |
| 5 | `lzctl apply` | Déployer les couches en ordre CAF |
| 6 | `lzctl status` | Vérifier l'état du déploiement |

Voir le [README](../README.md) pour le quick start complet.

## Référence

- [CLI Reference](cli-reference.md) — toutes les commandes, flags et variables d'environnement
- [Per-Command Docs](commands/) — documentation détaillée par commande

## Architecture

- [Decisions architecturales (ADRs)](architecture/decisions/)
  - [ADR-001: Go as CLI Language](architecture/decisions/001-go-as-cli-language.md)
  - [ADR-002: YAML Manifests as Source of Truth](architecture/decisions/002-yaml-manifests-as-source-of-truth.md)
  - [ADR-003: Terraform with AVM](architecture/decisions/003-terraform-with-avm.md)
  - [ADR-004: Bootstrap via az CLI](architecture/decisions/004-bootstrap-az-cli.md)
  - [ADR-005: Stateless CLI with GitOps](architecture/decisions/005-stateless-cli-gitops.md)

### Design Areas CAF

| Design Area | Couche | Page |
|-------------|--------|------|
| Resource Organisation | `management-groups` | [resource-org.md](architecture/design-areas/resource-org.md) |
| Identity & Access | `identity` | [identity.md](architecture/design-areas/identity.md) |
| Management & Monitoring | `management` | [management.md](architecture/design-areas/management.md) |
| Governance | `governance` | [governance.md](architecture/design-areas/governance.md) |
| Network Topology | `connectivity` | [network.md](architecture/design-areas/network.md) |
| Security | `security` | [security.md](architecture/design-areas/security.md) |
| Workload Vending | `landing-zones/` | [workload-vending.md](architecture/design-areas/workload-vending.md) |

## Opérations

- [State Management](operations/state-management.md) — gestion du lifecycle des fichiers d'état Terraform
- [Rollback](operations/rollback.md) — procédures de rollback standard et d'urgence
- [Drift Response](operations/drift-response.md) — traitement du drift d'infrastructure
- [Policy Incident](operations/policy-incident.md) — gestion des incidents de policy

## Développement

- [Contributing](contributing.md) — setup, conventions et workflow de développement
- [Testing](../TESTING.md) — guide des tests unitaires et d'intégration

## Upgrade

- [Upgrade Guide](upgrade/upgrade-guide.md) — compatibilité des versions, migrations
