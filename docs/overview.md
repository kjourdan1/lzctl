# lzctl en bref

`lzctl` est une CLI qui **standardise les stacks d'un dépôt IaC Azure et rend chaque
déploiement prévisible**. Cadrage détaillé : [product.md](product.md). Ces deux documents
sont à réaligner sur le brief
[`bmad-output/product-brief-lzctl-2026-11-13.md`](../bmad-output/product-brief-lzctl-2026-11-13.md),
qui fait foi pour la direction.

> Les briques marquées *(à construire)* sont décidées dans le cadrage mais pas encore
> développées.

## Objectif

Une **stack** est une ressource et ses à-côtés, déployés ensemble : un compte de stockage, ses
conteneurs, ses private endpoints. Dans la plupart des dépôts IaC :

- un déploiement peut écraser un changement fait dans le portail sans que le plan le montre ;
- les modules maison ne bougent plus, faute de voir ce qu'une nouvelle version casserait ;
- chaque stack est écrite à la main, et ses flux réseau se demandent à part.

`lzctl` répond par un **contrat de stack** : un fichier qui décrit la stack (type,
propriétaire, paramètres, flux réseau). Le contrat sert à la créer, à contrôler chaque PR, à
la surveiller chaque nuit et à la faire évoluer.

## Règles de fonctionnement

- Un humain ou un agent propose, un humain valide ; **seule une pipeline approuvée déploie**.
- Le code produit est du **Terraform standard**, qui fonctionne sans `lzctl`.
- **Lecture seule** pour tous les contrôles ; l'écriture passe toujours par une PR.
- **Déterministe** : `lzctl` ne contient pas de LLM. Un LLM peut l'appeler, plus tard.
- Un contrôle qui n'a pas pu lire le dit ; il ne passe jamais en silence.

## Ce que contient l'outil

| Famille | Commandes | Rôle |
|---|---|---|
| **Ne rien écraser** | `check`, `drift` étendu *(à construire)* | Écritures hors pipeline sur les ressources du plan, y compris ce que Terraform ne relit pas |
| **Faire évoluer les modules** | `module consumers`, `module diff`, `module impact` *(à construire)* | Qui consomme quoi, ce qu'une version change, ce qu'elle provoque chez chacun |
| **Créer une stack** | `stack new` *(à construire)* | Contrat, code et entrée de pipeline depuis le catalogue ; flux réseau rattachés à la PR |
| **Tenir le contrat** | `check` *(à construire)*, `validate` | Le plan reste dans le périmètre de la stack ; changements sensibles en tête |
| **Déployer** | `plan`, `apply`, pipelines générées | Validation, plan, garde des destructions, apply après approbation |
| **Automatiser la demande** | `discover`, `propose` *(à construire)* | De la demande à la PR ; base de la console |

## Démarrer sur un dépôt existant

Rien n'est déployé, rien n'est généré ; une identité en lecture seule suffit.

| Étape | Action | Résultat |
|---|---|---|
| 1. Contrôler une PR | `lzctl check --plan tfplan.json` *(à construire)* dans la pipeline | Les ressources du plan modifiées hors pipeline, avec l'identité et la date |
| 2. Surveiller | Relevé nocturne *(à construire)* | Une écriture hors pipeline est connue le lendemain |
| 3. Voir les modules | `lzctl module consumers` *(à construire)* | Qui utilise quel module, à quelle version |
| 4. Publier un module | `lzctl module impact` *(à construire)* dans la PR du module | Effet de la nouvelle version chez chaque consommateur |
| 5. Standardiser | `lzctl stack new` *(à construire)* pour les nouvelles stacks | Chaque stack naît avec son contrat |

## Le workflow visé

1. Un besoin est exprimé : un fichier de demande, plus tard une console en langage naturel.
2. `lzctl` produit le contrat, le code et la PR, avec le plan, les constats et les flux demandés.
3. Des humains valident.
4. La pipeline déploie, après un dernier contrôle.
5. Le relevé nocturne et l'analyse d'impact des modules prennent le relais.

`lzctl` est un outil de ce workflow, pas le workflow entier. La console et le LLM viennent en
dernier ; le LLM s'exécute dans le tenant du client et n'écrit pas le Terraform.

## État actuel

| Existe | Existe, expérimental | À construire |
|---|---|---|
| CLI, schéma `lzctl.yaml`, `validate`, pipelines GitHub Actions et Azure DevOps, orchestration `plan` et `apply`, `upgrade` | Génération de la fondation (management groups et connectivité ne passent pas `terraform init`), blueprints, `audit`, cycle de vie des policies, `rollback` | `check`, relevé nocturne, `module *`, contrat de stack, `stack new`, `discover`, `propose`, console |

## Ordre de travail

1. Dire vrai sur l'existant : CI qui valide la sortie générée.
2. Garde hors pipeline.
3. Modules : consommateurs, différence d'interface, impact.
4. Contrat de stack et `stack new`, pour un seul type.
5. Conformité au contrat.
6. `discover` minimal et `propose`.
7. Schéma de demande, interface pour agents, console.

Le détail et les raisons sont dans [product.md](product.md), § 11.
