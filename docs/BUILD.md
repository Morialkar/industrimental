# Compiler et lancer Industrimental

## Prérequis

- Go (la version exacte est dans `go.mod` ; Go la télécharge automatiquement si besoin).
- **Linux** : `sudo apt-get install libgl1-mesa-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev pkg-config`
- **macOS** : Xcode et ses outils en ligne de commande.

## Desktop (développement)

```sh
go run ./cmd/desktop
```

La fenêtre a le format d'un téléphone. Glisse à la souris pour faire défiler le panorama, `R` pour revenir au début.

Capture d'écran sans fenêtre (utilisée par l'intégration continue) :

```sh
go run ./cmd/desktop -screenshot scene.png -frames 20 -warmup 4
```

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

2. Dans Xcode, créer un projet *App* (interface Storyboard, langage Swift) dans `ios/`, nommé `Industrimental`, orientation portrait uniquement.
3. Glisser `Industrimental.xcframework` dans le projet (cible : *Embed & Sign*).
4. Dans `Main.storyboard`, donner au contrôleur de vue initial la classe `MobileEbitenViewController` (module : `Industrimental`).
5. Lancer sur un iPhone branché.

La coquille iOS sera versionnée dans `ios/` une fois créée sur le Mac.
