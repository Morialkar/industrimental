package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/sim"
	"github.com/morialkar/industrimental/internal/ui"
)

// shade darkens (f < 1) or lightens (f > 1) a color.
func shade(c color.RGBA, f float64) color.RGBA {
	m := func(v uint8) uint8 {
		x := float64(v) * f
		if f > 1 {
			x = float64(v) + (255-float64(v))*(f-1)
		}
		return uint8(math.Max(0, math.Min(255, x)))
	}
	return color.RGBA{m(c.R), m(c.G), m(c.B), c.A}
}

// resIcon draws a small picture of a resource, centered on cx, cy, about 2r
// wide. Placeholders until the era kit has painted icons, but each one is a
// distinct shape so that costs read at a glance.
func (g *Game) resIcon(dst *ebiten.Image, id string, cx, cy, r float64) {
	if dst == nil {
		return
	}
	c := resColor(id)
	dark, light := shade(c, 0.6), shade(c, 1.35)
	ink := rgba(0x1a, 0x10, 0x08, 0xb0)
	// p maps unit coordinates (-1..1) to the screen.
	p := func(pts ...float64) []float64 {
		out := make([]float64, len(pts))
		for i := 0; i < len(pts); i += 2 {
			out[i], out[i+1] = cx+pts[i]*r, cy+pts[i+1]*r
		}
		return out
	}
	el := func(x, y, rx, ry float64, col color.RGBA) { g.ellipse(dst, cx+x*r, cy+y*r, rx*r, ry*r, col) }
	ln := func(x0, y0, x1, y1, w float64, col color.RGBA) {
		g.thickLine(dst, cx+x0*r, cy+y0*r, cx+x1*r, cy+y1*r, w*r, col)
	}
	switch id {
	case "coquillages": // a scallop: fan and ridges
		g.poly(dst, p(0, 0.85, -0.95, -0.15, -0.6, -0.75, 0, -0.95, 0.6, -0.75, 0.95, -0.15), dark)
		g.poly(dst, p(0, 0.7, -0.8, -0.15, -0.5, -0.65, 0, -0.82, 0.5, -0.65, 0.8, -0.15), c)
		for _, a := range []float64{-0.6, -0.3, 0, 0.3, 0.6} {
			ln(0, 0.7, a*1.2, -0.7+math.Abs(a)*0.4, 0.09, dark)
		}
		g.quad(dst, cx-0.3*r, cy+0.6*r, cx+0.3*r, cy+0.6*r, cx+0.2*r, cy+0.95*r, cx-0.2*r, cy+0.95*r, dark)
	case "baies": // three berries and a leaf
		g.poly(dst, p(0.1, -0.5, 0.75, -0.95, 0.55, -0.35), rgb(0x5c, 0x8a, 0x3a))
		for _, b := range [][2]float64{{-0.4, 0.15}, {0.4, 0.15}, {0, -0.3}} {
			el(b[0], b[1]+0.05, 0.45, 0.45, dark)
			el(b[0], b[1], 0.42, 0.42, c)
			el(b[0]-0.14, b[1]-0.14, 0.12, 0.12, light)
		}
	case "silex": // a faceted nodule
		g.poly(dst, p(-0.9, 0.2, -0.5, -0.75, 0.3, -0.9, 0.95, -0.2, 0.7, 0.7, -0.3, 0.9), dark)
		g.poly(dst, p(-0.75, 0.15, -0.4, -0.62, 0.25, -0.75, 0.8, -0.2, 0.58, 0.58, -0.25, 0.75), c)
		g.poly(dst, p(-0.4, -0.62, 0.25, -0.75, 0.1, -0.05, -0.55, 0), light)
	case "bois": // a log seen from the end
		g.quad(dst, cx-0.9*r, cy-0.15*r, cx+0.5*r, cy-0.65*r, cx+0.75*r, cy+0.05*r, cx-0.65*r, cy+0.55*r, dark)
		g.quad(dst, cx-0.85*r, cy-0.1*r, cx+0.45*r, cy-0.55*r, cx+0.65*r, cy+0.0*r, cx-0.65*r, cy+0.45*r, c)
		el(0.6, -0.3, 0.38, 0.4, dark)
		el(0.6, -0.3, 0.32, 0.34, rgb(0xd9, 0xb0, 0x78))
		el(0.6, -0.3, 0.16, 0.17, rgb(0xb0, 0x82, 0x4c))
	case "viande": // a drumstick
		ln(0.2, 0.2, 0.75, 0.75, 0.24, rgb(0xf1, 0xe6, 0xd0))
		el(0.82, 0.68, 0.15, 0.15, rgb(0xf1, 0xe6, 0xd0))
		el(0.68, 0.84, 0.15, 0.15, rgb(0xf1, 0xe6, 0xd0))
		el(-0.2, -0.2, 0.68, 0.6, dark)
		el(-0.22, -0.24, 0.62, 0.54, c)
		el(-0.4, -0.42, 0.18, 0.14, light)
	case "peaux-brutes", "peaux-tannees": // a stretched hide
		g.poly(dst, p(0, -0.95, 0.55, -0.7, 0.95, -0.85, 0.75, -0.1, 0.95, 0.75, 0.4, 0.6, 0, 0.95, -0.4, 0.6, -0.95, 0.75, -0.75, -0.1, -0.95, -0.85, -0.55, -0.7), dark)
		g.poly(dst, p(0, -0.8, 0.5, -0.58, 0.78, -0.68, 0.6, -0.1, 0.78, 0.62, 0.35, 0.48, 0, 0.8, -0.35, 0.48, -0.78, 0.62, -0.6, -0.1, -0.78, -0.68, -0.5, -0.58), c)
		if id == "peaux-tannees" { // stitched and smoothed
			for _, y := range []float64{-0.4, 0, 0.4} {
				ln(-0.3, y, 0.3, y, 0.07, dark)
			}
		} else {
			el(-0.2, -0.15, 0.2, 0.14, dark)
			el(0.25, 0.25, 0.16, 0.12, dark)
		}
	case "argile": // a wet lump
		el(0, 0.2, 0.92, 0.62, dark)
		el(0, 0.15, 0.86, 0.56, c)
		el(-0.05, -0.25, 0.55, 0.4, c)
		el(-0.25, -0.3, 0.2, 0.12, light)
	case "outils-silex": // a hafted flint blade
		ln(-0.8, 0.85, 0.15, -0.1, 0.24, rgb(0x7a, 0x4e, 0x26))
		ln(-0.35, 0.4, -0.1, 0.15, 0.3, rgb(0xc9, 0xa6, 0x6b))
		g.poly(dst, p(0.05, -0.05, 0.2, -0.8, 0.95, -0.95, 0.75, -0.2), ink)
		g.poly(dst, p(0.1, -0.1, 0.25, -0.72, 0.85, -0.85, 0.68, -0.25), c)
		g.poly(dst, p(0.25, -0.72, 0.85, -0.85, 0.45, -0.4), light)
	case "grain": // an ear of wheat
		ln(0, 0.95, 0, -0.85, 0.08, dark)
		for i := 0; i < 4; i++ {
			y := -0.65 + float64(i)*0.32
			el(-0.2, y, 0.17, 0.27, c)
			el(0.2, y, 0.17, 0.27, c)
		}
		el(0, -0.82, 0.14, 0.22, c)
	case "sel": // a heap of white crystals
		g.poly(dst, p(-0.95, 0.7, -0.4, -0.3, 0, -0.6, 0.4, -0.3, 0.95, 0.7), rgb(0xa8, 0xb4, 0xbb))
		g.poly(dst, p(-0.8, 0.6, -0.35, -0.2, 0, -0.45, 0.35, -0.2, 0.8, 0.6), c)
		for _, s := range [][2]float64{{-0.3, 0.2}, {0.25, 0.1}, {0, -0.15}} {
			g.poly(dst, p(s[0], s[1]-0.15, s[0]+0.1, s[1], s[0], s[1]+0.15, s[0]-0.1, s[1]), rgb(0xff, 0xff, 0xff))
		}
	case "parures": // a necklace with a pendant
		for i := 0; i <= 8; i++ {
			a := math.Pi * (0.1 + 0.8*float64(i)/8)
			el(math.Cos(a)*0.75, -0.55+math.Sin(a)*0.75, 0.13, 0.13, c)
		}
		g.poly(dst, p(0, 0.25, 0.3, 0.55, 0, 0.98, -0.3, 0.55), dark)
		g.poly(dst, p(0, 0.33, 0.22, 0.56, 0, 0.88, -0.22, 0.56), rgb(0xe9, 0x78, 0x2a))
	case "ocre": // a bowl of pigment
		el(0, -0.1, 0.7, 0.35, c)
		el(0, -0.25, 0.4, 0.3, light)
		g.poly(dst, p(-0.9, 0, 0.9, 0, 0.6, 0.65, -0.6, 0.65), rgb(0x8a, 0x6a, 0x44))
		g.poly(dst, p(-0.9, 0, 0.9, 0, 0.8, 0.15, -0.8, 0.15), rgb(0x6e, 0x52, 0x33))
	case "poteries-crues": // a pot
		g.poly(dst, p(-0.35, -0.8, 0.35, -0.8, 0.3, -0.55, -0.3, -0.55), dark)
		el(0, 0.15, 0.78, 0.72, dark)
		el(0, 0.12, 0.72, 0.66, c)
		g.quad(dst, cx-0.3*r, cy-0.6*r, cx+0.3*r, cy-0.6*r, cx+0.42*r, cy-0.35*r, cx-0.42*r, cy-0.35*r, c)
		ln(-0.6, 0.05, 0.6, 0.05, 0.08, dark)
	case "fresques": // a stone slab with a painted animal
		g.poly(dst, p(-0.95, -0.75, 0.95, -0.85, 0.9, 0.85, -0.9, 0.8), rgb(0x5a, 0x50, 0x48))
		g.poly(dst, p(-0.82, -0.62, 0.82, -0.7, 0.78, 0.72, -0.78, 0.68), rgb(0xc9, 0xb8, 0x98))
		el(0, 0, 0.42, 0.22, rgb(0xa3, 0x3b, 0x22))
		el(0.45, -0.2, 0.15, 0.12, rgb(0xa3, 0x3b, 0x22))
		for _, x := range []float64{-0.3, -0.1, 0.15, 0.32} {
			ln(x, 0.1, x, 0.45, 0.07, rgb(0xa3, 0x3b, 0x22))
		}
	default:
		g.pentagon(dst, cx, cy, r*0.9, c)
	}
}

// --- costs as icons ---------------------------------------------------------

// chip is one icon and its number in a row of amounts, or a bare word (an
// arrow) when id is empty.
type chip struct {
	id    string
	qty   float64
	text  string
	short bool // the player has fewer than qty
	plus  bool // written +qty (a gain)
}

// costChips lists amounts to pay, marking those the stock cannot cover.
func (g *Game) costChips(a content.Amounts) []chip {
	out := make([]chip, 0, len(a))
	for _, id := range sortedAmounts(a) {
		out = append(out, chip{id: id, qty: a[id], short: g.sim.Stock(id) < a[id]-1e-9})
	}
	return out
}

// amountChips lists amounts without judging them.
func amountChips(a content.Amounts, plus bool) []chip {
	out := make([]chip, 0, len(a))
	for _, id := range sortedAmounts(a) {
		out = append(out, chip{id: id, qty: a[id], plus: plus})
	}
	return out
}

// recipeChips is "inputs → outputs" with icons.
func recipeChips(r content.Recipe, mult float64) []chip {
	var out []chip
	scale := func(a content.Amounts) content.Amounts {
		b := content.Amounts{}
		for id, v := range a {
			b[id] = v * mult
		}
		return b
	}
	if len(r.Inputs) > 0 {
		out = append(out, amountChips(scale(r.Inputs), false)...)
		out = append(out, chip{text: "→"})
	}
	return append(out, amountChips(scale(r.Outputs), false)...)
}

func (c chip) label() string {
	if c.id == "" {
		return c.text
	}
	s := sim.Format(c.qty)
	if c.plus {
		s = "+" + s
	}
	return s
}

// drawChips lays chips out in rows of width w from x, y, wrapping as needed,
// and returns the height used. With center, each row is centered on x+w/2.
// It measures the same way in both passes, so the Update pass can call it
// with a nil image to get the height.
func (g *Game) drawChips(dst *ebiten.Image, chips []chip, x, y, w, size float64, center bool, col color.Color) float64 {
	k := g.kit
	icon := size * 1.15
	widthOf := func(c chip) float64 {
		tw := k.Width(c.label(), size, true)
		if c.id == "" {
			return tw
		}
		return icon + 6 + tw
	}
	const space = 20.0
	// Split into rows.
	var rows [][]chip
	var row []chip
	rw := 0.0
	for _, c := range chips {
		cw := widthOf(c)
		if len(row) > 0 && rw+space+cw > w {
			rows = append(rows, row)
			row, rw = nil, 0
		}
		if len(row) > 0 {
			rw += space
		}
		row = append(row, c)
		rw += cw
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	lineH := size * 1.5
	if dst != nil {
		for i, r := range rows {
			total := 0.0
			for j, c := range r {
				if j > 0 {
					total += space
				}
				total += widthOf(c)
			}
			cx := x
			if center {
				cx = x + (w-total)/2
			}
			cy := y + float64(i)*lineH
			for _, c := range r {
				tc := col
				if c.short {
					tc = inkBad
				}
				if c.id != "" {
					g.resIcon(dst, c.id, cx+icon/2, cy+size*0.62, icon/2)
					cx += icon + 6
				}
				k.Shadowed(dst, c.label(), cx, cy, size, tc, ui.Left, true)
				cx += k.Width(c.label(), size, true) + space
			}
		}
	}
	return float64(len(rows)) * lineH
}

// costButton is a wooden button with its label on top and what it costs, as
// icons, underneath. Its height is about 1.5 btnH.
func (g *Game) costButton(dst *ebiten.Image, r ui.Rect, id, label string, chips []chip, enabled bool) ui.Click {
	c, held := g.kit.Control(dst, r, id, enabled)
	if dst == nil {
		return c
	}
	f := g.th.button
	if held {
		f = g.th.buttonDown
	}
	alpha := float32(1)
	if !enabled {
		alpha = 0.6
	}
	f.Draw(dst, r, mk(10), mk(20), mk(10), mk(20), alpha)
	col := color.Color(rgb(0xff, 0xf6, 0xe6))
	if !enabled {
		col = rgba(0xe8, 0xd8, 0xc0, 0xb0)
	}
	dy := 0.0
	if held {
		dy = 2
	}
	k := g.kit
	// Narrow buttons shrink their label rather than cut it.
	size := 24.0
	for size > 17 && k.Width(label, size, true) > r.W-mk(24) {
		size--
	}
	k.Shadowed(dst, k.Fit(label, size, r.W-mk(24), true), r.X+r.W/2, r.Y+18+dy+(24-size)/2, size, col, ui.Center, true)
	g.drawChips(dst, chips, r.X+mk(14), r.Y+56+dy, r.W-mk(28), 22, true, rgb(0xff, 0xe7, 0xa8))
	return c
}

const costBtnH = 120.0
