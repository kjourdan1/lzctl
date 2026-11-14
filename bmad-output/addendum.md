# Addendum au PRD de l'étape 1

Ce qui alourdirait le PRD : questions ouvertes pour l'architecture, et ce qui est reporté.

## Questions ouvertes pour l'architecture

| # | Question | Pourquoi elle compte |
|---|---|---|
| 1 | Nom et format du fichier de métadonnées d'une brique | Engage tout le catalogue ; doit pouvoir porter plus tard les flux et la description pour le plan de l'environnement |
| 2 | Format d'une offre : comment exprimer le raccordement d'une sortie de brique à l'entrée d'une autre | Détermine si une offre reste déclarative ou devient du code |
| 3 | Où vivent les ressources « entre briques » (droits d'une identité sur un coffre, par exemple) | Le PRD interdit toute ressource hors brique, hormis le resource group |
| 4 | Référence aux briques depuis une stack : chemin relatif ou référence Git | Chemin relatif dans cette étape ; les tags viendront avec un catalogue dans son propre dépôt |
| 5 | Nom du contrat dans la stack, et son rapport au fichier de demande | Le contrat est une trace d'origine, pas une source de régénération |
| 6 | Nom des commandes, face à `workload add` et `add-blueprint` existants | Éviter deux vocabulaires dans la même CLI |
| 7 | Langue des fichiers générés (README, texte de PR) | Le catalogue est celui de l'organisation ; la langue pourrait en dépendre |
| 8 | Ce qui est conservé, gelé ou marqué expérimental dans le code existant | Décidé à l'architecture, avec la correction du README |
| 9 | Version du catalogue citée dans le contrat et la PR, tant qu'il n'est pas versionné par tags | Le commit du dépôt est un candidat, sans casser le déterminisme |

## Reporté après l'étape 1, dans l'ordre retenu

1. Découverte du contexte (Azure en lecture, Terraform existant) et graphe validé par PR
2. Matrice des flux et plan de l'environnement, d'abord en Mermaid
3. Rapport de couverture
4. Interface MCP et proposition d'architecture hors catalogue
5. Plan interactif de l'environnement

Puis, sans ordre fixé : squelette de brique nouvelle, seconde offre (projet IA rattaché au
compte Foundry mutualisé), espace sur AKS mutualisé, environnements multiples et pipeline de
livraison applicative, attribution de plage IP, services mutualisés, Azure DevOps, console.

## Prérequis hors `lzctl`

Le dépôt d'infrastructure du tenant professionnel se prépare à la main, avant la preuve :
state distant sans clé d'accès, identités de pipeline fédérées (une pour le plan, une pour
l'apply), pipeline de PR avec plan commenté, environnement protégé pour l'apply. Le dépôt
`azure-platform` en fournit le modèle.
