package game

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/morialkar/industrimental/internal/save"
	"github.com/morialkar/industrimental/internal/sim"
)

// timeState is the wall clock as last seen by the game.
type timeState struct {
	ref    time.Time // reference for the offline gains
	lastUp time.Time // last Update: detects that the app was suspended
}

// start opens the save on the first tick: the native shells may only give the
// data directory after the Go package is initialized.
func (g *Game) start() {
	g.started = true
	g.mu.Lock()
	dir := g.opts.DataDir
	g.mu.Unlock()
	now := time.Now()
	g.lastTick = timeState{ref: now, lastUp: now}
	if dir == "" {
		return
	}
	g.mu.Lock()
	g.store = &save.Store{Dir: dir}
	g.mu.Unlock()
	state, err := g.store.Load()
	switch {
	case errors.Is(err, save.ErrNoSave):
		return
	case err != nil:
		log.Println("save:", err)
		return
	}
	if st, ok := state.Eras[currentEra]; ok {
		g.sim = sim.Restore(g.era, st)
	}
	if state.Locale != "" {
		g.locale = state.Locale
	}
	g.lastTick.ref = state.LastTick
	g.applyAway(now)
}

// catchUp notices a suspension of the app: pausing stops Update, so a long gap
// between two ticks is an absence.
func (g *Game) catchUp() {
	now := time.Now()
	if gap := now.Sub(g.lastTick.lastUp).Seconds(); gap > awayMinimum {
		g.applyAway(now)
	} else {
		_, g.lastTick.ref = sim.Away(g.lastTick.ref, now)
	}
	g.lastTick.lastUp = now
}

func (g *Game) applyAway(now time.Time) {
	away, ref := sim.Away(g.lastTick.ref, now)
	g.lastTick.ref = ref
	if away < awayMinimum {
		return
	}
	rep := g.sim.Offline(away, sim.DefaultOfflineCap)
	g.report = &rep
}

// Save writes the game to disk. Update calls it regularly; the native shells
// call it from their own thread when the app goes to the background, right
// after suspending the game, so that Update is not running at the same time.
func (g *Game) Save() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.store == nil {
		return
	}
	state := &save.GameState{
		CurrentEra: currentEra,
		Locale:     g.locale,
		Eras:       map[string]*save.EraState{currentEra: g.sim.State()},
		LastTick:   g.lastTick.ref,
	}
	if err := g.store.Save(state); err != nil {
		log.Println("save:", err)
	}
}

// reset erases the game: a debug tool of the prototype.
func (g *Game) reset() {
	g.mu.Lock()
	if g.store != nil {
		for i := 0; i < save.Slots; i++ {
			_ = os.Remove(g.store.SlotPath(i))
		}
	}
	g.mu.Unlock()
	g.sim = sim.New(g.era, uint64(time.Now().UnixNano()))
	g.report = nil
}

// prepareDemo builds an advanced game for captures.
func (g *Game) prepareDemo() {
	s := g.sim
	s.Offline(0, 0)
	for _, id := range []string{"baies", "silex", "bois", "coquillages", "peaux-brutes", "viande", "argile", "sel", "outils-silex", "parures", "grain", "ocre", "poteries-crues", "peaux-tannees", "fresques"} {
		s.Grant(id, 2e6)
	}
	for _, id := range []string{"main-habile", "feu-apprivoise", "premiers-paniers", "langage-troc", "sedentarisation", "le-cercle", "foires"} {
		_ = s.BuyNode(id)
	}
	for i := range g.era.Stations {
		def := &g.era.Stations[i]
		_ = s.Build(def.ID, def.Variants[i%len(def.Variants)].ID)
		_ = s.Upgrade(def.ID, 8)
	}
	s.Step(1)
}
