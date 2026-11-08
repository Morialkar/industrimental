package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/morialkar/industrimental/internal/assets"
	"github.com/morialkar/industrimental/internal/ui"
)

// The interface mockup is drawn for a 390-point-wide phone; the game's logical
// screen is 720 pixels wide. mk converts the mockup's sizes.
const mockScale = float64(ScreenW) / 390

func mk(v float64) float64 { return v * mockScale }

// Safe areas of a notched phone (iPhone 15 Pro Max: 59 and 34 points of 932).
const (
	safeTop    = 100.0
	safeBottom = 58.0
)

// Colors of text on the wooden kit.
var (
	inkLight  = rgb(0xff, 0xf3, 0xdd)
	inkSoft   = rgb(0xf3, 0xd9, 0xae)
	inkGold   = rgb(0xf2, 0xc1, 0x4e)
	inkGood   = rgb(0x8f, 0xd1, 0x6a)
	inkBad    = rgb(0xff, 0x8a, 0x6a)
	inkDark   = rgb(0x3a, 0x23, 0x12)
	scrimCol  = rgba(0, 0, 0, 0x73)
	sunkCol   = rgba(0x30, 0x16, 0x06, 0x59)
	goldSunk  = rgba(0xf2, 0xc1, 0x4e, 0x40)
	bandLock  = rgba(0x28, 0x12, 0x04, 0x8c)
	bandLock2 = rgba(0x28, 0x12, 0x04, 0x59)
)

// theme is the era's interface kit, cut and ready to draw.
type theme struct {
	panel, inset, button, buttonDown, tab, tabOn, bar, band ui.Frame

	round, slot, slotKey, plate, sep, board *ebiten.Image
	acts                                    [5]*ebiten.Image
	caps                                    map[string]*ebiten.Image
}

func loadTheme(era string) (*theme, error) {
	img := func(name string) (*ebiten.Image, error) {
		i, err := assets.UI(era, name)
		if err != nil {
			return nil, err
		}
		return ebiten.NewImageFromImage(i), nil
	}
	names, err := assets.UINames(era)
	if err != nil {
		return nil, err
	}
	all := map[string]*ebiten.Image{}
	for _, n := range names {
		if all[n], err = img(n); err != nil {
			return nil, err
		}
	}
	need := func(n string) *ebiten.Image {
		if all[n] == nil && err == nil {
			err = fmt.Errorf("theme: missing %s/%s", era, n)
		}
		return all[n]
	}
	// Slices measured on the kit, as in the mockup's border-image rules.
	t := &theme{
		panel:      ui.NewFrame(need("panneau"), 0.2, 0.2, 0.2, 0.2),
		inset:      ui.NewFrame(need("encart"), 0.2, 0.2, 0.2, 0.2),
		button:     ui.NewFrame(need("bouton"), 0.3, 0.14, 0.3, 0.14),
		buttonDown: ui.NewFrame(need("bouton_presse"), 0.3, 0.14, 0.3, 0.14),
		tab:        ui.NewFrame(need("onglet"), 0.4, 0.19, 0.4, 0.19),
		tabOn:      ui.NewFrame(need("onglet_actif"), 0.4, 0.19, 0.4, 0.19),
		bar:        ui.NewFrame(need("barre"), 0.45, 0.07, 0.45, 0.07),
		band:       ui.NewFrame(need("bandeau"), 0.45, 0.15, 0.45, 0.15),
		round:      need("bouton_rond"),
		slot:       need("case"),
		slotKey:    need("case_tournant"),
		plate:      need("plaque_rang"),
		sep:        need("separateur"),
		board:      need("fond_planche"),
		acts:       [5]*ebiten.Image{need("ic_capacites"), need("ic_marche"), need("ic_quetes"), need("ic_boutique"), need("ic_clan")},
		caps:       map[string]*ebiten.Image{},
	}
	for n, i := range all {
		if len(n) > 4 && n[:4] == "cap_" {
			t.caps[n[4:]] = i
		}
	}
	return t, err
}

// --- drawing helpers ------------------------------------------------------

// panel is the main wooden frame.
func (g *Game) panel(dst *ebiten.Image, r ui.Rect, border float64) {
	g.th.panel.Draw(dst, r, border, border, border, border, 1)
}

// insetBox is the carved inset used for cards.
func (g *Game) insetBox(dst *ebiten.Image, r ui.Rect, border float64) {
	g.th.inset.Draw(dst, r, border, border, border, border, 1)
}

// woodButton is a plank button with a label. Disabled buttons fade out.
func (g *Game) woodButton(dst *ebiten.Image, r ui.Rect, label string, enabled bool, size float64) ui.Click {
	return g.woodBtn(dst, r, label, label, enabled, size)
}

// woodBtn is a plank button whose test name differs from its label.
func (g *Game) woodBtn(dst *ebiten.Image, r ui.Rect, id, label string, enabled bool, size float64) ui.Click {
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
		alpha = 0.55
	}
	f.Draw(dst, r, mk(10), mk(20), mk(10), mk(20), alpha)
	col := color.Color(rgb(0xff, 0xf6, 0xe6))
	if !enabled {
		col = rgba(0xe8, 0xd8, 0xc0, 0xa0)
	}
	label = g.kit.Fit(label, size, r.W-mk(30), true)
	dy := 0.0
	if held {
		dy = 2
	}
	g.kit.Shadowed(dst, label, r.X+r.W/2, r.Y+(r.H-size*1.25)/2+dy, size, col, ui.Center, true)
	return c
}

// roundButton is the round wooden button with a glyph or a picture inside.
func (g *Game) roundButton(dst *ebiten.Image, r ui.Rect, label string, glyph string) ui.Click {
	c, held := g.kit.Control(dst, r, label, true)
	if dst == nil {
		return c
	}
	if held {
		r = r.Inset(2)
	}
	ui.Image(dst, g.th.round, r, ebiten.ColorScale{})
	if glyph != "" {
		g.kit.Text(dst, glyph, r.X+r.W/2, r.Y+r.H*0.18, r.H*0.5, inkDark, ui.Center, true)
	}
	return c
}

// progress draws a progress bar in the kit's frame.
func (g *Game) progress(dst *ebiten.Image, r ui.Rect, v float64) {
	if dst == nil {
		return
	}
	v = math.Max(0, math.Min(1, v))
	bx, by := mk(12), mk(4)
	g.th.bar.Draw(dst, r, by, bx, by, bx, 1)
	in := ui.Rect{X: r.X + bx*0.75, Y: r.Y + by*0.9, W: (r.W - bx*1.5) * v, H: r.H - 1.8*by}
	if in.W > 1 {
		vector.FillRect(dst, float32(in.X), float32(in.Y), float32(in.W), float32(in.H), rgb(0xe2, 0x6f, 0x1f), false)
		vector.FillRect(dst, float32(in.X), float32(in.Y), float32(in.W), float32(in.H*0.45), rgb(0xff, 0xc3, 0x5a), false)
	}
}

// plaque draws the small wooden plate behind a short label.
func (g *Game) plaque(dst *ebiten.Image, r ui.Rect) {
	ui.Image(dst, g.th.plate, r, ebiten.ColorScale{})
}

// separator draws the carved divider across a width.
func (g *Game) separator(dst *ebiten.Image, x, y, w float64) float64 {
	b := g.th.sep.Bounds()
	h := w * float64(b.Dy()) / float64(b.Dx())
	ui.Image(dst, g.th.sep, ui.Rect{X: x, Y: y, W: w, H: h}, ebiten.ColorScale{})
	return h
}

// tile fills a rectangle with the repeating board texture.
func (g *Game) tile(dst *ebiten.Image, r ui.Rect, size float64) {
	if dst == nil {
		return
	}
	sub := dst.SubImage(rectI(r)).(*ebiten.Image)
	b := g.th.board.Bounds()
	s := size / float64(b.Dx())
	for y := r.Y; y < r.Y+r.H; y += size {
		for x := r.X; x < r.X+r.W; x += size {
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(s, s)
			op.GeoM.Translate(x, y)
			sub.DrawImage(g.th.board, op)
		}
	}
}

// pentagon is the placeholder icon of a resource, until resource icons exist.
func (g *Game) pentagon(dst *ebiten.Image, cx, cy, r float64, c color.RGBA) {
	if dst == nil {
		return
	}
	pts := []float64{cx, cy - r, cx + r, cy - r*0.3, cx + r*0.7, cy + r, cx - r*0.7, cy + r, cx - r, cy - r*0.3}
	g.poly(dst, pts, c)
}

// resColors gives each resource of era 1 a color, as in the mockup.
var resColors = map[string]color.RGBA{
	"coquillages": rgb(0xf2, 0xc1, 0x4e), "baies": rgb(0xb5, 0x47, 0x6b), "bois": rgb(0x8b, 0x5a, 0x2b),
	"silex": rgb(0x6f, 0x74, 0x80), "peaux-brutes": rgb(0xa8, 0x70, 0x3f), "peaux-tannees": rgb(0xc7, 0x8b, 0x55),
	"outils-silex": rgb(0x9a, 0xa3, 0xad), "parures": rgb(0xe7, 0xd7, 0xb5), "ocre": rgb(0xc4, 0x55, 0x2d),
	"viande": rgb(0xa8, 0x3a, 0x32), "argile": rgb(0x9c, 0x6b, 0x4a), "sel": rgb(0xe6, 0xec, 0xef),
	"grain": rgb(0xd6, 0xb8, 0x66), "poteries-crues": rgb(0xb8, 0x7a, 0x52), "fresques": rgb(0x7d, 0x4e, 0x9e),
}

func resColor(id string) color.RGBA {
	if c, ok := resColors[id]; ok {
		return c
	}
	return rgb(0xcc, 0xcc, 0xcc)
}
