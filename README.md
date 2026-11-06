# Industrimental

Jeu incremental mobile (portrait, iOS et Android) : l'industrie humaine du troc à l'ère spatiale, en 10 ères qui remplacent le prestige. Écrit en Go avec [Ebitengine](https://ebitengine.org).

## Documents

- [Game Design Document](docs/GDD.md) — copie du [document vivant](https://claude.ai/artifact/F1vkuWbsfuwMhCVuMZG82a)
- [Plan de développement](docs/PLAN.md) — copie du [document vivant](https://claude.ai/artifact/3kYaVD4YnBafY64h1hp46t)
- [Compiler et lancer le jeu](docs/BUILD.md)

## Structure

| Dossier | Rôle |
| --- | --- |
| `cmd/desktop` | Build de développement : fenêtre au format téléphone |
| `mobile` | Point d'entrée lié par `ebitenmobile` pour Android et iOS |
| `internal/content` | Schéma des ères (ressources, stations, recettes, arbre, quêtes), chargement JSON et validation |
| `internal/content/data` | Données du jeu : `eras/*.json` et textes `locales/fr.json`, `locales/en.json` |
| `internal/save` | Sauvegarde versionnée, compressée, vérifiée, sur 3 emplacements tournants |
| `internal/fx` | Système de particules (logique pure, testée) |
| `internal/game` | Boucle Ebitengine ; pour l'instant la scène de test de la phase 0 |
| `android` | Coquille Kotlin minimale qui héberge le jeu |
| `ios` | Coquille Swift minimale qui héberge le jeu |

## Démarrage rapide

```sh
go test ./...
go run ./cmd/desktop
```
