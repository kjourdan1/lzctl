# Example Configurations

Ce répertoire contient des exemples de configurations `lzctl` prêtes à l'emploi.

| Scénario | Profil | Connectivité | CI/CD | Description |
|----------|--------|-------------|-------|-------------|
| [greenfield-standard](greenfield-standard/) | CAF Standard | Hub-Spoke + Firewall | GitHub Actions | Landing zone complète entreprise |
| [greenfield-lite](greenfield-lite/) | CAF Lite | Aucune | Azure DevOps | Configuration minimale pour sandbox/PoC |
| [brownfield](brownfield/) | — | — | — | Walkthrough import d'infrastructure existante |
| [pipeline-init](pipeline-init/) | Input one-shot | Configurable | GitHub Actions / Azure DevOps | Onboarding headless (`init --from-file --ci`) |

## Utilisation

```bash
# Copier un exemple comme base de travail
cp -r docs/examples/greenfield-standard/lzctl.yaml ./lzctl.yaml

# Ou utiliser le wizard interactif
lzctl init

# Ou mode pipeline non-interactif
lzctl init --ci --from-file docs/examples/pipeline-init/lzctl-init-input.yaml
```

## Validation

Chaque configuration peut être validée avec :

```bash
lzctl validate
```
