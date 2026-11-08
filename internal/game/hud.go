package game

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/sim"
	"github.com/morialkar/industrimental/internal/ui"
)

// Screens laid over the world.
const (
	scrWorld   = ""
	scrStation = "station"
	scrMarket  = "market"
	scrQuests  = "quests"
	scrShop    = "shop"
	scrClan    = "clan"
	scrEra     = "era"
	scrTree    = "tree"
)

// Layout of the main screen, in logical pixels (see mk for the mockup's units).
var (
	eraBtnRect   = ui.Rect{X: (ScreenW - 340) / 2, Y: safeTop - 10, W: 340, H: 60}
	resBarRect   = ui.Rect{X: 14, Y: safeTop + 60, W: ScreenW - 28, H: 88}
	questPinRect = ui.Rect{X: 14, Y: safeTop + 160, W: ScreenW * 0.74, H: 150}
	harvestRect  = ui.Rect{X: ScreenW - 22 - 192, Y: 560, W: 192, H: 192}
	actionBarY   = float64(ScreenH) - safeBottom - 150
	worldRect    = ui.Rect{X: 0, Y: questPinRect.Y + questPinRect.H, W: ScreenW, H: actionBarY - questPinRect.Y - questPinRect.H}
)

// floatText is a "+12 baies" bubble rising and fading.
type floatText struct {
	x, y  float64
	text  string
	t     float64
	small bool
	world bool // follows the panorama
}

// layout is the whole interface. It runs in Update with dst nil (input and
// actions) and in Draw with the screen (painting), see package ui.
func (g *Game) layout(dst *ebiten.Image) {
	modal := g.screen != scrWorld || g.report != nil
	if dst == nil {
		g.updateFloats()
	}
	if !modal && dst == nil {
		g.worldTap()
	}
	// Underneath an overlay the main screen stays painted but takes no input.
	if !modal || dst != nil {
		g.drawHUD(dst)
		g.drawHarvest(dst)
		g.drawActionBar(dst)
	}
	g.drawFloats(dst)
	// A screen opened during this pass must not also take the tap that opened it.
	fresh := dst == nil && g.openedAt == g.frame
	switch {
	case fresh:
	case g.report != nil:
		g.drawAway(dst)
	case g.screen == scrTree:
		g.drawTree(dst)
	case g.screen != scrWorld:
		g.drawSheet(dst)
	default:
		g.sheetT = 0
	}
	g.drawToast(dst)
}

// open shows a screen over the world.
func (g *Game) open(screen string) {
	g.screen = screen
	g.openedAt = g.frame
	g.sheetT = 0
	g.kit.Reset("sheet")
}

// pressPoint is where the current or last press started.
func (g *Game) pressPoint() (float64, float64) {
	_, _, x, y := g.kit.Released()
	return x, y
}

func (g *Game) closeScreen() {
	g.screen = scrWorld
	g.sheet = ""
	g.pick = ""
	g.focusQuest, g.focusPending = "", false
}

// --- world ----------------------------------------------------------------

// worldTap handles a tap on the panorama: the bush harvests, a station opens
// its sheet.
func (g *Game) worldTap() {
	k := g.kit
	if !k.Tapped(worldRect) || k.Tapped(harvestRect) {
		return
	}
	_, _, x, y := k.Released()
	for i := range g.era.Stations {
		def := &g.era.Stations[i]
		sx := slotX(i) - g.scrollX
		if math.Abs(x-sx) > slotSpacing/2 || y < groundY-stationScale*340 || y > groundY+110 {
			continue
		}
		onPlate := y > groundY
		if len(def.Harvest) > 0 && !onPlate {
			g.harvest(x, y)
			return
		}
		g.openStation(def.ID)
		return
	}
}

func (g *Game) openStation(id string) {
	g.sheet = id
	g.pick = ""
	g.open(scrStation)
}

// harvest gathers by hand, with a bubble and sparks where the finger is.
func (g *Game) harvest(x, y float64) {
	id := g.sim.HarvestStation()
	got, err := g.sim.Harvest(id)
	if err != nil {
		g.act(err)
		return
	}
	g.floats = append(g.floats, floatText{x: x, y: y - 20, text: "+" + g.amountsText(got)})
	g.sparks.X, g.sparks.Y = x, y
	g.sparkFx.Emit(g.sparks, 10)
}

// drawStationLabels writes the plates under the stations of the panorama.
func (g *Game) drawStationLabels(dst *ebiten.Image) {
	k := g.kit
	for i := range g.era.Stations {
		def := &g.era.Stations[i]
		x := slotX(i) - g.scrollX
		if x < -300 || x > ScreenW+300 {
			continue
		}
		name := g.text(def.NameKey)
		w := k.Width(name, 22, true) + 60
		g.plaque(dst, ui.Rect{X: x - w/2, Y: groundY + 14, W: w, H: 48})
		k.Shadowed(dst, name, x, groundY+24, 22, inkLight, ui.Center, true)
		st := g.sim.Station(def.ID)
		label := g.tr("ui.toBuild")
		if st.Built {
			label = fmt.Sprintf("%s %d", g.tr("ui.level"), st.Level)
		}
		k.Shadowed(dst, label, x, groundY+68, 21, inkGold, ui.Center, true)
	}
	// The first tap is the hardest to guess: say it above the bush.
	if !g.sim.Station("carriere-silex").Built {
		x := slotX(0) - g.scrollX
		bob := 6 * math.Sin(g.t*4)
		k.Shadowed(dst, g.tr("ui.tap.harvest"), x, groundY-stationScale*360+bob, 22, inkLight, ui.Center, true)
	}
}

// --- HUD ------------------------------------------------------------------

func (g *Game) drawHUD(dst *ebiten.Image) {
	k := g.kit
	// Era name: opens the timeline.
	c, held := k.Control(dst, eraBtnRect, "era", true)
	if c.Tap {
		g.open(scrEra)
	}
	if dst != nil {
		r := eraBtnRect
		if held {
			r = r.Inset(2)
		}
		g.plaque(dst, r)
		name := fmt.Sprintf(g.tr("ui.era.index"), g.era.Index) + " · " + g.text(g.era.NameKey)
		w := k.Width(name, 24, true) + 12 + k.Width(g.tr("ui.era.timeline"), 19, true)
		x0 := r.X + (r.W-w)/2
		k.Shadowed(dst, name, x0, r.Y+14, 24, inkLight, ui.Left, true)
		k.Shadowed(dst, g.tr("ui.era.timeline"), x0+w, r.Y+18, 19, inkSoft, ui.Right, true)
	}
	g.drawResources(dst)
	g.drawQuestPin(dst)
}

// shownResources lists the resources worth a chip: the currency, then what
// the player has or makes.
func (g *Game) shownResources() []*content.Resource {
	var out []*content.Resource
	for i := range g.era.Resources {
		r := &g.era.Resources[i]
		if r.ID == g.era.Currency || g.sim.ResourceUnlocked(r.ID) {
			out = append(out, r)
		}
	}
	return out
}

const chipW = 150.0

func (g *Game) drawResources(dst *ebiten.Image) {
	k := g.kit
	r := resBarRect
	res := g.shownResources()
	inner := ui.Rect{X: r.X + 40, Y: r.Y + 10, W: r.W - 80, H: r.H - 20}
	maxOff := math.Max(0, float64(len(res))*(chipW+8)-8-inner.W)
	if dst == nil {
		if dx, _, horiz, ok := k.Drag(); ok && horiz && k.PressedIn(r) {
			if !g.resDrag {
				g.resDrag, g.resBase = true, g.resScroll
			}
			g.resScroll = g.resBase - dx
		} else {
			g.resDrag = false
		}
		g.resScroll = math.Max(0, math.Min(maxOff, g.resScroll))
		return
	}
	g.th.band.Draw(dst, r, mk(6), mk(26), mk(6), mk(26), 1)
	sub := dst.SubImage(rectI(inner)).(*ebiten.Image)
	for i, rd := range res {
		x := inner.X + float64(i)*(chipW+8) - g.resScroll
		if x+chipW < inner.X || x > inner.X+inner.W {
			continue
		}
		chip := ui.Rect{X: x, Y: inner.Y, W: chipW, H: inner.H}
		bg := sunkCol
		if rd.ID == g.era.Currency {
			bg = goldSunk
		}
		k.Fill(sub, chip, bg)
		g.pentagon(sub, x+22, chip.Y+chip.H/2, 15, resColor(rd.ID))
		k.Shadowed(sub, sim.Format(g.sim.Stock(rd.ID)), x+44, chip.Y+4, 25, inkLight, ui.Left, true)
		rate := g.sim.Rate(rd.ID)
		line, col := g.text(rd.NameKey), inkSoft
		if math.Abs(rate) > 1e-6 {
			line, col = signed(rate)+"/s", inkGood
			if rate < 0 {
				col = inkBad
			}
		}
		k.Shadowed(sub, k.Fit(line, 18, chipW-50, false), x+44, chip.Y+36, 18, col, ui.Left, false)
	}
}

// pinnedQuest is the active main quest closest to being done.
func (g *Game) pinnedQuest() *content.Quest {
	var best *content.Quest
	bestV := -1.0
	for _, q := range g.sim.ActiveQuests() {
		if !q.Main {
			continue
		}
		if v, _ := g.questProgress(q); v > bestV {
			best, bestV = q, v
		}
	}
	return best
}

func (g *Game) drawQuestPin(dst *ebiten.Image) {
	k := g.kit
	q := g.pinnedQuest()
	if q == nil {
		return
	}
	r := questPinRect
	c, held := k.Control(dst, r, "quest-pin", true)
	if c.Tap {
		g.open(scrQuests)
		g.focusQuest, g.focusPending = q.ID, true
	}
	if dst == nil {
		return
	}
	if held {
		r = r.Inset(2)
	}
	g.panel(dst, r, mk(14))
	x, w := r.X+28, r.W-56
	k.Shadowed(dst, strings.ToUpper(g.tr("ui.quest.pin")), x, r.Y+18, 17, inkSoft, ui.Left, true)
	k.Shadowed(dst, k.Fit(g.text(q.NameKey), 25, w, true), x, r.Y+44, 25, inkLight, ui.Left, true)
	v, count := g.questProgress(q)
	g.progress(dst, ui.Rect{X: x, Y: r.Y + 82, W: w, H: 26}, v)
	k.Shadowed(dst, count, x, r.Y+112, 18, inkSoft, ui.Left, false)
	if len(q.Reward) > 0 {
		k.Shadowed(dst, k.Fit("+"+g.amountsText(q.Reward), 18, w*0.55, false), x+w, r.Y+112, 18, inkGold, ui.Right, false)
	}
}

// --- harvest button -------------------------------------------------------

func (g *Game) drawHarvest(dst *ebiten.Image) {
	k := g.kit
	id := g.sim.HarvestStation()
	if id == "" {
		return
	}
	r := harvestRect
	label := g.tr("ui.harvest")
	c, held := k.Control(dst, r, label, true)
	if c.Tap {
		g.harvest(r.X+r.W/2, r.Y+r.H/3)
	}
	if dst == nil {
		return
	}
	if held {
		r = r.Inset(5)
	}
	ui.Image(dst, g.th.round, r, ebiten.ColorScale{})
	ic := r.Inset(r.W * 0.18)
	if art := g.th.caps["main-habile"]; art != nil {
		ui.Image(dst, art, ic, ebiten.ColorScale{})
	}
	plate := ui.Rect{X: harvestRect.X - 4, Y: harvestRect.Y + harvestRect.H - 30, W: harvestRect.W + 8, H: 76}
	g.plaque(dst, plate)
	k.Shadowed(dst, label, plate.X+plate.W/2, plate.Y+10, 23, inkLight, ui.Center, true)
	k.Shadowed(dst, k.Fit("+"+g.amountsText(g.sim.HarvestValue(id)), 19, plate.W-30, true), plate.X+plate.W/2, plate.Y+40, 19, inkGold, ui.Center, true)
}

// --- action bar -----------------------------------------------------------

func (g *Game) drawActionBar(dst *ebiten.Image) {
	k := g.kit
	bar := ui.Rect{X: 0, Y: actionBarY, W: ScreenW, H: ScreenH - actionBarY + 30}
	g.th.panel.Draw(dst, bar, mk(18), mk(16), mk(10), mk(16), 1)
	labels := [5]string{g.tr("ui.act.tree"), g.tr("ui.act.market"), g.tr("ui.act.quests"), g.tr("ui.act.shop"), g.tr("ui.act.clan")}
	screens := [5]string{scrTree, scrMarket, scrQuests, scrShop, scrClan}
	badges := [5]string{}
	if g.anyInvest() {
		badges[0] = "!"
	}
	if n := g.deliverable(); n > 0 {
		badges[2] = fmt.Sprint(n)
	}
	w := (ScreenW - 2*mk(16)) / 5
	for i, label := range labels {
		r := ui.Rect{X: mk(16) + float64(i)*w, Y: actionBarY + mk(14), W: w, H: 128}
		c, held := k.Control(dst, r, label, true)
		if c.Tap {
			g.open(screens[i])
		}
		if dst == nil {
			continue
		}
		size := 78.0
		ir := ui.Rect{X: r.X + (r.W-size)/2, Y: r.Y + 4, W: size, H: size}
		if held {
			ir = ir.Inset(3)
			ir.Y += 2
		}
		ui.Image(dst, g.th.acts[i], ir, ebiten.ColorScale{})
		k.Shadowed(dst, label, r.X+r.W/2, r.Y+88, 21, inkLight, ui.Center, true)
		if b := badges[i]; b != "" {
			bx, by := ir.X+ir.W-6, ir.Y+4
			g.ellipse(dst, bx, by, 17, 17, rgb(0x14, 0x0a, 0x04))
			g.ellipse(dst, bx, by, 15, 15, rgb(0xe2, 0x73, 0x5a))
			k.Text(dst, b, bx, by-12, 18, inkLight, ui.Center, true)
		}
	}
}

// anyInvest tells whether a tree node can be bought right now.
func (g *Game) anyInvest() bool {
	for i := range g.era.Tree {
		if g.sim.CanInvest(g.era.Tree[i].ID) {
			return true
		}
	}
	return false
}

// deliverable counts the quests and contracts that can be handed in.
func (g *Game) deliverable() int {
	n := 0
	for _, q := range g.sim.ActiveQuests() {
		if goalDelivers(g.sim.CurrentStep(q).Goal) && g.sim.CanDeliver(q) {
			n++
		}
	}
	for _, c := range g.sim.Contracts() {
		if g.sim.Stock(c.Template.Resource) >= c.Amount {
			n++
		}
	}
	return n
}

// --- bubbles and toast ----------------------------------------------------

func (g *Game) updateFloats() {
	kept := g.floats[:0]
	for _, f := range g.floats {
		f.t += dt
		if f.t < 1.1 {
			kept = append(kept, f)
		}
	}
	g.floats = kept
	// A production bubble over one station after the other.
	g.bubbleT += dt
	if g.bubbleT < 0.9 {
		return
	}
	g.bubbleT = 0
	built := []int{}
	for i := range g.era.Stations {
		if g.sim.Station(g.era.Stations[i].ID).Built {
			built = append(built, i)
		}
	}
	if len(built) == 0 {
		return
	}
	g.bubbleN++
	i := built[g.bubbleN%len(built)]
	st := g.sim.Station(g.era.Stations[i].ID)
	v := g.variantOf(&g.era.Stations[i], st.Variant)
	if v == nil || st.Speed <= 0 {
		return
	}
	for _, id := range sortedAmounts(v.Recipe.Outputs) {
		per := v.Recipe.Outputs[id] * st.Cycles * st.Speed * 0.9
		if per <= 0 {
			continue
		}
		g.floats = append(g.floats, floatText{x: slotX(i), y: groundY - stationScale*330, text: "+" + sim.Format(per) + " " + strings.ToLower(g.resName(id)), small: true, world: true})
		break
	}
}

func (g *Game) drawFloats(dst *ebiten.Image) {
	if dst == nil {
		return
	}
	for _, f := range g.floats {
		x := f.x
		if f.world {
			x -= g.scrollX
		}
		y := f.y - 85*easeOut(math.Min(1, f.t))
		a := 1 - math.Max(0, f.t-0.5)/0.6
		size, col := 30.0, rgba(0xff, 0xff, 0xff, uint8(255*a))
		if f.small {
			size, col = 21, rgba(0xf7, 0xf0, 0xc7, uint8(255*a))
		}
		g.kit.Text(dst, f.text, x, y+2, size, rgba(0x10, 0x08, 0x02, uint8(200*a)), ui.Center, true)
		g.kit.Text(dst, f.text, x, y, size, col, ui.Center, true)
	}
	g.drawSparks(dst)
}

func (g *Game) drawToast(dst *ebiten.Image) {
	if g.toastT <= 0 {
		return
	}
	if dst == nil {
		g.toastT--
		return
	}
	r := ui.Rect{X: 50, Y: actionBarY - 100, W: ScreenW - 100, H: 76}
	g.insetBox(dst, r, mk(12))
	g.kit.Fill(dst, r.Inset(14), rgba(0x20, 0x10, 0x04, 0xc0))
	g.kit.Shadowed(dst, g.kit.Fit(g.toast, 24, r.W-50, false), ScreenW/2, r.Y+22, 24, inkLight, ui.Center, false)
}
