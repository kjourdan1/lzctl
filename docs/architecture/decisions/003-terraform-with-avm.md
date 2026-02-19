# ADR-003: Terraform with Azure Verified Modules

- **Status:** Accepted
- **Date:** 2026-02-16

## Context

lzctl génère de l'Infrastructure as Code pour déployer des Azure Landing Zones. Il faut choisir l'outil IaC et les modules à utiliser.

## Decision

Utiliser **Terraform** avec des **Azure Verified Modules (AVM)** maintenus par Microsoft.

## Rationale

- **Terraform** — standard industriel pour l'IaC multi-cloud, large adoption, écosystème mature
- **AVM** — modules officiels Microsoft, testés, documentés, avec des versions semver
- **Pas de dépendance runtime** — le code Terraform généré fonctionne sans lzctl
- **Versions pinées** — reproductibilité des déploiements
- **State séparé par couche** — blast radius réduit, parallélisme possible

## Alternatives considérées

| Option | Rejetée car |
|--------|-------------|
| Bicep | Pas de state management natif, moins de modules réutilisables |
| Pulumi | Base d'utilisateurs plus petite, lock-in SDK |
| ARM Templates | Verbeux, pas de modules, pas de state |

## Consequences

- Chaque couche plateforme est un dossier Terraform indépendant
- Les modules AVM sont référencés par `source` + `version` dans les fichiers `.tf`
- `lzctl upgrade` vérifie les nouvelles versions AVM
- Le code généré est standard Terraform — maintenable sans lzctl
