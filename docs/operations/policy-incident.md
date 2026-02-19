# Policy Incident Response

Procédure de réponse quand une Azure Policy bloque un déploiement ou génère des alertes de non-conformité.

## Types d'incidents

| Type | Description | Urgence |
|------|-------------|---------|
| **Blocking** | Une policy `Deny` bloque un déploiement légitime | Haute |
| **Non-compliance** | Des ressources existantes ne respectent pas une policy | Moyenne |
| **False positive** | Une policy signale à tort une non-conformité | Basse |

## Procédure — Policy bloquante

### 1. Identifier la policy

```bash
# Voir l'état des policies
lzctl policy status

# Comparer local vs déployé
lzctl policy diff
```

### 2. Créer une exemption temporaire

```bash
# Scaffolder une exemption
lzctl policy create --type exemption --name "temp-deploy-fix"
```

L'exemption est créée dans `policies/exemptions/` avec une date d'expiration obligatoire.

### 3. Résoudre

- **Si la policy est correcte** : modifier le code Terraform pour être conforme
- **Si la policy est trop restrictive** : ajuster la définition de policy
- **Si c'est un faux positif** : créer un report upstream

### 4. Retirer l'exemption

Après résolution, supprimer l'exemption et redéployer :

```bash
lzctl policy deploy
lzctl policy verify
```

## Procédure — Non-conformité

### 1. Générer le rapport de conformité

```bash
lzctl policy verify
```

### 2. Créer des tâches de remédiation

```bash
lzctl policy remediate
```

### 3. Vérifier la résolution

```bash
lzctl audit
```

## Prévention

- Toujours tester les policies en mode audit d'abord : `lzctl policy test`
- Utiliser `lzctl policy verify` avant de passer en enforcement
- Documenter les exemptions avec une justification et une date d'expiration
