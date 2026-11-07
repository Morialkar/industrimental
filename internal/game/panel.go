package game

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/sim"
	"github.com/morialkar/industrimental/internal/ui"
)

const (
	tabStations = iota
	tabMarket
	tabTree
	tabQuests
	tabEra
	tabCount
)

const (
	topBarH  = 112.0
	gaugesY  = 836.0
	gaugesH  = 64.0
	panelY   = gaugesY + gaugesH
	tabsH    = 104.0
	tabsY    = ScreenH - tabsH - 26 // room for the home indicator
	padding  = 18.0
	btnH     = 76.0 // at least 44 points on a phone
	rowGap   = 14.0
	sizeBody = 27.0
	sizeHead = 31.0
	sizeNote = 24.0
)

var (
	panoramaRect = ui.Rect{X: 0, Y: topBarH, W: ScreenW, H: gaugesY - topBarH}

	colBar      = rgba(0x2b, 0x22, 0x18, 0xf0)
	colPanel    = rgb(0x2b, 0x22, 0x18)
	colRow      = rgb(0x3a, 0x2e, 0x21)
	colText     = rgb(0xf5, 0xea, 0xd6)
	colDim      = rgb(0xb8, 0xa8, 0x8e)
	colAccent   = rgb(0xd9, 0x7a, 0x2b)
	colGood     = rgb(0x8f, 0xb8, 0x6a)
	colBad      = rgb(0xe0, 0x7a, 0x5f)
	colSheet    = rgb(0x33, 0x29, 0x1d)
	colScrim    = rgba(0, 0, 0, 0x90)
	styleMain   = ui.Style{Fill: colAccent, Disabled: rgb(0x4a, 0x3b, 0x2a), Text: rgb(0x20, 0x16, 0x0c), TextDisabled: colDim, Size: 26}
	styleSecond = ui.Style{Fill: rgb(0x5a, 0x48, 0x33), Disabled: rgb(0x40, 0x34, 0x26), Text: colText, TextDisabled: colDim, Size: 26}
)

func (g *Game) text(key string) string { return g.bundle.Text(g.locale, key) }
func (g *Game) tr(key string) string   { return g.bundle.Text(g.locale, key) }

func (g *Game) resName(id string) string {
	for i := range g.era.Resources {
		if g.era.Resources[i].ID == id {
			return g.text(g.era.Resources[i].NameKey)
		}
	}
	return id
}

func (g *Game) stationName(id string) string {
	for i := range g.era.Stations {
		if g.era.Stations[i].ID == id {
			return g.text(g.era.Stations[i].NameKey)
		}
	}
	return id
}

func (g *Game) nodeName(id string) string {
	for i := range g.era.Tree {
		if g.era.Tree[i].ID == id {
			return g.text(g.era.Tree[i].NameKey)
		}
	}
	return id
}

func (g *Game) amountsText(a content.Amounts) string {
	if len(a) == 0 {
		return g.tr("ui.free")
	}
	parts := make([]string, 0, len(a))
	for _, id := range sortedAmounts(a) {
		parts = append(parts, sim.Format(a[id])+" "+g.resName(id))
	}
	return strings.Join(parts, " · ")
}

func (g *Game) recipeText(r content.Recipe) string {
	out := g.amountsText(r.Outputs)
	if len(r.Inputs) > 0 {
		return g.amountsText(r.Inputs) + " → " + out
	}
	return out
}

func sortedAmounts(a content.Amounts) []string {
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (g *Game) act(err error) {
	switch {
	case err == nil:
	case errors.Is(err, sim.ErrAfford):
		g.say(g.tr("ui.toast.afford"))
	default:
		g.say(g.tr("ui.toast.locked"))
	}
}

func (g *Game) say(s string) { g.toast, g.toastT = s, 150 }

// layout is the whole interface. It runs in Update with dst nil (input and
// actions) and in Draw with the screen (painting), see package ui.
func (g *Game) layout(dst *ebiten.Image) {
	k := g.kit
	modal := g.sheet != "" || g.report != nil

	if dst == nil && !modal && k.Tapped(panoramaRect) {
		x, _, _ := k.Pointer()
		if math.Abs(x+g.scrollX-slotX(0)) < 200 {
			g.act(g.sim.Harvest(g.era.Stations[0].ID))
		}
	}

	g.drawTopBar(dst)
	g.drawGauges(dst)
	k.Fill(dst, ui.Rect{X: 0, Y: panelY, W: ScreenW, H: ScreenH - panelY}, colPanel)

	// Underneath a sheet the panel stays painted but takes no input.
	if !modal || dst != nil {
		g.drawPanelBody(dst)
		g.drawTabs(dst)
	}
	if g.toastT > 0 {
		if dst == nil {
			g.toastT--
		}
		k.Fill(dst, ui.Rect{X: 40, Y: panelY - 76, W: ScreenW - 80, H: 64}, rgba(0x10, 0x0a, 0x04, 0xe0))
		k.Text(dst, g.toast, ScreenW/2, panelY-60, sizeBody, colText, ui.Center, false)
	}
	switch {
	case g.report != nil:
		g.drawAway(dst)
	case g.sheet != "":
		g.drawSheet(dst)
	default:
		g.sheetT = 0
	}
}

func (g *Game) drawTopBar(dst *ebiten.Image) {
	k := g.kit
	k.Fill(dst, ui.Rect{X: 0, Y: 0, W: ScreenW, H: topBarH}, colBar)
	cur := g.era.Currency
	k.Text(dst, sim.Format(g.sim.Stock(cur))+" "+g.resName(cur), padding, 62, sizeHead, colAccent, ui.Left, true)
	k.Text(dst, g.text(g.era.NameKey), ScreenW-padding, 62, sizeHead, colText, ui.Right, true)
}

func (g *Game) drawGauges(dst *ebiten.Image) {
	k := g.kit
	k.Fill(dst, ui.Rect{X: 0, Y: gaugesY, W: ScreenW, H: gaugesH}, colBar)
	c := g.sim.Completion()
	branch := 0.0
	for _, a := range content.Axes {
		branch = math.Max(branch, g.sim.BranchProgress(a))
	}
	gauges := []struct {
		label string
		value float64
	}{
		{g.tr("ui.gauge.branch"), branch},
		{g.tr("ui.gauge.stations"), float64(c.StationsBuilt) / float64(max(1, c.StationsTotal))},
		{g.tr("ui.gauge.quests"), float64(c.QuestsDone) / float64(max(1, c.QuestsTotal))},
	}
	w := (ScreenW - 4*padding) / 3
	for i, it := range gauges {
		x := padding + float64(i)*(w+padding)
		k.Text(dst, it.label, x, gaugesY+6, 22, colText, ui.Left, false)
		k.Fill(dst, ui.Rect{X: x, Y: gaugesY + 40, W: w, H: 12}, rgb(0x4a, 0x3b, 0x2a))
		k.Fill(dst, ui.Rect{X: x, Y: gaugesY + 40, W: w * math.Min(1, it.value), H: 12}, colGood)
	}
}

func (g *Game) drawTabs(dst *ebiten.Image) {
	k := g.kit
	labels := [tabCount]string{g.tr("ui.tab.stations"), g.tr("ui.tab.market"), g.tr("ui.tab.tree"), g.tr("ui.tab.quests"), g.tr("ui.tab.era")}
	w := float64(ScreenW) / tabCount
	for i, label := range labels {
		st := styleSecond
		if i == g.tab {
			st = styleMain
		}
		st.Size = 24
		r := ui.Rect{X: float64(i) * w, Y: tabsY, W: w, H: tabsH}.Inset(4)
		if k.Button(dst, r, label, true, st).Tap {
			g.tab = i
		}
	}
}

func (g *Game) drawPanelBody(dst *ebiten.Image) {
	k := g.kit
	area := ui.Rect{X: 0, Y: panelY, W: ScreenW, H: tabsY - panelY}
	l := k.BeginList(dst, fmt.Sprintf("tab%d", g.tab), area, g.contentH[g.tab])
	defer l.End()
	y := l.Y() + rowGap
	d := l.Target()
	switch g.tab {
	case tabStations:
		y = g.tabStations(d, y)
	case tabMarket:
		y = g.tabMarket(d, y)
	case tabTree:
		y = g.tabTree(d, y)
	case tabQuests:
		y = g.tabQuests(d, y)
	case tabEra:
		y = g.tabEra(d, y)
	}
	g.contentH[g.tab] = y - l.Y() + rowGap
}

// row paints the background of a list row.
func (g *Game) row(d *ebiten.Image, y, h float64) ui.Rect {
	r := ui.Rect{X: padding / 2, Y: y, W: ScreenW - padding, H: h}
	g.kit.Fill(d, r, colRow)
	return r
}

func (g *Game) header(d *ebiten.Image, y float64, s string) float64 {
	g.kit.Text(d, s, padding, y+6, sizeHead, colAccent, ui.Left, true)
	return y + 50
}

// note writes wrapped text in a row and returns the next y.
func (g *Game) note(d *ebiten.Image, s string, y float64, c color.Color) float64 {
	return y + g.kit.Paragraph(d, s, padding*1.5, y, sizeNote, ScreenW-3*padding, c)
}

func (g *Game) noteHeight(s string) float64 {
	return float64(len(g.kit.Wrap(s, sizeNote, ScreenW-3*padding, false))) * sizeNote * 1.3
}

// --- stations -------------------------------------------------------------

func (g *Game) variantOf(def *content.Station, id string) *content.Variant {
	for i := range def.Variants {
		if def.Variants[i].ID == id {
			return &def.Variants[i]
		}
	}
	return nil
}

func (g *Game) tabStations(d *ebiten.Image, y float64) float64 {
	k := g.kit
	for i := range g.era.Stations {
		def := &g.era.Stations[i]
		st := g.sim.Station(def.ID)
		name := g.text(def.NameKey)
		switch {
		case st.Built:
			v := g.variantOf(def, st.Variant)
			status := g.statusText(st)
			h := 96 + btnH + rowGap + g.noteHeight(status)
			r := g.row(d, y, h)
			k.Text(d, name, padding*1.5, y+10, sizeHead, colText, ui.Left, true)
			k.Text(d, fmt.Sprintf("%s %d", g.tr("ui.level"), st.Level), r.X+r.W-padding, y+10, sizeHead, colAccent, ui.Right, true)
			k.Text(d, g.text(v.NameKey), padding*1.5, y+52, sizeNote, colDim, ui.Left, false)
			g.note(d, status, y+84, statusColor(st))
			by := y + h - btnH - rowGap/2
			cost := g.sim.UpgradeCost(def.ID, 1)
			click := k.Button(d, ui.Rect{X: padding * 1.5, Y: by, W: 420, H: btnH}, "+1 : "+g.amountsText(cost), g.sim.CanAfford(cost), shrink(styleMain, g.amountsText(cost)))
			switch {
			case click.Tap:
				g.act(g.sim.Upgrade(def.ID, 1))
			case click.Long:
				// A long press buys as many levels as the stock can pay.
				g.act(g.sim.Upgrade(def.ID, g.sim.MaxUpgrades(def.ID)))
			}
			if k.Button(d, ui.Rect{X: padding*1.5 + 432, Y: by, W: ScreenW - 2*padding*1.5 - 432, H: btnH}, g.tr("ui.sheet"), true, styleSecond).Tap {
				g.sheet = def.ID
			}
			y += h + rowGap
		case g.sim.StationAvailable(def.ID):
			h := 184.0
			if len(def.Harvest) > 0 {
				h += btnH + rowGap
			}
			g.row(d, y, h)
			k.Text(d, name, padding*1.5, y+10, sizeHead, colText, ui.Left, true)
			k.Text(d, g.amountsText(def.UnlockCost), padding*1.5, y+52, sizeNote, colDim, ui.Left, false)
			by := y + 184 - btnH - rowGap/2
			if k.Button(d, ui.Rect{X: padding * 1.5, Y: by, W: ScreenW - 3*padding, H: btnH}, g.tr("ui.build"), g.sim.CanAfford(def.UnlockCost), styleMain).Tap {
				g.sheet = def.ID
			}
			if len(def.Harvest) > 0 {
				if k.Button(d, ui.Rect{X: padding * 1.5, Y: by + btnH + rowGap, W: ScreenW - 3*padding, H: btnH}, g.tr("ui.harvest")+" : +"+g.amountsText(def.Harvest), true, styleSecond).Tap {
					g.act(g.sim.Harvest(def.ID))
				}
			}
			y += h + rowGap
		default:
			g.row(d, y, 96)
			k.Text(d, name, padding*1.5, y+10, sizeHead, colDim, ui.Left, true)
			k.Text(d, g.tr("ui.needs")+" : "+g.nodeName(def.RequiresNode), padding*1.5, y+52, sizeNote, colDim, ui.Left, false)
			y += 96 + rowGap
		}
	}
	return y
}

func shrink(st ui.Style, label string) ui.Style {
	if len(label) > 26 {
		st.Size = 20
	} else if len(label) > 18 {
		st.Size = 23
	}
	return st
}

func (g *Game) statusText(st sim.StationStatus) string {
	switch {
	case st.Retooling > 0:
		return g.tr("ui.retooling") + " : " + sim.FormatDuration(st.Retooling)
	case st.Limiting != "":
		return fmt.Sprintf("%s : %s (%.0f %%)", g.tr("ui.limiting"), g.resName(st.Limiting), st.Speed*100)
	default:
		return fmt.Sprintf("%s : ×%s", g.tr("ui.speed"), sim.Format(st.Cycles))
	}
}

func statusColor(st sim.StationStatus) color.Color {
	if st.Retooling > 0 || st.Limiting != "" {
		return colBad
	}
	return colGood
}

// drawSheet is the sliding sheet of a station: story, variants, retooling.
func (g *Game) drawSheet(dst *ebiten.Image) {
	k := g.kit
	var def *content.Station
	for i := range g.era.Stations {
		if g.era.Stations[i].ID == g.sheet {
			def = &g.era.Stations[i]
		}
	}
	if def == nil {
		g.sheet = ""
		return
	}
	if dst == nil {
		g.sheetT = math.Min(1, g.sheetT+dt*6)
	}
	top := 420 + (1-easeOut(g.sheetT))*(ScreenH-420)
	k.Fill(dst, ui.Rect{X: 0, Y: 0, W: ScreenW, H: ScreenH}, colScrim)
	k.Fill(dst, ui.Rect{X: 0, Y: top, W: ScreenW, H: ScreenH - top}, colSheet)

	st := g.sim.Station(def.ID)
	y := top + 20
	k.Text(dst, g.text(def.NameKey), padding, y, sizeHead+4, colText, ui.Left, true)
	y += 54
	if def.DescKey != "" {
		y += k.Paragraph(dst, g.text(def.DescKey), padding, y, sizeNote, ScreenW-2*padding, colDim) + 12
	}
	cost := def.UnlockCost
	if st.Built {
		cost = g.sim.RetoolCost(def.ID)
	}
	for i := range def.Variants {
		v := &def.Variants[i]
		desc := g.recipeText(v.Recipe)
		if eff := g.effectsText(v.Effects); eff != "" {
			desc += " · " + eff
		}
		lines := k.Wrap(desc, sizeNote, ScreenW-3*padding-210, false)
		h := math.Max(44+float64(len(lines))*sizeNote*1.3+14, btnH+24)
		k.Fill(dst, ui.Rect{X: padding / 2, Y: y, W: ScreenW - padding, H: h}, colRow)
		k.Text(dst, g.text(v.NameKey), padding, y+8, sizeBody, colText, ui.Left, true)
		k.Paragraph(dst, desc, padding, y+44, sizeNote, ScreenW-3*padding-210, colDim)
		bx := ui.Rect{X: ScreenW - padding - 200, Y: y + (h-btnH)/2, W: 200, H: btnH}
		switch {
		case st.Built && st.Variant == v.ID:
			k.Text(dst, g.tr("ui.current"), bx.X+100, bx.Y+22, sizeNote, colGood, ui.Center, true)
		case st.Built:
			if k.Button(dst, bx, g.tr("ui.retool"), g.sim.CanAfford(cost), styleSecond).Tap {
				g.act(g.sim.Retool(def.ID, v.ID))
				g.sheet = ""
			}
		default:
			if k.Button(dst, bx, g.tr("ui.build"), g.sim.CanAfford(cost), styleMain).Tap {
				g.act(g.sim.Build(def.ID, v.ID))
				g.sheet = ""
			}
		}
		y += h + rowGap
	}
	label := g.tr("ui.cost")
	if st.Built {
		label = g.tr("ui.retool")
	}
	k.Text(dst, label+" : "+g.amountsText(cost), padding, y, sizeNote, colDim, ui.Left, false)
	if k.Button(dst, ui.Rect{X: padding, Y: tabsY, W: ScreenW - 2*padding, H: tabsH - 8}, g.tr("ui.close"), true, styleSecond).Tap {
		g.sheet = ""
	}
}

func easeOut(t float64) float64 { return 1 - (1-t)*(1-t)*(1-t) }

func (g *Game) effectsText(effects []content.Effect) string {
	parts := make([]string, 0, len(effects))
	for _, fx := range effects {
		target := ""
		switch {
		case fx.Station == "*" || fx.Resource == "*":
			target = g.tr("ui.fx.all")
		case fx.Station != "":
			target = g.stationName(fx.Station)
		case fx.Resource != "":
			target = g.resName(fx.Resource)
		}
		sign := "+"
		if fx.Kind == content.EffectRetoolDiscount || fx.Kind == content.EffectInputSaving {
			sign = "-"
		}
		s := fmt.Sprintf("%s%.0f %% %s", sign, fx.Value*100, g.tr("ui.fx."+string(fx.Kind)))
		if target != "" {
			s += " " + target
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " · ")
}

// --- market ---------------------------------------------------------------

func (g *Game) tabMarket(d *ebiten.Image, y float64) float64 {
	k := g.kit
	if !g.sim.MarketOpen() {
		g.note(d, g.tr("ui.market.closed"), y, colDim)
		y += 60
		return g.npcDeals(d, y)
	}
	if open := g.sim.Contracts(); len(open) > 0 {
		y = g.header(d, y, g.tr("ui.market.contracts"))
		for _, c := range open {
			g.row(d, y, 104+btnH-40)
			k.Text(d, g.text(c.Template.NameKey), padding*1.5, y+8, sizeBody, colText, ui.Left, true)
			k.Text(d, sim.Format(c.Amount)+" "+g.resName(c.Template.Resource), padding*1.5, y+46, sizeNote, colDim, ui.Left, false)
			k.Text(d, g.tr("ui.time.left")+" "+sim.FormatDuration(c.Left), padding*1.5, y+80, sizeNote, colDim, ui.Left, false)
			label := g.tr("ui.deliver") + " +" + sim.Format(c.Reward)
			if k.Button(d, ui.Rect{X: ScreenW - padding*1.5 - 260, Y: y + 34, W: 260, H: btnH}, label, g.sim.Stock(c.Template.Resource) >= c.Amount, shrink(styleMain, label)).Tap {
				g.act(g.sim.DeliverContract(c.Index))
			}
			y += 144 + rowGap
		}
	}
	y = g.npcDeals(d, y)
	y = g.header(d, y, g.tr("ui.market.resources"))
	for i := range g.era.Resources {
		r := &g.era.Resources[i]
		if r.Kind == content.KindCurrency {
			continue
		}
		stock := g.sim.Stock(r.ID)
		g.row(d, y, 104+btnH)
		arrow := "▲"
		arrowCol := colGood
		if g.sim.Trend(r.ID) < 0 {
			arrow, arrowCol = "▼", colBad
		}
		k.Text(d, g.text(r.NameKey), padding*1.5, y+8, sizeBody, colText, ui.Left, true)
		k.Text(d, sim.Format(g.sim.Price(r.ID))+" "+arrow, ScreenW-padding*1.5, y+8, sizeBody, arrowCol, ui.Right, true)
		rate := g.sim.Rate(r.ID)
		k.Text(d, fmt.Sprintf("%s (%s/s)", sim.Format(stock), signed(rate)), padding*1.5, y+48, sizeNote, colDim, ui.Left, false)
		w := (ScreenW - 3*padding - 2*rowGap) / 3
		by := y + 92
		if k.Button(d, ui.Rect{X: padding * 1.5, Y: by, W: w, H: btnH}, g.tr("ui.sell.quarter"), stock >= 4, shrink(styleSecond, g.tr("ui.sell.quarter"))).Tap {
			_, err := g.sim.Sell(r.ID, stock/4)
			g.act(err)
		}
		if k.Button(d, ui.Rect{X: padding*1.5 + w + rowGap, Y: by, W: w, H: btnH}, g.tr("ui.sell.all"), stock >= 1, shrink(styleSecond, g.tr("ui.sell.all"))).Tap {
			_, err := g.sim.Sell(r.ID, stock)
			g.act(err)
		}
		cost := g.sim.BuyCost(r.ID, 10)
		if k.Button(d, ui.Rect{X: padding*1.5 + 2*(w+rowGap), Y: by, W: w, H: btnH}, g.tr("ui.buy.ten"), g.sim.Stock(g.era.Currency) >= cost, shrink(styleSecond, g.tr("ui.buy.ten"))).Tap {
			g.act(g.sim.Buy(r.ID, 10))
		}
		y += 104 + btnH + rowGap
	}
	return y
}

func signed(v float64) string {
	if v >= 0 {
		return "+" + sim.Format(v)
	}
	return sim.Format(v)
}

func (g *Game) npcDeals(d *ebiten.Image, y float64) float64 {
	k := g.kit
	shown := false
	for i := range g.era.NPCs {
		n := &g.era.NPCs[i]
		if !g.sim.NPCPresent(n.ID) || len(n.Deals)+len(n.Buys) == 0 {
			continue
		}
		if !shown {
			y = g.header(d, y, g.tr("ui.market.npcs"))
			shown = true
		}
		desc := g.text(n.DescKey)
		h := 56 + g.noteHeight(desc) + 8
		for range n.Deals {
			h += btnH + 50
		}
		for range n.Buys {
			h += btnH + rowGap
		}
		g.row(d, y, h)
		k.Text(d, g.text(n.NameKey), padding*1.5, y+8, sizeBody, colText, ui.Left, true)
		k.Text(d, fmt.Sprintf("%s %.0f", g.tr("ui.affinity"), g.sim.Affinity(n.ID)), ScreenW-padding*1.5, y+8, sizeNote, colDim, ui.Right, false)
		cy := g.note(d, desc, y+48, colDim) + 8
		for j := range n.Deals {
			deal := &n.Deals[j]
			give := g.sim.DealGive(n.ID, deal)
			k.Text(d, g.text(deal.NameKey), padding*1.5, cy, sizeNote, colText, ui.Left, false)
			cy += 36
			label := g.amountsText(give) + " → " + g.amountsText(deal.Get)
			if k.Button(d, ui.Rect{X: padding * 1.5, Y: cy, W: ScreenW - 3*padding, H: btnH}, label, g.sim.CanAfford(give), shrink(styleSecond, label)).Tap {
				g.act(g.sim.Trade(n.ID, deal.ID))
			}
			cy += btnH + 14
		}
		for _, id := range sortedBuys(n.Buys) {
			stock := g.sim.Stock(id)
			label := fmt.Sprintf("%s %s ×%.1f", g.tr("ui.sells.to"), g.resName(id), n.Buys[id])
			if k.Button(d, ui.Rect{X: padding * 1.5, Y: cy, W: ScreenW - 3*padding, H: btnH}, label, g.sim.MarketOpen() && stock >= 1, shrink(styleSecond, label)).Tap {
				_, err := g.sim.SellTo(n.ID, id, stock)
				g.act(err)
			}
			cy += btnH + rowGap
		}
		y += h + rowGap
	}
	return y
}

func sortedBuys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- tree -----------------------------------------------------------------

func (g *Game) tabTree(d *ebiten.Image, y float64) float64 {
	k := g.kit
	branches := []string{content.BranchTrunk}
	for _, a := range content.Axes {
		branches = append(branches, string(a))
	}
	for _, b := range branches {
		y = g.header(d, y, g.tr("ui.branch."+b))
		for i := range g.era.Tree {
			n := &g.era.Tree[i]
			if n.Branch != b {
				continue
			}
			owned := g.sim.HasNode(n.ID)
			avail := g.sim.NodeAvailable(n.ID)
			if !owned && !avail {
				// A locked node only shows its name, to keep the list short.
				g.row(d, y, 52)
				k.Text(d, g.text(n.NameKey), padding*1.5, y+10, sizeNote, colDim, ui.Left, false)
				y += 52 + 6
				continue
			}
			eff := g.effectsText(n.Effects)
			cost := g.amountsText(n.Cost)
			width := ScreenW - 3*padding - 190
			tall := 48.0
			lines := k.Wrap(cost, sizeNote, width, false)
			tall += float64(len(lines)) * sizeNote * 1.3
			var effLines []string
			if eff != "" {
				effLines = k.Wrap(eff, sizeNote, width, false)
				tall += float64(len(effLines)) * sizeNote * 1.3
			}
			h := math.Max(tall+12, btnH+24)
			g.row(d, y, h)
			title := g.text(n.NameKey)
			titleCol := colText
			if n.EraKey {
				title = "♦ " + title
				titleCol = colAccent
			}
			k.Text(d, title, padding*1.5, y+8, sizeBody, titleCol, ui.Left, true)
			cy := y + 46
			if eff != "" {
				cy += k.Paragraph(d, eff, padding*1.5, cy, sizeNote, width, colGood)
			}
			k.Paragraph(d, cost, padding*1.5, cy, sizeNote, width, colDim)
			bx := ui.Rect{X: ScreenW - padding*1.5 - 180, Y: y + (h-btnH)/2, W: 180, H: btnH}
			if owned {
				k.Text(d, g.tr("ui.bought"), bx.X+90, bx.Y+22, sizeNote, colGood, ui.Center, true)
			} else if k.Button(d, bx, g.tr("ui.buy"), g.sim.CanAfford(n.Cost), styleMain).Tap {
				g.act(g.sim.BuyNode(n.ID))
			}
			y += h + rowGap
		}
		y += rowGap
	}
	return y
}

// --- quests ---------------------------------------------------------------

func (g *Game) tabQuests(d *ebiten.Image, y float64) float64 {
	active := g.sim.ActiveQuests()
	if len(active) == 0 {
		g.note(d, g.tr("ui.quests.none"), y, colDim)
		return y + 60
	}
	y = g.header(d, y, g.tr("ui.quests.main"))
	for _, q := range active {
		if q.Main {
			y = g.questRow(d, y, q)
		}
	}
	first := true
	for _, q := range active {
		if q.Main {
			continue
		}
		if first {
			y = g.header(d, y+rowGap, g.tr("ui.quests.optional"))
			first = false
		}
		y = g.questRow(d, y, q)
	}
	return y
}

func (g *Game) questRow(d *ebiten.Image, y float64, q *content.Quest) float64 {
	k := g.kit
	step := g.sim.CurrentStep(q)
	body := g.text(step.TextKey)
	goal := g.goalText(step.Goal)
	h := 54 + g.noteHeight(body) + g.noteHeight(goal) + 12
	left, running := g.sim.QuestDeadline(q.ID)
	if running {
		h += 34
	}
	canDeliver := g.sim.CanDeliver(q)
	needs := goalDelivers(step.Goal)
	if needs {
		h += btnH + rowGap
	}
	g.row(d, y, h)
	title := g.text(q.NameKey)
	if len(q.Steps) > 1 {
		title += fmt.Sprintf(" (%d/%d)", g.sim.QuestStep(q.ID)+1, len(q.Steps))
	}
	k.Text(d, title, padding*1.5, y+8, sizeBody, colText, ui.Left, true)
	cy := g.note(d, body, y+48, colDim)
	cy = g.note(d, goal, cy+4, colGood)
	if running {
		k.Text(d, g.tr("ui.time.left")+" "+sim.FormatDuration(left), padding*1.5, cy, sizeNote, colBad, ui.Left, false)
		cy += 34
	}
	if needs {
		if k.Button(d, ui.Rect{X: padding * 1.5, Y: cy + 6, W: ScreenW - 3*padding, H: btnH}, g.tr("ui.deliver"), canDeliver, styleMain).Tap {
			g.act(g.sim.Deliver(q.ID))
		}
	}
	return y + h + rowGap
}

func goalDelivers(goal content.Goal) bool {
	if len(goal.Any) > 0 {
		return slices.ContainsFunc(goal.Any, goalDelivers)
	}
	return len(goal.Deliver) > 0
}

// goalText says what a goal still asks for, with the current progress.
func (g *Game) goalText(goal content.Goal) string {
	if len(goal.Any) > 0 {
		parts := make([]string, 0, len(goal.Any))
		for _, alt := range goal.Any {
			parts = append(parts, g.goalText(alt))
		}
		return strings.Join(parts, " "+g.tr("ui.goal.or")+" ")
	}
	var parts []string
	progress := func(verb string, a content.Amounts, cur func(string) float64) {
		for _, id := range sortedAmounts(a) {
			parts = append(parts, fmt.Sprintf("%s %s / %s %s", verb, sim.Format(math.Min(cur(id), a[id])), sim.Format(a[id]), g.resName(id)))
		}
	}
	progress(g.tr("ui.goal.have"), goal.Have, g.sim.Stock)
	progress(g.tr("ui.goal.deliver"), goal.Deliver, g.sim.Stock)
	progress(g.tr("ui.goal.produced"), goal.Produced, g.sim.Produced)
	if goal.Earned > 0 {
		parts = append(parts, fmt.Sprintf("%s %s / %s %s", g.tr("ui.goal.earned"), sim.Format(math.Min(g.sim.State().Earned, goal.Earned)), sim.Format(goal.Earned), g.resName(g.era.Currency)))
	}
	for _, id := range goal.Stations {
		parts = append(parts, g.tr("ui.goal.station")+" "+g.stationName(id))
	}
	levels := make([]string, 0, len(goal.Levels))
	for id := range goal.Levels {
		levels = append(levels, id)
	}
	sort.Strings(levels)
	for _, id := range levels {
		parts = append(parts, fmt.Sprintf("%s %s : %d / %d", g.tr("ui.goal.level"), g.stationName(id), g.sim.Station(id).Level, goal.Levels[id]))
	}
	for _, id := range goal.Nodes {
		parts = append(parts, g.tr("ui.goal.node")+" "+g.nodeName(id))
	}
	return strings.Join(parts, " · ")
}

// --- era ------------------------------------------------------------------

func (g *Game) tabEra(d *ebiten.Image, y float64) float64 {
	k := g.kit
	c := g.sim.Completion()
	y = g.header(d, y, g.tr("ui.era.conditions"))
	g.row(d, y, 140)
	keyMark := func(ok bool) (string, color.Color) {
		if ok {
			return "● ", colGood
		}
		return "○ ", colDim
	}
	lines := []struct {
		label string
		ok    bool
	}{
		{g.tr("ui.era.keys"), len(c.Keys) > 0},
		{fmt.Sprintf("%s : %d / %d", g.tr("ui.gauge.stations"), c.StationsBuilt, c.StationsTotal), c.StationsBuilt == c.StationsTotal},
		{fmt.Sprintf("%s : %d / %d", g.tr("ui.gauge.quests"), c.QuestsDone, c.QuestsTotal), c.QuestsDone == c.QuestsTotal},
	}
	for i, l := range lines {
		mark, col := keyMark(l.ok)
		k.Text(d, mark+l.label, padding*1.5, y+10+float64(i)*42, sizeBody, col, ui.Left, false)
	}
	y += 140 + rowGap

	if end := g.sim.Ending(); end != "" {
		g.note(d, g.tr("ui.era.ended"), y, colAccent)
		y += 60
	}
	y = g.header(d, y, g.tr("ui.era.choose"))
	for i := range g.era.Endings {
		e := &g.era.Endings[i]
		owned := slices.Contains(c.Keys, e.Axis)
		desc := g.text(e.DescKey)
		patent := g.tr("ui.era.patent") + " " + g.text(e.PatentKey) + " : " + g.text(e.PatentDescKey)
		h := 56 + g.noteHeight(desc) + g.noteHeight(patent) + 12
		portal := owned && c.Complete() && g.sim.Ending() == ""
		if portal {
			h += btnH + rowGap
		}
		g.row(d, y, h)
		col := colDim
		if owned {
			col = colAccent
		}
		k.Text(d, g.text(e.NameKey), padding*1.5, y+8, sizeBody, col, ui.Left, true)
		k.Text(d, g.tr("ui.branch."+string(e.Axis)), ScreenW-padding*1.5, y+10, sizeNote, colDim, ui.Right, false)
		cy := g.note(d, desc, y+50, colDim)
		cy = g.note(d, patent, cy+4, colText)
		if portal {
			if k.Button(d, ui.Rect{X: padding * 1.5, Y: cy + 6, W: ScreenW - 3*padding, H: btnH}, g.tr("ui.era.portal"), true, styleMain).Tap {
				g.act(g.sim.Finish(e.Axis))
			}
		}
		y += h + rowGap
	}
	y += rowGap
	if k.Button(d, ui.Rect{X: padding, Y: y, W: ScreenW - 2*padding, H: btnH}, g.tr("ui.era.language"), true, styleSecond).Tap {
		g.locale = map[string]string{"fr": "en", "en": "fr"}[g.locale]
	}
	y += btnH + rowGap
	if k.Button(d, ui.Rect{X: padding, Y: y, W: ScreenW - 2*padding, H: btnH}, g.tr("ui.era.reset"), true, styleSecond).Long {
		g.reset()
	}
	return y + btnH + rowGap
}

// --- absence --------------------------------------------------------------

// drawAway is the "while you were away" screen.
func (g *Game) drawAway(dst *ebiten.Image) {
	k := g.kit
	rep := g.report
	k.Fill(dst, ui.Rect{X: 0, Y: 0, W: ScreenW, H: ScreenH}, colScrim)
	top := 360.0
	k.Fill(dst, ui.Rect{X: 30, Y: top, W: ScreenW - 60, H: ScreenH - 2*top + 100}, colSheet)
	y := top + 24
	k.Text(dst, g.tr("ui.away.title"), ScreenW/2, y, sizeHead+4, colAccent, ui.Center, true)
	y += 58
	k.Text(dst, fmt.Sprintf("%s %s · %s %s", g.tr("ui.away.time"), sim.FormatDuration(rep.Away), g.tr("ui.away.counted"), sim.FormatDuration(rep.Counted)), ScreenW/2, y, sizeNote, colDim, ui.Center, false)
	y += 50
	ids := make([]string, 0, len(rep.Gains))
	for id, q := range rep.Gains {
		if q > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		k.Paragraph(dst, g.tr("ui.away.nothing"), 60, y, sizeBody, ScreenW-120, colText)
	}
	for _, id := range ids {
		if y > ScreenH-top-50 {
			break
		}
		k.Text(dst, g.resName(id), 60, y, sizeBody, colText, ui.Left, false)
		k.Text(dst, "+"+sim.Format(rep.Gains[id]), ScreenW-60, y, sizeBody, colGood, ui.Right, true)
		y += 42
	}
	if k.Button(dst, ui.Rect{X: 60, Y: ScreenH - top - 10, W: ScreenW - 120, H: btnH + 8}, g.tr("ui.away.ok"), true, styleMain).Tap {
		g.report = nil
	}
}
