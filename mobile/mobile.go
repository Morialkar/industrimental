// Package mobile is the entry point bound by ebitenmobile for Android and iOS.
//
//	ebitenmobile bind -target android -javapkg com.morialkar.industrimental -o android/app/libs/industrimental.aar ./mobile
//	ebitenmobile bind -target ios -o ios/Industrimental.xcframework ./mobile
package mobile

import (
	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/morialkar/industrimental/internal/game"
)

func init() {
	g, err := game.New(game.Options{})
	if err != nil {
		panic(err)
	}
	mobile.SetGame(g)
}

// Dummy is required: gomobile only binds packages that export a symbol.
func Dummy() {}
