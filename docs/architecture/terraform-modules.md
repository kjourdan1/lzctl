# Modules, providers et versions Terraform en entreprise

Recommandations pour les dépôts que `lzctl` génère et pour les modules qu'une organisation
écrit autour d'eux. Complète l'[ADR-003](decisions/003-terraform-with-avm.md) (Terraform avec
AVM).

## Principes

1. **AVM pour les ressources, modules maison pour les standards.** On n'écrit pas un module
   par type de ressource : AVM le fait déjà, testé et maintenu par Microsoft. Les modules
   maison sont des **patterns** qui composent AVM et y appliquent les règles de
   l'organisation (nommage, tags, diagnostics, private endpoints, verrous).
2. **Une seule implémentation du nommage**, consommée partout. Un nommage recopié dans chaque
   module finit avec des séparateurs et une casse différents d'une ressource à l'autre.
3. **Tout est figé, rien n'est monté à la main.** Versions de Terraform, de providers et de
   modules sont fixées dans le code ; un outil (Renovate) propose les montées de version par PR.
4. **Une seule branche, des environnements en données.** Pas de branche par environnement.

## Modules

### Composition

```
Stack racine (une par landing zone ou couche plateforme, un state chacune)
  └─ module pattern de l'organisation      ex. « key-vault-privé »
       ├─ module de nommage                 nom calculé, jamais saisi
       ├─ AVM res-keyvault-vault            ressource
       └─ AVM res-network-privateendpoint   ressource
```

### Contrat d'un module
- Variables typées : `object({...})` plutôt que `map(any)`, avec `validation` sur les champs
  obligatoires (tags exigés, formats, plages).
- Sorties stables : une sortie renommée casse tous les consommateurs ; on ajoute la nouvelle,
  on déprécie l'ancienne, on la retire au majeur suivant.
- Blocs `moved` pour toute réorganisation interne, afin qu'aucun consommateur ne voie de
  destruction.
- Aucune configuration de provider dans un module : `provider` est déclaré par la stack racine.

### Versionnage
- **SemVer strict** : correctif = patch, ajout compatible = mineur, rupture = majeur.
- `CHANGELOG.md` par module ; le tag est posé par la pipeline au merge, pas à la main.
- Les consommateurs référencent une version exacte (`?ref=v1.4.2` ou `version = "1.4.2"`) et
  la font monter par PR.

### Distribution

| Modèle | Adapté à | Avantages | Inconvénients |
|---|---|---|---|
| Un dépôt par module, tag `vX.Y.Z` | Plusieurs équipes, propriétaires différents | Droits et cycles de vie séparés | Beaucoup de dépôts ; montées de version dispersées |
| Un dépôt de modules, tag par module (`naming/v1.2.0`) | Une équipe plateforme | Une seule CI, refontes transverses simples | Chaque consommateur clone tout le dépôt |
| Registre privé (Terraform Cloud, auto-hébergé) | Grandes organisations | Découverte, contraintes `~>` | Un service de plus à exploiter |

Source Git avec Azure DevOps :

```hcl
module "key_vault" {
  source = "git::https://dev.azure.com/<org>/<projet>/_git/terraform-modules//key-vault-private?ref=key-vault-private/v1.4.2"
}
```

Dans la pipeline, l'agent s'authentifie avec le jeton du job, sans PAT stocké :

```bash
git config --global http.https://dev.azure.com/.extraheader "AUTHORIZATION: bearer $SYSTEM_ACCESSTOKEN"
```

Le projet doit autoriser l'accès des pipelines aux dépôts de modules (paramètre « Protect
access to repositories in YAML pipelines » et autorisations par dépôt).

## Providers

| Où | Règle | Pourquoi |
|---|---|---|
| Module | Version minimale seulement : `version = ">= 4.0"` | La stack racine arbitre ; un plafond serré dans un module bloque les montées de version de tous ses consommateurs |
| Stack racine | Contrainte pessimiste : `version = "~> 4.37"` | Correctifs acceptés, ruptures refusées |
| Stack racine | `.terraform.lock.hcl` commité, avec les empreintes de toutes les plateformes | Même binaire partout ; sinon le fichier généré sur un poste échoue sur l'agent |

```bash
terraform providers lock -platform=linux_amd64 -platform=darwin_arm64 -platform=windows_amd64
```

Sur un agent auto-hébergé, un cache évite de retélécharger les providers à chaque job :
`TF_PLUGIN_CACHE_DIR=/var/cache/terraform-plugins`.

## Version de Terraform

Une version cible commune à toutes les stacks, déclarée deux fois :
- `required_version = "~> 1.13.0"` dans le code, qui refuse explicitement une autre version ;
- `.terraform-version` à la racine, lu par `tfenv`, les tâches de pipeline et Renovate.

Des stacks sur des versions différentes finissent par se comporter différemment et rendent
chaque migration coûteuse.

## Mises à jour : Renovate

Renovate (disponible pour GitHub et Azure DevOps) ouvre une PR par montée de version :

| Cible | Détection |
|---|---|
| Version de Terraform | `.terraform-version`, `required_version` |
| Providers | `required_providers` |
| Modules AVM | `source` + `version` (registre Terraform) |
| Modules maison | `source = "git::...?ref=..."` |
| Actions, tâches, images | Workflows et pipelines |

Chaque PR passe la même CI que le reste (voir ci-dessous) : une rupture d'interface d'un
module AVM se voit dans la PR de montée de version, pas des mois plus tard.

## Tests

| Quand | Quoi | Accès Azure |
|---|---|---|
| Chaque PR | `fmt`, `validate`, tflint (règles `azurerm`), checkov, doc à jour (terraform-docs) | Non |
| Chaque PR | `terraform test` avec `mock_provider` : variables, validations, sorties, nommage | Non |
| Chaque nuit | `examples/` de chaque module déployés puis détruits dans une subscription bac à sable | Oui, coût borné |
| Chaque PR de `lzctl` | `terraform init` et `validate` sur la sortie de chaque profil et blueprint | Non |

## Environnements et pipelines

- **Une branche `main`**, un fichier de variables par environnement
  (`env/dev.tfvars`, `env/prd.tfvars`) ou un dossier par environnement. Une branche par
  environnement fait diverger le code : on ne sait plus ce qui tourne en production.
- **Environnements Azure DevOps (ou GitHub) avec approbation** pour l'apply ; le plan est
  publié dans la PR.
- **Templates de pipeline versionnés** dans leur propre dépôt et référencés par tag
  (`ref: refs/tags/v2.1.0`), jamais par branche : un changement de template ne casse pas toutes
  les pipelines d'un coup.
- **Aucun secret dans les pipelines** : service connections par workload identity federation
  (OIDC), secrets applicatifs dans Key Vault (groupes de variables liés ou lecture à
  l'exécution). Un secret écrit en clair dans un fichier de pipeline est compromis.
- **Un state par stack et par environnement**, backend en authentification Entra, sans clé
  d'accès au compte de stockage.

## Ce que `lzctl` applique déjà

| Recommandation | Dans `lzctl` |
|---|---|
| AVM pour les ressources | Templates de couches et blueprints |
| Un state par couche | `backend.hcl` partagé, une clé par couche |
| Versions AVM figées et vérifiées | `version` explicite, `lzctl upgrade` |
| Validation | `lzctl validate` (schéma, CIDR, `terraform validate`) |

À compléter : CI qui valide la sortie de chaque profil, Renovate sur les templates, module
de nommage unique, templates de pipeline versionnés par tag.
