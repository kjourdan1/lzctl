# Journal des décisions

| Date | Action | Détail |
|---|---|---|
| 2026-11-13 | Brief créé | `product-brief-lzctl-2026-11-13.md` |
| 2026-11-13 | Direction | Platform as a service sur Azure ; les contrôles de pipeline deviennent du suivi, pas le produit |
| 2026-11-13 | Catalogue | Appartient à l'organisation, dans son dépôt ; `lzctl` orchestre |
| 2026-11-13 | Unité livrée | Une stack dans une subscription existante, un resource group par stack |
| 2026-11-13 | Fondation et vending | Hors du cœur ; étape amont pour les organisations qui possèdent leur tenant |
| 2026-11-13 | LLM et console | Secondaires ; LLM dans le tenant du client, n'écrit pas le Terraform |
| 2026-11-13 | Première offre | Application web interne légère, écrite dans le catalogue |
| 2026-11-13 | Ordre | `lzctl` d'abord ; Cockpit est la première demande réelle servie par `lzctl`, pas une stack écrite à la main au préalable |
| 2026-11-13 | Vie de la stack | Générée une fois, puis reprise à la main ; le contrat est une trace d'origine |
| 2026-11-13 | Catalogue | Briques composables ; rapport de couverture ; squelette de brique nouvelle |
| 2026-11-13 | Contexte | Découvert (Azure en lecture, Terraform existant), validé par PR avant de servir |
| 2026-11-13 | Priorité après l'étape 1 | Conception, attente entre équipes et flux : contexte en graphe, matrice des flux, plan de l'environnement, couverture, puis MCP |
| 2026-11-13 | Plan de l'environnement | Concept repris de Floorplan (carte à niveaux, demande en pointillé), format propre à l'infrastructure ; Mermaid d'abord |
| 2026-11-13 | Hors catalogue | Un LLM propose une architecture typée par MCP (hypothèses, questions, briques à créer) ; `lzctl` valide et dessine, ne génère pas de code |
| 2026-11-13 | Entrées et sortie | Demande, catalogue, contexte de la plateforme → une PR (stack, contrat, README, demande d'origine) |
