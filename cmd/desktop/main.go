// Command desktop runs the game in a phone-shaped window for development.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/morialkar/industrimental/internal/game"
)

func main() {
	shot := flag.String("screenshot", "", "save a PNG of the screen after -frames frames, then quit")
	frames := flag.Int("frames", 30, "frame at which -screenshot is taken")
	warmup := flag.Float64("warmup", 0, "seconds of simulation to run before the first frame")
	demo := flag.Bool("demo", false, "start from an advanced game, without saving")
	tab := flag.Int("tab", 0, "with -demo, screen to open: 0 world, 1 market, 2 skills, 3 quests, 4 timeline, 5 clan, 6 shop, 7 station")
	flag.Parse()

	opts := game.Options{ScreenshotPath: *shot, ScreenshotFrame: *frames, Warmup: *warmup, Demo: *demo, Tab: *tab}
	if *shot == "" && !*demo {
		// Captures and demos never touch the real save.
		dir, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		opts.DataDir = filepath.Join(dir, "Industrimental")
	}
	g, err := game.New(opts)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowTitle("Industrimental")
	ebiten.SetWindowSize(game.ScreenW/2, game.ScreenH/2)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	if *shot != "" {
		// Headless captures (Xvfb in CI) have no real display to sync with.
		ebiten.SetVsyncEnabled(false)
	}
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
