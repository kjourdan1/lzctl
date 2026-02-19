# Drift Response

Procédure de réponse quand du drift d'infrastructure est détecté.

## Détection

Le drift est détecté via :
- `lzctl drift` — scan local à la demande
- Pipeline CI/CD planifié (hebdomadaire) — crée automatiquement une issue/work item

## Classification

| Type | Description | Action |
|------|-------------|--------|
| **Ajout** | Ressource créée hors Terraform | Importer ou supprimer |
| **Modification** | Attribut modifié manuellement | Corriger en code ou reverter |
| **Suppression** | Ressource supprimée manuellement | Re-déployer ou mettre à jour le code |

## Procédure

### 1. Identifier le drift

```bash
# Scan complet
lzctl drift

# Scan d'une couche spécifique
lzctl drift --layer connectivity

# Sortie JSON pour analyse
lzctl drift --json
```

### 2. Analyser

- Vérifier si le changement est intentionnel (maintenance, incident)
- Identifier la couche et les ressources impactées
- Évaluer l'impact (blast radius)

### 3. Résoudre

**Option A — Aligner le code sur le réel :**
```bash
# Mettre à jour la configuration Terraform
# Puis valider
lzctl validate
lzctl plan --layer <couche>
```

**Option B — Revenir à l'état déclaré :**
```bash
# Re-appliquer la configuration Terraform
lzctl apply --layer <couche>
```

**Option C — Importer la ressource :**
```bash
# Si une ressource a été ajoutée manuellement
lzctl import --resource-group <rg> --layer <couche>
```

### 4. Prévenir

- Activer les Azure Policy `Deny` pour empêcher les modifications manuelles
- Restreindre les droits d'écriture directe via RBAC
- Documenter la résolution dans l'issue/work item

## Escalation

| Severity | Critère | SLA |
|----------|---------|-----|
| Critical | Drift sur management-groups ou identity | 4h |
| High | Drift sur connectivity ou governance | 24h |
| Medium | Drift sur management | 72h |
| Low | Drift sur landing-zones | Sprint suivant |
