# lzctl en bref

`lzctl` (« landing zone control ») est une CLI qui **garde le contrôle des landing zones
Azure dans la durée**. Cadrage détaillé : [product.md](product.md).

> Les briques marquées *(à construire)* sont décidées dans le cadrage mais pas encore
> développées.

## Objectif

Une landing zone se crée en un jour et vit des années. C'est pendant ces années que les
écarts s'accumulent : versions qui divergent, ressources créées à la main, policies qui
refusent au dernier moment, services mutualisés recréés plusieurs fois, et plus personne ne
sait ce qui est déployé au regard de ce qui est dans le code.

`lzctl` sert à **savoir où en est le parc, le remettre en ordre, et livrer de nouveaux
environnements sans recréer le désordre**, au-dessus de ce que Microsoft fournit déjà :
l'accélérateur IaC ALZ pour l'amorçage, Azure Verified Modules pour les ressources, et les
analyseurs (checkov, tflint, PSRule for Azure) pour l'analyse de code.

## Règles de fonctionnement

- Un humain génère et propose ; **seule une pipeline approuvée déploie**.
- Le code produit est du **Terraform standard**, qui fonctionne sans `lzctl`.
- **Lecture seule par défaut** ; l'écriture passe toujours par une PR.
- Les décisions (archétype, blueprint, rattachement) suivent la **table de décision de
  l'organisation**, et chaque résultat cite la règle appliquée.

## Ce que contient l'outil

| Famille | Commandes | Rôle |
|---|---|---|
| **Voir** | `discover` *(à construire)*, `status`, `audit`, `drift` | Inventaire en lecture seule, écart aux règles de l'organisation, dérive entre code et réalité |
| **Reprendre** | `init --adopt` *(à construire)* | Reprendre une fondation existante (accélérateur ALZ ou faite à la main) sans rien regénérer |
| **Anticiper** | `preflight` *(à construire)*, `validate`, `plan` | Refus de policies prévus avant l'apply, contrôles, plan publié dans la PR |
| **Remettre en ordre** | `scan`, `remediate` *(à construire)*, `import` | Hygiène du parc, plan de correction priorisé, reprise de l'existant dans le code |
| **Livrer** | `workload add`, `add-blueprint`, `--from-request` *(à construire)* | Nouvel environnement généré et proposé par PR, avec réutilisation des services mutualisés |
| **Maintenir** | `upgrade` (et Renovate) | Montées de version proposées par PR |
| **Ouvrir aux agents** | `lzctl mcp` *(à construire)* | Les mêmes briques, utilisables par un agent, avec les mêmes garde-fous |

## Jour 1 sur un tenant existant

Cas d'un parc déjà en place, ou d'une mission de remise en ordre. **Rien n'est déployé avant
l'étape 5.**

| Étape | Action | Droits nécessaires | Résultat |
|---|---|---|---|
| 1. Déclarer le périmètre | `lzctl.yaml` : `tenant`, `managementGroup` ou `subscriptions`, selon ce que l'on possède réellement | Aucun | Ce que `lzctl` gère, et ce qu'il se contente de lire |
| 2. Découvrir | `lzctl discover` *(à construire)* avec une identité en lecture seule | Reader et Cost Management Reader sur le périmètre | Inventaire (hiérarchie, subscriptions, réseaux et plages IP, services mutualisés, policies héritées) avec ce qui n'a pas pu être lu |
| 3. Reprendre et évaluer | `lzctl init --adopt` *(à construire)*, `lzctl audit`, `lzctl scan` *(à construire)* sur les dépôts existants, en reprenant les résultats de checkov ou PSRule | Lecture des dépôts | `lzctl.yaml` qui décrit l'existant ; écarts à CAF et aux règles de l'organisation ; hygiène du parc (versions, lock files, branches par environnement, secrets) |
| 4. Planifier | `lzctl remediate` *(à construire)* | Aucun | Écarts classés par risque et par effort, chacun avec sa preuve et la PR qui le corrigera |
| 5. Reprendre dans le code | `lzctl import`, puis une PR par lot | Pipeline existante | L'existant entre dans le code sans être recréé |
| 6. Contrôler en continu | `drift`, `audit` et `scan` planifiés dans la pipeline | Identité en lecture seule | Rapport régulier ; chaque nouvel écart se voit |
| 7. Livrer | Nouvelles landing zones ou zones applicatives par PR, puis par demande depuis une console | Pipeline approuvée | Le parc grandit sans recréer le désordre |

## Sur un tenant neuf

| Étape | Outil | Résultat |
|---|---|---|
| 1. Amorcer la fondation | **Accélérateur IaC ALZ de Microsoft** (module PowerShell ALZ) : dépôts, pipelines GitHub ou Azure DevOps, identités fédérées, state, hiérarchie et policies ALZ | Fondation conforme à la référence Microsoft, maintenue par Microsoft |
| 2. Prendre le relais | `lzctl init --adopt` *(à construire)* : lit ce que l'accélérateur a créé et écrit `lzctl.yaml`, sans regénérer ce qui existe | `lzctl` devient le contrôleur de la fondation |
| 3. Poser les règles de l'organisation | Table de décision, catalogue d'offres, registre des services mutualisés, conventions de nommage | L'offre interne est décrite dans le code |
| 4. Livrer les premières landing zones | `lzctl workload add`, puis PR, approbation et apply | Chaque environnement naît conforme |
| 5. Contrôler en continu | Comme pour un tenant existant | Le tenant ne dérive pas |

À partir de l'étape 2, un tenant neuf et un tenant existant se gèrent de la même façon ; le
neuf démarre simplement sans écarts.

## Deux missions types

1. **Remettre de l'ordre** dans un parc existant : le parcours « jour 1 sur un tenant
   existant » ci-dessus, de la découverte au contrôle continu.
2. **Construire une offre interne de platform as a service** : catalogue d'offres, table de
   décision, demandes livrées par PR, réutilisation des services mutualisés, suivi par
   landing zone. Une console ou un portail existant dépose les demandes ; `lzctl` décide,
   génère et contrôle.

## État actuel

| Existe et fonctionne | Existe, à réparer | À construire |
|---|---|---|
| CLI, schéma `lzctl.yaml`, `validate`, `audit`, `drift`, `upgrade`, `import`, orchestration `plan` et `apply`, couche management groups | Couche connectivité ; blueprints, placés dans le hub au lieu de la landing zone | `discover`, `init --adopt`, périmètre `subscriptions`, `scan`, `remediate`, `preflight`, demande par PR, serveur MCP |

## Ordre de travail

1. Réparer la génération et la couvrir en CI.
2. `discover`.
3. Mode adoption (`init --adopt`).
4. Périmètre `subscriptions` et zones applicatives.
5. `scan` et `remediate`.
6. `preflight`.
7. Catalogue et templates surchargeables.
8. Demande, résolveur, IPAM, réutilisation.
9. Interface pour agents (MCP).
10. Connectivité de niveau tenant, seulement si l'accélérateur ALZ ne la couvre pas.

Le détail et les raisons de chaque priorité sont dans [product.md](product.md), § 11.
