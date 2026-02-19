# ADR-005: Stateless CLI with GitOps

- **Status:** Accepted
- **Date:** 2026-02-16

## Context

lzctl doit s'intégrer dans un workflow d'équipe. Il faut décider si le CLI maintient un état local ou non.

## Decision

lzctl est un CLI **stateless** qui suit un workflow **GitOps**.

## Rationale

- **Pas d'état local** — tout vit dans `lzctl.yaml` + Git + Terraform state
- **Reproductible** — n'importe quel membre de l'équipe obtient le même résultat
- **GitOps natif** — PR = review + plan, merge = apply
- **CI/CD indépendant** — les pipelines générés n'ont pas de dépendance runtime sur lzctl
- **Pas de serveur** — pas de daemon, pas de base de données locale

## Conséquences

- lzctl ne fait jamais de `git push`
- L'état de déploiement est dans Terraform state (Azure Storage)
- Les pipelines CI/CD appellent Terraform directement
- `lzctl status` lit `lzctl.yaml` + Git log (pas d'état persistant)
- Chaque commande est idempotente et peut être relancée sans effet de bord
