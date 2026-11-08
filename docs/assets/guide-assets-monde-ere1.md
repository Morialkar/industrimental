# Industrimental — Guide des assets du monde, ère 1 (Troc)

Mise à jour : 8 novembre 2026. Ce guide couvre les images du **monde** (stations, personnages, portraits, icônes de ressources, décor, particules). Les éléments d'**interface** (cadres, boutons, cases de capacités, icônes de la barre d'actions et des capacités) ont leur propre kit : `kit-interface-ere1.md`, dans un projet ChatGPT séparé.

Version de référence : le document vivant https://claude.ai/artifact/DUhvWC9YDmEq8KVtz4VKDu

## Méthode de travail

1. **Un projet ChatGPT par ère** (« Industrimental · Ère 1 ») avec `instructions-projet-chatgpt-ere1.txt` dans ses instructions.
2. **Commence par l'image de référence** (image 1 du lot de test). Une fois validée, joins-la à chaque demande avec la phrase « Match the style of the attached reference image exactly. »
3. **Un élément par image**, avec le gabarit de son type. Les prompts restent en anglais.
4. **Régénère plutôt que de corriger** si l'angle, la lumière ou les couleurs dérivent.
5. **Nomme le fichier** selon la convention ci-dessous.
6. **Livraison** : dépose les PNG dans le dépôt sous `assets/src/e01/` (ou envoie-les à Claude).

### Convention de nommage

`e<ère>_<type>_<sujet>[_<précision>].png`, en minuscules, mots séparés par des tirets. Le **slug** d'une station est son identifiant interne dans le jeu : il ne change pas quand son nom affiché change.

| Type | Exemple |
| --- | --- |
| Base de station (palier 1, 25 ou 100) | `e01_station_atelier-taille_n1.png` |
| Partie mobile | `e01_part_atelier-taille_bras.png` |
| Accessoire de spécialisation | `e01_prop_atelier-taille_parures.png` |
| Personnage de scène | `e01_char_tailleur.png` |
| Portrait de PNJ | `e01_npc_bob_neutre.png` |
| Icône de ressource | `e01_icon_parures.png` |
| Plan de décor | `e01_bg_plan2.png` |
| Texture de particule (commune) | `fx_fumee.png` |

## Bible de style (ère 1)

Le texte complet est dans `instructions-projet-chatgpt-ere1.txt`. Points clés :

- Low-poly semi-réaliste, facettes plates visibles (150 à 400 polygones), jamais lisse ni photoréaliste.
- Lumière douce venant d'en haut à gauche ; caméra de côté légèrement plongeante (environ 20°), quasi orthographique.
- Un seul sujet centré sur fond transparent ; pas de sol, d'ombre portée, de texte ni de contour.
- Palette de l'ère 1 : *earthy ochres, warm clay browns, sage green, bone white, with fire-orange accents*.

## Spécifications techniques

| Type | Format ChatGPT | Cadrage | Taille à l'écran |
| --- | --- | --- | --- |
| Base de station | Carré 1024 × 1024 | Bâtiment entier, centré, environ 80 % de la largeur, base vers le bas | Environ 300 px de large |
| Partie mobile | Carré 1024 × 1024 | Pièce seule, pivot au centre exact | Selon la station |
| Accessoire de spécialisation | Carré 1024 × 1024 | Objet seul, centré | Environ 100 px |
| Personnage de scène | Portrait 1024 × 1536 | Corps entier, de profil, pose neutre | Environ 60 px de haut |
| Portrait de PNJ | Carré 1024 × 1024 | Tête et épaules, trois-quarts vers la droite | Environ 200 px |
| Icône de ressource | Carré 1024 × 1024 | Objet seul, environ 80 % de l'image | 48 à 128 px |
| Plan de décor | Paysage 1536 × 1024 | Bande horizontale, bords raccordables | Panorama répété |
| Texture de particule | Carré 1024 × 1024 | Forme blanche et grise, centrée | 8 à 64 px |

- Pas d'ombre au sol ni de pièces mobiles dans les bases de stations : elles sont ajoutées en code.
- Si le fond transparent échoue : fond uni magenta (#FF00FF).

## Gabarits de prompt

### Base de station

```
Match the style of the attached reference image exactly.
Subject: {STATION}, a building from the Neolithic era, at stage {STAGE}: {STAGE_DESCRIPTION}.
Show the whole structure, centered, about 80% of the canvas width, its base near the bottom of the image.
Do NOT include these parts, they will be added separately: {MOVING_PARTS}.
No people. Transparent background, PNG, square 1024x1024.
```

Paliers : `n1` = rudimentaire, construit à la main ; `n25` = agrandi, mieux organisé ; `n100` = florissant, décoré, fier.

### Partie mobile

```
Match the style of the attached reference image exactly.
Subject: only the {PART} of the {STATION}, isolated, in a neutral resting pose.
Its pivot point ({PIVOT}) must sit at the exact center of the image.
Transparent background, PNG, square 1024x1024.
```

### Accessoire de spécialisation

```
Match the style of the attached reference image exactly.
Subject: {PROP}, a small add-on placed next to the {STATION} to show its specialization.
Centered, transparent background, PNG, square 1024x1024.
```

### Personnage de scène

```
Match the style of the attached reference image exactly.
Subject: a {ROLE} from the Neolithic era, full body, seen from the side, neutral standing pose, arms slightly away from the body.
Simple low-poly figure with a friendly look. Transparent background, PNG, portrait 1024x1536.
```

### Portrait de PNJ (3 expressions)

```
Match the style of the attached reference image exactly.
Subject: head-and-shoulders portrait of {NPC_NAME}, {NPC_DESCRIPTION}. Three-quarter view, facing right, friendly caricature, expressive face.
Expression: neutral. Transparent background, PNG, square 1024x1024.
```

Pour les deux autres expressions, dans la même conversation : « Same character, same framing, same lighting. Change only the expression to {joyful | panicked}. »

### Icône de ressource

```
Match the style of the attached reference image exactly.
Subject: {RESOURCE} as a game resource icon. One chunky, instantly readable object, slightly exaggerated proportions, filling about 80% of the canvas.
Transparent background, PNG, square 1024x1024.
```

### Plan de décor

```
Match the style of the attached reference image exactly.
Subject: layer {N} of a side-scrolling parallax background for the Neolithic era: {LAYER_DESCRIPTION}.
Wide horizontal strip; the left and right edges must match so the image can repeat seamlessly.
{TRANSPARENCY}. PNG, landscape 1536x1024.
```

Plans : `plan1` = ciel (opaque) ; `plan2` = lointain ; `plan3` = sol des stations ; `plan4` = premier plan décoratif.

### Texture de particule (une seule fois pour tout le jeu)

```
Subject: a single {SHAPE} particle sprite for a game effect, white and light grey only, soft low-poly facets, centered.
Transparent background, PNG, square 1024x1024.
```

Formes : bouffée de fumée, étincelle, feuille, grain de poussière, bulle, éclat de pierre, goutte, flamme.

## Lot de test

Les 10 prompts prêts à coller sont dans `prompts-lot-test-ere1.md` (atelier de taille, Foyer du clan, icônes parures et silex, portrait de Bob, décor plan 2, fumée).

## Liste complète de l'ère 1

L'ère 1 demande environ 120 images du monde. Chaque station a 3 bases (`n1`, `n25`, `n100`), ses parties mobiles et un accessoire par spécialisation.

### Stations (environ 75 images)

| Station (nom validé) | Slug du fichier | Parties mobiles (pivot) | Accessoires de spécialisation |
| --- | --- | --- | --- |
| Coin de cueillette | `cueillette` | Branche qui se balance (base de la branche) | Baies sauvages : panier de baies · Herbes du clan : botte d'herbes · Herbes à tresser : rouleau d'herbes tressées |
| Carrière de silex | `carriere-silex` | Pic en bois de cerf (manche) | Silex de taille : tas de silex fin · Pierres à moudre : meules · Galets de parure : galets polis |
| Camp de chasse | `camp-chasse` | Peau qui sèche au vent (haut du cadre) | Chasse au gros gibier : trophée · Pêche : filet et poissons · Chasse au piège : collets |
| Atelier de taille | `atelier-taille` | Bras du tailleur (coude) | Pointes de chasse : râtelier de pointes · Outils agricoles : faucilles et houes · Parures d'échange : présentoir de parures |
| Foyer du clan | `feu-camp` | Broche tournante (axe) | Fumoir : claie de fumage · Foyer à ocre : four à ocre · Assemblée du soir : cercle de bûches |
| Atelier des peaux | `tannerie` | Peau tendue qui oscille (haut du cadre) | Cuir épais : pile de cuirs · Fourrures de prestige : fourrures suspendues · Outres : outres gonflées |
| Premiers champs | `champ` | Bande d'épis qui ondule (pied des tiges) | Blé sauvage : gerbe de blé · Lentilles : paniers de lentilles · Lin : bottes de lin |
| Terres salées | `source-salee` | Seau et corde (point d'attache) | Évaporation : bassins d'évaporation · Mine de sel : entrée de mine · Sel marin : sel et coquillages |
| Argilière | `fosse-argile` | Mains qui pétrissent (poignets) | Jarres d'argile : jarres · Figurines d'argile : figurines · Tuiles d'argile : piles de tuiles |
| Place du troc | `cercle-troc` | Totem à rubans (pied du mât) | Pierre du troc : pierre gravée · Foire saisonnière : étals de foire · Conseil des anciens : sièges des anciens |

### Personnages, PNJ, icônes, décor et particules

| Groupe | Contenu | Images |
| --- | --- | --- |
| Personnages de scène | Cueilleuse, tailleur, chasseur, tanneuse, potier, porteur, enfant, ancien, marchand, berger | 10 |
| Portraits de PNJ | Bob, Amna, Nick, L'Ancienne × neutre / joyeux / paniqué | 12 |
| Icônes de ressources | Coquillages, baies, silex, bois, viande, peaux brutes, argile, outils en silex, peaux tannées, grain, sel, parures, ocre, poteries crues, peintures rupestres | 15 |
| Décor | Plans 1 à 4 + une tuile de sol | 5 |
| Particules (communes à tout le jeu) | Fumée, étincelle, feuille, poussière, bulle, éclat, goutte, flamme | 8 |

Noms de fichiers des icônes de ressources : `e01_icon_coquillages`, `e01_icon_baies`, `e01_icon_silex`, `e01_icon_bois`, `e01_icon_viande`, `e01_icon_peaux-brutes`, `e01_icon_argile`, `e01_icon_outils-silex`, `e01_icon_peaux-tannees`, `e01_icon_grain`, `e01_icon_sel`, `e01_icon_parures`, `e01_icon_ocre`, `e01_icon_poteries-crues`, `e01_icon_fresques` (peintures rupestres ; le slug reste `fresques`).

## Contrôle qualité

- [ ] Fond transparent (ou magenta uni), aucun sol ni ombre portée.
- [ ] Même angle de caméra que la référence (légèrement en plongée, vue de côté).
- [ ] Lumière venant d'en haut à gauche.
- [ ] Facettes visibles, pas de rendu lisse ni photoréaliste.
- [ ] Couleurs dans la palette de l'ère.
- [ ] Aucun texte, logo, cadre ou contour noir.
- [ ] Base de station : aucune pièce mobile ni personnage.
- [ ] Partie mobile : le pivot est bien au centre.
- [ ] Lisible en petit : réduite à environ 10 %, la silhouette se reconnaît encore.
