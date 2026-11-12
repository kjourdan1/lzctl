# lzctl : cadrage produit

Statut : cadrage, à valider avant tout développement. Date : 2026-11-11.
Ce document fixe **à quoi sert `lzctl`, pour qui, et comment on l'utilise**. Résumé et mise
en route : [overview.md](overview.md). Architecture technique : [architecture.md](architecture.md).

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

> `lzctl` (« landing zone control ») garde le contrôle des landing zones **après** leur
> création : il lit l'existant, le confronte aux règles de l'organisation, livre de nouveaux
> environnements par PR et remet de l'ordre là où le parc a dérivé, sans remplacer les
> modules, les pipelines ni les outils de Microsoft.

## 2 bis. Positionnement : au-dessus de Microsoft, pas à côté

| Besoin | Qui le fait | Rôle de `lzctl` |
|---|---|---|
| Amorcer une fondation ALZ (dépôts, pipelines, identités, state) | Accélérateur IaC ALZ de Microsoft (module PowerShell ALZ) | **Ne le refait pas.** S'en sert au jour 0, puis prend le relais |
| Ressources et patterns Azure | Azure Verified Modules | Les consomme, version figée |
| Libre-service d'environnements à partir de templates | Azure Deployment Environments (Dev Center), portails existants | Peut leur fournir des définitions, ou servir de moteur derrière un portail |
| Analyse de code IaC (sécurité, bonnes pratiques) | checkov, tflint, PSRule for Azure, Defender for Cloud | **Ne les concurrence pas** : lit leurs résultats et les rattache à chaque landing zone |
| Conformité des ressources déployées | Azure Policy, Defender for Cloud | Lit l'état de conformité ; anticipe les refus avant l'apply |
| **Contrôle continu du parc de landing zones** : inventaire, écart aux règles de l'organisation, dérive, cycle de vie, demandes, remise en ordre | **Personne de façon intégrée** | **Le cœur de `lzctl`** |

La différence se joue dans la continuité et l'exploitation : un parc de landing zones vit
des années après sa création, et c'est là que les écarts s'accumulent.

### Deux missions types
1. **Remettre de l'ordre** dans un parc existant (plusieurs subscriptions créées à des
   époques et par des équipes différentes) : inventaire, écarts, plan de remise en ordre
   priorisé, reprise dans le code, puis contrôle continu.
2. **Construire une offre interne « platform as a service »** : catalogue d'offres, table de
   décision, demandes livrées par PR, réutilisation des services mutualisés, suivi par
   landing zone.

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
| Jour 0, une fois | Ingénieur plateforme | Poste ou Cloud Shell | Tenant neuf : accélérateur IaC ALZ de Microsoft, puis `lzctl init --adopt`. Tenant existant : `doctor`, `discover`, `init --adopt` | Fondation amorcée par Microsoft ou existante, reprise par `lzctl` sans rien regénérer |
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
| **Hygiène du parc** | `scan` (nouveau) | Ce que les analyseurs de code ne font pas : **vue de parc** sur tous les dépôts (versions de Terraform, de providers et de modules, lock files, branches par environnement, landing zones sans code) ; **intègre** les résultats de checkov, tflint ou PSRule au lieu de les refaire ; propose une configuration Renovate | À construire |
| **Plan de remise en ordre** | `remediate` (nouveau) | À partir de `discover`, `audit` et `scan` : écarts classés par risque et par effort, chacun avec sa preuve et la PR qui le corrige (import, déplacement, mise à niveau) | À construire |
| **Adoption** | `init --adopt` (nouveau) | Lit une fondation existante (créée par l'accélérateur IaC ALZ ou à la main) et écrit `lzctl.yaml` qui la décrit, sans regénérer ce qui existe ; `lzctl` en devient le contrôleur | À construire |
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
| 3 | **Mode adoption** (`init --adopt`) : reprendre une fondation créée par l'accélérateur IaC ALZ ou existante, sans rien regénérer | Condition pour se placer au-dessus de Microsoft ; point d'entrée des tenants neufs comme existants |
| 4 | **Périmètre `subscriptions` et zones applicatives** | Le cas le plus fréquent en entreprise |
| 5 | **`scan`** (hygiène de parc, intégration des analyseurs existants) et **`remediate`** | Mission « remettre de l'ordre » ; valeur immédiate, sans déploiement |
| 6 | **`preflight`** sur les policies effectives | Fiabilise chaque livraison |
| 7 | **Catalogue et templates surchargeables** (modules, nommage, pipelines de l'organisation) | Condition d'adoption dans un existant |
| 8 | **Demande, résolveur, IPAM, réutilisation** (`--from-request`) | Contrat de la console |
| 9 | **Interface pour agents** (serveur MCP, § 13) | Rend chaque brique utilisable par un agent, avec les mêmes garde-fous |
| 10 | **Connectivité de niveau tenant** (hub, firewall, règles en données), seulement si l'accélérateur ALZ ne la couvre pas | Le jour 0 relève de Microsoft |

Validation de bout en bout : un tenant vierge (périmètre `tenant`), amorcé par l'accélérateur
IaC ALZ puis repris par `init --adopt`, une fois les points 1 à 3 verts ; le périmètre
`subscriptions` se valide en simulant une entité dans une sous-hiérarchie de ce tenant, avec
des policies posées au-dessus d'elle.

## 12. Modèle de confiance

Lecture seule garantit l'innocuité, pas l'exactitude. Chaque sortie de `discover`, `audit`,
`scan` et `preflight` :
- **affiche sa couverture** : scopes lus, scopes invisibles (droits insuffisants, plan de
  données, objets Entra), horodatage ; absence de preuve n'est pas preuve d'absence ;
- porte **une preuve par constat** (ID de ressource, `fichier:ligne`) et **la règle
  versionnée** qui l'a produit ;
- est **déterministe** : même entrée, même sortie ;
- accepte des **exceptions justifiées et versionnées** pour les faux positifs.

L'outil s'exécute dans la pipeline de l'organisation, avec son identité, sans appel sortant
hors API Azure et de la forge Git ; binaire signé, code ouvert. Les références restent la
conformité Azure Policy et Defender for Cloud.

**Décisions** (archétype, blueprint, rattachement) : `lzctl` exécute une **table de décision
de l'organisation** (une table par défaut est fournie comme proposition). Chaque résultat cite
la règle appliquée ; sans règle applicable, la demande va à un humain.

**Blueprints** : construits sur AVM ; `terraform test` ; checkov et PSRule for Azure (règles
WAF) ; déploiement de nuit en bac à sable **sous les policies réelles de l'organisation**
(zéro refus exigé) ; coût estimé ; versionnés et adoptés par les architectes de
l'organisation, qui en sont propriétaires.

## 13. Interface pour agents

`lzctl` est un bon outil pour un agent : commandes déterministes, sorties JSON, lecture par
défaut, écriture uniquement par PR. Un serveur MCP (`lzctl mcp`) expose les briques comme des
outils.

| Niveau | Outils | Effet |
|---|---|---|
| Lecture | `discover`, `audit`, `drift`, `scan`, `preflight`, `explain` (pourquoi cette décision, cette règle, cet écart) | Aucun |
| Proposition | `propose` (génère le code d'une demande ou d'une remise en ordre et ouvre une PR) | Une branche et une PR ; rien n'est appliqué |
| Application | Aucun | Réservée à la pipeline approuvée |

Garde-fous propres aux agents :
- **les données lues dans le tenant sont des données, pas des consignes** : noms, tags ou
  descriptions de ressources peuvent contenir du texte piégé ; les sorties d'outils le
  signalent comme contenu non fiable ;
- les faits viennent des outils, jamais du modèle ; le modèle explique, résume et enchaîne ;
- l'identité utilisée est celle de l'agent, en lecture seule, plus une application Git qui ne
  peut qu'ouvrir des PR ; chaque appel d'outil est journalisé ;
- sorties bornées et paginées, schémas stables et versionnés, codes d'erreur explicites.

Usages visés : un assistant de remise en ordre qui enchaîne découverte, audit et plan de
correction ; l'agent de cadrage de la console, qui appelle le résolveur ; l'agent de revue,
qui explique une PR à l'équipe plateforme. Complémentaire du serveur MCP Azure de Microsoft :
celui-ci donne accès aux ressources, `lzctl` apporte les règles et le contexte des landing
zones.

## 14. Questions ouvertes
- `discover` : quel format pour le modèle de l'existant (fichier versionné dans Git, ou
  produit à chaque exécution) ?
- `scan` : lire les dépôts en local, ou par les API GitHub et Azure DevOps ?
- `preflight` : évaluer les policies par simulation locale des règles, ou par l'API What-If
  et l'évaluation de conformité d'Azure ?
- Surcharge des templates : par dossier de templates de l'organisation, ou par un catalogue
  de modules déclaré dans `lzctl.yaml` ?
- Nom des commandes : `discover` remplace-t-il `assess` et `status --live`, ou les complète ?
