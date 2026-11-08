package game

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/ui"
)

// The skills screen, after the mockup: the common trunk on top, four lines
// fanning out of "Le Cercle", then one board per branch in a carousel. The
// whole tree scrolls up and down; the boards slide sideways.
const (
	treeMargin = 30.0
	trunkRowH  = 132.0
	boardRowH  = 150.0
	boardHead  = 56.0
	fanH       = 70.0
	tabsH      = 72.0
	infoH      = 262.0
	nodeSize   = 92.0
	keySize    = 112.0
)

var (
	treeTop   = safeTop + 92
	infoRect  = ui.Rect{X: treeMargin, Y: ScreenH - safeBottom - infoH, W: ScreenW - 2*treeMargin, H: infoH}
	treeArea  = ui.Rect{X: 0, Y: treeTop, W: ScreenW, H: infoRect.Y - treeTop - 8}
	boardW    = ScreenW - 2*treeMargin
	trunkH    = 64 + 2*trunkRowH
	boardH    = boardHead + content.BoardRows*boardRowH + 24
	lineWarm  = color.RGBA{0xf2, 0xd7, 0x9a, 0xff}
	lineCold  = color.RGBA{0xe8, 0xd9, 0xb8, 0x5a}
	glowGold  = color.RGBA{0xff, 0xd3, 0x6b, 0xff}
	axisOrder = content.Axes
)

// nodeCenter is where a node sits on its board, relative to the board's
// top-left corner (or the trunk's).
func nodeCenter(n *content.Node) (float64, float64) {
	if n.Branch == content.BranchTrunk {
		cw := (boardW - 48) / content.TrunkCols
		x := 24 + (float64(n.Col)-0.5)*cw
		if n.ID == "le-cercle" || (n.Row == content.TrunkRows && n.Col == 2) {
			x += cw / 2 // the gate to the branches sits in the middle
		}
		return x, 56 + (float64(n.Row)-1)*trunkRowH + nodeSize/2
	}
	cw := (boardW - 40) / content.BoardCols
	return 20 + (float64(n.Col)-0.5)*cw, boardHead + (float64(n.Row)-1)*boardRowH + keySize/2 + 4
}

func (g *Game) drawTree(dst *ebiten.Image) {
	k := g.kit
	if dst == nil {
		g.sheetT = math.Min(1, g.sheetT+dt*6)
		g.treeMotion()
	}
	g.th.panel.Draw(dst, ui.Rect{X: 0, Y: 0, W: ScreenW, H: ScreenH}, mk(26), mk(18), mk(18), mk(18), 1)
	title := g.tr("ui.tree.title") + " · " + g.text(g.era.NameKey)
	k.Shadowed(dst, title, treeMargin+16, safeTop-4, 36, inkLight, ui.Left, true)
	sub := fmt.Sprintf(g.tr("ui.tree.sub"), g.sim.StationsBuilt(), g.sim.RowsOpen(), content.BoardRows)
	k.Shadowed(dst, k.Fit(sub, 20, ScreenW-200, false), treeMargin+16, safeTop+44, 20, inkSoft, ui.Left, false)
	if g.roundButton(dst, ui.Rect{X: ScreenW - treeMargin - 82, Y: safeTop - 10, W: 76, H: 76}, g.tr("ui.close"), "×").Tap {
		g.closeScreen()
		return
	}

	contentH := trunkH + fanH + tabsH + boardH + 50
	l := k.BeginList(dst, "tree", treeArea, contentH)
	d := l.Target()
	y := l.Y()

	trunk := ui.Rect{X: treeMargin, Y: y, W: boardW, H: trunkH}
	g.insetBox(d, trunk, mk(12))
	k.Shadowed(d, strings.ToUpper(g.tr("ui.branch.trunk")), trunk.X+26, trunk.Y+20, 16, inkSoft, ui.Left, true)
	g.drawLinks(d, content.BranchTrunk, trunk.X, trunk.Y)
	for i := range g.era.Tree {
		if n := &g.era.Tree[i]; n.Branch == content.BranchTrunk {
			cx, cy := nodeCenter(n)
			g.drawNode(d, n, trunk.X+cx, trunk.Y+cy, nodeSize*0.86)
		}
	}
	y += trunkH
	g.drawFan(d, y)
	y += fanH

	// Tabs stick to the top of the scrolling area once it scrolls past them.
	tabsY := math.Max(y, treeArea.Y)
	boardsY := y + tabsH
	g.drawBoards(d, boardsY)
	if tabsY > y {
		k.Fill(d, ui.Rect{X: 0, Y: tabsY - 6, W: ScreenW, H: tabsH + 6}, rgba(0x4a, 0x2c, 0x16, 0xf2))
	}
	g.drawTreeTabs(d, tabsY)
	l.End()

	g.drawJumps(dst, l.Offset(), contentH)
	g.drawNodeInfo(dst)
}

// treeMotion slides the boards: the finger drags them, then they snap.
func (g *Game) treeMotion() {
	k := g.kit
	dx, _, horiz, ok := k.Drag()
	if ok && horiz && k.PressedIn(treeArea) {
		if !g.treeDrag {
			g.treeDrag, g.treeBase = true, g.treePos
		}
		g.treePos = g.treeBase - dx/ScreenW
		g.treePos = math.Max(-0.3, math.Min(float64(len(axisOrder))-0.7, g.treePos))
		return
	}
	if g.treeDrag {
		g.treeDrag = false
		moved := g.treePos - g.treeBase
		target := math.Round(g.treeBase)
		if math.Abs(moved) > 0.15 {
			target += math.Copysign(1, moved)
		}
		g.treeBoard = int(math.Max(0, math.Min(float64(len(axisOrder)-1), target)))
	}
	g.treePos += (float64(g.treeBoard) - g.treePos) * math.Min(1, dt*12)
}

func (g *Game) drawTreeTabs(d *ebiten.Image, y float64) {
	k := g.kit
	x := treeMargin
	w := (boardW - 3*6) / 4
	for i, a := range axisOrder {
		r := ui.Rect{X: x + float64(i)*(w+6), Y: y + 6, W: w, H: tabsH - 12}
		label := g.tr("ui.branch." + string(a))
		c, _ := k.Control(d, r, "tab "+string(a), true)
		if c.Tap {
			g.treeBoard = i
		}
		if d == nil {
			continue
		}
		on := i == g.treeBoard
		f, col := g.th.tab, inkSoft
		if on {
			f, col = g.th.tabOn, rgb(0xff, 0xf6, 0xe6)
		}
		f.Draw(d, r, mk(6), mk(18), mk(6), mk(18), 1)
		k.Shadowed(d, k.Fit(label, 21, r.W-24, true), r.X+r.W/2, r.Y+(r.H-26)/2, 21, col, ui.Center, true)
	}
}

// drawFan draws the four lines that leave "Le Cercle" towards the boards and
// follow them as they slide.
func (g *Game) drawFan(d *ebiten.Image, y float64) {
	if d == nil {
		return
	}
	var cercle *content.Node
	for i := range g.era.Tree {
		if g.era.Tree[i].ID == "le-cercle" {
			cercle = &g.era.Tree[i]
		}
	}
	if cercle == nil {
		return
	}
	cx, _ := nodeCenter(cercle)
	x0 := treeMargin + cx
	y0 := y - 14
	for i := range axisOrder {
		x1 := ScreenW/2 + (float64(i)-g.treePos)*ScreenW*0.62
		near := math.Abs(float64(i)-g.treePos) < 0.5
		col, wd := lineCold, float32(3)
		if near {
			col, wd = color.RGBA{0xf2, 0xc1, 0x4e, 0xff}, 4.5
		}
		px, py := x0, y0
		for s := 1; s <= 16; s++ {
			t := float64(s) / 16
			// Cubic curve: down from the gate, then across to the board.
			bx := cubic(x0, x0, x1, x1, t)
			by := cubic(y0, y0+fanH*0.6, y+fanH*0.4, y+fanH+4, t)
			vector.StrokeLine(d, float32(px), float32(py), float32(bx), float32(by), wd, col, true)
			px, py = bx, by
		}
	}
}

func cubic(a, b, c, e, t float64) float64 {
	u := 1 - t
	return u*u*u*a + 3*u*u*t*b + 3*u*t*t*c + t*t*t*e
}

func (g *Game) drawBoards(d *ebiten.Image, y float64) {
	k := g.kit
	for i, a := range axisOrder {
		bx := treeMargin + (float64(i)-g.treePos)*ScreenW
		if bx > ScreenW || bx+boardW < 0 {
			continue
		}
		r := ui.Rect{X: bx, Y: y, W: boardW, H: boardH}
		if d != nil {
			g.tile(d, r, 300)
			k.Fill(d, ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 4}, rgba(0x28, 0x12, 0x04, 0x99))
			k.Fill(d, ui.Rect{X: r.X, Y: r.Y + r.H - 4, W: r.W, H: 4}, rgba(0x28, 0x12, 0x04, 0x99))
			k.Fill(d, ui.Rect{X: r.X, Y: r.Y, W: 4, H: r.H}, rgba(0x28, 0x12, 0x04, 0x99))
			k.Fill(d, ui.Rect{X: r.X + r.W - 4, Y: r.Y, W: 4, H: r.H}, rgba(0x28, 0x12, 0x04, 0x99))
			k.Shadowed(d, g.tr("ui.branch."+string(a)), r.X+22, r.Y+16, 28, inkLight, ui.Left, true)
			if key := g.keyOf(a); key != nil {
				k.Shadowed(d, k.Fit(g.text(key.NameKey), 19, r.W*0.55, false), r.X+r.W-22, r.Y+24, 19, inkSoft, ui.Right, false)
			}
		}
		g.drawLinks(d, string(a), r.X, r.Y)
		for j := range g.era.Tree {
			if n := &g.era.Tree[j]; n.Branch == string(a) {
				cx, cy := nodeCenter(n)
				size := nodeSize
				if n.EraKey {
					size = keySize
				}
				g.drawNode(d, n, r.X+cx, r.Y+cy, size)
			}
		}
		g.drawLockBand(d, r)
	}
	// Dots under the boards say which one is shown.
	if d != nil {
		dy := y + boardH + 22
		for i := range axisOrder {
			x := ScreenW/2 + (float64(i)-1.5)*26
			c := rgba(0x28, 0x12, 0x04, 0xa0)
			if i == g.treeBoard {
				c = rgb(0xf2, 0xc1, 0x4e)
			}
			g.ellipse(d, x, dy, 7, 7, c)
		}
	}
}

func (g *Game) keyOf(a content.Axis) *content.Node {
	for i := range g.era.Tree {
		if n := &g.era.Tree[i]; n.EraKey && n.Branch == string(a) {
			return n
		}
	}
	return nil
}

// drawLockBand covers the rows not open yet, from the first locked one down.
func (g *Game) drawLockBand(d *ebiten.Image, board ui.Rect) {
	if d == nil {
		return
	}
	first := 0
	for r := 1; r <= content.BoardRows; r++ {
		if !g.sim.RowOpen(r) {
			first = r
			break
		}
	}
	if first == 0 {
		return
	}
	top := board.Y + boardHead + float64(first-1)*boardRowH - 4
	band := ui.Rect{X: board.X + 4, Y: top, W: board.W - 8, H: board.Y + board.H - 4 - top}
	sub := d.SubImage(rectI(band)).(*ebiten.Image)
	g.kit.Fill(sub, band, bandLock2)
	for x := band.X - band.H; x < band.X+band.W; x += 32 {
		vector.StrokeLine(sub, float32(x), float32(band.Y+band.H), float32(x+band.H), float32(band.Y), 11, bandLock, false)
	}
	need := g.era.RowStations[first]
	label := fmt.Sprintf(g.tr("ui.tree.rowLocked"), first, need, g.sim.StationsBuilt(), need)
	w := g.kit.Width(label, 21, true) + 36
	pill := ui.Rect{X: band.X + (band.W-w)/2, Y: band.Y + 46, W: w, H: 46}
	g.kit.Fill(d, pill, rgba(0, 0, 0, 0xa8))
	g.kit.Text(d, label, pill.X+pill.W/2, pill.Y+10, 21, inkLight, ui.Center, true)
}

// drawLinks draws the prerequisite arrows between the nodes of one board.
func (g *Game) drawLinks(d *ebiten.Image, branch string, ox, oy float64) {
	if d == nil {
		return
	}
	for i := range g.era.Tree {
		n := &g.era.Tree[i]
		if n.Branch != branch {
			continue
		}
		reqs := append(slices.Clone(n.RequiresAll), n.RequiresAny...)
		for _, id := range reqs {
			r := g.nodeDef(id)
			if r == nil || r.Branch != branch {
				continue
			}
			ax, ay := nodeCenter(r)
			bx, by := nodeCenter(n)
			ax, ay, bx, by = ax+ox, ay+oy, bx+ox, by+oy
			col, w := lineCold, float32(3)
			if g.sim.HasNode(id) {
				col, w = lineWarm, 4
			}
			dashed := slices.Contains(n.RequiresAny, id)
			g.arrow(d, ax, ay, bx, by, n.EraKey, col, w, dashed)
		}
	}
}

// arrow joins two node centers, stopping at their edges, with a head.
func (g *Game) arrow(d *ebiten.Image, ax, ay, bx, by float64, toKey bool, col color.RGBA, w float32, dashed bool) {
	dx, dy := bx-ax, by-ay
	l := math.Hypot(dx, dy)
	if l < 1 {
		return
	}
	ux, uy := dx/l, dy/l
	r0, r1 := nodeSize*0.52, nodeSize*0.52
	if toKey {
		r1 = keySize * 0.52
	}
	if math.Abs(dy) > 1 {
		r0 += 26 // room for the node's name under it
	}
	sx, sy := ax+ux*r0, ay+uy*r0
	ex, ey := bx-ux*r1, by-uy*r1
	seg := math.Hypot(ex-sx, ey-sy)
	if seg < 8 {
		return
	}
	if dashed {
		for t := 0.0; t < seg; t += 18 {
			t2 := math.Min(seg, t+10)
			vector.StrokeLine(d, float32(sx+ux*t), float32(sy+uy*t), float32(sx+ux*t2), float32(sy+uy*t2), w, col, true)
		}
	} else {
		vector.StrokeLine(d, float32(sx), float32(sy), float32(ex), float32(ey), w, col, true)
	}
	// Head.
	hx, hy := ex, ey
	px, py := -uy, ux
	g.poly(d, []float64{hx, hy, hx - ux*11 + px*7, hy - uy*11 + py*7, hx - ux*11 - px*7, hy - uy*11 - py*7}, col)
}

// drawNode draws one skill: its slot, its picture or glyph, its rank plate
// and its name; tapping selects it.
func (g *Game) drawNode(d *ebiten.Image, n *content.Node, cx, cy, size float64) {
	k := g.kit
	r := ui.Rect{X: cx - size/2, Y: cy - size/2, W: size, H: size}
	if c, _ := k.Control(d, r.Inset(-8), "node "+n.ID, true); c.Tap {
		g.selNode = n.ID
	}
	if d == nil {
		return
	}
	rank := g.sim.Rank(n.ID)
	off := rank == 0 && !g.sim.NodeUnlocked(n.ID)
	var cs ebiten.ColorScale
	if off {
		cs.Scale(0.5, 0.47, 0.45, 1)
	}
	if g.sim.CanInvest(n.ID) {
		a := 0.45 + 0.4*math.Sin(g.t*4.5)
		g.ellipse(d, cx, cy, size*0.66, size*0.66, rgba(glowGold.R, glowGold.G, glowGold.B, uint8(110*a)))
	}
	if g.selNode == n.ID {
		g.ellipse(d, cx, cy, size*0.72, size*0.72, rgba(0xff, 0xe8, 0xa0, 0x8c))
	}
	slot := g.th.slot
	if n.EraKey {
		slot = g.th.slotKey
	}
	ui.Image(d, slot, r, cs)
	inset := 0.21
	if n.EraKey {
		inset = 0.23
	}
	in := ui.Rect{X: r.X + r.W*inset, Y: r.Y + r.H*inset, W: r.W * (1 - 2*inset), H: r.H * (1 - 2*inset)}
	if art := g.th.caps[n.Art]; art != nil {
		ui.Image(d, art, in, cs)
	} else {
		g.glyph(d, n, cx, cy, in.W*0.4, off)
	}
	m := n.Ranks()
	plate := ui.Rect{X: r.X + r.W - 44, Y: r.Y + r.H - 22, W: 66, H: 30}
	g.plaque(d, plate)
	col := rgb(0xff, 0xe8, 0xbf)
	if rank >= m {
		col = inkGold
	}
	k.Shadowed(d, fmt.Sprintf("%d/%d", rank, m), plate.X+plate.W/2, plate.Y+5, 17, col, ui.Center, true)
	nameCol := rgb(0xff, 0xe9, 0xc4)
	if n.EraKey {
		nameCol = inkGold
	}
	if off {
		nameCol = rgb(0xc8, 0xb4, 0x96)
	}
	cw := (boardW - 40) / content.BoardCols
	k.Shadowed(d, k.Fit(g.text(n.NameKey), 17, cw-10, true), cx, r.Y+r.H+12, 17, nameCol, ui.Center, true)
}

// glyph stands in for a node's picture until it has one.
func (g *Game) glyph(d *ebiten.Image, n *content.Node, cx, cy, s float64, off bool) {
	c := rgba(0x3a, 0x23, 0x12, 0xc8)
	if off {
		c = rgba(0x3a, 0x23, 0x12, 0x80)
	}
	switch (n.Row + n.Col) % 4 {
	case 0:
		g.poly(d, []float64{cx, cy - s, cx + s, cy, cx, cy + s, cx - s, cy}, c)
	case 1:
		g.ellipse(d, cx, cy, s, s, c)
		g.ellipse(d, cx, cy, s*0.55, s*0.55, rgba(0x8a, 0x5a, 0x34, 0xff))
		g.ellipse(d, cx, cy, s*0.3, s*0.3, c)
	case 2:
		g.poly(d, []float64{cx, cy - s, cx + s, cy + s*0.8, cx - s, cy + s*0.8}, c)
	default:
		g.poly(d, []float64{cx - s, cy - s*0.7, cx + s, cy - s*0.7, cx + s, cy - s*0.25, cx - s, cy - s*0.25}, c)
		g.poly(d, []float64{cx - s*0.7, cy, cx + s*0.7, cy, cx + s*0.7, cy + s, cx - s*0.7, cy + s}, c)
	}
}

// drawJumps shows the Top and Turning point shortcuts at the bottom right.
func (g *Game) drawJumps(dst *ebiten.Image, offset, contentH float64) {
	x := ScreenW - treeMargin - 230
	y := infoRect.Y - 96
	if offset < contentH-treeArea.H-24 {
		if g.woodButton(dst, ui.Rect{X: x, Y: y, W: 230, H: 80}, "▼ "+g.tr("ui.tree.bottom"), true, 22).Tap {
			g.kit.ScrollTo("tree", contentH)
		}
		y -= 90
	}
	if offset > 24 {
		if g.woodButton(dst, ui.Rect{X: x, Y: y, W: 230, H: 80}, "▲ "+g.tr("ui.tree.top"), true, 22).Tap {
			g.kit.Reset("tree")
		}
	}
}

// drawNodeInfo is the panel under the tree: what the selected node does,
// what its next rank costs and the button to invest.
func (g *Game) drawNodeInfo(dst *ebiten.Image) {
	k := g.kit
	in := g.card(dst, infoRect.X, infoRect.Y, infoRect.W, infoRect.H)
	n := g.nodeDef(g.selNode)
	if n == nil {
		k.Shadowed(dst, g.tr("ui.tree.hint.title"), in.X, in.Y, 28, inkLight, ui.Left, true)
		k.ShadowedParagraph(dst, g.tr("ui.tree.hint"), in.X, in.Y+44, smallSize, in.W, inkSoft, false)
		return
	}
	rank, m := g.sim.Rank(n.ID), n.Ranks()
	k.Shadowed(dst, k.Fit(g.text(n.NameKey), 28, in.W-120, true), in.X, in.Y, 28, inkLight, ui.Left, true)
	k.Shadowed(dst, fmt.Sprintf("%d / %d", rank, m), in.X+in.W, in.Y, 28, inkGold, ui.Right, true)
	y := in.Y + 42
	effect := ""
	if len(n.Effects) > 0 {
		effect = g.effectsText(n.Effects, 1/float64(m))
		if m > 1 {
			effect += " " + g.tr("ui.tree.perRank")
		}
		effect += "."
	}
	if n.ID == "main-habile" {
		effect += " " + fmt.Sprintf(g.tr("ui.tree.harvestNow"), int(math.Round(g.sim.HarvestShare()*100)))
	}
	if n.DescKey != "" {
		effect = strings.TrimSpace(effect + " « " + g.text(n.DescKey) + " »")
	}
	if effect != "" {
		lines := k.Wrap(effect, 20, in.W, false)
		if len(lines) > 3 {
			lines = append(lines[:2], k.Fit(strings.Join(lines[2:], " "), 20, in.W, false))
		}
		for _, l := range lines {
			k.Shadowed(dst, l, in.X, y, 20, inkLight, ui.Left, false)
			y += 26
		}
	}
	why := ""
	switch {
	case g.sim.NodeMaxed(n.ID):
		why = g.tr("ui.tree.maxed")
	case n.Branch != content.BranchTrunk && !g.sim.RowOpen(n.Row):
		need := g.era.RowStations[n.Row]
		why = fmt.Sprintf(g.tr("ui.tree.rowLocked"), n.Row, need, g.sim.StationsBuilt(), need)
	case !g.sim.NodeReqsMet(n.ID):
		names := []string{}
		for _, id := range n.RequiresAll {
			names = append(names, g.nodeName(id))
		}
		sep := " " + g.tr("ui.tree.and") + " "
		req := strings.Join(names, sep)
		if len(n.RequiresAny) > 0 {
			alt := []string{}
			for _, id := range n.RequiresAny {
				alt = append(alt, g.nodeName(id))
			}
			if req != "" {
				req += sep
			}
			req += strings.Join(alt, " "+g.tr("ui.goal.or")+" ")
		}
		why = fmt.Sprintf(g.tr("ui.tree.requires"), req)
	}
	bottom := in.Y + in.H
	btn := ui.Rect{X: in.X + in.W - 220, Y: bottom - 80, W: 220, H: 80}
	if cost := g.sim.NextRankCost(n.ID); cost != nil {
		k.Shadowed(dst, g.tr("ui.tree.nextRank"), in.X, bottom-84, 19, inkSoft, ui.Left, false)
		g.drawChips(dst, g.costChips(cost), in.X, bottom-56, in.W-240, 21, false, inkLight)
	}
	if why != "" {
		k.Shadowed(dst, k.Fit(why, 19, in.W-240, false), in.X, bottom-24, 19, inkSoft, ui.Left, false)
	}
	label := g.tr("ui.tree.invest")
	if n.EraKey {
		label = g.tr("ui.tree.accomplish")
	}
	if g.woodButton(dst, btn, label, g.sim.CanInvest(n.ID), 25).Tap {
		if err := g.sim.BuyNode(n.ID); err != nil {
			g.act(err)
		} else {
			g.sparks.X, g.sparks.Y = btn.X+btn.W/2, btn.Y
			g.sparkFx.Emit(g.sparks, 14)
		}
	}
}
