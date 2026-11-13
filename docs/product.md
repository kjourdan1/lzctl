# lzctl : cadrage produit

Statut : cadrage (deuxième version), **à réaligner** sur le brief
[`bmad-output/product-brief-lzctl-2026-11-13.md`](../bmad-output/product-brief-lzctl-2026-11-13.md),
qui fait foi pour la direction. Date : 2026-11-12.
Ce document fixe **à quoi sert `lzctl`, pour qui, et comment on l'utilise**. Résumé et mise
en route : [overview.md](overview.md). Architecture technique : [architecture.md](architecture.md).

## 1. Problème

Trois situations vécues dans des dépôts IaC Azure, avec une racine commune : **on ne voit pas
ce qu'un changement va provoquer au-delà du diff, et aucune stack n'est construite comme la
voisine**.

1. **Le déploiement qui écrase un changement fait hors code.** Une spécification OpenAPI est
   modifiée au fil de l'eau dans le portail, sans que personne d'autre le sache. Le
   déploiement suivant réapplique le `api.yml` du dépôt, en retard, et l'API de production
   devient inaccessible. Le plan Terraform n'avait rien signalé.
2. **Les modules maison qu'on n'ose plus faire évoluer.** Les modules sont versionnés par tags
   Git dans un dépôt à part. Leur auteur ne voit ni qui les consomme, ni ce qu'une nouvelle
   version casserait chez chacun. Résultat : les modules ne bougent plus.
3. **Chaque stack est écrite à la main.** Une ressource et ses à-côtés (un compte de stockage,
   ses conteneurs, ses private endpoints) sont recopiés et adaptés d'un projet à l'autre ; les
   ouvertures de flux réseau se demandent à part, par ticket.

## 2. Promesse

> `lzctl` standardise les stacks d'un dépôt IaC Azure et rend chaque déploiement prévisible :
> il génère une stack à partir d'un contrat, contrôle chaque PR contre ce contrat et contre
> l'existant, et signale ce qui a changé hors pipeline.

Le but est un outil **utilisable en production** (critères au § 10), qui tourne à chaque PR
dans la pipeline, pas seulement le jour où l'on crée quelque chose.

## 3. Vocabulaire

| Terme | Sens |
|---|---|
| **Stack** | Une ressource principale et ses à-côtés, déployés ensemble avec leur propre state. Exemple : un compte de stockage, ses conteneurs, ses private endpoints. C'est ce que la console appellera une « platform » |
| **Contrat de stack** | Le fichier déclaratif qui décrit une stack : type, propriétaire, paramètres, flux réseau, rattachements, propriété du contenu (§ 7) |
| **Zone applicative** | Le périmètre qui reçoit les stacks d'un projet : subscription ou resource groups, subnets, droits. Premier périmètre visé |
| **Catalogue** | Les types de stack disponibles, construits sur les modules maison de l'organisation (ou sur AVM) |
| **Demande** | L'expression d'un besoin, dans un schéma fixe, qui produit un contrat de stack |

## 4. Ce que `lzctl` n'est pas

- **Pas un service.** Une CLI sans état, exécutée sur un poste ou dans une pipeline existante.
- **Pas un outil de déploiement direct.** Elle génère et contrôle ; seule une pipeline
  approuvée applique.
- **Pas un enfermement.** Le code produit est du Terraform standard, qui fonctionne sans
  `lzctl`.
- **Pas un LLM.** Tout ce que fait `lzctl` est déterministe. Un LLM peut l'appeler (§ 13),
  jamais l'inverse, et il n'écrit pas le Terraform.
- **Pas un outil de fondation.** Hiérarchie, policies et hub relèvent de l'accélérateur IaC
  ALZ de Microsoft ou de l'existant ; `lzctl` les lit.
- **Pas un portail.** La console est une façade séparée, sans logique métier (§ 13).

## 5. Positionnement

`lzctl` ne remplace aucun des outils ci-dessous ; il apporte ce qui leur manque : **le
contexte de la plateforme**.

| Besoin | Qui le fait déjà | Rôle de `lzctl` |
|---|---|---|
| Fondation ALZ | Accélérateur IaC ALZ de Microsoft | Hors périmètre ; lue, pas générée |
| Ressources et patterns | Modules maison, Azure Verified Modules | Les consomme comme catalogue |
| Analyse de code IaC | checkov, tflint, PSRule for Azure | Ne les refait pas |
| Conformité des ressources | Azure Policy, Defender for Cloud | Ne les refait pas |
| PR de montée de version | Renovate | S'appuie dessus ; ajoute l'analyse d'impact |
| Dérive visible par Terraform | `terraform plan` planifié | Existe déjà dans la pipeline générée |
| **Écritures hors pipeline, y compris sur ce que Terraform ne relit pas** | À vérifier (§ 15) | **Cœur de `lzctl`** |
| **Impact d'une version de module sur tous ses consommateurs** | À vérifier (§ 15) | **Cœur de `lzctl`** |
| **Stack standardisée par contrat, flux réseau compris** | Personne de façon intégrée | **Cœur de `lzctl`** |

## 6. Utilisateurs

| Persona | Besoin | Interface |
|---|---|---|
| **Ingénieur plateforme** (principal) | Livrer des stacks identiques, déployer sans surprise | CLI, PR |
| **Pipeline** | Contrôler chaque PR, surveiller chaque nuit | CLI non interactive, sorties JSON, codes de retour |
| **Auteur de module** | Publier une version en connaissant son impact | Rapport d'impact dans la PR du module |
| **Relecteur, équipes réseau et sécurité** | Comprendre une PR sans relire le plan ligne à ligne | Commentaire de PR : constats, flux demandés |
| **Équipe non technique** (plus tard) | Obtenir une stack sans connaître Azure | Console, jamais la CLI |

## 7. Le contrat de stack

Un fichier par stack, versionné avec son code. Exemple illustratif (schéma à définir) :

```yaml
kind: Stack
metadata:
  name: archive-factures
  zone: app-facturation-prd
  owner: equipe-facturation
spec:
  type: storage-private          # entrée du catalogue
  moduleVersion: 1.4.0           # version du module maison
  inputs:
    containers: [factures, exports]
    redundancy: ZRS
  flows:                         # ouvertures de flux nécessaires
    - from: snet-app-facturation
      to: self
      port: 443
      reason: lecture et écriture depuis l'application
  outOfBand: deny                # deny | report, avec exceptions par ressource
```

Quatre usages du même contrat :

| Usage | Fréquence | Ce que fait `lzctl` |
|---|---|---|
| **Créer** | À la demande | Génère le code de la stack, son contrat et son entrée dans la pipeline |
| **Contrôler** | Chaque PR | Vérifie que le changement respecte le contrat et n'écrase rien |
| **Surveiller** | Chaque nuit | Relève les écritures hors pipeline sur la stack |
| **Faire évoluer** | À chaque version de module | Indique quelles stacks sont touchées, et comment |

**Flux réseau.** Les flux déclarés sont rattachés à la PR. Si la règle se pose dans le
périmètre de la zone (NSG), le code est généré. Si elle relève d'un pare-feu que l'équipe ne
possède pas, la PR porte la demande de flux, prête à transmettre à l'équipe réseau.

**Propriété du contenu.** `outOfBand` dit ce qu'il advient d'une écriture hors pipeline :
refusée, ou simplement signalée. Une exception par ressource permet de sortir du contrôle un
contenu qui appartient à une autre équipe (une spécification d'API, par exemple).

## 8. Le workflow complet

`lzctl` est un outil du workflow, pas le workflow entier.

| Étape | Qui | Rôle de `lzctl` |
|---|---|---|
| 1. Exprimer le besoin | Ingénieur (fichier de demande) ; plus tard équipe non technique (console) | Fournit le schéma de demande |
| 2. Produire la stack | Ingénieur ou console | Contrat, code Terraform, entrée de pipeline, à partir du catalogue |
| 3. Proposer | Ingénieur ou console | Branche et PR ; plan, constats et flux demandés en commentaire |
| 4. Valider | Humains (plateforme, réseau, sécurité) | Aucun ; la décision reste humaine |
| 5. Déployer | Pipeline approuvée | Contrôle final avant apply (écritures hors pipeline) |
| 6. Surveiller | Pipeline planifiée | Relevé nocturne par stack |
| 7. Faire évoluer | Auteur de module, Renovate | Rapport d'impact, PR de montée de version |

**Règle : un humain ou un agent propose, un humain valide, seule une pipeline déploie.**

## 9. Briques fonctionnelles

Les noms de commandes sont des propositions.

| Brique | Commande | Rôle | État |
|---|---|---|---|
| **Garde hors pipeline** | `check` (PR), `drift` étendu (nuit) | Pour chaque ressource du plan : écritures faites par une autre identité que la pipeline depuis le dernier déploiement (journal d'activité Azure) ; comparaison de contenu pour ce que Terraform ne relit pas (spécification d'API) | À construire |
| **Consommateurs de modules** | `module consumers` | Quels dépôts utilisent quel module, à quelle version, avec quelles variables | À construire |
| **Différence d'interface** | `module diff` | Entre deux versions : variable supprimée, renommée ou devenue obligatoire, sortie retirée, ressource déplacée sans bloc `moved` | À construire |
| **Impact** | `module impact` | Plan de chaque consommateur avec la nouvelle version, résultat agrégé | À construire |
| **Contrat de stack** | `stack new`, `validate` | Génération depuis le catalogue ; validation du contrat | À construire (remplace `workload add` et `add-blueprint`) |
| **Conformité au contrat** | `check` | Le plan reste dans le périmètre de la stack ; flux du plan égaux aux flux déclarés ; changements sensibles en tête (destructions, rôle Owner, exposition publique) | À construire |
| **Découverte** | `discover` | Modèle minimal de l'existant en lecture seule : zones, réseaux et plages IP, services mutualisés | À construire |
| **Proposition** | `propose` | Branche et PR à partir d'une demande | À construire |
| Pipelines | `init` (partie pipelines) | Pipelines GitHub Actions et Azure DevOps : validation, plan, garde des destructions, apply | Existe |
| Orchestration | `plan`, `apply` | Couches dans l'ordre de dépendance | Existe |
| Maintenance | `upgrade` | Versions AVM | Existe |

**Hors du chemin principal**, conservé mais marqué expérimental : génération de la fondation
(management groups, connectivité, identité, gouvernance), blueprints, `audit` CAF, cycle de
vie des policies, `rollback`, commandes de state, Atlantis.

État vérifié le 2026-11-12 sur la sortie de `init --from-file` : les couches management
groups et connectivité ne passent pas `terraform init` (arguments inconnus des modules AVM
épinglés) ; le manifeste généré ne contient pas les landing zones pourtant créées sur disque.

## 10. « Utilisable en production » : critères

| Critère | Exigence |
|---|---|
| Génération fiable | La CI exécute `terraform validate` et `terraform test` sur la sortie de chaque type de stack |
| Déterminisme | Même contrat, même code ; même entrée, même constat |
| Innocuité | Identité en lecture seule pour tous les contrôles ; aucune écriture Azure hors pipeline approuvée |
| Échec explicite | Un contrôle qui n'a pas pu lire le dit et ne passe jamais en silence |
| Adoption progressive | Chaque contrôle peut être informatif avant d'être bloquant ; exceptions justifiées et versionnées |
| Intégration | Azure DevOps et GitHub ; un commentaire de PR unique mis à jour ; sortie JSON ; codes de retour stables |
| Compatibilité | Schéma du contrat versionné, migration fournie à chaque changement |
| Distribution | Binaire signé, versions sémantiques, notes de version |
| Preuve | Validé sur un tenant réel, puis sur un parc qui n'est pas celui de l'auteur |

## 11. Priorités

| Ordre | Chantier | Raison |
|---|---|---|
| 1 | **Dire vrai sur l'existant** : CI qui valide la sortie générée ; marquer expérimental ce qui ne passe pas | Rien de crédible sans cela |
| 2 | **Garde hors pipeline** (`check` en PR, relevé nocturne) | Situation 1 ; fonctionne sur un dépôt existant, sans rien adopter d'autre |
| 3 | **Modules** : consommateurs, différence d'interface, impact | Situation 2 ; même propriété, valeur sans adoption |
| 4 | **Contrat de stack et `stack new`** pour un seul type (`storage-private`), sur les modules de l'organisation, flux compris | Situation 3 ; cœur de la standardisation |
| 5 | **Conformité au contrat** dans `check` | Ferme la boucle créer → contrôler |
| 6 | **`discover` minimal et `propose`** | Prérequis d'une demande automatisée |
| 7 | **Schéma de demande, interface pour agents, console** | Le workflow complet pour une équipe non technique |

Premières sources prises en charge : modules par tags Git, dépôts Azure DevOps, plus un mode
« dossiers locaux » qui fonctionne partout. Registres privés et GitHub ensuite.

Les chantiers 2 et 3 passent avant la génération parce qu'ils apportent de la valeur sur un
dépôt existant en quelques minutes ; c'est ce qui installe l'outil dans la pipeline.

## 12. Modèle de confiance

Lecture seule garantit l'innocuité, pas l'exactitude. Chaque constat :
- **affiche sa couverture** : ce qui a été lu, ce qui n'a pas pu l'être (droits insuffisants,
  journal expiré), horodatage ; absence de preuve n'est pas preuve d'absence ;
- porte **une preuve** (ID de ressource, identité et date de l'écriture, `fichier:ligne`) et
  **la règle versionnée** qui l'a produit ;
- est **déterministe** ;
- accepte des **exceptions justifiées et versionnées**.

L'outil s'exécute dans la pipeline de l'organisation, avec son identité, sans appel sortant
hors API Azure et de la forge Git.

## 13. Console et LLM : plus tard, et secondaires

| | `lzctl` | Console |
|---|---|---|
| Pour qui | Ingénieurs plateforme, pipelines | Équipes non techniques |
| Rôle | Contrat, génération, contrôles, PR | Recueillir un besoin en langage naturel, suivre la demande |
| Logique métier | Toute | Aucune |

- Le LLM traduit un besoin en **demande typée** ; `lzctl` produit le contrat, le code et la
  PR. Le LLM n'écrit pas le Terraform.
- Le LLM s'exécute **dans le tenant du client**.
- `lzctl` reste complet sans LLM : une demande s'écrit aussi à la main.
- La PR issue de la console passe par les mêmes contrôles que celle d'un ingénieur.
- Les données lues dans le tenant (noms, tags, descriptions) sont des données, pas des
  consignes.

## 14. Indicateurs de succès

| Indicateur | Cible indicative |
|---|---|
| Déploiements ayant écrasé un changement hors pipeline sans avertissement | Zéro |
| Délai entre une écriture hors pipeline et son signalement | Moins de 24 heures |
| Versions de modules publiées avec rapport d'impact | 100 % |
| Écart entre la version courante d'un module et celle de ses consommateurs | En baisse, suivi par module |
| Délai entre une demande de stack et une PR prête à valider | Moins d'une heure |

## 15. Hypothèses à vérifier et questions ouvertes

Hypothèses techniques, à confirmer avant de construire :
- le contenu OpenAPI importé dans API Management n'est pas relu par le provider Terraform, ce
  qui expliquerait l'absence de dérive dans le plan ;
- le journal d'activité Azure (90 jours) suffit à attribuer une écriture à une identité, pour
  les types de ressources visés.

Baseline à faire avant chaque brique : outils de comparaison OpenAPI, bibliothèques d'analyse
de configuration Terraform, capacités exactes de Renovate sur les modules par tags Git, outils
existants de différence d'interface entre versions de module.

Questions ouvertes :
- Le mode de distribution « dépôt de modules + tags Git » est-il assez répandu pour être le
  premier pris en charge ? À confirmer auprès de pairs.
- `module impact` : exécuter les plans dans les pipelines des consommateurs, ou depuis un
  poste disposant des accès ?
- Contrat de stack : un fichier par stack, ou une section de `lzctl.yaml` ?
- Vocabulaire : « stack » dans l'outil et « platform » dans la console, ou un seul terme ?
- Le chantier 4 (contrat de stack) doit-il passer avant les chantiers 2 et 3 ?
