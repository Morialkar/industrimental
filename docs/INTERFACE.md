# Kit d'interface : essai de la phase 1

Recommandation : **garder le kit maison** (`internal/ui`). `ebitenui` fonctionne, mais tout ce qui fait la sensation au doigt reste à écrire de toute façon.

L'essai a porté sur la même liste de stations et la même fiche à 3 variantes, faites avec chacun des deux (`ebitenui` v0.7.3, prototype non conservé).

| Critère | Kit maison | `ebitenui` |
| --- | --- | --- |
| Défilement au doigt | Oui | Non : molette ou barre seulement, à écrire soi-même |
| Inertie de liste | Oui | Non, à écrire soi-même |
| Glisser sans déclencher le bouton sous le doigt | Oui, testé | Non prévu : le clic part si le doigt finit sur le bouton |
| Appui long | Oui, testé | Oui, événement de widget |
| Fiche coulissante animée | Oui | Non : fenêtre modale sans animation |
| Mise en page automatique | **Non** : coordonnées à la main | **Oui** : lignes, grilles, ancres, retours à la ligne |
| Canevas zoomable pour l'arbre | À écrire | À écrire |
| Volume | 390 lignes, sans dépendance | 14 000 lignes de widgets et 168 lignes pour l'essai |
| Jouer l'interface depuis un test | Oui (`internal/game/play_test.go`) | Pas de point d'entrée équivalent |

Le vrai coût du kit maison est la mise en page : `internal/game/panel.go` place tout à la main. La phase 2 doit y ajouter un conteneur ligne/colonne avant que l'interface finale grossisse.

Limite de l'essai : seul le kit maison a été lancé sur un téléphone. Le comportement d'`ebitenui` au doigt vient de la lecture de son code et d'un lancement sur desktop.
