# Product Brief — lzctl

> Version: 1.0 | Date: 2026-02-16

## Résumé

**lzctl** est un outil CLI open-source (Go, binaire unique) qui bootstrap et maintient des Azure Landing Zones alignées avec le Cloud Adoption Framework (CAF).

Il génère des repositories Terraform production-ready utilisant des Azure Verified Modules (AVM), connectés à des pipelines CI/CD (GitHub Actions ou Azure DevOps), suivant un workflow GitOps où les PR déclenchent des plans et les merges déclenchent des déploiements.

## Problème

Les équipes plateforme Azure font face à :
- **Setup ad-hoc** — chaque tenant a une configuration Terraform unique, non-standardisée
- **Policies manuelles** — appliquées via le portail, sans versioning ni review
- **Pas de visibilité** — personne ne sait ce qui est réellement déployé vs en code
- **Pas de CI/CD plateforme** — les changements passent directement en production
- **Brownfield risqué** — les environnements existants sont trop risqués à terraformiser

## Solution

lzctl ajoute la **couche d'orchestration manquante** au-dessus du chemin recommandé par Microsoft :

1. **Scaffolding** — un wizard interactif génère un repository Terraform complet
2. **Validation** — schéma JSON, cross-validation CIDR/UUID, terraform validate
3. **Orchestration** — plan/apply multi-couche en ordre de dépendance CAF
4. **Day-2 Ops** — drift detection, module upgrade, policy lifecycle, state management
5. **Brownfield** — audit CAF, import progressif de ressources existantes

## Architecture

```
lzctl (binaire Go)
    ↓ génère
lzctl.yaml → Terraform (AVM) → Azure Landing Zone
    ↓ orchestre
CI/CD (GitHub Actions / Azure DevOps)
```

### Couches CAF (ordre de déploiement)

1. `management-groups` — Resource Organisation
2. `identity` — Identity & Access
3. `management` — Management & Monitoring
4. `governance` — Azure Policies
5. `connectivity` — Hub-Spoke ou vWAN

### Principes

| Principe | Description |
|----------|-------------|
| Stateless | Pas d'état local — tout dans lzctl.yaml + Git + Terraform state |
| Terraform natif | Code généré fonctionne sans lzctl |
| GitOps | PR = review + plan, merge = apply |
| State as First-Class | Versioning, soft delete, health checks |

## Audience cible

- Équipes plateforme Azure (Cloud Engineers, Platform Engineers)
- Consultants déployant des landing zones pour leurs clients
- Organisations adoptant le Cloud Adoption Framework

## Différenciation

| Feature | lzctl | ALZ Terraform Module | Manual Terraform |
|---------|-------|---------------------|------------------|
| Scaffolding interactif | ✅ | ❌ | ❌ |
| Validation cross-field | ✅ | ❌ | ❌ |
| Drift detection | ✅ | ❌ | Manuel |
| Module upgrade | ✅ | ❌ | Manuel |
| Policy-as-Code lifecycle | ✅ | ❌ | ❌ |
| State lifecycle management | ✅ | ❌ | ❌ |
| Brownfield import | ✅ | ❌ | Manuel |
| CI/CD generation | ✅ | ❌ | Manuel |

## Stack technique

| Composant | Technologie |
|-----------|-------------|
| Langage | Go 1.24 |
| CLI framework | Cobra + Viper |
| IaC | Terraform >= 1.5 |
| Modules | Azure Verified Modules (AVM) |
| Auth | Azure CLI + Workload Identity Federation |
| State | Azure Storage (versioning, soft delete, blob lease locking) |
| CI/CD | GitHub Actions ou Azure DevOps |
| Templates | Go text/template (embed.FS) |
| Validation | JSON Schema (embarqué) |

## Licence

Apache License 2.0
