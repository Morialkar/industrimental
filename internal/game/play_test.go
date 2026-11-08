package game

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/morialkar/industrimental/internal/content"
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
		f.t.Fatalf("no button %q on screen (screen %q); buttons: %v", label, f.g.screen, f.g.kit.RecordedLabels())
	}
	return r
}

func (f *finger) tap(label string) {
	f.t.Helper()
	r := f.button(label)
	f.tapAt(r.X+r.W/2, r.Y+r.H/2)
}

func (f *finger) tapAt(x, y float64) {
	f.t.Helper()
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

// swipe drags the finger from (x, y) by (dx, dy).
func (f *finger) swipe(x, y, dx, dy float64) {
	f.tick(x, y, true)
	for i := 1; i <= 10; i++ {
		f.tick(x+dx*float64(i)/10, y+dy*float64(i)/10, true)
	}
	f.tick(x+dx, y+dy, false)
	f.tick(x+dx, y+dy, false)
}

// openBush taps the plate of the first station in the panorama.
func (f *finger) openBush() {
	f.t.Helper()
	f.tapAt(slotX(0)-f.g.scrollX, groundY+30)
	f.settle()
	if f.g.screen != scrStation || f.g.sheet != "cueillette" {
		f.t.Fatalf("the plate should open the bush's sheet, got %q %q", f.g.screen, f.g.sheet)
	}
}

func TestTheOpeningThroughTheInterface(t *testing.T) {
	f := newFinger(t, Options{})
	g := f.g
	for i := 0; i < 6; i++ {
		f.tap("Récolter")
	}
	if got := g.sim.Stock("baies"); got != 6 {
		t.Fatalf("after 6 taps: %g berries", got)
	}

	// Bob's barter, in the market, even before the market opens.
	f.tap("Marché")
	f.settle()
	if g.screen != scrMarket {
		t.Fatalf("screen = %q", g.screen)
	}
	f.tap("deal bob silex")
	if g.sim.Stock("silex") != 1 {
		t.Fatalf("silex = %g", g.sim.Stock("silex"))
	}
	f.tap("Fermer")
	if g.screen != scrWorld {
		t.Fatalf("the sheet should close, screen = %q", g.screen)
	}

	for i := 0; i < 8; i++ {
		f.tap("Récolter")
	}
	f.openBush()
	// The main screen takes no input while a sheet is open.
	if _, ok := g.kit.Recorded("Marché"); ok {
		t.Error("the action bar should be disabled under a sheet")
	}
	f.tap("spec herbes")
	f.tap("Construire")
	if st := g.sim.Station("cueillette"); !st.Built || st.Variant != "herbes" {
		t.Fatalf("the bush should be built with the picked specialization: %+v", st)
	}
}

func TestTappingTheBushInThePanorama(t *testing.T) {
	f := newFinger(t, Options{})
	f.tapAt(slotX(0), groundY-60)
	if f.g.sim.Stock("baies") != 1 {
		t.Errorf("baies = %g", f.g.sim.Stock("baies"))
	}
	if len(f.g.floats) == 0 {
		t.Error("a harvest should show a bubble")
	}
}

func TestLongPressBuysAllLevels(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	g := f.g
	f.openBush()
	before := g.sim.Station("cueillette").Level
	label, ok := g.kit.RecordedPrefix("upgrade")
	if !ok {
		t.Fatalf("no upgrade button: %v", g.kit.RecordedLabels())
	}
	f.tap(label)
	if got := g.sim.Station("cueillette").Level; got != before+1 {
		t.Fatalf("a tap buys one level: %d -> %d", before, got)
	}
	label, _ = g.kit.RecordedPrefix("upgrade")
	f.hold(label)
	if got := g.sim.Station("cueillette").Level; got <= before+2 {
		t.Fatalf("a long press should buy many levels: %d -> %d", before+1, got)
	}
}

func TestSwipeScrollsWithoutPressing(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	g := f.g
	f.tap("Marché")
	f.settle()
	silex := g.sim.Stock("silex")
	f.swipe(360, 1200, 0, -500)
	if off := g.kit.ScrollOffset("sheet"); off < 300 {
		t.Errorf("the sheet should have scrolled, offset = %g", off)
	}
	if got := g.sim.Stock("silex"); got < silex-1 || len(g.sim.State().TotalSold) > 0 {
		t.Error("a swipe must not press the button under the finger")
	}
}

func TestTheSkillTree(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	g := f.g
	f.tap("Capacités")
	f.settle()
	if g.screen != scrTree {
		t.Fatalf("screen = %q", g.screen)
	}
	f.tap("node etals")
	if g.selNode != "etals" {
		t.Fatalf("selected = %q", g.selNode)
	}
	f.tap("Investir")
	if g.sim.Rank("etals") != 1 {
		t.Fatalf("one tap buys one rank, rank = %d", g.sim.Rank("etals"))
	}
	// A sideways swipe moves to the next branch without scrolling the tree.
	f.swipe(500, 1000, -400, 10)
	f.settle()
	if g.treeBoard != 1 {
		t.Errorf("board = %d after a swipe", g.treeBoard)
	}
	if off := g.kit.ScrollOffset("tree"); off != 0 {
		t.Errorf("a sideways swipe scrolled the tree to %g", off)
	}
	f.tap("tab community")
	f.settle()
	if g.treeBoard != 3 || math.Abs(g.treePos-3) > 0.05 {
		t.Errorf("the tab should bring board 3, board = %d at %g", g.treeBoard, g.treePos)
	}
	f.tap("▼ Tournant")
	f.settle()
	if g.kit.ScrollOffset("tree") < 300 {
		t.Error("the Turning point shortcut should scroll down")
	}
	f.tap("Fermer")
	if g.screen != scrWorld {
		t.Error("the tree should close")
	}
}

func TestSaveThenComeBackLater(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "saves")
	f := newFinger(t, Options{DataDir: dir})
	g := f.g
	for i := 0; i < 8; i++ {
		f.tap("Récolter")
	}
	f.openBush()
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
		f.tap("Récolter")
	}
	f.openBush()
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

func TestEveryScreenLaysOut(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	for _, label := range []string{"Marché", "Quêtes", "Boutique", "Clan", "era", "quest-pin"} {
		f.tap(label)
		f.settle()
		if f.g.screen == scrWorld {
			t.Fatalf("%q should open a screen", label)
		}
		f.tap("Fermer")
		if f.g.screen != scrWorld {
			t.Fatalf("%q should close", label)
		}
	}
	// Tapping above a sheet closes it too.
	f.tap("Clan")
	f.settle()
	f.tapAt(360, 150)
	if f.g.screen != scrWorld {
		t.Error("a tap on the world above the sheet should close it")
	}
}

func TestThePinnedQuestOpensAndHighlights(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	g := f.g
	pinned := g.pinnedQuest()
	if pinned == nil {
		t.Fatal("the demo should pin a quest")
	}
	f.tap("quest-pin")
	f.settle()
	if g.screen != scrQuests || g.focusQuest != pinned.ID {
		t.Fatalf("screen %q, focus %q, want the journal on %q", g.screen, g.focusQuest, pinned.ID)
	}
	if g.focusPending {
		t.Error("the journal should have scrolled to the quest")
	}
	f.tap("Fermer")
	if g.focusQuest != "" {
		t.Error("closing the journal ends the highlight")
	}
}

func TestTappingAStationOpensItsModal(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	g := f.g
	f.tapAt(slotX(1)-g.scrollX, groundY-80)
	f.settle()
	if g.screen != scrStation || g.sheet != g.era.Stations[1].ID {
		t.Fatalf("screen %q sheet %q", g.screen, g.sheet)
	}
	if _, ok := g.kit.RecordedPrefix("upgrade"); !ok {
		t.Errorf("the modal should offer the upgrade: %v", g.kit.RecordedLabels())
	}
}

func TestAResourceSheetSaysWhereItComesFrom(t *testing.T) {
	f := newFinger(t, Options{Demo: true})
	g := f.g
	f.tap("res baies")
	f.settle()
	if g.screen != scrResource || g.sheet != "baies" {
		t.Fatalf("screen %q sheet %q, want the sheet of baies", g.screen, g.sheet)
	}
	src := g.producers("baies")
	if len(src) == 0 {
		t.Fatal("something makes baies")
	}
	see := "see " + src[0].st.ID + " " + src[0].v.ID
	f.tap(see)
	f.settle()
	if g.screen != scrStation || g.sheet != src[0].st.ID {
		t.Fatalf("Voir should open the station: screen %q sheet %q", g.screen, g.sheet)
	}
	f.tap("Fermer")
	f.settle()
	if g.screen != scrResource || g.sheet != "baies" {
		t.Fatalf("closing the station comes back to the resource: screen %q sheet %q", g.screen, g.sheet)
	}
	f.tap("Fermer")
	if g.screen != scrWorld {
		t.Fatalf("then back to the world, got %q", g.screen)
	}
}

func TestEveryResourceHasASource(t *testing.T) {
	g := newFinger(t, Options{Demo: true}).g
	for _, r := range g.era.Resources {
		if r.Kind == content.KindCurrency {
			continue
		}
		if line, _ := g.sourceLine(r.ID); line == g.tr("ui.res.none") {
			t.Errorf("%s: the market would say nothing makes it", r.ID)
		}
	}
}
