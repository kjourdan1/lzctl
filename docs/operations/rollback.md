# Rollback

Procédures de rollback pour les couches plateforme.

## Principe

Le rollback s'effectue en **ordre inverse CAF** :
1. `connectivity` (en premier — dépendances en aval)
2. `governance`
3. `management`
4. `identity`
5. `management-groups` (en dernier — fundation)

Chaque couche a son propre state file, ce qui limite le blast radius.

## Rollback via lzctl

### Rollback complet

```bash
# Prévisualiser
lzctl rollback --dry-run

# Exécuter (avec confirmation)
lzctl rollback

# Sans confirmation (CI)
lzctl rollback --auto-approve
```

### Rollback d'une couche spécifique

```bash
lzctl rollback --layer connectivity
```

## Rollback via state snapshot

Si un apply a corrompu l'état, restaurer depuis un snapshot :

### 1. Lister les snapshots disponibles

```bash
lzctl state list
```

### 2. Identifier le snapshot à restaurer

```bash
# Via Azure CLI
az storage blob list \
  --account-name <storage-account> \
  --container-name tfstate \
  --include s \
  --query "[?name=='platform-connectivity.tfstate'].{name:name, snapshot:snapshot, lastModified:properties.lastModified}" \
  --output table
```

### 3. Restaurer le snapshot

```bash
az storage blob copy start \
  --account-name <storage-account> \
  --destination-container tfstate \
  --destination-blob platform-connectivity.tfstate \
  --source-uri "https://<storage-account>.blob.core.windows.net/tfstate/platform-connectivity.tfstate?snapshot=<snapshot-id>" \
  --auth-mode login
```

### 4. Vérifier

```bash
lzctl plan --layer connectivity
```

## Rollback d'urgence

En cas d'incident critique :

1. **Snapshot immédiat** : `lzctl state snapshot --all --tag "pre-emergency"`
2. **Identifier la couche** : `lzctl drift`
3. **Rollback ciblé** : `lzctl rollback --layer <couche> --auto-approve`
4. **Vérifier** : `lzctl plan` (doit montrer zéro changement)
5. **Post-mortem** : documenter l'incident et les actions correctives

## Prévention

- Toujours exécuter `lzctl plan` avant `lzctl apply`
- Utiliser les pipelines CI/CD avec review (PR) pour les changements
- Activer le blob versioning et soft delete sur le state backend
- Vérifier avec `lzctl state health`
