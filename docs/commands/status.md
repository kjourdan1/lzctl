# lzctl status

Affiche un aperçu de l'état du projet landing zone.

## Synopsis

```bash
lzctl status [flags]
```

## Description

Lit `lzctl.yaml` et affiche :
- Métadonnées du projet (nom, tenant, région)
- Couches plateforme activées et leur état
- Landing zones configurées
- Informations Git (branche, dernier commit)

## Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `--live` | `false` | Interroger Azure pour vérifier l'état réel |

## Sortie

```
Project: contoso-platform
Tenant:  00000000-0000-0000-0000-000000000000
Region:  westeurope

Platform Layers:
  LAYER                STATUS
  management-groups    ✅ active
  identity             ✅ active
  management           ✅ active
  governance           ✅ active
  connectivity         ✅ active

Landing Zones: 2
  NAME       ARCHETYPE   CONNECTED
  app-prod   corp        yes
  sandbox    sandbox     no

Git: main (abc1234) — 2026-02-19
```

## Exemples

```bash
# Status local
lzctl status

# Status live (requiert Azure)
lzctl status --live

# JSON
lzctl status --json
```

## Voir aussi

- [drift](drift.md) — détecter les changements
- [doctor](doctor.md) — vérifier l'environnement
