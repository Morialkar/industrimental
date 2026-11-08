# Compiler et lancer Industrimental

## Prérequis

- Go (la version exacte est dans `go.mod` ; Go la télécharge automatiquement si besoin).
- **Linux** : `sudo apt-get install libgl1-mesa-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev pkg-config`
- **macOS** : Xcode et ses outils en ligne de commande.

## Desktop (développement)

```sh
go run ./cmd/desktop
```

La fenêtre a le format d'un téléphone. Glisse à la souris pour faire défiler le monde. La sauvegarde est dans le dossier de configuration de l'utilisateur, sous `Industrimental`.

Capture d'écran sans fenêtre (utilisée par l'intégration continue) :

```sh
go run ./cmd/desktop -screenshot scene.png -frames 20 -warmup 4
```

Pour voir un écran sur une partie avancée sans toucher à la sauvegarde : `-demo -tab 2` (0 monde, 1 marché, 2 capacités, 3 quêtes, 4 frise, 5 clan, 6 boutique, 7 fiche de station).

## Équilibrage

```sh
go run ./cmd/balance
```

Joue l'ère une fois par fin, sur 5 sessions de 5 minutes par jour, et affiche le temps de chacune. `-v` détaille les étapes et l'état final, `-curves dossier` écrit les courbes en CSV. `go test ./internal/balance` échoue si une fin de l'ère 1 sort de la plage de 24 à 48 h.

## Tests

```sh
go test ./...
```

Les données de contenu sont validées au démarrage et par `go test` : références de ressources, arbre sans cycle, une Clé d'ère par axe, textes présents en français et en anglais.

## Android

L'intégration continue produit un APK de debug à chaque push (artefact `industrimental-debug-apk` de l'onglet Actions sur GitHub). Pour le construire localement :

1. Installer Android Studio (SDK + NDK) et Java 17.
2. Lier le jeu Go en bibliothèque Android :

   ```sh
   go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@v2.10.5
   ebitenmobile bind -target android -androidapi 24 \
     -javapkg com.morialkar.industrimental \
     -o android/app/libs/industrimental.aar ./mobile
   ```

3. Ouvrir le dossier `android/` dans Android Studio et lancer l'app sur le téléphone, ou en ligne de commande : `gradle -p android installDebug`.

Téléphones de référence : Xiaomi Mi A2 (entrée de gamme), Xiaomi Mi A3 (milieu de gamme, cible 60 fps), Galaxy Tab FE (tablette).

## iOS (sur Mac)

1. Lier le jeu Go en framework iOS :

   ```sh
   go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@v2.10.5
   ebitenmobile bind -target ios -o ios/Industrimental.xcframework ./mobile
   ```

   Le framework est ignoré par git : refaire cette étape après un clone et à chaque changement du code Go.

2. Ouvrir `ios/Industrimental.xcodeproj` dans Xcode, choisir son équipe dans *Signing & Capabilities* et lancer sur un iPhone branché. En ligne de commande, depuis `ios/` :

   ```sh
   xcodebuild -project Industrimental.xcodeproj -scheme Industrimental \
     -destination 'id=<UDID>' -derivedDataPath build \
     DEVELOPMENT_TEAM=<ÉQUIPE> -allowProvisioningUpdates build
   xcrun devicectl device install app --device <UDID> \
     build/Build/Products/Debug-iphoneos/Industrimental.app
   xcrun devicectl device process launch --device <UDID> com.morialkar.industrimental
   ```

   `xcrun devicectl list devices` donne l'UDID. L'équipe de signature n'est pas versionnée.

Le framework est une bibliothèque statique : il est lié à l'app sans être embarqué, ne pas le passer en *Embed & Sign*.
