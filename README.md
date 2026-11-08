# Industrimental

Jeu incremental mobile (portrait, iOS et Android) : l'industrie humaine du troc à l'ère spatiale, en 10 ères qui remplacent le prestige. Écrit en Go avec [Ebitengine](https://ebitengine.org).

## Documents

- [Game Design Document](docs/GDD.md) — copie du [document vivant](https://claude.ai/artifact/F1vkuWbsfuwMhCVuMZG82a)
- [Plan de développement](docs/PLAN.md) — copie du [document vivant](https://claude.ai/artifact/3kYaVD4YnBafY64h1hp46t)
- [Compiler et lancer le jeu](docs/BUILD.md)
- [Essai du kit d'interface](docs/INTERFACE.md)

## Structure

| Dossier | Rôle |
| --- | --- |
| `cmd/desktop` | Build de développement : fenêtre au format téléphone |
| `mobile` | Point d'entrée lié par `ebitenmobile` pour Android et iOS |
| `internal/content` | Schéma des ères (ressources, stations, recettes, arbre, quêtes), chargement JSON et validation |
| `internal/content/data` | Données du jeu : `eras/*.json` et textes `locales/fr.json`, `locales/en.json` |
| `internal/save` | Sauvegarde versionnée, compressée, vérifiée, sur 3 emplacements tournants |
| `internal/fx` | Système de particules (logique pure, testée) |
| `internal/sim` | Économie d'une ère en Go pur : stations, goulets, marché, arbre, quêtes, hors-ligne |
| `internal/balance` | Joueur simulé : une stratégie par fin, sur le rythme d'un joueur régulier |
| `internal/ui` | Kit d'interface maison : boutons, listes, cadres en 9 morceaux (voir [l'essai](docs/INTERFACE.md)) |
| `internal/assets` | Images embarquées : kit d'interface de l'ère 1 (bois) |
| `internal/game` | Boucle Ebitengine : monde plein écran, barre d'actions, fiches, arbre des capacités, sauvegarde, absences |
| `cmd/balance` | Durée de chaque fin et courbes de progression, sans écran |
| `android` | Coquille Kotlin minimale qui héberge le jeu |
| `ios` | Coquille Swift minimale qui héberge le jeu |

## Démarrage rapide

```sh
go test ./...
go run ./cmd/desktop
```
