# Product Brief: lzctl

**Date:** 2026-11-13
**Author:** kjourdan1
**Status:** Brouillon à valider
**Version:** 0.1

---

## 1. Executive Summary

`lzctl` est une CLI qui aide une équipe plateforme à offrir un « platform as a service » sur
Azure : une demande, exprimée dans un schéma fixe, devient une stack Terraform conforme au
catalogue de l'organisation, proposée par PR et suivie dans le temps. Le catalogue appartient
à l'organisation ; `lzctl` l'orchestre.

**Key Points:**
- Problem: ajouter une plateforme veut dire écrire une stack à la main, avec ses décisions (réseau, flux, réutilisation) refaites à chaque fois.
- Solution: un moteur déterministe entre une demande et une PR, puis un suivi par stack.
- Target Users: ingénieurs plateforme ; à terme, équipes non techniques via une console.
- Primary Metric: délai entre une demande et une PR prête à valider.

---

## 2. Problem Statement

### The Problem

Dans un dépôt IaC Azure, chaque nouvelle plateforme (une ressource et ses à-côtés) est écrite
à la main. Les mêmes décisions sont reprises à chaque fois : quelle plage IP, quels flux
ouvrir, créer un service ou se rattacher à un service mutualisé, quelles tailles. Le résultat
dépend de la personne qui écrit, et rien ne relie ensuite la stack à la demande d'origine.

### Who Experiences This Problem

**Primary Users:**
- Ingénieurs plateforme qui livrent des stacks pour des équipes applicatives
- Relecteurs (plateforme, réseau, sécurité) qui valident les PR

**Secondary Users:**
- Équipes applicatives ou métier qui attendent leur plateforme

### Current Situation

**How Users Currently Handle This:**
Copie d'une stack voisine, adaptation à la main, demande de flux réseau par ticket séparé.
Dans `azure-platform`, l'audit de novembre 2026 le constate : « ajouter une plateforme veut
dire écrire une stack à la main ».

**Pain Points:**
- Stacks hétérogènes : aucune n'est construite comme la voisine
- Flux réseau et plages IP traités hors de la PR
- Services mutualisés recréés au lieu d'être réutilisés
- Après la création : déploiement qui écrase un changement fait dans le portail ; modules maison qu'on n'ose plus faire évoluer faute de voir l'impact

**Cas vécu (anonymisé).** Un besoin réglementaire de flux de facturation a demandé trois
Functions, plusieurs pipelines Data Factory et des ouvertures de flux (private endpoints vers
runtimes d'intégration et agents de build, Functions vers Entra). Le module Function
n'existait pas : il a fallu le créer, mettre à jour d'autres modules, industrialiser les
pipelines et rendre le tout duplicable entre environnements. Trois mois d'allers-retours. Le
besoin ne correspondait à aucune brique unique, et l'une d'elles manquait : c'est le cas courant.

### Impact & Urgency

**Impact if Unsolved:**
Délais de mise à disposition longs, parc incohérent, charge de relecture élevée.

**Why Now:**
La mise en place d'un platform as a service est l'objectif d'une mission à venir ; le dépôt
`azure-platform` et l'application Cockpit fournissent deux terrains d'essai immédiats.

**Frequency:**
À chaque nouvelle plateforme et à chaque modification d'une plateforme existante. Fréquence
réelle à mesurer en mission.

---

## 3. Target Users

### User Personas

#### Persona 1: Ingénieur plateforme

- **Role:** Livre et maintient les stacks dans une ou plusieurs subscriptions
- **Goals:** Livrer vite, de façon identique, sans surprise au déploiement
- **Pain Points:** Travail répétitif, décisions refaites, relectures longues
- **Technical Proficiency:** Élevée (Terraform, Azure, pipelines)
- **Usage Pattern:** CLI sur son poste et dans la pipeline, à chaque PR

#### Persona 2: Demandeur non technique (plus tard)

- **Role:** Membre d'une équipe métier ou applicative
- **Goals:** Obtenir une plateforme sans connaître Azure
- **Pain Points:** Tickets, délais, vocabulaire technique
- **Technical Proficiency:** Faible
- **Usage Pattern:** Console en langage naturel ; n'utilise jamais la CLI

### User Needs

**Must Have (MVP):**
- Écrire une demande dans un schéma fixe et obtenir une stack générée depuis le catalogue de l'organisation
- Voir dans la PR la décision prise et la règle qui l'a produite
- Garder du Terraform standard, qui fonctionne sans `lzctl`

**Should Have:**
- Plage IP attribuée et flux réseau rattachés à la PR
- Rattachement à un service mutualisé plutôt que création, quand une règle le permet
- Savoir dès le dépôt de la demande quelles briques manquent au catalogue

**Nice to Have (Future):**
- Console en langage naturel ; plan de l'environnement interrogeable par un LLM
- Proposition d'architecture quand aucune brique n'existe
- Suivi par stack : écritures hors pipeline, impact d'une version de module

---

## 4. Proposed Solution

### Solution Overview

Un moteur déterministe, sans état, exécuté dans la pipeline existante. Il lit une demande, la
résout avec les règles de l'organisation, génère la stack depuis une offre du catalogue et
ouvre une PR. Seule une pipeline approuvée déploie. L'unité livrée est une stack dans une
subscription existante, un resource group par stack.

Trois entrées : la **demande** (ce qui est voulu), le **catalogue** (ce qui peut être livré)
et le **contexte de la plateforme** (ce qui existe et les standards : sources de modules,
nommage, plages IP libres, hub à raccorder, services mutualisés comme une passerelle d'API).
Le contexte est découvert par lecture d'Azure (droit Reader) et des fichiers Terraform
existants, puis validé par l'équipe plateforme dans une PR ; seul le contexte validé sert aux
décisions.
Une sortie : une PR, dans une branche du dépôt ou dans un dépôt dédié, qui contient la stack,
son contrat, son README et la demande d'origine telle qu'elle a été formulée.

Le but est de mâcher le travail de l'ingénieur plateforme : traduire un besoin métier en code
d'infrastructure conforme, qu'il relit et ajuste, au lieu de l'écrire depuis zéro.

Une offre est une couche mince : un appel aux modules de l'organisation (ou à AVM), un schéma
de paramètres et des règles. Elle ne réécrit pas les ressources.

### Key Capabilities

1. **Schéma de demande et résolveur à règles**
   - Description: Une demande typée est résolue en offre, tailles et contrôles par une table de règles versionnée ; chaque résultat cite sa règle. Sans règle applicable, la demande va à un humain.
   - User Value: Une demande automatisée devient acceptable pour l'équipe qui valide.

2. **Catalogue de l'organisation**
   - Description: Les briques vivent dans le dépôt de l'organisation. Une demande se résout en une composition de briques ; `lzctl` rend un rapport de couverture (briques disponibles, briques manquantes) et fournit le squelette d'une brique nouvelle aux conventions en place.
   - User Value: L'outil s'adapte aux standards en place au lieu d'imposer les siens.

3. **Réseau et réutilisation**
   - Description: Plage IP attribuée ; matrice des flux calculée (chaque brique déclare ses besoins, le contexte dit où sont les agents, les runtimes et le hub) et rattachée à la PR ; rattachement à un service mutualisé du contexte validé.
   - User Value: Les décisions les plus pénibles à la main sont prises et tracées.

4. **Contrat et suivi de stack**
   - Description: La stack est générée une fois, puis reprise à la main ; les changements suivants sont des PR ordinaires. Le contrat issu de la demande reste dans la stack comme trace d'origine. La brique fournit aussi la structure par environnement et la pipeline de livraison applicative.
   - User Value: L'ingénieur garde la main sur le code ; l'origine de chaque stack reste lisible.

5. **Plan de l'environnement**
   - Description: Le contexte validé est un graphe (zones, stacks, réseaux, services mutualisés, flux). `lzctl` en tire une carte lisible à plusieurs niveaux de détail, en langage courant pour le métier et en vue technique pour l'ingénieur. Une demande s'y dessine en pointillé : ce qu'elle ajouterait, à quoi elle se raccorderait, quels flux elle ouvrirait, quelles briques manquent. Inspiré de Floorplan pour le concept, pas pour le format, qui est conçu pour du code applicatif.
   - User Value: Métier, développeurs, réseau et plateforme lisent le même schéma avant d'écrire une ligne ; les flux se voient d'emblée au lieu d'être découverts un par un.

6. **Proposition d'architecture hors catalogue**
   - Description: Quand aucune brique ne couvre le besoin, un LLM interroge `lzctl` (contexte, conventions, couverture) par MCP et rédige une proposition typée, dessinée en pointillé sur le plan. Elle liste ses hypothèses, ses questions ouvertes et les briques à créer. `lzctl` la valide et la rend ; il ne génère aucun code à partir d'elle.
   - User Value: La conception avec le métier et les développeurs part d'un schéma discutable au lieu d'une page blanche.

### Unique Value Proposition

Une CLI sans état qui porte les décisions propres à Azure (réseau, flux, réutilisation) de
façon déterministe, et dont la seule sortie est une PR. Rien à héberger, aucun nouveau chemin
vers la production. La console et le LLM sont interchangeables.

### MVP Scope

**Core Features for Launch:**
- Schéma de demande, validation, résolveur minimal
- Une offre, composée de briques dès le départ : application web interne légère (Container Apps avec mise à l'échelle à zéro, stockage, Key Vault, authentification Entra), écrite dans le catalogue ; Cockpit en est la première demande réelle
- Génération de la stack, de son contrat et de son README dans une PR qui cite la demande d'origine, plan commenté ; le README porte un schéma de la stack (Mermaid)
- Critère d'acceptation : la demande « Cockpit » produit une PR dont la stack se déploie sur le tenant professionnel sans retouche manuelle
- CI qui exécute `terraform validate` sur la sortie de l'offre

**Deferred to Future Phases:**
- Seconde offre (projet IA rattaché au compte Foundry mutualisé), puis espace sur AKS mutualisé
- Ordre retenu après l'étape 1, d'après les trois postes qui ont le plus pesé dans le cas vécu (conception, attente entre équipes, flux découverts un par un) :
  1. Découverte du contexte (Azure et Terraform existant) et graphe validé
  2. Matrice des flux et plan de l'environnement, d'abord en Mermaid dans la PR
  3. Rapport de couverture
  4. Interface MCP et proposition d'architecture hors catalogue
  5. Plan interactif
- Squelette de brique nouvelle
- Environnements multiples et pipeline de livraison applicative par brique
- Attribution de plage IP, rattachement aux services mutualisés
- Demandes de modification automatisées : écartées tant que la stack est générée une fois
- Garde hors pipeline, impact des versions de modules
- Console sur `ai-gateway`
- Vending de subscriptions : étape amont, hors du cœur, pour les organisations qui possèdent leur tenant

---

## 5. Goals & Constraints

### Business Goals

- Étape 1 : pièce de portfolio, utilisée sur un tenant professionnel (Cockpit) et sur `azure-platform`
- Étape 2 : outil utilisé en mission pour mettre en place un platform as a service
- Étape 3 : projet open source avec un premier utilisateur extérieur

### Constraints

**Non-Negotiable Constraints:**
- Un humain valide, seule une pipeline approuvée déploie ; aucun agent n'écrit sur Azure
- Le LLM s'exécute dans le tenant du client et n'écrit pas le Terraform ; `lzctl` reste complet sans lui
- Rien de l'infrastructure d'un client dans un dépôt public
- Budget Azure personnel de quelques dizaines d'euros

**Technical Constraints:**
- Un seul mainteneur
- Go, Terraform standard, Azure DevOps et GitHub
- L'existant génère du code qui ne s'initialise plus sur deux couches de fondation ; ce code sort du chemin principal

---

## 6. Success Metrics

### Primary Metrics

**Délai entre une demande et une PR prête à valider**
- Baseline: inconnue ; à relever en mission
- Target: moins d'une heure
- Timeline: fin de l'étape 1
- Measurement: horodatage de la demande et de la PR

**Fidélité de la génération**
- Baseline: sans objet
- Target: stack générée déployée sans retouche manuelle
- Timeline: fin de l'étape 1
- Measurement: différence entre la PR générée et le commit déployé

**Portabilité du catalogue**
- Baseline: zéro contexte
- Target: la même offre livrée dans deux contextes (subscription seule ; tenant complet)
- Timeline: 6 mois
- Measurement: deux PR publiques, deux déploiements

### Success Vision

- **3 months:** une demande écrite à la main produit la stack de Cockpit, déployée sur le tenant professionnel
- **6 months:** deux offres, deux contextes, attribution de plage IP et flux dans la PR
- **12 months:** utilisé en mission ; console de démonstration ; premier utilisateur extérieur

---

## 7. Market & Competition

### Market Context

**Market Size:** non estimée ; ce n'est pas un objectif à ce stade.

**Key Trends:**
- Ingénierie de plateforme : portails internes et libre-service d'infrastructure
- Agents LLM en amont des demandes, avec exigence de garde-fous déterministes

**Target Segment:** équipes plateforme sur Azure avec Terraform, sans plateforme Terraform
commerciale ni portail interne abouti ; en particulier les entités qui ne possèdent pas leur tenant.

### Competitive Landscape

#### Azure Deployment Environments
- Strengths: service Microsoft, catalogue dans Git, libre-service intégré au portail développeur
- Weaknesses: Terraform passe par une image de conteneur personnalisée ; synchronisation manuelle du catalogue ; déploie directement, sans PR relue
- Positioning: environnements de développement à la demande

#### Backstage (Software Templates)
- Strengths: formulaire, squelette Terraform rendu, PR ouverte ; écosystème large
- Weaknesses: un portail à héberger et maintenir ; le gabarit ne décide rien (pas de règles, pas d'attribution IP, pas de rattachement)
- Positioning: portail développeur généraliste ; pourrait appeler `lzctl` comme moteur

#### HCP Terraform (provisionnement sans code)
- Strengths: modules du registre privé offerts en libre-service, notification des nouvelles versions de module aux workspaces
- Weaknesses: offre commerciale liée à l'édition ; un workspace par demande, hors du flux PR de l'organisation
- Positioning: libre-service pour les clients de la plateforme HashiCorp

### Our Differentiation

**Advantages:**
- Sans état et sans service à héberger ; la sortie est une PR dans le flux existant
- Décisions propres à Azure prises par règles et tracées

**Gaps to Close:**
- Pas d'interface ; un seul mainteneur ; aucune preuve en production
- Attribution de plage IP : s'appuyer sur Azure IPAM (projet Microsoft, avec provider Terraform) plutôt que la refaire, à évaluer

---

## 8. Risks & Assumptions

### High-Priority Risks

**Risk 1: Dispersion**
- Probability: Élevée
- Impact: Élevé
- Mitigation: Une offre, un critère d'acceptation ; rien d'autre avant la fin de l'étape 1
- Owner: kjourdan1

**Risk 2: Le moteur de catalogue réinvente les modules Terraform**
- Probability: Moyenne
- Impact: Élevé
- Mitigation: Une offre reste une couche mince au-dessus des modules ; pas de gabarit de ressources
- Owner: kjourdan1

**Risk 3: Recouvrement avec Backstage ou Azure Deployment Environments**
- Probability: Moyenne
- Impact: Moyen
- Mitigation: Se placer comme moteur appelable par un portail ; concentrer l'effort sur les règles, le réseau et la réutilisation
- Owner: kjourdan1

**Risk 4: Confidentialité en mission**
- Probability: Moyenne
- Impact: Élevé
- Mitigation: Vérifier les clauses de propriété et de confidentialité ; catalogue et demandes du client hors du dépôt public
- Owner: kjourdan1

### Critical Assumptions

- Une offre peut s'exprimer comme une couche mince au-dessus de modules existants
- Le périmètre « stack dans une subscription » correspond à celui de la mission
- Les relecteurs acceptent une PR générée si la règle appliquée est citée

**Validation Plan:**
Écrire l'offre dans le catalogue, servir la demande Cockpit, déployer. Confronter le
périmètre et le catalogue à la réalité de la mission dès son démarrage.

---

## 9. Dependencies

### Internal Dependencies

- Cockpit comme première demande réelle, sur le tenant professionnel
- `azure-platform` : pipeline de PR existante, vending de subscriptions, `ai-gateway` comme base de la future console

### External Dependencies

- Modules Azure Verified Modules ou modules de l'organisation
- Forge Git (GitHub, Azure DevOps) pour les PR

### Current Blockers

- Génération de fondation cassée — Resolution: la sortir du chemin principal et la marquer expérimentale
- `docs/product.md` orienté vers les contrôles de pipeline — Resolution: le réaligner sur ce brief après validation

---

## 10. Next Steps

### Immediate Actions

1. Valider ce brief (offre de départ, périmètre de la première version, ordre des étapes)
2. Figer l'architecture cible de Cockpit et la découper en briques
3. Rédiger le PRD de l'étape 1 : schéma de demande, format d'une offre, critère du plan vide

### Recommended Handoff

**Hand off to:** Product Manager (PRD)

**Required before handoff:**
- Brief validé par l'auteur
- Architecture cible de Cockpit figée : elle définit la première offre

---

## Appendix

### Research Sources

- [Azure Deployment Environments : structure de catalogue](https://learn.microsoft.com/en-us/azure/deployment-environments/best-practice-catalog-structure)
- [Azure Deployment Environments : modèle d'extensibilité](https://learn.microsoft.com/en-ca/Azure/deployment-environments/concept-extensibility-model)
- [Terraform avec Backstage (Spacelift)](https://spacelift.io/blog/terraform-backstage)
- [HCP Terraform : provisionnement sans code](https://developer.hashicorp.com/validated-designs/terraform-operating-guides-scale/nocode-provisioning)
- [Azure IPAM](https://github.com/Azure/ipam)
- [Floorplan](https://github.com/haydenschmitty/floorplan) : carte interactive d'une application, plans en pointillé avec hypothèses et questions ouvertes (lu le 2026-11-13, version 0.8.0)

### Stakeholders Consulted

- kjourdan1 — auteur et unique mainteneur

### Additional Notes

La recherche de concurrence est un premier passage sur des sources en ligne, non vérifiées
par essai. Crossplane, Radius et les plateformes Terraform commerciales autres que HCP n'ont
pas été examinés.

---

**Document Status:** Brouillon à valider
**Last Updated:** 2026-11-13
