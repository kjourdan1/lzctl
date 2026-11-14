# Product Requirements Document (PRD)

**Project Name:** lzctl — étape 1
**Version:** 0.1
**Date:** 2026-11-14
**Author:** kjourdan1 (PM)
**Status:** Brouillon à valider
**Track:** Quick Flow

> Fait foi pour le *quoi* et le *pourquoi*. Le *comment* relève de l'architecture.
> Détails reportés et questions ouvertes : `addendum.md`. Décisions : `decision-log.md`.
> Source : `product-brief-lzctl-2026-11-13.md`.

---

## Executive Summary

**Problem Statement:** Ajouter une plateforme dans un dépôt IaC Azure veut dire écrire une
stack à la main. L'ingénieur plateforme refait à chaque fois la traduction d'un besoin en code
conforme aux standards de son organisation.

**Proposed Solution:** Une demande écrite dans un schéma fixe est résolue contre le catalogue
de l'organisation et produit, en une commande, une stack Terraform composée de briques, avec
son contrat, son README et le texte de la PR. La stack est générée une fois, puis reprise à la
main.

**Business Value:** Prouver le moteur de bout en bout sur un cas réel (l'application Cockpit),
avant toute fonction de réseau, de contexte ou d'IA.

**Target Outcome:** La demande « Cockpit » produit une stack qui se déploie sur le tenant
professionnel sans retouche manuelle.

---

## Project Overview

### Background
`lzctl` existe comme générateur de fondation Azure Landing Zones ; deux de ses couches ne
s'initialisent plus. Le brief le réoriente vers l'aide à la mise en place d'un platform as a
service : le catalogue appartient à l'organisation, `lzctl` l'orchestre.

### Current State → Desired State
- **Current:** des gabarits de ressources embarqués dans le binaire ; aucune notion de demande, de brique ni d'offre.
- **Desired:** un catalogue de briques dans le dépôt de l'organisation ; une demande typée ; une stack générée de façon déterministe, proposée par PR.

### Stakeholders
| Stakeholder | Role | Interest | Influence |
|-------------|------|----------|-----------|
| kjourdan1 | Auteur, mainteneur, premier utilisateur | Outil démontrable, puis utilisable en mission | Décide de tout |
| Ingénieur plateforme (persona) | Utilisateur de la CLI | Ne plus écrire une stack depuis zéro | Valide l'utilité |
| Relecteur de PR (persona) | Valide ce qui sera déployé | Comprendre la demande et la décision sans relire tout le code | Peut bloquer l'adoption |

---

## Goals and Objectives

### Business Goals
1. **G1** — Prouver le moteur sur un cas réel : Cockpit déployé depuis une demande.
2. **G2** — Poser un format de catalogue que l'organisation possède et qui reste utilisable sans `lzctl`.
3. **G3** — Rétablir la confiance dans ce que `lzctl` génère.

### User Goals
1. Écrire une demande courte et obtenir une stack conforme, prête à relire.
2. Lire dans la PR ce qui a été demandé, ce qui a été décidé et pourquoi.

---

## Functional Requirements

### FR-001: Schéma de demande — MUST
**Description:** Une demande est un fichier dans un schéma fixe et versionné. Elle porte le besoin dans les mots du demandeur, l'offre voulue, le propriétaire et les paramètres.
**Acceptance Criteria:**
- Une demande valide est acceptée ; la commande rend le code de retour de succès.
- Un champ inconnu, manquant ou mal typé est refusé, avec le nom du champ et la règle violée.
- Le texte du besoin est conservé mot pour mot dans tout ce qui est généré.
- Le schéma est exportable, pour qu'un portail ou un LLM puisse produire des demandes valides.
**Related Epic:** EPIC-002

---

### FR-002: Brique = module Terraform et métadonnées — MUST
**Description:** Une brique est un module Terraform ordinaire accompagné d'un fichier de métadonnées : description en langage courant, schéma des paramètres, flux réseau déclarés.
**Acceptance Criteria:**
- Une brique sans fichier de métadonnées, ou dont les métadonnées sont invalides, est refusée avec la raison.
- Chaque paramètre déclaré correspond à une variable du module ; chaque variable obligatoire du module est couverte ; tout écart est signalé.
- Le champ des flux déclarés est présent dans le schéma et validé ; il n'est pas encore exploité.
- Le module passe `terraform validate` seul, sans `lzctl`.
**Related Epic:** EPIC-001

---

### FR-003: Offre = composition de briques — MUST
**Description:** Une offre nomme les briques qui la composent, la façon dont elles se raccordent entre elles et les paramètres qu'elle expose à la demande.
**Acceptance Criteria:**
- Une offre qui cite une brique absente du catalogue est refusée.
- Un raccordement vers une sortie inexistante, ou une dépendance circulaire entre briques, est refusé.
- Une commande valide le catalogue entier et rend un code de retour d'échec à la première anomalie signalée.
**Related Epic:** EPIC-001

---

### FR-004: Résolution déterministe — MUST
**Description:** `lzctl` résout une demande en offre, briques et paramètres, sans appel à Azure ni à un LLM.
**Acceptance Criteria:**
- Le résultat cite l'offre et les briques retenues, avec leur version.
- Une demande qui vise une offre absente du catalogue est refusée comme « hors catalogue », avec un code de retour distinct ; aucun fichier n'est généré.
- Un paramètre hors des valeurs permises par l'offre est refusé, avec les valeurs permises.
**Related Epic:** EPIC-002

---

### FR-005: Génération de la stack — MUST
**Description:** À partir d'une demande résolue, `lzctl` écrit une stack Terraform dans son propre dossier : un resource group, les briques appelées comme modules, leurs raccordements.
**Acceptance Criteria:**
- La stack générée passe `terraform fmt -check` et `terraform validate`.
- La stack ne contient aucune ressource écrite hors des briques, hormis le resource group et la configuration du backend et des providers.
- Si le dossier de la stack existe déjà, la commande refuse d'écrire et le dit.
- Une option de simulation liste les fichiers qui seraient écrits, sans rien écrire.
**Related Epic:** EPIC-002

---

### FR-006: Contrat et README de la stack — MUST
**Description:** Chaque stack générée porte son contrat (la demande résolue) et un README qui la décrit.
**Acceptance Criteria:**
- Le contrat contient la demande d'origine, l'offre, les briques avec leur version et les paramètres résolus.
- Le README contient le besoin mot pour mot, la description de chaque brique en langage courant, un schéma Mermaid des briques et de leurs raccordements, et les étapes de déploiement.
- Le contrat et le README ne contiennent ni identifiant de subscription ou de tenant, ni secret.
**Related Epic:** EPIC-002

---

### FR-007: Texte de la PR — MUST
**Description:** `lzctl` produit la description de la PR, prête à coller ou à passer à la forge.
**Acceptance Criteria:**
- Elle contient la demande d'origine mot pour mot, l'offre et les briques retenues avec la raison, et la liste des fichiers générés.
- Elle signale explicitement que la stack a été générée et par quelle version de `lzctl` et du catalogue.
**Related Epic:** EPIC-003

---

### FR-008: Ouverture de la PR sur GitHub — SHOULD
**Description:** Une commande crée la branche, y committe la stack et ouvre la PR, avec l'identité Git et la forge déjà configurées par l'utilisateur.
**Acceptance Criteria:**
- La PR est ouverte sur une branche dédiée à la demande ; rien n'est poussé sur la branche principale.
- Si l'identité ou les droits manquent, la commande échoue sans rien pousser et le dit.
- `lzctl` n'approuve ni ne fusionne jamais une PR.
**Related Epic:** EPIC-003

---

### FR-009: Offre « application web interne légère » — MUST
**Description:** Le catalogue du tenant professionnel, dans un dépôt d'infrastructure à part, contient une offre de trois briques : application conteneurisée avec mise à l'échelle à zéro et authentification Entra, stockage de tables, coffre de secrets.
**Acceptance Criteria:**
- La demande « Cockpit » produit une stack qui se déploie par la pipeline du dépôt sans retouche manuelle du code généré.
- L'application déployée n'est accessible qu'au compte Entra autorisé.
- L'application lit ses secrets dans le coffre par identité managée ; aucun secret n'est stocké dans le dépôt ni dans la configuration de l'application.
- Si l'image de Cockpit n'est pas prête, le critère se vérifie avec une image de démonstration publique.
**Related Epic:** EPIC-004

---

### FR-010: Vérification en CI de ce qui est généré — MUST
**Description:** Le dépôt `lzctl` contient un catalogue d'exemple et une demande d'exemple ; la CI génère la stack et la valide.
**Acceptance Criteria:**
- La CI exécute `terraform init` sans backend et `terraform validate` sur la stack générée ; un échec fait échouer la CI.
- La CI compare la sortie à une sortie de référence versionnée ; toute différence fait échouer la CI.
**Related Epic:** EPIC-005

---

### FR-011: Explication d'une résolution — COULD
**Description:** Une commande affiche, sans rien écrire, comment une demande serait résolue.
**Acceptance Criteria:**
- Elle affiche l'offre, les briques, les paramètres résolus et les refus éventuels.
- Elle offre une sortie JSON.
**Related Epic:** EPIC-002

---

### FR-012: Marquage de l'existant hors chemin principal — SHOULD
**Description:** Les commandes existantes qui ne servent pas ce PRD sont signalées comme expérimentales. La liste exacte se décide à l'étape d'architecture.
**Acceptance Criteria:**
- L'aide de chaque commande concernée porte la mention « expérimental ».
- Aucune commande existante n'est supprimée dans cette étape.
**Related Epic:** EPIC-005

---

## Non-Functional Requirements

### NFR-001: Déterminisme — MUST (Reliability)
**Description:** Même demande et même version du catalogue donnent les mêmes fichiers.
**Acceptance / Threshold:** Zéro octet de différence entre deux générations successives ; aucun horodatage ni valeur aléatoire dans les fichiers générés.
**Measurement Method:** Test en CI qui génère deux fois et compare.

### NFR-002: Innocuité — MUST (Security)
**Description:** La validation, la résolution et la génération n'appellent pas Azure et ne demandent aucun droit.
**Acceptance / Threshold:** Zéro appel réseau vers Azure ; les commandes réussissent sans identifiants Azure.
**Measurement Method:** Test en CI exécuté sans identifiants ; recherche de secrets sur la sortie générée, zéro constat.

### NFR-003: Indépendance du code généré — MUST (Maintainability)
**Description:** La stack générée fonctionne sans `lzctl`.
**Acceptance / Threshold:** `terraform init`, `validate` et `plan` réussissent sur une machine sans le binaire `lzctl` ; aucun provider ni outil propre à `lzctl` n'est requis.
**Measurement Method:** Job de CI qui valide la sortie de référence sans installer `lzctl`.

### NFR-004: Messages d'erreur exploitables — SHOULD (Usability)
**Description:** Tout refus dit où et pourquoi.
**Acceptance / Threshold:** 100 % des refus nomment le fichier, le champ et la règle ; codes de retour documentés et distincts pour succès, demande invalide, catalogue invalide et hors catalogue.
**Measurement Method:** Tests unitaires sur chaque cas de refus.

### NFR-005: Temps de génération — SHOULD (Performance)
**Description:** La génération ne ralentit pas une pipeline.
**Acceptance / Threshold:** Moins de 5 secondes pour une offre de 10 briques au plus, hors `terraform init`.
**Measurement Method:** Mesure dans la CI sur le catalogue d'exemple.

---

## Epics and User Stories (Outline)

> Esquisse (user story) ; les stories détaillées viennent à l'étape suivante. En tant que… je veux… afin de…

### EPIC-001: Catalogue — briques et offres
**Business Value:** Un format que l'organisation possède et qui survit à `lzctl`.
**User Segments:** Ingénieur plateforme
**Related Requirements:** FR-002, FR-003

**User Stories (sketch):**
- **STORY-001:** En tant qu'ingénieur plateforme, je veux décrire une brique par un module et un fichier de métadonnées, afin qu'elle soit réutilisable et lisible par le métier.
  - Étant donné un module sans métadonnées, quand je valide le catalogue, alors la brique est refusée avec la raison.
- **STORY-002:** En tant qu'ingénieur plateforme, je veux composer une offre à partir de briques, afin de livrer un ensemble cohérent en une demande.
  - Étant donné une offre qui cite une brique absente, quand je valide le catalogue, alors l'offre est refusée.
- **STORY-003:** En tant qu'ingénieur plateforme, je veux valider tout le catalogue en une commande, afin de le contrôler dans la pipeline.

---

### EPIC-002: De la demande à la stack
**Business Value:** Le cœur du produit : traduire un besoin en code conforme.
**User Segments:** Ingénieur plateforme
**Related Requirements:** FR-001, FR-004, FR-005, FR-006, FR-011

**User Stories (sketch):**
- **STORY-004:** En tant qu'ingénieur plateforme, je veux valider une demande contre son schéma, afin de refuser tôt ce qui est mal formé.
- **STORY-005:** En tant qu'ingénieur plateforme, je veux résoudre une demande en offre et briques, afin de savoir ce qui sera livré avant de générer.
  - Étant donné une demande pour une offre absente, quand je la résous, alors elle est refusée comme hors catalogue et rien n'est écrit.
- **STORY-006:** En tant qu'ingénieur plateforme, je veux générer la stack dans son dossier, afin de la relire et de l'ajuster.
  - Étant donné un dossier de stack existant, quand je génère, alors la commande refuse d'écrire.
- **STORY-007:** En tant que relecteur, je veux un contrat et un README dans la stack, afin de comprendre d'où elle vient et ce qu'elle contient.
- **STORY-008:** En tant qu'ingénieur plateforme, je veux voir une résolution sans rien écrire, afin de corriger ma demande.

---

### EPIC-003: Proposition
**Business Value:** Un seul objet à relire, qui porte la demande et la décision.
**User Segments:** Ingénieur plateforme, relecteur
**Related Requirements:** FR-007, FR-008

**User Stories (sketch):**
- **STORY-009:** En tant que relecteur, je veux lire dans la PR la demande d'origine et la décision prise, afin de valider sans reconstituer le contexte.
- **STORY-010:** En tant qu'ingénieur plateforme, je veux ouvrir la PR en une commande, afin de ne pas refaire les gestes Git à la main.

---

### EPIC-004: Preuve — catalogue du tenant professionnel et Cockpit
**Business Value:** Un cas réel, déployé, public.
**User Segments:** kjourdan1
**Related Requirements:** FR-009

**User Stories (sketch):**
- **STORY-011:** En tant qu'ingénieur plateforme, je veux un dépôt d'infrastructure avec son catalogue, ses demandes et ses stacks, afin de reproduire la disposition d'une mission.
- **STORY-012:** En tant qu'ingénieur plateforme, je veux les trois briques de l'offre « application web interne légère », afin de servir la demande Cockpit.
- **STORY-013:** En tant que demandeur, je veux que la demande « Cockpit » aboutisse à une application déployée et protégée, afin de l'utiliser.

---

### EPIC-005: Socle de confiance
**Business Value:** Ce que `lzctl` annonce est vérifié à chaque changement.
**User Segments:** kjourdan1, tout lecteur du dépôt
**Related Requirements:** FR-010, FR-012

**User Stories (sketch):**
- **STORY-014:** En tant que mainteneur, je veux que la CI valide et compare la stack générée, afin qu'une régression se voie tout de suite.
- **STORY-015:** En tant que lecteur du dépôt, je veux voir quelles commandes sont expérimentales, afin de savoir sur quoi m'appuyer.

---

## Prioritization Summary (MoSCoW)

| Priority | Requirements | Rationale |
|----------|--------------|-----------|
| Must | FR-001 à FR-007, FR-009, FR-010, NFR-001 à NFR-003 | Le chemin minimal de la demande à une stack déployée, et la preuve qu'il tient |
| Should | FR-008, FR-012, NFR-004, NFR-005 | Confort et lisibilité ; la PR peut s'ouvrir à la main en attendant |
| Could | FR-011 | Aide à la mise au point d'une demande |
| Won't (this release) | Voir Out of Scope | Reporté après la preuve Cockpit |

---

## Success Metrics

| Metric | Baseline | Target | Measurement Method | Frequency |
|--------|----------|--------|--------------------|-----------|
| Retouches manuelles du code généré avant déploiement de Cockpit | Sans objet | Zéro | Différence entre la PR générée et le commit déployé | Une fois, à la fin de l'étape |
| Délai entre la demande écrite et la PR ouverte | Inconnu | Moins d'une heure | Horodatage du fichier de demande et de la PR | À chaque demande |
| Différence entre deux générations identiques | Sans objet | Zéro octet | Test en CI | À chaque changement |

---

## Assumptions and Dependencies

### Assumptions
1. Une offre s'exprime comme une couche mince au-dessus de modules ; aucune ressource n'a besoin d'être écrite hors d'une brique.
2. Dans cette étape, le catalogue et les stacks vivent dans le même dépôt ; les briques sont référencées sans versionnement par tags.
3. Une seule offre rend le résolveur trivial (la demande nomme l'offre) ; la table de règles vient avec la seconde offre.
4. Un seul environnement suffit pour Cockpit.

### Dependencies
| Dependency | Type | Owner | Status | Risk | Mitigation |
|------------|------|-------|--------|------|------------|
| Dépôt d'infrastructure du tenant professionnel (state, identités de pipeline, pipeline de PR) | Interne | kjourdan1 | À créer, à la main | Moyen : travail préalable hors `lzctl` | Reprendre le modèle de `azure-platform` |
| Image de conteneur de Cockpit | Interne | kjourdan1 | Non commencée | Moyen | Image de démonstration publique pour le critère d'acceptation |
| Modules Terraform sous-jacents (AVM ou ressources du provider) | Externe | Microsoft, HashiCorp | Disponibles | Faible : évolution d'interface | Versions épinglées dans les briques |

---

## Constraints

- **Technical:** Go, Terraform standard, GitHub pour cette étape ; un seul mainteneur ; budget Azure de quelques euros par mois pour Cockpit.
- **Business:** Rien de confidentiel dans les dépôts publics ; un humain valide, seule la pipeline déploie.
- **Timeline:** Pas d'échéance fixée ; l'étape se termine au critère d'acceptation de FR-009, pas à une date.

---

## Out of Scope

| Excluded | Reason | Revisit? |
|----------|--------|----------|
| Découverte du contexte, matrice des flux, plan de l'environnement | Demandent un modèle de l'existant ; premier chantier après l'étape 1 | Oui, en priorité |
| Rapport de couverture, squelette de brique nouvelle | N'ont de sens qu'avec plusieurs offres | Oui |
| Interface MCP, proposition d'architecture, console, LLM | Le moteur doit être prouvé d'abord | Oui |
| Attribution de plage IP, services mutualisés | L'offre de l'étape 1 n'en a pas besoin | Oui |
| Azure DevOps, dépôt dédié par stack | Une seule forge pour la preuve | Oui, pour la mission |
| Plusieurs environnements, pipeline de livraison applicative | Un seul environnement suffit à Cockpit | Oui |
| Demandes de modification automatisées, régénération | La stack est générée une fois | Seulement si ce choix est remis en cause |
| Génération de fondation, vending de subscriptions | Hors du cœur du produit | Non, sauf besoin avéré |

---

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation | Owner |
|------|--------|-------------|------------|-------|
| Le format de brique se révèle trop pauvre ou trop lourd à l'usage | Élevé | Moyenne | Le valider sur trois briques réelles avant de le figer ; schéma versionné | kjourdan1 |
| Le travail préalable (dépôt, state, identités) retarde la preuve | Moyen | Élevée | Le traiter en premier, hors `lzctl`, sur le modèle existant | kjourdan1 |
| Élargissement du périmètre avant la preuve Cockpit | Élevé | Élevée | La liste Out of Scope fait foi ; tout ajout passe par le journal des décisions | kjourdan1 |
| L'offre exige une ressource hors brique (droits entre briques, par exemple) | Moyen | Moyenne | Prévoir le raccordement dans l'offre ; à trancher en architecture | kjourdan1 |

---

## Traceability Matrix

| Requirement | Business Goal | Epic | User Story | Status |
|-------------|---------------|------|------------|--------|
| FR-001 | G1 | EPIC-002 | STORY-004 | À faire |
| FR-002 | G2 | EPIC-001 | STORY-001 | À faire |
| FR-003 | G2 | EPIC-001 | STORY-002, STORY-003 | À faire |
| FR-004 | G1 | EPIC-002 | STORY-005 | À faire |
| FR-005 | G1 | EPIC-002 | STORY-006 | À faire |
| FR-006 | G1 | EPIC-002 | STORY-007 | À faire |
| FR-007 | G1 | EPIC-003 | STORY-009 | À faire |
| FR-008 | G1 | EPIC-003 | STORY-010 | À faire |
| FR-009 | G1 | EPIC-004 | STORY-011, STORY-012, STORY-013 | À faire |
| FR-010 | G3 | EPIC-005 | STORY-014 | À faire |
| FR-011 | G1 | EPIC-002 | STORY-008 | À faire |
| FR-012 | G3 | EPIC-005 | STORY-015 | À faire |
| NFR-001 | G3 | (transverse) | STORY-014 | À faire |
| NFR-002 | G3 | (transverse) | — | À faire |
| NFR-003 | G2 | (transverse) | STORY-014 | À faire |
| NFR-004 | G1 | (transverse) | — | À faire |
| NFR-005 | G1 | (transverse) | — | À faire |

---

## Handoff

- **To Architecture:** trancher les formats (métadonnées de brique, offre, demande, contrat), la façon de raccorder les briques, la référence aux modules, le nom des commandes face à l'existant, et la liste du code conservé, gelé ou marqué expérimental. Le README du dépôt se corrige à ce moment-là.
- **To Sprint/Story Planning:** l'esquisse des epics ci-dessus sert de source.
- **Open questions / overflow:** voir `addendum.md`.

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 0.1 | 2026-11-14 | kjourdan1 | Première version |
