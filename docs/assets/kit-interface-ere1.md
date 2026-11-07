# Industrimental — Kit d'interface de l'ère 1 (bois)

À utiliser dans un **projet ChatGPT séparé** (« Industrimental · Interface ») dont les instructions contiennent `instructions-projet-chatgpt-ui.txt`. Ne mélange pas avec le projet des images du monde : les règles de cadrage sont différentes.

## Comment ça marche

L'interface est assemblée à partir de ces images : le code ne dessine plus de « fausse matière ». La plupart des cadres sont en **9 morceaux** : les 4 coins restent intacts, les bords s'étirent, le centre s'étire ou se répète. C'est ce qui permet d'utiliser un seul cadre pour une petite fiche comme pour un grand panneau.

```
┌────┬──────────────┬────┐
│coin│ bord (étiré) │coin│
├────┼──────────────┼────┤
│bord│ centre       │bord│
│    │ (étiré/répété)│    │
├────┼──────────────┼────┤
│coin│ bord (étiré) │coin│
└────┴──────────────┴────┘
```

Règles pour tous les cadres :

- Bordure de **même épaisseur sur les 4 côtés** (environ 1/9 de l'image).
- Ornements **uniquement dans les coins**, en miroir.
- Bords droits **simples et réguliers** entre les coins.
- Centre **uni**, sans motif ni ornement.

Je m'occupe du détourage, du découpage et des mesures. Si un cadre ne respecte pas ces règles, régénère-le plutôt que de le corriger.

## Ordre de production

1. Génère d'abord **UI-01 (panneau principal)** : il devient la référence de matière. Joins-le à toutes les demandes suivantes avec « Match the style of the attached reference image exactly. »
2. Génère ensuite le reste du lot de base (UI-02 à UI-14) : c'est ce qu'il me faut pour refaire la maquette.
3. Les icônes de capacités (UI-20 et suivantes) viennent après, avec la référence UI-08.

Dépose les PNG dans le dépôt sous `assets/src/ui/e01/` (ou envoie-les-moi ici) avec le nom indiqué.

## Lot de base (14 images)

| N° | Fichier | Élément | Format |
| --- | --- | --- | --- |
| UI-01 | `ui_e01_panneau.png` | Panneau principal (9 morceaux) | 1024 × 1024 |
| UI-02 | `ui_e01_encart.png` | Encart creusé (9 morceaux) | 1024 × 1024 |
| UI-03 | `ui_e01_bouton.png` | Bouton planche, normal | 1536 × 1024 |
| UI-04 | `ui_e01_bouton_presse.png` | Bouton planche, enfoncé | 1536 × 1024 |
| UI-05 | `ui_e01_bouton_rond.png` | Gros bouton rond (Récolter) | 1024 × 1024 |
| UI-06 | `ui_e01_onglet.png` | Onglet, normal | 1536 × 1024 |
| UI-07 | `ui_e01_onglet_actif.png` | Onglet, actif | 1536 × 1024 |
| UI-08 | `ui_e01_case.png` | Case de capacité | 1024 × 1024 |
| UI-09 | `ui_e01_case_tournant.png` | Case du Tournant historique | 1024 × 1024 |
| UI-10 | `ui_e01_plaque_rang.png` | Plaquette de rang (« 2/5 ») | 1536 × 1024 |
| UI-11 | `ui_e01_barre.png` | Cadre de barre de progression | 1536 × 1024 |
| UI-12 | `ui_e01_separateur.png` | Séparateur orné | 1536 × 1024 |
| UI-13 | `ui_e01_fond_planche.png` | Fond de la planche des capacités (répétable) | 1024 × 1024 |
| UI-14 | `ui_e01_bandeau.png` | Bandeau des ressources (3 morceaux) | 1536 × 1024 |

Plus les 5 icônes de la barre d'actions (UI-15 à UI-19).

### UI-01 — Panneau principal (référence)

```
Subject: a game UI panel frame in carved honey-brown wood, for menus and info sheets.
Ornate corner pieces: each corner is a carved wooden knot lashed with rope, with a small inlaid polished river stone and a dab of red ochre. The four corners mirror each other.
The border has the same thickness on all four sides (about one ninth of the image). Between the corners, the straight edges are plain bevelled wood, uniform so they can be stretched.
The center is an even, plain wood board with soft grain and no ornament.
Flat front view, fills the whole square image edge to edge. Transparent background outside the frame. PNG, square 1024x1024.
```

### UI-02 — Encart creusé

```
Match the style of the attached reference image exactly.
Subject: a recessed inset panel, as if carved into the wood of the main panel: a sunken, slightly darker wood surface with a thin bevelled rim.
Small, discreet corner details only; straight edges plain and uniform; same rim thickness on all four sides; plain center with no ornament.
Flat front view, fills the image edge to edge. Transparent background. PNG, square 1024x1024.
```

### UI-03 — Bouton planche (normal)

```
Match the style of the attached reference image exactly.
Subject: a wide wooden plank button for a game UI, raised and inviting to press: bevelled edges, a fire-orange ochre-painted face with visible wood grain, small rope lashings at both short ends.
The button is a horizontal rectangle about three times wider than tall, centered, filling about 90% of the image width. The long edges are plain and uniform so the button can be stretched; ornaments only at the two ends.
No text. Flat front view. Transparent background. PNG, landscape 1536x1024.
```

### UI-04 — Bouton planche (enfoncé)

```
Match the style of the attached reference image exactly.
Subject: the exact same wooden plank button, in its pressed state: pushed in, face slightly darker, highlights reduced, inner shadow along the top edge.
Same size, same position and same outline as the normal button so the two images line up perfectly.
No text. Flat front view. Transparent background. PNG, landscape 1536x1024.
```

### UI-05 — Gros bouton rond (Récolter)

```
Match the style of the attached reference image exactly.
Subject: a large round button like a carved wooden shield or a cross-section of a log, with a rim of small polished stones and rope wrapped around it, a slightly domed center for an icon to sit on.
Circular, centered, filling about 90% of the image. No text, no icon in the center.
Flat front view. Transparent background. PNG, square 1024x1024.
```

### UI-06 — Onglet (normal)

```
Match the style of the attached reference image exactly.
Subject: a small wooden tab for switching pages in a game UI, shaped like a short plank with a rounded top edge and a carved notch at each end, muted wood color, not highlighted.
Horizontal, about three times wider than tall, centered, filling about 90% of the image width; the long edges plain and uniform so it can be stretched.
No text. Flat front view. Transparent background. PNG, landscape 1536x1024.
```

### UI-07 — Onglet (actif)

```
Match the style of the attached reference image exactly.
Subject: the exact same wooden tab in its active state: brighter wood, a thin fire-orange ochre inlay along the top edge and a soft warm glow.
Same size, same position and same outline as the normal tab so the two images line up perfectly.
No text. Flat front view. Transparent background. PNG, landscape 1536x1024.
```

### UI-08 — Case de capacité

```
Match the style of the attached reference image exactly.
Subject: a square socket that holds a skill icon in a skill tree, like the icon slots of Diablo II but in carved wood: a thick bevelled square frame with small rope lashings at the corners and a recessed, empty, slightly darker square opening in the middle.
The opening is empty (no icon inside) and takes about 70% of the frame. Centered, filling about 90% of the image.
Flat front view. Transparent background. PNG, square 1024x1024.
```

### UI-09 — Case du Tournant historique

```
Match the style of the attached reference image exactly.
Subject: a special, more prestigious version of the skill socket, for the final node of a branch: the same carved wooden square frame, larger corner ornaments, small inlaid amber and polished stones, a warm golden glow around the rim, and an empty recessed opening in the middle.
Centered, filling about 90% of the image. No icon inside.
Flat front view. Transparent background. PNG, square 1024x1024.
```

### UI-10 — Plaquette de rang

```
Match the style of the attached reference image exactly.
Subject: a tiny wooden plaque used to display a skill rank like "2/5" in a game UI: a small dark wooden rectangle with a bevelled light rim, two tiny pegs at the ends.
Horizontal, about two and a half times wider than tall, centered, filling about 60% of the image width. Blank surface: no text or numbers.
Flat front view. Transparent background. PNG, landscape 1536x1024.
```

### UI-11 — Cadre de barre de progression

```
Match the style of the attached reference image exactly.
Subject: an empty progress bar frame in carved wood: a long horizontal groove with a bevelled rim, small rope-lashed caps at both ends, and an empty dark channel inside where the fill will go.
Very elongated horizontal shape (about eight times wider than tall), centered, filling about 95% of the image width; the long edges plain and uniform so it can be stretched.
Flat front view. Transparent background. PNG, landscape 1536x1024.
```

### UI-12 — Séparateur orné

```
Match the style of the attached reference image exactly.
Subject: a thin horizontal ornamental divider for a game UI: a carved wooden rod with a small centerpiece (a polished stone bead wrapped in rope) and tapered ends.
Very thin and elongated, centered, filling about 90% of the image width.
Flat front view. Transparent background. PNG, landscape 1536x1024.
```

### UI-13 — Fond de la planche des capacités

```
Match the style of the attached reference image exactly.
Subject: a seamless tileable background texture for a skill tree board: a slightly darker warm wood surface with faint carved guide lines and subtle charcoal and ochre stains, calm enough for icons to stand out on top.
No frame, no border, no ornament, no vignette. The texture must repeat seamlessly on all four edges.
Flat front view, fills the whole image. PNG, square 1024x1024.
```

### UI-14 — Bandeau des ressources

```
Match the style of the attached reference image exactly.
Subject: a long horizontal wooden bar for the top of a game screen, where resource counters will be shown: a carved plank with ornate rope-lashed ends and a plain, uniform middle section that can be stretched.
Very elongated horizontal shape (about six times wider than tall), centered, filling about 95% of the image width. No text, no icons.
Flat front view. Transparent background. PNG, landscape 1536x1024.
```

### UI-15 à UI-19 — Icônes de la barre d'actions

Un prompt par icône, avec ce gabarit :

```
Match the style of the attached reference image exactly.
Subject: a painted game icon representing {ICON}, in the warm low-poly style of the game, with a thin carved wooden rim, readable at 48 pixels.
Centered, filling about 80% of the image. No text.
Flat front view. Transparent background. PNG, square 1024x1024.
```

| N° | Fichier | {ICON} |
| --- | --- | --- |
| UI-15 | `ui_e01_ic_capacites.png` | a small carved wooden tree with glowing branch nodes (skills) |
| UI-16 | `ui_e01_ic_marche.png` | a primitive balance scale made of a stick and two woven baskets, one holding shells |
| UI-17 | `ui_e01_ic_quetes.png` | a rolled hide scroll tied with a cord, with a small ochre mark |
| UI-18 | `ui_e01_ic_boutique.png` | an open woven pouch spilling a few glowing sparks and shell beads |
| UI-19 | `ui_e01_ic_clan.png` | three small figures around a campfire, seen as simple low-poly silhouettes |

## Icônes de capacités (après le lot de base)

Une icône peinte par nœud de l'arbre, placée dans la case UI-08. On commence par les 10 nœuds dont les noms sont stables (tronc commun et Tournants historiques). Les autres suivront quand leurs noms seront validés.

```
Match the style of the attached reference image exactly.
Subject: a painted skill icon for "{NOM}": {DESCRIPTION}. Warm low-poly painted style, strong silhouette, readable at 48 pixels, with a soft dark vignette at the edges so it sits well inside a recessed socket.
Square composition filling the whole image, no frame, no text.
PNG, square 1024x1024.
```

| N° | Fichier | {NOM} | {DESCRIPTION} |
| --- | --- | --- | --- |
| UI-20 | `ui_e01_cap_main-habile.png` | Main habile | a skilled hand catching a falling berry mid-air |
| UI-21 | `ui_e01_cap_feu.png` | Le feu apprivoisé | a small fire held safely in a ring of stones, sparks rising |
| UI-22 | `ui_e01_cap_langage.png` | Langage du troc | two hands exchanging a shell for a flint, with simple gesture lines |
| UI-23 | `ui_e01_cap_sedentarisation.png` | Sédentarisation | a first hut planted firmly in the ground next to a small field |
| UI-24 | `ui_e01_cap_paniers.png` | Premiers paniers | a woven basket overflowing with berries and grain |
| UI-25 | `ui_e01_cap_cercle.png` | Le Cercle | a ring of standing stones with four paths leaving it |
| UI-26 | `ui_e01_cap_pacte.png` | Le Pacte des Marchands | two hands shaking above a pile of shells, with a goat looking on |
| UI-27 | `ui_e01_cap_revolution.png` | La Révolution des Champs | golden wheat field with a flint sickle and a full granary |
| UI-28 | `ui_e01_cap_maitrise-feu.png` | Maîtrise du feu | a glowing kiln with a fired clay pot and a bowl of ochre |
| UI-29 | `ui_e01_cap_conteurs.png` | Le Cercle des conteurs | a storyteller by the fire, a cave painting of a giant mammoth behind |

## Contrôle qualité

- [ ] Fond transparent (ou magenta uni), rien derrière l'élément.
- [ ] Vue parfaitement de face, sans perspective.
- [ ] Aucun texte, chiffre ni lettre.
- [ ] Pour un cadre : bordure d'épaisseur égale, ornements seulement dans les coins, bords et centre unis.
- [ ] Pour une paire normal / actif ou normal / enfoncé : même taille et même contour.
- [ ] Chaleureux et lumineux : aucun motif de dark fantasy.
- [ ] Lisible en petit : réduite à 10 %, la forme se reconnaît encore.
