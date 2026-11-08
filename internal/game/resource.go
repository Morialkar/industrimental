package game

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/sim"
	"github.com/morialkar/industrimental/internal/ui"
)

// source is one specialization of a station that makes a resource.
type source struct {
	st *content.Station
	v  *content.Variant
}

// producers lists the station specializations whose recipe outputs id, in
// the order of the panorama.
func (g *Game) producers(id string) []source {
	var out []source
	for i := range g.era.Stations {
		def := &g.era.Stations[i]
		for j := range def.Variants {
			if def.Variants[j].Recipe.Outputs[id] > 0 {
				out = append(out, source{def, &def.Variants[j]})
			}
		}
	}
	return out
}

// sourceLine says in a few words where a resource comes from right now: the
// station making it, else the next step to take.
func (g *Game) sourceLine(id string) (string, bool) {
	srcs := g.producers(id)
	var making []string
	for _, s := range srcs {
		if st := g.sim.Station(s.st.ID); st.Built && st.Variant == s.v.ID && !contains(making, g.text(s.st.NameKey)) {
			making = append(making, g.text(s.st.NameKey))
		}
	}
	if len(making) > 0 {
		return fmt.Sprintf(g.tr("ui.res.from"), strings.Join(making, ", ")), true
	}
	for i := range g.era.Stations {
		if g.era.Stations[i].Harvest[id] > 0 {
			return g.tr("ui.res.byHand"), true
		}
	}
	for _, s := range srcs {
		if st := g.sim.Station(s.st.ID); st.Built {
			return fmt.Sprintf(g.tr("ui.res.specialize"), g.text(s.st.NameKey), g.text(s.v.NameKey)), false
		}
	}
	for _, s := range srcs {
		if g.sim.StationAvailable(s.st.ID) {
			return fmt.Sprintf(g.tr("ui.res.toBuild"), g.text(s.st.NameKey)), false
		}
	}
	if len(srcs) > 0 {
		return fmt.Sprintf(g.tr("ui.res.locked"), g.nodeName(srcs[0].st.RequiresNode), g.text(srcs[0].st.NameKey)), false
	}
	return g.tr("ui.res.none"), false
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// openResource shows the sheet of a resource, over the screen already open.
func (g *Game) openResource(id string) {
	g.push()
	g.sheet = id
	g.open(scrResource)
}

// push remembers the open screen so that closing the next one comes back to it.
func (g *Game) push() {
	if g.screen != scrWorld {
		g.back = append(g.back, backScreen{g.screen, g.sheet})
	}
}

// resourceBody is the sheet of one resource: what you have, where it comes
// from, what it is for, and what it sells for.
func (g *Game) resourceBody(d *ebiten.Image, x, y, w float64) float64 {
	k := g.kit
	id := g.sheet
	var res *content.Resource
	for i := range g.era.Resources {
		if g.era.Resources[i].ID == id {
			res = &g.era.Resources[i]
		}
	}
	if res == nil {
		return y
	}
	// A big icon and the stock.
	if d != nil {
		box := ui.Rect{X: x, Y: y, W: 150, H: 150}
		g.insetBox(d, box, mk(12))
		g.resIcon(d, id, box.X+75, box.Y+75, 50)
		k.Shadowed(d, g.tr("ui.res.have"), x+176, y+14, smallSize, inkSoft, ui.Left, false)
		k.Shadowed(d, sim.Format(g.sim.Stock(id)), x+176, y+44, 44, inkLight, ui.Left, true)
		rate := g.sim.Rate(id)
		col := inkSoft
		if rate > 1e-6 {
			col = inkGood
		} else if rate < -1e-6 {
			col = inkBad
		}
		k.Shadowed(d, signed(rate)+"/s", x+176, y+102, bodySize, col, ui.Left, true)
	}
	y += 150 + gap
	if res.DescKey != "" {
		y = g.quip(d, g.text(res.DescKey), x, y, w)
	}

	// Where it comes from.
	y = g.groupTitle(d, g.tr("ui.res.origin"), x, y)
	any := false
	for i := range g.era.Stations {
		if g.era.Stations[i].Harvest[id] > 0 {
			any = true
			y += k.ShadowedParagraph(d, g.tr("ui.res.byHand")+" · "+g.text(g.era.Stations[i].NameKey), x, y, bodySize, w, inkLight, false) + gap
		}
	}
	for _, s := range g.producers(id) {
		any = true
		y = g.sourceCard(d, s, x, y, w)
	}
	for i := range g.era.NPCs {
		n := &g.era.NPCs[i]
		for j := range n.Deals {
			if n.Deals[j].Get[id] > 0 {
				any = true
				y += k.ShadowedParagraph(d, fmt.Sprintf(g.tr("ui.res.trade"), g.text(n.NameKey)), x, y, bodySize, w, inkLight, false) + gap
			}
		}
	}
	if !any {
		y += k.ShadowedParagraph(d, g.tr("ui.res.none"), x, y, bodySize, w, inkSoft, false) + gap
	}

	// What it is for.
	if uses := g.usesOf(id); len(uses) > 0 {
		y = g.groupTitle(d, g.tr("ui.res.uses"), x, y)
		for _, u := range uses {
			y += k.ShadowedParagraph(d, "• "+u, x, y, smallSize, w, inkLight, false) + 6
		}
		y += gap
	}

	// What it sells for.
	if res.Kind != content.KindCurrency && g.sim.MarketOpen() {
		y = g.groupTitle(d, g.tr("ui.act.market"), x, y)
		y = g.priceLine(d, id, x, y, w)
		y = g.tradeButtons(d, id, x, y, w)
	}
	return y
}

// sourceCard shows one specialization making the resource: its recipe in
// icons, where it stands, and a button to the station.
func (g *Game) sourceCard(d *ebiten.Image, s source, x, y, w float64) float64 {
	k := g.kit
	st := g.sim.Station(s.st.ID)
	chips := recipeChips(s.v.Recipe, 1)
	inner := w - 48 - 160
	ch := g.drawChips(nil, chips, 0, 0, inner, 21, false, inkLight)
	h := 40 + 36 + ch + 34
	in := g.card(d, x, y, w, h)
	k.Shadowed(d, k.Fit(g.text(s.st.NameKey)+" · "+g.text(s.v.NameKey), smallSize+1, inner, true), in.X, in.Y, smallSize+1, inkLight, ui.Left, true)
	g.drawChips(d, chips, in.X, in.Y+36, inner, 21, false, inkLight)
	status, col := "", inkSoft
	switch {
	case st.Built && st.Variant == s.v.ID:
		status, col = fmt.Sprintf("%s %d · %s/s", g.tr("ui.level"), st.Level, sim.Format(s.v.Recipe.Outputs[g.sheet]*st.Cycles*st.Speed)), inkGood
	case st.Built:
		status, col = g.tr("ui.res.otherSpec"), inkGold
	case g.sim.StationAvailable(s.st.ID):
		status, col = g.tr("ui.station.notBuilt"), inkGold
	default:
		status = fmt.Sprintf(g.tr("ui.res.needs"), g.nodeName(s.st.RequiresNode))
	}
	k.Shadowed(d, k.Fit(status, 19, inner, false), in.X, in.Y+36+ch+4, 19, col, ui.Left, false)
	btn := ui.Rect{X: in.X + in.W - 140, Y: in.Y + (in.H-btnH)/2, W: 140, H: btnH}
	if g.woodBtn(d, btn, "see "+s.st.ID+" "+s.v.ID, g.tr("ui.res.see"), true, 22).Tap {
		g.push()
		g.sheet = s.st.ID
		g.pick = s.v.ID
		g.open(scrStation)
	}
	return y + h + gap
}

// usesOf lists, in words, what consumes a resource.
func (g *Game) usesOf(id string) []string {
	var out []string
	add := func(s string) {
		if !contains(out, s) {
			out = append(out, s)
		}
	}
	for i := range g.era.Stations {
		def := &g.era.Stations[i]
		name := g.text(def.NameKey)
		for j := range def.Variants {
			if def.Variants[j].Recipe.Inputs[id] > 0 {
				add(fmt.Sprintf(g.tr("ui.res.useRecipe"), name, g.text(def.Variants[j].NameKey)))
			}
		}
		if def.UnlockCost[id] > 0 && !g.sim.Station(def.ID).Built {
			add(fmt.Sprintf(g.tr("ui.res.useBuild"), name))
		}
		if def.LevelCost[id] > 0 && g.sim.Station(def.ID).Built {
			add(fmt.Sprintf(g.tr("ui.res.useLevel"), name))
		}
	}
	for i := range g.era.Tree {
		n := &g.era.Tree[i]
		if n.Cost[id] > 0 && g.sim.NodeAvailable(n.ID) {
			add(g.tr("ui.res.useTree"))
			break
		}
	}
	for _, q := range g.sim.ActiveQuests() {
		if step := g.sim.QuestStep(q.ID); step < len(q.Steps) && questWants(q.Steps[step].Goal, id) {
			add(fmt.Sprintf(g.tr("ui.res.useQuest"), g.text(q.NameKey)))
		}
	}
	return out
}

// priceLine is "3.2 each ▲" and the stock, on one line.
func (g *Game) priceLine(d *ebiten.Image, id string, x, y, w float64) float64 {
	k := g.kit
	arrow, col := "▲", inkGood
	if g.sim.Trend(id) < 0 {
		arrow, col = "▼", inkBad
	}
	if d != nil {
		price := fmt.Sprintf(g.tr("ui.res.each"), sim.Format(g.sim.Price(id))) + " " + arrow
		pw := k.Width(price, bodySize, true)
		k.Shadowed(d, price, x+w, y, bodySize, col, ui.Right, true)
		g.resIcon(d, g.era.Currency, x+w-pw-24, y+bodySize*0.62, 15)
		k.Shadowed(d, g.tr("ui.res.price"), x, y, bodySize, inkSoft, ui.Left, false)
	}
	return y + bodySize*1.7
}

// tradeButtons are sell a quarter, sell all and buy ten, each with what it
// brings or costs.
func (g *Game) tradeButtons(d *ebiten.Image, id string, x, y, w float64) float64 {
	stock := g.sim.Stock(id)
	cur := g.era.Currency
	bw := (w - 2*10) / 3
	quarter := stock / 4
	gain := func(v float64) []chip { return []chip{{id: cur, qty: v, plus: true}} }
	if g.costButton(d, ui.Rect{X: x, Y: y, W: bw, H: costBtnH}, "sell-quarter "+id, g.tr("ui.sell.quarter"), gain(g.sim.SellValue(id, quarter)), stock >= 4).Tap {
		_, err := g.sim.Sell(id, quarter)
		g.act(err)
	}
	if g.costButton(d, ui.Rect{X: x + bw + 10, Y: y, W: bw, H: costBtnH}, "sell-all "+id, g.tr("ui.sell.all"), gain(g.sim.SellValue(id, stock)), stock >= 1).Tap {
		_, err := g.sim.Sell(id, stock)
		g.act(err)
	}
	cost := g.sim.BuyCost(id, 10)
	buy := []chip{{id: cur, qty: cost, short: g.sim.Stock(cur) < cost}}
	if g.costButton(d, ui.Rect{X: x + 2*(bw+10), Y: y, W: bw, H: costBtnH}, "buy-ten "+id, g.tr("ui.buy.ten"), buy, g.sim.Stock(cur) >= cost).Tap {
		g.act(g.sim.Buy(id, 10))
	}
	return y + costBtnH + gap
}

// questWants tells whether a quest goal asks for a resource.
func questWants(goal content.Goal, id string) bool {
	if goal.Have[id] > 0 || goal.Produced[id] > 0 || goal.Deliver[id] > 0 {
		return true
	}
	for _, alt := range goal.Any {
		if questWants(alt, id) {
			return true
		}
	}
	return false
}
