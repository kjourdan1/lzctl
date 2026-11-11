# lzctl : cadrage produit

Statut : cadrage, à valider avant tout développement. Date : 2026-11-11.
Ce document fixe **à quoi sert `lzctl`, pour qui, et comment on l'utilise**. L'architecture
technique est dans [architecture.md](architecture.md).

## 1. Problème

Dans une organisation sur Azure, chaque nouveau projet a besoin d'un environnement prêt :
subscription ou resource groups, réseau raccordé, droits, policies, identités, budget,
pipeline. En pratique :

- la mise à disposition prend des semaines de tickets et de copier-coller entre dépôts
  Terraform ;
- chaque copie dérive des standards (versions de Terraform et de providers hétérogènes,
  modules en retard, nommage incohérent) ;
- les refus de policies se découvrent au moment de l'apply ;
- personne ne sait précisément ce qui est déployé au regard de ce qui est dans le code ;
- des services mutualisés existent (passerelle d'API, cluster, base partagée), mais chaque
  projet recrée les siens.

## 2. Promesse

> `lzctl` lit l'existant, décide avec des règles explicites, génère du Terraform standard et
> le livre par PR : des environnements conformes en heures plutôt qu'en semaines, sans
> remplacer les modules ni les pipelines de l'organisation.

## 3. Ce que `lzctl` n'est pas

- **Pas un service.** C'est une CLI sans état ; elle s'exécute sur un poste ou dans une
  pipeline existante. Rien de plus à héberger.
- **Pas un outil de déploiement direct.** Elle génère et orchestre ; seule une pipeline
  approuvée applique.
- **Pas un enfermement.** Le code produit est du Terraform standard, qui fonctionne sans
  `lzctl`.
- **Pas un remplaçant de l'existant.** Elle s'adapte aux modules, au nommage et aux templates
  de pipeline de l'organisation.
- **Pas un portail.** La console est une façade séparée, sans logique métier (§ 8).

## 4. Utilisateurs

| Persona | Besoin | Interface |
|---|---|---|
| **Ingénieur plateforme** | Poser et faire évoluer la fondation, livrer des environnements, garder le parc à jour | CLI, PR |
| **Pipeline** | Valider, planifier, appliquer, surveiller sans humain au clavier | CLI en mode non interactif, sorties JSON, codes de retour |
| **Équipe sécurité et conformité** | Savoir ce qui est conforme et ce qui dérive | Rapports `audit`, `drift`, contrôle préalable des policies |
| **Équipe projet ou métier** | Obtenir un environnement sans connaître Azure | Console (demande en langage naturel), jamais la CLI |
| **FinOps, direction** | Délai de mise à disposition, coûts, inventaire fiable | Console, rapports |

## 5. Périmètres : ce que l'on possède vraiment

Le périmètre se déclare dans `lzctl.yaml`. Il détermine ce que `lzctl` génère, et ce qu'il se
contente de lire.

| Périmètre | Situation typique | Ce que `lzctl` gère | Ce que `lzctl` lit seulement |
|---|---|---|---|
| `tenant` | Organisation qui possède son tenant (greenfield ou refonte) | Management groups, policies, identités de plateforme, management, connectivité (hub), landing zones (subscriptions) | — |
| `managementGroup` | Filiale à qui le groupe délègue une branche de la hiérarchie | Sous-hiérarchie déléguée, ses policies, ses subscriptions, éventuellement son hub | Hiérarchie et policies du groupe au-dessus |
| `subscriptions` | Entité « spoke » d'un grand groupe : quelques subscriptions, un VNet attribué ou non | **Zones applicatives** dans les subscriptions : resource groups, subnets (si le VNet est à elle), RBAC, identités, budget, diagnostics, Key Vault | Policies héritées, hub, DNS et services du groupe |

Une **zone applicative** est l'équivalent d'une landing zone au niveau d'une subscription :
l'unité livrée à un projet quand on ne crée pas de subscription.

## 6. Cycle d'utilisation

| Moment | Qui | Où | Commandes | Effet |
|---|---|---|---|---|
| Jour 0, une fois | Ingénieur plateforme | Poste ou Cloud Shell | `doctor`, `discover`, `init` | État de l'existant, fondation générée, state et identités fédérées amorcés |
| Préparer un changement | Ingénieur plateforme | Poste, puis PR | `workload add`, `add-blueprint`, `validate` | Code généré dans une branche ; rien n'est déployé |
| Valider une PR | Pipeline | GitHub Actions, Azure DevOps | `validate`, `preflight`, `plan` | Contrôles, refus de policies anticipés, plan publié dans la PR |
| Déployer | Pipeline | Environnement protégé | `apply` | Application après approbation humaine |
| Surveiller | Pipeline planifiée | GitHub Actions, Azure DevOps | `drift`, `audit`, `scan` | Rapports, alertes |
| Maintenir | Bot | PR | `upgrade` (et Renovate) | Montées de version proposées par PR |

**Règle : un humain génère et propose, seule une pipeline déploie.**

## 7. Briques fonctionnelles

| Brique | Commande | Rôle | État |
|---|---|---|---|
| **Découverte** | `discover` (nouveau ; regroupe `assess`, `status --live`) | Modèle de l'existant en lecture seule : hiérarchie, subscriptions, réseaux et plages IP, services mutualisés, policies héritées. Sert à toutes les autres briques | À construire |
| **Contrôle préalable des policies** | `preflight` (nouveau) | Confronte le code à générer aux policies effectives (héritées comprises) ; signale les refus avant l'apply | À construire |
| **Audit du parc** | `scan` (nouveau) | Parcourt des dépôts Terraform et pipelines existants : versions de Terraform et de providers, modules en retard, secrets en clair, branches par environnement, absence de lock file ; propose une configuration Renovate | À construire |
| Génération | `init`, `workload add`, `add-blueprint` | Code Terraform standard, AVM ou modules de l'organisation | Existe ; templates à réparer (§ 11) |
| Décision | Résolveur (via `workload add --from-request`) | Demande métier → offre, archétype, blueprint, tailles, rattachement à un service mutualisé ou création | À construire |
| IPAM | Intégré au résolveur | Plage libre attribuée, contrôlée contre l'existant découvert | À construire |
| Orchestration | `plan`, `apply`, `rollback` | Couches dans l'ordre de dépendance | Existe |
| Conformité | `audit`, `drift`, `policy *` | Score CAF, dérive, cycle de vie des policies | Existe |
| Maintenance | `upgrade` | Versions AVM | Existe |
| Reprise | `import`, `select` | Import de l'existant | Existe |

## 8. CLI et console

| | `lzctl` (CLI) | Console |
|---|---|---|
| Pour qui | Ingénieurs plateforme, pipelines | Équipes projet ; vue d'ensemble pour l'équipe plateforme |
| Rôle | Le moteur : découvrir, décider, générer, valider, orchestrer, auditer | La façade : recueillir une demande, suivre son statut, afficher coûts, dérive, conformité |
| Logique métier | Toute : schéma de demande, résolveur, IPAM, règles de réutilisation | Aucune ; elle dépose une demande en PR et lit les sorties JSON de `lzctl` |
| Exécution | Pipeline | N'exécute jamais `lzctl` ; la PR déclenche la pipeline |

Une seule source de vérité pour les règles : si la console en avait, elles divergeraient.
Une organisation qui a déjà un portail peut l'utiliser à la place de la console : le contrat
est la demande (schéma) et la PR, pas l'interface.

## 9. Cas d'usage et valeur

### A. Organisation qui possède son tenant (greenfield)
Fondation ALZ complète en quelques heures depuis un fichier déclaratif, landing zones par PR,
audit et dérive continus.

### B. Entité dans un grand groupe (périmètre `subscriptions`)
Situation : le groupe possède le tenant, la hiérarchie, les policies et le hub ; l'entité a
des subscriptions avec des resource groups, du Terraform, des pipelines et des validations.

| Apport | Valeur |
|---|---|
| `scan` du parc existant | Valeur dès le premier jour, sans rien déployer : rapport d'hygiène et plan de remise à niveau |
| `preflight` sur les policies du groupe | Plus de déploiement refusé découvert à l'apply |
| Zones applicatives par demande | Un projet obtient un environnement conforme en heures |
| Réutilisation de l'existant | Les services mutualisés découverts sont proposés avant toute création |
| Adaptation aux standards | Le code généré utilise les modules, le nommage et les templates de pipeline de l'entité ; son Terraform existant continue de tourner |

### C. Reprise d'un existant non géré (brownfield)
Découverte, évaluation contre CAF, import dans le code, puis gestion comme le reste.

## 10. Indicateurs de succès

| Indicateur | Cible indicative |
|---|---|
| Délai entre une demande et un environnement prêt (approbation comprise) | Moins d'une journée |
| Part des environnements conformes au score d'audit | 100 % à la création |
| Refus de policy découverts à l'apply | Zéro (anticipés par `preflight`) |
| Écart de version entre stacks (Terraform, providers) | Une version cible, écarts proposés en PR |
| Actions manuelles dans le portail Azure pour livrer un environnement | Zéro |

## 11. Priorités révisées

| Ordre | Chantier | Raison |
|---|---|---|
| 1 | **Réparer et tester la génération existante** : CI qui exécute `init` et `validate` sur la sortie de chaque profil et blueprint ; blueprints placés dans la landing zone ou la zone applicative, pas dans le hub ; contrat de sorties explicite entre couches | Une génération cassée ruine toute démonstration |
| 2 | **`discover`** (lecture seule) | Base de toutes les autres briques |
| 3 | **Périmètre `subscriptions` et zones applicatives** | Le cas le plus fréquent en entreprise |
| 4 | **`scan`** du parc Terraform et des pipelines | Valeur immédiate, sans déploiement |
| 5 | **`preflight`** sur les policies effectives | Fiabilise chaque livraison |
| 6 | **Catalogue et templates surchargeables** (modules, nommage, pipelines de l'organisation) | Condition d'adoption dans un existant |
| 7 | **Demande, résolveur, IPAM, réutilisation** (`--from-request`) | Contrat de la console |
| 8 | **Connectivité de niveau tenant** (hub, firewall, règles en données) | Utile au périmètre `tenant`, moins prioritaire |

Validation de bout en bout : un tenant vierge (périmètre `tenant`), une fois le point 1 vert ;
le périmètre `subscriptions` se valide en simulant une entité dans une sous-hiérarchie de ce
tenant, avec des policies posées au-dessus d'elle.

## 12. Questions ouvertes
- `discover` : quel format pour le modèle de l'existant (fichier versionné dans Git, ou
  produit à chaque exécution) ?
- `scan` : lire les dépôts en local, ou par les API GitHub et Azure DevOps ?
- `preflight` : évaluer les policies par simulation locale des règles, ou par l'API What-If
  et l'évaluation de conformité d'Azure ?
- Surcharge des templates : par dossier de templates de l'organisation, ou par un catalogue
  de modules déclaré dans `lzctl.yaml` ?
- Nom des commandes : `discover` remplace-t-il `assess` et `status --live`, ou les complète ?
