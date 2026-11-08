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

## Intégration de la maquette (8 novembre)

L'interface du jeu suit maintenant la [maquette](assets/maquette-interface.html) et son kit d'interface de l'ère 1 :

- Le monde occupe tout l'écran. Par-dessus : le nom de l'ère (qui ouvre la frise), le bandeau des ressources, la quête épinglée, le bouton Récolter et la barre de 5 actions (Capacités, Marché, Quêtes, Boutique, Clan). Toucher une station ouvre sa fiche.
- Les cadres, boutons, onglets, cases, barres et icônes viennent du kit (`internal/assets/ui/e01`, les mêmes WebP que la maquette). Le kit les dessine en 9 morceaux (`ui.Frame`) avec les découpes de la maquette.
- Les capacités reprennent la grille de la maquette : tronc commun fixe, une planche de 3 × 6 par branche dans un carrousel, rangées ouvertes selon le nombre de bâtiments construits (`rowStations`), rangs payés en ressources de l'ère.
- Le défilement se verrouille sur un axe : un glissement de côté fait tourner le carrousel sans faire défiler la liste.

Pour garder l'équilibrage, les rangs se partagent le coût et l'effet de l'ancien nœud : acheter tous les rangs revient au même qu'avant (voir `content.Node`). La récolte au toucher vaut 5 % de la production par seconde, +5 points par rang de Main habile.

Reste provisoire : les polices (Go au lieu d'Alegreya), les icônes de ressources (pentagones de couleur), le décor du monde et les stations (formes géométriques), et les nœuds sans illustration (glyphes).
