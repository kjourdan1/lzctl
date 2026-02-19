# ADR-002: YAML Manifest as Source of Truth

- **Status:** Accepted
- **Date:** 2026-02-16

## Context

lzctl a besoin d'un format de configuration déclaratif qui serve de source de vérité pour l'ensemble du projet landing zone.

## Decision

Utiliser un fichier unique `lzctl.yaml` avec un schéma versionné (`apiVersion: lzctl/v1`, `kind: LandingZone`) comme source de vérité.

## Rationale

- **Un seul fichier** — simplicité, versionnable dans Git, facile à reviewer en PR
- **apiVersion versionné** — permet des migrations de schéma futures
- **Validated by JSON Schema** — schéma embarqué dans le binaire pour validation offline
- **Déclaratif** — l'utilisateur décrit l'état désiré, lzctl orchestre
- **Standard YAML** — supporte les commentaires, lisible, large support d'outillage

## Schema

```yaml
apiVersion: lzctl/v1
kind: LandingZone

metadata:
  name: string
  tenant: uuid
  primaryRegion: string

spec:
  platform:
    managementGroups: ...
    connectivity: ...
    identity: ...
    management: ...
  governance: ...
  stateBackend: ...
  landingZones: []
  cicd: ...
```

## Consequences

- Toutes les commandes lisent `lzctl.yaml` comme entrée
- `lzctl validate` vérifie le manifeste contre le schéma JSON embarqué
- Les changements de schéma nécessitent un bump de `apiVersion`
