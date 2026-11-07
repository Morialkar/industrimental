// Package mobile is the entry point bound by ebitenmobile for Android and iOS.
//
//	ebitenmobile bind -target android -javapkg com.morialkar.industrimental -o android/app/libs/industrimental.aar ./mobile
//	ebitenmobile bind -target ios -o ios/Industrimental.xcframework ./mobile
package mobile

import (
	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/morialkar/industrimental/internal/game"
)

var theGame *game.Game

func init() {
	g, err := game.New(game.Options{})
	if err != nil {
		panic(err)
	}
	theGame = g
	mobile.SetGame(g)
}

// SetDataDir tells the game where to keep its saves. The native shell calls it
// once, at startup, before the first frame.
func SetDataDir(dir string) { theGame.SetDataDir(dir) }

// Save writes the game to disk. The native shell calls it when the app goes to
// the background: a suspended app may never get another chance.
func Save() { theGame.Save() }
