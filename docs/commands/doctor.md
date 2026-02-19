# lzctl doctor

Vérifie les prérequis et la santé de l'environnement.

## Synopsis

```bash
lzctl doctor
```

## Description

Vérifie la présence et la version des outils nécessaires au fonctionnement de lzctl :

### Outils vérifiés

| Outil | Version min | Obligatoire |
|-------|-------------|-------------|
| Terraform | >= 1.5 | ✅ |
| Azure CLI | >= 2.50 | ✅ |
| Git | >= 2.30 | ✅ |
| GitHub CLI | any | ❌ optionnel |

### Vérifications Azure

| Check | Description |
|-------|-------------|
| Session Azure | `az account show` retourne une session valide |
| Accès management groups | Accès en lecture au root management group |
| Resource providers | `Microsoft.Management`, `Microsoft.Authorization`, `Microsoft.Network`, `Microsoft.ManagedIdentity` enregistrés |

### Vérification du state backend

| Check | Description |
|-------|-------------|
| Storage account accessible | Le storage account tagué `purpose=terraform-state` est accessible |

## Sortie

```
═══ lzctl doctor ═══

Tools:
  ✅  terraform   v1.9.0
  ✅  az          v2.65.0
  ✅  git         v2.45.0
  ⚠️  gh          not found (optional)

Auth:
  ✅  Azure session active (tenant: contoso.onmicrosoft.com)
  ✅  Management group access verified

Azure:
  ✅  Microsoft.Management registered
  ✅  Microsoft.Authorization registered
  ✅  Microsoft.Network registered
  ✅  Microsoft.ManagedIdentity registered

State Backend:
  ✅  Storage account accessible
```

## Exit codes

| Code | Signification |
|------|---------------|
| 0 | Tous les checks critiques passent |
| 1 | Un ou plusieurs checks critiques échouent |

## Voir aussi

- [init](init.md) — lancer doctor avant init
- [state health](../operations/state-management.md) — vérification détaillée du state backend
