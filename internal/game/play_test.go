package game

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/morialkar/industrimental/internal/ui"
)

// finger plays the real interface through the same input path as a phone:
// it presses the buttons by their label, tick after tick.
type finger struct {
	t *testing.T
	g *Game
}

func newFinger(t *testing.T, opts Options) *finger {
	t.Helper()
	g, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	g.kit.Record()
	f := &finger{t, g}
	f.tick(0, 0, false) // the first tick opens the save
	return f
}

func (f *finger) tick(x, y float64, down bool) {
	f.t.Helper()
	f.g.kit.Feed(x, y, down)
	f.g.tickWith()
}

func (f *finger) button(label string) ui.Rect {
	f.t.Helper()
	r, ok := f.g.kit.Recorded(label)
	if !ok {
		f.t.Fatalf("no button %q on screen (tab %d); buttons: %v", label, f.g.tab, f.g.kit.RecordedLabels())
	}
	return r
}

func (f *finger) tap(label string) {
	f.t.Helper()
	r := f.button(label)
	x, y := r.X+r.W/2, r.Y+r.H/2
	f.tick(x, y, true)
	f.tick(x, y, false)
	f.tick(x, y, false) // lets the screen settle and the buttons be listed again
}

func (f *finger) hold(label string) {
	f.t.Helper()
	r := f.button(label)
	x, y := r.X+r.W/2, r.Y+r.H/2
	for i := 0; i < 40; i++ {
		f.tick(x, y, true)
	}
	f.tick(x, y, false)
	f.tick(x, y, false)
}

// settle waits for animations to finish.
func (f *finger) settle() {
	for i := 0; i < 30; i++ {
		f.tick(0, 0, false)
	}
}

// swipe drags the panel upward from its middle.
func (f *finger) swipe(dy float64) {
	y := panelY + 300
	f.tick(360, y, true)
	for i := 1; i <= 10; i++ {
		f.tick(360, y-dy*float64(i)/10, true)
	}
	f.tick(360, y-dy, false)
	f.tick(360, y-dy, false)
}

func TestTheOpeningThroughTheInterface(t *testing.T) {
	f := newFinger(t, Options{})
	g := f.g
	f.tick(0, 0, false)
	gather := "Cueillir : +1 Baies"
	for i := 0; i < 6; i++ {
		f.tap(gather)
	}
	if got := g.sim.Stock("baies"); got != 6 {
		t.Fatalf("after 6 taps: %g berries", got)
	}

	// Bob's barter, in the market tab, even before the market opens.
	f.tap("Marché")
	if g.tab != tabMarket {
		t.Fatalf("tab = %d", g.tab)
	}
	f.tap("5 Baies → 1 Silex")
	if g.sim.Stock("silex") != 1 {
		t.Fatalf("silex = %g", g.sim.Stock("silex"))
	}

	// Build the bush from its sliding sheet: choose a variant, the sheet closes.
	f.tap("Stations")
	for i := 0; i < 8; i++ {
		f.tap(gather)
	}
	if got := g.sim.Stock("baies"); got < 5 {
		t.Fatalf("not enough berries: %g", got)
	}
	f.tap("Construire")
	f.settle()
	if g.sheet != "cueillette" {
		t.Fatalf("the sheet of the bush should be open, got %q", g.sheet)
	}
	// The panel below takes no input while a sheet is open.
	if _, ok := g.kit.Recorded("Marché"); ok {
		t.Error("the tabs should be disabled under a sheet")
	}
	f.tap("Construire")
	if g.sheet != "" || !g.sim.Station("cueillette").Built {
		t.Fatalf("the bush should be built, sheet = %q", g.sheet)
	}
}

func TestTappingTheBushInThePanorama(t *testing.T) {
	f := newFinger(t, Options{})
	f.tick(360, 600, true)
	f.tick(360, 600, false)
	if f.g.sim.Stock("baies") != 1 {
		t.Errorf("baies = %g", f.g.sim.Stock("baies"))
	}
}

func TestLongPressBuysAllLevels(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	g := f.g
	before := g.sim.Station("cueillette").Level
	label, ok := g.kit.RecordedPrefix("+1 : ")
	if !ok {
		t.Fatalf("no upgrade button: %v", g.kit.RecordedLabels())
	}
	f.tap(label)
	if got := g.sim.Station("cueillette").Level; got != before+1 {
		t.Fatalf("a tap buys one level: %d -> %d", before, got)
	}
	label, _ = g.kit.RecordedPrefix("+1 : ")
	f.hold(label)
	if got := g.sim.Station("cueillette").Level; got <= before+2 {
		t.Fatalf("a long press should buy many levels: %d -> %d", before+1, got)
	}
}

func TestSwipeScrollsWithoutPressing(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	g := f.g
	level := g.sim.Station("cueillette").Level
	f.swipe(400)
	if off := g.kit.ScrollOffset("tab0"); off < 300 {
		t.Errorf("the list should have scrolled, offset = %g", off)
	}
	if g.sim.Station("cueillette").Level != level {
		t.Error("a swipe must not press the button under the finger")
	}
	f.tap("Arbre")
	f.tap("Stations")
}

func TestSaveThenComeBackLater(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "saves")
	f := newFinger(t, Options{DataDir: dir})
	g := f.g
	for i := 0; i < 8; i++ {
		f.tap("Cueillir : +1 Baies")
	}
	f.tap("Construire")
	f.settle()
	f.tap("Construire")
	if !g.sim.Station("cueillette").Built {
		t.Fatal("the bush should be built")
	}
	g.lastTick.ref = g.lastTick.ref.Add(-time.Hour) // the player left an hour ago
	g.Save()

	back := newFinger(t, Options{DataDir: dir})
	if !back.g.sim.Station("cueillette").Built {
		t.Fatal("the save should bring the bush back")
	}
	if back.g.report == nil {
		t.Fatal("coming back after an hour should show the absence screen")
	}
	if got := back.g.report.Gains["baies"]; got < 3600 {
		t.Errorf("berries gained in an hour = %g", got)
	}
	back.tap("Merci, le clan")
	if back.g.report != nil {
		t.Error("the absence screen should close")
	}
}

func TestTheClockGoingBackGivesNothing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "saves")
	f := newFinger(t, Options{DataDir: dir})
	for i := 0; i < 8; i++ {
		f.tap("Cueillir : +1 Baies")
	}
	f.tap("Construire")
	f.settle()
	f.tap("Construire")
	f.g.lastTick.ref = f.g.lastTick.ref.Add(24 * time.Hour) // saved "in the future"
	f.g.Save()
	baies := f.g.sim.Stock("baies")

	back := newFinger(t, Options{DataDir: dir})
	if back.g.report != nil {
		t.Error("no absence when the clock went backward")
	}
	if got := back.g.sim.Stock("baies"); got > baies+1 {
		t.Errorf("berries jumped from %g to %g", baies, got)
	}
}

func TestEverySafeTabLaysOut(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	for _, tab := range []string{"Marché", "Arbre", "Quêtes", "Ère", "Stations"} {
		f.tap(tab)
	}
	f.tap("Fiche")
	f.settle()
	if f.g.sheet == "" {
		t.Fatal("the details button should open the sheet")
	}
	f.tap("Fermer")
	if f.g.sheet != "" {
		t.Error("the sheet should close")
	}
}
