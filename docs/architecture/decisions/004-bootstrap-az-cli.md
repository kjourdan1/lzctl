# ADR-004: Bootstrap via az CLI

- **Status:** Accepted
- **Date:** 2026-02-16

## Context

Avant de pouvoir utiliser Terraform, il faut un state backend (Azure Storage Account). C'est un problème chicken-and-egg : on ne peut pas utiliser Terraform pour créer le backend qui stocke l'état de Terraform.

## Decision

Le bootstrap du state backend utilise des commandes `az` CLI directement, pas Terraform.

## Rationale

- **Évite le chicken-and-egg** — pas besoin de state pour créer le state backend
- **az CLI toujours disponible** — prérequis déjà vérifié par `lzctl doctor`
- **Idempotent** — les commandes `az` sont idempotentes par défaut
- **Transparent** — l'utilisateur peut voir et comprendre les commandes exécutées
- **Pas de state orphelin** — pas de state file pour le bootstrap lui-même

## Implementation

Le bootstrap crée :
1. Resource group (`rg-<project>-tfstate-<region>`)
2. Storage account (versioning, soft delete, TLS 1.2, encryption)
3. Blob container (`tfstate`)
4. Managed identity + RBAC
5. Federated credentials OIDC (pour CI/CD)

## Consequences

- Le package `internal/bootstrap/` utilise `exec.Command("az", ...)` 
- Le bootstrap est optionnel (`--no-bootstrap`)
- Le storage account est tagué `purpose=terraform-state` pour la découverte
