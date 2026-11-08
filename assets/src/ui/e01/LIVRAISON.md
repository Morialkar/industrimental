# Industrimental — UI Kit Ère 1 (livraison finale)

**29 / 29 PNG officiels validés**, dans `assets/src/ui/e01/`.

- Les 29 fichiers reprennent les noms et dimensions du guide de production.
- Le bonus `bonus/ui_e01_cap_premier_tisserand_bonus.png` est conservé séparément, hors du lot officiel.
- `note.md` (à la racine) explique le statut du bonus et sa relation au GDD.
- `validated_manifest.json` (à la racine) donne les métadonnées et SHA-256 de chaque PNG.
- `controle_qualite.md` détaille les vérifications techniques effectuées et les tests restant à faire dans le moteur.

**Cible d’intégration :** `assets/src/ui/e01/` dans le projet de jeu.

Les assets de capacité UI-20 à UI-29 se placent dans le cadre UI-08 sans fusionner leurs fichiers sources avec ce dernier. Les assets de fond peuvent être opaques, contrairement aux éléments d'interface détourés.

# Industrimental — notes du UI Kit Ère 1

## Asset bonus : Le Premier Tisserand

- **Fichier :** `bonus/ui_e01_cap_premier_tisserand_bonus.png`
- **Statut :** illustration bonus / concept graphique exploratoire, **hors du lot officiel de 29 assets** de l’Ère 1.
- **Description :** métier à tisser en bois, cordages et tissu rouge et écru ; format carré 1024 × 1024, fond sombre.
- **Référence dans le jeu :** le document *Industrimental — Game Design Document* mentionne une station **« Tisserand » à l’ère 3 — Guildes** (liste des stations de cette ère). Il cite également le textile parmi les ressources de l’ère 5 — Manufactures.
- **Ce qui n’est pas confirmé :** aucune capacité officiellement nommée **« Le Premier Tisserand »** n’est définie dans le GDD ou le guide de l’interface Ère 1 consultés. Cette icône ne confirme ni une fonctionnalité implémentée, ni son apparition en ère 1.
- **Proposition :** réserver l’illustration à une exploration de l’interface de la station Tisserand à l’ère 3 ou à un éventuel nœud futur, sous réserve de validation du game design.

## État du ZIP final

- **29/29 assets officiels validés**, rangés dans `assets/src/ui/e01/`.
- **UI-28 — Maîtrise du feu** (`ui_e01_cap_maitrise-feu.png`) : validé et inclus.
- **UI-29 — Le Cercle des conteurs** (`ui_e01_cap_conteurs.png`) : validé définitivement et inclus.
- **Le Premier Tisserand** reste un bonus exploratoire dans `bonus/`, **non comptabilisé** dans les 29 PNG officiels.
- `validated_manifest.json` recense les 29 PNG officiels et le bonus avec leurs empreintes SHA-256 et chemins de livraison.
- `controle_qualite.md` résume les tests de conformité technique et les limites de validation moteur.

*Ce fichier note.md se trouve à la racine de l'archive finale.*

# Contrôle qualité — Industrimental UI Kit Ère 1

## Vérifications réalisées

- **29 fichiers PNG officiels** présents et conformes aux 29 noms du manifeste de production.
- **1 PNG bonus** présent dans `bonus/` : Le Premier Tisserand.
- **29/29 tailles de canvas conformes** aux dimensions du guide (1024×1024 ou 1536×1024 selon l'asset).
- Tous les PNG, bonus inclus, ont été **ouverts, vérifiés puis décodés intégralement** sans erreur.
- **UI-21 — Le feu apprivoisé** : une copie tronquée du dossier de travail a été remplacée par le PNG source intact déjà approuvé.
- Les empreintes **SHA-256** sont fournies par le manifeste pour chaque fichier image.
- ZIP testé après création par contrôle des **CRC** et comparaison des SHA-256 à l'intérieur de l'archive.
- `note.md` présent **à la racine**, contenant le statut du bonus et les références Game Design Document.

## Remarques d'intégration

- Les images qui couvrent toute la surface (textures de fond et vignettes de compétences) peuvent être **opaques** ; les composants isolés ont une transparence selon leur rôle.
- Les aperçus d'icônes à 48 px et d'insertion dans UI-08 ont été examinés pendant la production. La validation finale de la lisibilité et de l'alignement doit être faite **dans le moteur et à l'échelle d'utilisation réelle**.
- Les éléments extensibles (9-slice / 3-slice) n'ont pas été testés dans le moteur de jeu ; seule leur présence et leurs dimensions ont été contrôlées pour cette livraison.
- Les arts de compétences UI-20 à UI-29 doivent être placés **à l'intérieur** de UI-08, non fusionnés avec son cadre.

## Résultat

**Livraison techniquement conforme : 29 assets officiels + 1 bonus documenté.**
