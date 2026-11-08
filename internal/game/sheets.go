package game

import (
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
	sheetPad  = 46.0
	bodySize  = 24.0
	smallSize = 21.0
	titleSize = 26.0
	btnH      = 84.0 // at least 44 points on a phone
	gap       = 14.0
)

// drawSheet is the sliding sheet over the world: a title, a close button and
// a scrolling body that depends on the screen.
func (g *Game) drawSheet(dst *ebiten.Image) {
	k := g.kit
	if dst == nil {
		g.sheetT = math.Min(1, g.sheetT+dt*6)
	}
	title, sub := g.sheetTitle()
	headH := 196.0
	// A station opens as a modal in the middle of the screen; the other
	// screens slide up from the bottom.
	modal := g.screen == scrStation || g.screen == scrResource
	var frame ui.Rect
	if modal {
		maxH := float64(ScreenH) - safeTop - safeBottom - 20
		h := math.Min(maxH, headH+g.sheetH+40)
		top := safeTop + 10 + (maxH-h)/2 + (1-easeOut(g.sheetT))*80
		frame = ui.Rect{X: 16, Y: top, W: ScreenW - 32, H: h}
	} else {
		h := math.Min(ScreenH*0.8, headH+g.sheetH+safeBottom+50)
		top := ScreenH - h + (1-easeOut(g.sheetT))*h
		frame = ui.Rect{X: 0, Y: top, W: ScreenW, H: h + 60}
	}
	top := frame.Y
	k.Fill(dst, ui.Rect{X: 0, Y: 0, W: ScreenW, H: ScreenH}, scrimCol)
	if dst == nil && g.sheetT >= 1 && k.Tapped(ui.Rect{X: 0, Y: 0, W: ScreenW, H: ScreenH}) && !frame.Contains(g.pressPoint()) {
		g.closeScreen()
		return
	}
	g.panel(dst, frame, mk(24))
	x, w := sheetPad, ScreenW-2*sheetPad
	if modal {
		x, w = frame.X+sheetPad-12, frame.W-2*(sheetPad-12)
	}
	k.Shadowed(dst, k.Fit(title, 36, w-100, true), x, top+40, 36, inkLight, ui.Left, true)
	if sub != "" {
		k.Shadowed(dst, k.Fit(sub, smallSize, w-100, false), x, top+88, smallSize, inkSoft, ui.Left, false)
	}
	if g.roundButton(dst, ui.Rect{X: x + w - 72, Y: top + 34, W: 72, H: 72}, g.tr("ui.close"), "×").Tap {
		g.closeScreen()
		return
	}
	g.separator(dst, x-10, top+112, w+20)
	bottom := float64(ScreenH) - safeBottom - 10
	if modal {
		bottom = frame.Y + frame.H - 30
	}
	area := ui.Rect{X: 0, Y: top + headH, W: ScreenW, H: bottom - top - headH}
	l := k.BeginList(dst, "sheet", area, g.sheetH)
	d := l.Target()
	y0 := l.Y()
	g.bodyY0 = y0
	var y float64
	switch g.screen {
	case scrStation:
		y = g.stationBody(d, x, y0, w)
	case scrResource:
		y = g.resourceBody(d, x, y0, w)
	case scrMarket:
		y = g.marketBody(d, x, y0, w)
	case scrQuests:
		y = g.questsBody(d, x, y0, w)
	case scrShop:
		y = y0 + k.ShadowedParagraph(d, g.tr("ui.shop.text"), x, y0, bodySize, w, inkLight, false)
	case scrClan:
		y = g.clanBody(d, x, y0, w)
	case scrEra:
		y = g.eraBody(d, x, y0, w)
	}
	l.End()
	g.sheetH = y - y0 + gap
}

func (g *Game) sheetTitle() (string, string) {
	switch g.screen {
	case scrStation:
		def := g.stationDef(g.sheet)
		if def == nil {
			return "", ""
		}
		st := g.sim.Station(def.ID)
		if !st.Built {
			return g.text(def.NameKey), g.tr("ui.station.notBuilt")
		}
		return g.text(def.NameKey), fmt.Sprintf("%s %d · %s", g.tr("ui.level"), st.Level, g.outputsText(def, st))
	case scrResource:
		return g.resName(g.sheet), g.tr("ui.res.subtitle")
	case scrMarket:
		return g.tr("ui.act.market"), g.tr("ui.market.hint")
	case scrQuests:
		c := g.sim.Completion()
		return g.tr("ui.quests.title"), fmt.Sprintf(fmt.Sprintf(g.tr("ui.era.index"), g.era.Index)+" · "+g.tr("ui.quests.progress"), c.QuestsDone, c.QuestsTotal)
	case scrShop:
		return g.tr("ui.act.shop"), ""
	case scrClan:
		return g.tr("ui.act.clan"), ""
	case scrEra:
		return g.tr("ui.era.title"), fmt.Sprintf(g.tr("ui.era.index"), g.era.Index) + " · " + g.text(g.era.NameKey)
	}
	return "", ""
}

// outputsText is what a station makes per second right now.
func (g *Game) outputsText(def *content.Station, st sim.StationStatus) string {
	v := g.variantOf(def, st.Variant)
	if v == nil {
		return ""
	}
	parts := []string{}
	for _, id := range sortedAmounts(v.Recipe.Outputs) {
		parts = append(parts, fmt.Sprintf("%s %s/s", sim.Format(v.Recipe.Outputs[id]*st.Cycles*st.Speed), strings.ToLower(g.resName(id))))
	}
	return strings.Join(parts, " · ")
}

// card paints a carved inset and returns its inner area.
func (g *Game) card(d *ebiten.Image, x, y, w, h float64) ui.Rect {
	g.insetBox(d, ui.Rect{X: x, Y: y, W: w, H: h}, mk(12))
	return ui.Rect{X: x + 24, Y: y + 20, W: w - 48, H: h - 40}
}

func (g *Game) paraH(s string, size, w float64) float64 {
	return float64(len(g.kit.Wrap(s, size, w, false))) * size * 1.3
}

// quip writes a humor line with an accent bar on its left.
func (g *Game) quip(d *ebiten.Image, s string, x, y, w float64) float64 {
	h := g.paraH(s, smallSize+1, w-20)
	g.kit.Fill(d, ui.Rect{X: x, Y: y, W: 5, H: h}, rgb(0xe9, 0x78, 0x2a))
	g.kit.ShadowedParagraph(d, "« "+s+" »", x+18, y, smallSize+1, w-20, inkSoft, false)
	return y + h + gap
}

// statRow writes a label on the left and a value on the right.
func (g *Game) statRow(d *ebiten.Image, label, value string, x, y, w float64, col color.Color) float64 {
	g.kit.Shadowed(d, label, x, y, bodySize, inkSoft, ui.Left, false)
	g.kit.Shadowed(d, g.kit.Fit(value, bodySize, w-g.kit.Width(label, bodySize, false)-24, true), x+w, y, bodySize, col, ui.Right, true)
	return y + bodySize*1.6
}

// --- station --------------------------------------------------------------

func (g *Game) stationBody(d *ebiten.Image, x, y, w float64) float64 {
	k := g.kit
	def := g.stationDef(g.sheet)
	if def == nil {
		return y
	}
	st := g.sim.Station(def.ID)
	y = g.stationPicture(d, def, st, x, y, w)
	if def.DescKey != "" {
		y = g.quip(d, g.text(def.DescKey), x, y, w)
	}
	if !st.Built && !g.sim.StationAvailable(def.ID) {
		y = g.statRow(d, g.tr("ui.needs"), g.nodeName(def.RequiresNode)+" ("+g.tr("ui.act.tree")+")", x, y, w, inkGold)
		k.Shadowed(d, g.tr("ui.cost"), x, y, bodySize, inkSoft, ui.Left, false)
		return y + g.drawChips(d, g.costChips(def.UnlockCost), x+140, y-4, w-140, bodySize, false, inkLight) + gap
	}
	if st.Built {
		col := inkGood
		if st.Retooling > 0 || st.Limiting != "" {
			col = inkBad
		}
		k.Shadowed(d, g.statusText(st), x, y, smallSize, col, ui.Left, false)
		y += smallSize * 1.6
		cost := g.sim.UpgradeCost(def.ID, 1)
		label := fmt.Sprintf(g.tr("ui.upgrade"), st.Level+1)
		click := g.costButton(d, ui.Rect{X: x, Y: y, W: w, H: costBtnH}, "upgrade", label, g.costChips(cost), g.sim.CanAfford(cost))
		switch {
		case click.Tap:
			g.act(g.sim.Upgrade(def.ID, 1))
		case click.Long:
			// A long press buys as many levels as the stock can pay.
			g.act(g.sim.Upgrade(def.ID, g.sim.MaxUpgrades(def.ID)))
		}
		y += costBtnH + gap + 6
		y = g.statRow(d, g.tr("ui.station.spec"), g.text(g.variantOf(def, st.Variant).NameKey), x, y, w, inkLight)
	}
	pick := g.pick
	if pick == "" {
		pick = st.Variant
		if pick == "" {
			pick = def.Variants[0].ID
		}
	}
	// Specializations: one tile each.
	n := float64(len(def.Variants))
	tw := (w - (n-1)*10) / n
	th := 0.0
	for i := range def.Variants {
		th = math.Max(th, 40+g.paraH(g.text(def.Variants[i].NameKey), smallSize, tw-36)+g.drawChips(nil, recipeChips(def.Variants[i].Recipe, 1), 0, 0, tw-36, 19, false, inkSoft))
	}
	th += 20
	for i := range def.Variants {
		v := &def.Variants[i]
		r := ui.Rect{X: x + float64(i)*(tw+10), Y: y, W: tw, H: th}
		if c, _ := k.Control(d, r, "spec "+v.ID, true); c.Tap {
			g.pick = v.ID
		}
		if d != nil {
			if v.ID == pick {
				k.Fill(d, r.Inset(-3), rgba(0xff, 0xd3, 0x6b, 0x70))
			}
			g.insetBox(d, r, mk(9))
			cy := r.Y + 18 + k.ShadowedParagraph(d, g.text(v.NameKey), r.X+18, r.Y+18, smallSize, tw-36, inkLight, true)
			col := inkSoft
			if v.ID == st.Variant {
				col = inkGood
			}
			g.drawChips(d, recipeChips(v.Recipe, 1), r.X+18, cy+4, tw-36, 19, false, col)
		}
	}
	y += th + gap
	if v := g.variantOf(def, pick); v != nil && len(v.Effects) > 0 {
		y += k.ShadowedParagraph(d, g.effectsText(v.Effects, 1), x, y, smallSize, w, inkGood, false) + 6
	}
	if !st.Built {
		if g.costButton(d, ui.Rect{X: x, Y: y, W: w, H: costBtnH}, g.tr("ui.build"), g.tr("ui.build"), g.costChips(def.UnlockCost), g.sim.CanAfford(def.UnlockCost)).Tap {
			g.act(g.sim.Build(def.ID, pick))
			g.pick = ""
		}
		return y + costBtnH + gap
	}
	k.ShadowedParagraph(d, g.tr("ui.station.retoolHint"), x, y, 19, w, inkSoft, false)
	y += g.paraH(g.tr("ui.station.retoolHint"), 19, w) + gap
	if pick != st.Variant {
		cost := g.sim.RetoolCost(def.ID)
		if g.costButton(d, ui.Rect{X: x, Y: y, W: w, H: costBtnH}, "retool", g.tr("ui.retool"), g.costChips(cost), g.sim.CanAfford(cost)).Tap {
			g.act(g.sim.Retool(def.ID, pick))
			g.pick = ""
		}
		y += costBtnH + gap
	}
	return y
}

// stationPicture shows the building in a little window of sky: large, and
// faded when it is not built yet.
func (g *Game) stationPicture(d *ebiten.Image, def *content.Station, st sim.StationStatus, x, y, w float64) float64 {
	const h = 330.0
	if d != nil {
		box := ui.Rect{X: x, Y: y, W: w, H: h}
		g.insetBox(d, box, mk(12))
		in := box.Inset(16)
		sub := d.SubImage(rectI(in)).(*ebiten.Image)
		g.kit.Fill(sub, in, rgb(0x9f, 0xc4, 0xcf))
		g.kit.Fill(sub, ui.Rect{X: in.X, Y: in.Y + in.H*0.45, W: in.W, H: in.H * 0.55}, rgb(0xe6, 0xd3, 0xa4))
		ground := in.Y + in.H - 46
		g.kit.Fill(sub, ui.Rect{X: in.X, Y: ground, W: in.W, H: in.H}, rgb(0x8a, 0x77, 0x46))
		cx := in.X + in.W/2 + 40
		first := def.ID == g.era.Stations[0].ID
		withPen(0.72, ground, 1, func() {
			switch {
			case first:
				g.drawBush(sub, cx-40)
			case st.Built:
				g.drawWorkshop(sub, cx, st.Speed)
			default:
				g.drawEmptySlot(sub, cx-40)
				withPen(0.72, ground, 0.35, func() { g.drawWorkshop(sub, cx, 0) })
			}
		})
		if st.Built {
			label := fmt.Sprintf("%s %d", g.tr("ui.level"), st.Level)
			pw := g.kit.Width(label, 22, true) + 50
			pr := ui.Rect{X: box.X + 26, Y: box.Y + 24, W: pw, H: 46}
			g.plaque(d, pr)
			g.kit.Shadowed(d, label, pr.X+pr.W/2, pr.Y+10, 22, inkGold, ui.Center, true)
		}
	}
	return y + h + gap
}

// --- market ---------------------------------------------------------------

func (g *Game) groupTitle(d *ebiten.Image, s string, x, y float64) float64 {
	g.kit.Shadowed(d, strings.ToUpper(s), x, y+6, 19, inkSoft, ui.Left, true)
	return y + 42
}

func (g *Game) marketBody(d *ebiten.Image, x, y, w float64) float64 {
	k := g.kit
	if !g.sim.MarketOpen() {
		y += k.ShadowedParagraph(d, g.tr("ui.market.closed"), x, y, bodySize, w, inkSoft, false) + gap
		return g.npcDeals(d, x, y, w)
	}
	// What you have comes first, then the orders, then the merchants.
	y = g.groupTitle(d, g.tr("ui.market.resources"), x, y)
	for i := range g.era.Resources {
		r := &g.era.Resources[i]
		if r.Kind == content.KindCurrency || !g.sim.ResourceUnlocked(r.ID) {
			continue
		}
		y = g.marketCard(d, r, x, y, w)
	}
	if open := g.sim.Contracts(); len(open) > 0 {
		y = g.groupTitle(d, g.tr("ui.market.contracts"), x, y)
		for _, c := range open {
			h := 40 + 3*bodySize*1.4 + btnH
			in := g.card(d, x, y, w, h)
			k.Shadowed(d, k.Fit(g.text(c.Template.NameKey), titleSize, in.W, true), in.X, in.Y, titleSize, inkLight, ui.Left, true)
			k.Shadowed(d, sim.Format(c.Amount)+" "+strings.ToLower(g.resName(c.Template.Resource))+" · "+g.npcName(c.Template.NPC), in.X, in.Y+38, smallSize, inkSoft, ui.Left, false)
			k.Shadowed(d, g.tr("ui.time.left")+" "+sim.FormatDuration(c.Left), in.X, in.Y+68, smallSize, inkBad, ui.Left, true)
			label := g.tr("ui.deliver") + " · +" + sim.Format(c.Reward) + " " + strings.ToLower(g.resName(g.era.Currency))
			if g.woodButton(d, ui.Rect{X: in.X, Y: in.Y + 104, W: in.W, H: btnH}, label, g.sim.Stock(c.Template.Resource) >= c.Amount, 23).Tap {
				g.act(g.sim.DeliverContract(c.Index))
			}
			y += h + gap
		}
	}
	return g.npcDeals(d, x, y, w)
}

// marketCard is one resource on the market: what it is, how much you have,
// where it comes from, and the trades with what each brings. Tapping the top
// of the card opens the resource sheet.
func (g *Game) marketCard(d *ebiten.Image, r *content.Resource, x, y, w float64) float64 {
	k := g.kit
	h := 40 + 160 + costBtnH
	in := g.card(d, x, y, w, h)
	info := ui.Rect{X: in.X - 20, Y: in.Y - 16, W: in.W + 40, H: 150}
	if c, held := k.Control(d, info, "res-info "+r.ID, true); c.Tap {
		g.openResource(r.ID)
	} else if held {
		k.Fill(d, info, rgba(0xff, 0xff, 0xff, 0x18))
	}
	g.resIcon(d, r.ID, in.X+24, in.Y+18, 24)
	k.Shadowed(d, g.text(r.NameKey), in.X+60, in.Y, titleSize, inkLight, ui.Left, true)
	k.Shadowed(d, g.tr("ui.res.details"), in.X+in.W, in.Y+4, smallSize, inkSoft, ui.Right, false)
	stock := g.sim.Stock(r.ID)
	rate := g.sim.Rate(r.ID)
	have := fmt.Sprintf("%s %s", g.tr("ui.res.have"), sim.Format(stock))
	if rate > 1e-6 || rate < -1e-6 {
		have += fmt.Sprintf(" (%s/s)", signed(rate))
	}
	k.Shadowed(d, have, in.X, in.Y+48, bodySize, inkLight, ui.Left, true)
	src, ok := g.sourceLine(r.ID)
	col := inkSoft
	if !ok {
		col = inkGold
	}
	k.Shadowed(d, k.Fit(src, smallSize, in.W, false), in.X, in.Y+86, smallSize, col, ui.Left, false)
	g.priceLine(d, r.ID, in.X, in.Y+118, in.W)
	g.tradeButtons(d, r.ID, in.X, in.Y+160, in.W)
	return y + h + gap
}

func (g *Game) npcDeals(d *ebiten.Image, x, y, w float64) float64 {
	k := g.kit
	shown := false
	for i := range g.era.NPCs {
		n := &g.era.NPCs[i]
		if !g.sim.NPCPresent(n.ID) || len(n.Deals)+len(n.Buys) == 0 {
			continue
		}
		if !shown {
			y = g.groupTitle(d, g.tr("ui.market.npcs"), x, y)
			shown = true
		}
		h := 40 + 44 + float64(len(n.Deals))*(costBtnH+44) + float64(len(n.Buys))*(costBtnH+10)
		in := g.card(d, x, y, w, h)
		k.Shadowed(d, g.text(n.NameKey), in.X, in.Y, titleSize, inkLight, ui.Left, true)
		k.Shadowed(d, fmt.Sprintf("%s %.0f", g.tr("ui.affinity"), g.sim.Affinity(n.ID)), in.X+in.W, in.Y+4, smallSize, inkSoft, ui.Right, false)
		cy := in.Y + 44
		for j := range n.Deals {
			deal := &n.Deals[j]
			give := g.sim.DealGive(n.ID, deal)
			k.Shadowed(d, k.Fit(g.text(deal.NameKey), smallSize, in.W, false), in.X, cy, smallSize, inkSoft, ui.Left, false)
			cy += 36
			chips := append(g.costChips(give), chip{text: "→"})
			chips = append(chips, amountChips(deal.Get, true)...)
			if g.costButton(d, ui.Rect{X: in.X, Y: cy, W: in.W, H: costBtnH}, "deal "+n.ID+" "+deal.ID, g.tr("ui.trade"), chips, g.sim.CanAfford(give)).Tap {
				g.act(g.sim.Trade(n.ID, deal.ID))
			}
			cy += costBtnH + 8
		}
		for _, id := range sortedBuys(n.Buys) {
			stock := g.sim.Stock(id)
			label := fmt.Sprintf(g.tr("ui.sells.toAll"), strings.ToLower(g.resName(id)), n.Buys[id])
			chips := []chip{{id: id, qty: stock}, {text: "→"}, {id: g.era.Currency, qty: g.sim.SellToValue(n.ID, id, stock), plus: true}}
			if g.costButton(d, ui.Rect{X: in.X, Y: cy, W: in.W, H: costBtnH}, "sell-to "+n.ID+" "+id, label, chips, g.sim.MarketOpen() && stock >= 1).Tap {
				_, err := g.sim.SellTo(n.ID, id, stock)
				g.act(err)
			}
			cy += costBtnH + 10
		}
		y += h + gap
	}
	return y
}

// --- quests ---------------------------------------------------------------

func (g *Game) questsBody(d *ebiten.Image, x, y, w float64) float64 {
	active := g.sim.ActiveQuests()
	if len(active) == 0 {
		return y + g.kit.ShadowedParagraph(d, g.tr("ui.quests.none"), x, y, bodySize, w, inkSoft, false)
	}
	byProgress := func(qs []*content.Quest) []*content.Quest {
		sort.SliceStable(qs, func(i, j int) bool {
			a, _ := g.questProgress(qs[i])
			b, _ := g.questProgress(qs[j])
			return a > b
		})
		return qs
	}
	var main, side []*content.Quest
	for _, q := range active {
		if q.Main {
			main = append(main, q)
		} else {
			side = append(side, q)
		}
	}
	if len(main) > 0 {
		y = g.groupTitle(d, g.tr("ui.quests.main"), x, y)
		for _, q := range byProgress(main) {
			y = g.questCard(d, q, x, y, w)
		}
	}
	if len(side) > 0 {
		y = g.groupTitle(d, g.tr("ui.quests.optional"), x, y+4)
		for _, q := range byProgress(side) {
			y = g.questCard(d, q, x, y, w)
		}
	}
	return y
}

func (g *Game) questCard(d *ebiten.Image, q *content.Quest, x, y, w float64) float64 {
	k := g.kit
	step := g.sim.CurrentStep(q)
	body := g.text(step.TextKey)
	goal := g.goalText(step.Goal)
	iw := w - 48
	h := 40 + 36 + g.paraH(body, smallSize, iw) + 6 + g.paraH(goal, 19, iw) + 8 + 30 + 34
	left, running := g.sim.QuestDeadline(q.ID)
	if running {
		h += 32
	}
	needs := goalDelivers(step.Goal)
	if needs {
		h += btnH + 10
	}
	can := g.sim.CanDeliver(q)
	focus := q.ID == g.focusQuest
	if focus && d == nil && g.focusPending && g.sheetT >= 0.6 {
		// Bring the quest picked on the main screen to the top of the journal.
		g.kit.ScrollTo("sheet", y-g.bodyY0-gap)
		g.focusPending = false
	}
	switch {
	case d == nil:
	case focus:
		a := 0.55 + 0.45*math.Sin(g.t*5)
		k.Fill(d, ui.Rect{X: x - 7, Y: y - 7, W: w + 14, H: h + 14}, rgba(0xff, 0xd3, 0x6b, uint8(90+110*a)))
	case needs && can:
		k.Fill(d, ui.Rect{X: x - 4, Y: y - 4, W: w + 8, H: h + 8}, rgba(0xf2, 0xc1, 0x4e, 0x70))
	}
	in := g.card(d, x, y, w, h)
	who := ""
	if q.NPC != "" {
		who = g.npcName(q.NPC)
	}
	k.Shadowed(d, k.Fit(g.text(q.NameKey), titleSize, in.W-k.Width(who, 19, false)-20, true), in.X, in.Y, titleSize, inkLight, ui.Left, true)
	k.Shadowed(d, who, in.X+in.W, in.Y+5, 19, inkSoft, ui.Right, false)
	cy := in.Y + 36
	cy += k.ShadowedParagraph(d, body, in.X, cy, smallSize, in.W, inkSoft, false) + 6
	cy += k.ShadowedParagraph(d, goal, in.X, cy, 19, in.W, inkGood, false) + 8
	if running {
		k.Shadowed(d, g.tr("ui.time.left")+" "+sim.FormatDuration(left), in.X, cy, smallSize, inkBad, ui.Left, true)
		cy += 32
	}
	v, count := g.questProgress(q)
	g.progress(d, ui.Rect{X: in.X, Y: cy, W: in.W, H: 26}, v)
	cy += 30
	k.Shadowed(d, count, in.X, cy, 19, inkSoft, ui.Left, false)
	if len(q.Reward) > 0 {
		k.Shadowed(d, k.Fit("+"+g.amountsText(q.Reward), 19, in.W*0.6, false), in.X+in.W, cy, 19, inkGold, ui.Right, false)
	}
	cy += 34
	if needs {
		if g.woodBtn(d, ui.Rect{X: in.X, Y: cy + 4, W: in.W, H: btnH}, "deliver "+q.ID, g.tr("ui.deliver"), can, 24).Tap {
			g.act(g.sim.Deliver(q.ID))
		}
	}
	return y + h + gap
}

// --- clan -----------------------------------------------------------------

func (g *Game) clanBody(d *ebiten.Image, x, y, w float64) float64 {
	k := g.kit
	y = g.groupTitle(d, g.tr("ui.clan.npcs"), x, y)
	for i := range g.era.NPCs {
		n := &g.era.NPCs[i]
		desc := g.text(n.DescKey)
		h := 40 + 40 + g.paraH(desc, smallSize, w-48)
		in := g.card(d, x, y, w, h)
		col := inkLight
		if !g.sim.NPCPresent(n.ID) {
			col = inkSoft
		}
		k.Shadowed(d, g.text(n.NameKey), in.X, in.Y, titleSize, col, ui.Left, true)
		k.Shadowed(d, fmt.Sprintf("%s %.0f", g.tr("ui.affinity"), g.sim.Affinity(n.ID)), in.X+in.W, in.Y+4, smallSize, inkSoft, ui.Right, false)
		k.ShadowedParagraph(d, desc, in.X, in.Y+40, smallSize, in.W, inkSoft, false)
		y += h + gap
	}
	y = g.groupTitle(d, g.tr("ui.clan.settings"), x, y+4)
	if g.woodButton(d, ui.Rect{X: x, Y: y, W: w, H: btnH}, g.tr("ui.era.language"), true, 24).Tap {
		g.locale = map[string]string{"fr": "en", "en": "fr"}[g.locale]
	}
	y += btnH + gap
	if g.woodButton(d, ui.Rect{X: x, Y: y, W: w, H: btnH}, g.tr("ui.era.reset"), true, 24).Long {
		g.reset()
		g.closeScreen()
	}
	return y + btnH + gap
}

// --- era timeline ---------------------------------------------------------

func (g *Game) eraBody(d *ebiten.Image, x, y, w float64) float64 {
	k := g.kit
	c := g.sim.Completion()
	y = g.groupTitle(d, g.tr("ui.era.conditions"), x, y)
	branch := 0.0
	for _, a := range content.Axes {
		branch = math.Max(branch, g.sim.BranchProgress(a))
	}
	lines := []struct {
		label string
		ok    bool
		v     float64
	}{
		{g.tr("ui.era.keys"), len(c.Keys) > 0, branch},
		{fmt.Sprintf("%s : %d / %d", g.tr("ui.gauge.stations"), c.StationsBuilt, c.StationsTotal), c.StationsBuilt == c.StationsTotal, float64(c.StationsBuilt) / float64(max(1, c.StationsTotal))},
		{fmt.Sprintf("%s : %d / %d", g.tr("ui.gauge.quests"), c.QuestsDone, c.QuestsTotal), c.QuestsDone == c.QuestsTotal, float64(c.QuestsDone) / float64(max(1, c.QuestsTotal))},
	}
	in := g.card(d, x, y, w, 40+float64(len(lines))*74)
	for i, l := range lines {
		ly := in.Y + float64(i)*74
		mark, col := "○ ", inkSoft
		if l.ok {
			mark, col = "● ", inkGood
		}
		k.Shadowed(d, mark+l.label, in.X, ly, bodySize, col, ui.Left, true)
		g.progress(d, ui.Rect{X: in.X, Y: ly + 36, W: in.W, H: 24}, l.v)
	}
	y = in.Y + in.H + 20 + gap
	if g.sim.Ending() != "" {
		y += k.ShadowedParagraph(d, g.tr("ui.era.ended"), x, y, bodySize, w, inkGold, true) + gap
	}
	y = g.groupTitle(d, g.tr("ui.era.choose"), x, y)
	for i := range g.era.Endings {
		e := &g.era.Endings[i]
		owned := slices.Contains(c.Keys, e.Axis)
		desc := g.text(e.DescKey)
		legacy := g.tr("ui.era.patent") + " « " + g.text(e.PatentKey) + " » : " + g.text(e.PatentDescKey)
		iw := w - 48
		h := 40 + 40 + g.paraH(desc, smallSize, iw) + 8 + g.paraH(legacy, smallSize, iw)
		passage := owned && c.Complete() && g.sim.Ending() == ""
		if passage {
			h += btnH + gap
		}
		in := g.card(d, x, y, w, h)
		col := inkLight
		if owned {
			col = inkGold
		}
		k.Shadowed(d, k.Fit(g.text(e.NameKey), titleSize, in.W*0.62, true), in.X, in.Y, titleSize, col, ui.Left, true)
		k.Shadowed(d, g.tr("ui.branch."+string(e.Axis)), in.X+in.W, in.Y+4, smallSize, inkSoft, ui.Right, false)
		cy := in.Y + 40
		cy += k.ShadowedParagraph(d, desc, in.X, cy, smallSize, in.W, inkSoft, false) + 8
		cy += k.ShadowedParagraph(d, legacy, in.X, cy, smallSize, in.W, inkLight, false)
		if passage {
			if g.woodButton(d, ui.Rect{X: in.X, Y: cy + gap, W: in.W, H: btnH}, g.tr("ui.era.portal"), true, 24).Tap {
				g.act(g.sim.Finish(e.Axis))
			}
		}
		y += h + gap
	}
	return y
}

// --- absence --------------------------------------------------------------

// drawAway is the "while you were away" screen.
func (g *Game) drawAway(dst *ebiten.Image) {
	k := g.kit
	rep := g.report
	k.Fill(dst, ui.Rect{X: 0, Y: 0, W: ScreenW, H: ScreenH}, scrimCol)
	ids := make([]string, 0, len(rep.Gains))
	for id, q := range rep.Gains {
		if q > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	rows := math.Min(float64(max(1, len(ids))), 12)
	h := 200 + rows*44 + btnH + 40
	top := (ScreenH - h) / 2
	r := ui.Rect{X: 30, Y: top, W: ScreenW - 60, H: h}
	g.panel(dst, r, mk(24))
	y := top + 44
	k.Shadowed(dst, g.tr("ui.away.title"), ScreenW/2, y, 36, inkLight, ui.Center, true)
	y += 56
	k.Shadowed(dst, fmt.Sprintf("%s %s · %s %s", g.tr("ui.away.time"), sim.FormatDuration(rep.Away), g.tr("ui.away.counted"), sim.FormatDuration(rep.Counted)), ScreenW/2, y, smallSize, inkSoft, ui.Center, false)
	y += 20
	y += g.separator(dst, 70, y, ScreenW-140) + 10
	if len(ids) == 0 {
		k.ShadowedParagraph(dst, g.tr("ui.away.nothing"), 80, y, bodySize, ScreenW-160, inkLight, false)
	}
	for i, id := range ids {
		if float64(i) >= rows {
			break
		}
		g.resIcon(dst, id, 92, y+16, 15)
		k.Shadowed(dst, g.resName(id), 116, y, bodySize, inkLight, ui.Left, false)
		k.Shadowed(dst, "+"+sim.Format(rep.Gains[id]), ScreenW-80, y, bodySize, inkGood, ui.Right, true)
		y += 44
	}
	if g.woodButton(dst, ui.Rect{X: 80, Y: top + h - btnH - 40, W: ScreenW - 160, H: btnH}, g.tr("ui.away.ok"), true, 26).Tap {
		g.report = nil
	}
}
