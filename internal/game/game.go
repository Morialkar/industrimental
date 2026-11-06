// Package game is the Ebitengine entry point shared by the desktop and mobile
// builds. In phase 0 it shows a test scene: a procedural low-poly station with
// moving parts, a parallax panorama that scrolls under the finger, and a
// continuous smoke plume of about 500 particles.
package game

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/fx"
)

// Logical resolution: portrait, 9:19.5, scaled to the device screen.
const (
	ScreenW = 720
	ScreenH = 1560
	tps     = 60
	dt      = 1.0 / tps
)

// Options tweak the game for tooling (screenshots in CI, benchmarks).
type Options struct {
	// ScreenshotPath, if set, saves the screen to this PNG after
	// ScreenshotFrame frames and then quits.
	ScreenshotPath  string
	ScreenshotFrame int
	// Warmup simulates this many seconds before the first frame, so that a
	// capture shows the scene in its steady state (smoke already formed).
	Warmup float64
}

// Game implements ebiten.Game.
type Game struct {
	opts    Options
	eraName string

	frame   int
	t       float64 // seconds since start
	scrollX float64 // panorama offset, in logical pixels
	velX    float64 // inertia after a drag
	dragX   int
	drag    bool

	sky       *ebiten.Image
	hills     [3]ridge
	particles *fx.System
	smoke     fx.Emitter
	emitAcc   float64
	puff      *ebiten.Image // soft round particle texture
	white     *ebiten.Image // 1x1 white source for triangles

	done bool
}

// New loads the content and builds the test scene.
func New(opts Options) (*Game, error) {
	b, err := content.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("content: %w", err)
	}
	era, ok := b.Era("e01")
	if !ok {
		return nil, fmt.Errorf("content: era e01 missing")
	}
	g := &Game{
		opts:      opts,
		eraName:   b.Text("fr", era.NameKey),
		particles: fx.NewSystem(2000, 1),
		sky:       newSky(),
		puff:      newPuff(64),
	}
	base := ebiten.NewImage(3, 3)
	base.Fill(color.White)
	g.white = base.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	rng := rand.New(rand.NewPCG(3, 5))
	g.hills = [3]ridge{
		newRidge(rng, 0.15, 520, 140, rgb(0x8e, 0xa8, 0x8f)),
		newRidge(rng, 0.45, 650, 110, rgb(0x9c, 0x8a, 0x55)),
		newRidge(rng, 1.00, 780, 60, rgb(0x7b, 0x6a, 0x3e)),
	}
	g.smoke = fx.Emitter{
		Spread: 10, Direction: -math.Pi / 2, Cone: 0.25,
		SpeedMin: 40, SpeedMax: 80, LifeMin: 2.5, LifeMax: 3.5,
		SizeMin: 18, SizeMax: 30, Grow: 22, Gravity: -6,
	}
	for i := 0; i < int(opts.Warmup*tps); i++ {
		g.step()
	}
	return g, nil
}

// Update runs at a fixed 60 ticks per second.
func (g *Game) Update() error {
	if g.done {
		return ebiten.Termination
	}
	g.frame++
	g.handleDrag()
	g.step()
	return nil
}

// step advances the scene simulation by one tick.
func (g *Game) step() {
	g.t += dt
	// Smoke: about 170 particles per second x 3 s of life = ~500 alive.
	g.emitAcc += 170 * dt
	n := int(g.emitAcc)
	g.emitAcc -= float64(n)
	sx, sy := g.chimneyTop()
	g.smoke.X, g.smoke.Y = sx, sy
	g.particles.Emit(g.smoke, n)
	g.particles.Update(dt)
}

func (g *Game) handleDrag() {
	x, _ := ebiten.CursorPosition()
	if ids := ebiten.AppendTouchIDs(nil); len(ids) > 0 {
		x, _ = ebiten.TouchPosition(ids[0])
	}
	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || len(ebiten.AppendTouchIDs(nil)) > 0
	switch {
	case pressed && !g.drag:
		g.drag, g.dragX, g.velX = true, x, 0
	case pressed && g.drag:
		d := float64(x - g.dragX)
		g.scrollX -= d
		g.velX = -d / dt
		g.dragX = x
	case !pressed && g.drag:
		g.drag = false
	default:
		g.scrollX += g.velX * dt
		g.velX *= math.Pow(0.05, dt) // inertia decays to 5% per second
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.scrollX, g.velX = 0, 0
	}
}

// Draw renders one frame.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.sky, nil)
	for i := range g.hills {
		g.hills[i].draw(screen, g.white, g.scrollX)
	}
	g.drawStation(screen)
	g.drawParticles(screen)
	g.drawPanel(screen)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(
		"Industrimental - phase 0\nEre : %s\nFPS %.0f  TPS %.0f\nParticules : %d / %d\nGlisse pour faire defiler",
		g.eraName, ebiten.ActualFPS(), ebiten.ActualTPS(), g.particles.Len(), g.particles.Cap()), 16, 16)

	if g.opts.ScreenshotPath != "" && g.frame >= g.opts.ScreenshotFrame && !g.done {
		g.done = true
		if err := saveScreenshot(screen, g.opts.ScreenshotPath); err != nil {
			fmt.Fprintln(os.Stderr, "screenshot:", err)
		}
	}
}

// Layout keeps a fixed logical resolution; Ebitengine scales it to the screen.
func (g *Game) Layout(_, _ int) (int, int) { return ScreenW, ScreenH }

// --- station -----------------------------------------------------------

const (
	stationX = 360.0 // world position of the station (scrolls with the ground)
	groundY  = 780.0
)

func (g *Game) stationScreenX() float64 { return stationX - g.scrollX }

func (g *Game) chimneyTop() (float64, float64) {
	return g.stationScreenX() + 120, groundY - 330
}

// drawStation draws a flat-shaded "workshop" made of separate layers: base,
// chimney, a rotating wheel and an oscillating hammer arm.
func (g *Game) drawStation(dst *ebiten.Image) {
	x, y := g.stationScreenX(), groundY

	// Ground shadow (drawn in code, never baked into assets).
	g.ellipse(dst, x, y+6, 230, 22, rgba(0x30, 0x24, 0x10, 0x55))

	// Base: stone platform + walls + two-facet roof.
	g.quad(dst, x-210, y, x+210, y, x+190, y-40, x-190, y-40, rgb(0x8d, 0x84, 0x77))
	g.quad(dst, x-170, y-40, x+170, y-40, x+170, y-200, x-170, y-200, rgb(0xb0, 0x83, 0x52))
	g.quad(dst, x-170, y-200, x, y-200, x, y-40, x-170, y-40, rgb(0xa0, 0x74, 0x46)) // shaded half
	g.tri(dst, x-200, y-200, x, y-310, x, y-200, rgb(0x9a, 0x5b, 0x3a))
	g.tri(dst, x, y-310, x+200, y-200, x, y-200, rgb(0x7f, 0x47, 0x2c))
	g.quad(dst, x-40, y-40, x+40, y-40, x+40, y-130, x-40, y-130, rgb(0x4a, 0x35, 0x22)) // door

	// Chimney.
	g.quad(dst, x+95, y-230, x+145, y-230, x+140, y-330, x+100, y-330, rgb(0x6e, 0x66, 0x5d))

	// Wheel: rotating part with its own pivot.
	g.wheel(dst, x-245, y-120, 85, g.t*1.2)

	// Hammer arm: oscillates around its pivot.
	a := -0.6 + 0.5*math.Sin(g.t*4)
	px, py := x+215, y-150
	ex, ey := px+130*math.Cos(a), py+130*math.Sin(a)
	g.thickLine(dst, px, py, ex, ey, 14, rgb(0x5c, 0x40, 0x28))
	g.ellipse(dst, ex, ey, 26, 20, rgb(0x8a, 0x8a, 0x84))
	g.ellipse(dst, px, py, 10, 10, rgb(0x3b, 0x2a, 0x1a))
}

func (g *Game) wheel(dst *ebiten.Image, cx, cy, r, angle float64) {
	const sides = 10
	for i := 0; i < sides; i++ {
		a0 := angle + float64(i)*2*math.Pi/sides
		a1 := angle + float64(i+1)*2*math.Pi/sides
		shade := 0.85 + 0.15*math.Cos(a0+math.Pi/4) // light from the upper left
		c := rgb(uint8(0x9b*shade), uint8(0x6b*shade), uint8(0x3f*shade))
		g.tri(dst, cx, cy, cx+r*math.Cos(a0), cy+r*math.Sin(a0), cx+r*math.Cos(a1), cy+r*math.Sin(a1), c)
	}
	for i := 0; i < 4; i++ {
		a := angle + float64(i)*math.Pi/2
		g.thickLine(dst, cx, cy, cx+r*0.9*math.Cos(a), cy+r*0.9*math.Sin(a), 8, rgb(0x5c, 0x40, 0x28))
	}
	g.ellipse(dst, cx, cy, 14, 14, rgb(0x3b, 0x2a, 0x1a))
}

// --- particles & panel --------------------------------------------------

func (g *Game) drawParticles(dst *ebiten.Image) {
	size := float64(g.puff.Bounds().Dx())
	g.particles.Each(func(p *fx.Particle) {
		op := &ebiten.DrawImageOptions{}
		s := p.Size / size
		op.GeoM.Translate(-size/2, -size/2)
		op.GeoM.Rotate(p.Angle)
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(p.X, p.Y)
		a := float32(p.Alpha() * 0.55)
		op.ColorScale.Scale(0.93*a, 0.92*a, 0.9*a, a)
		dst.DrawImage(g.puff, op)
	})
}

// drawPanel sketches where the management panel will live (bottom 45%).
func (g *Game) drawPanel(dst *ebiten.Image) {
	top := float64(ScreenH) * 0.55
	g.quad(dst, 0, top+60, ScreenW, top+60, ScreenW, ScreenH, 0, ScreenH, rgba(0x2b, 0x22, 0x18, 0xee))
	tabs := 5
	w := float64(ScreenW) / float64(tabs)
	for i := 0; i < tabs; i++ {
		c := rgba(0x4a, 0x3b, 0x2a, 0xff)
		if i == 0 {
			c = rgba(0xd9, 0x7a, 0x2b, 0xff)
		}
		g.quad(dst, float64(i)*w+6, ScreenH-110, float64(i+1)*w-6, ScreenH-110, float64(i+1)*w-6, ScreenH-20, float64(i)*w+6, ScreenH-20, c)
	}
}

// --- drawing helpers (flat-shaded triangles) -----------------------------

func (g *Game) tri(dst *ebiten.Image, x0, y0, x1, y1, x2, y2 float64, c color.RGBA) {
	g.poly(dst, []float64{x0, y0, x1, y1, x2, y2}, c)
}

func (g *Game) quad(dst *ebiten.Image, x0, y0, x1, y1, x2, y2, x3, y3 float64, c color.RGBA) {
	g.poly(dst, []float64{x0, y0, x1, y1, x2, y2, x3, y3}, c)
}

// poly fills a convex polygon given as x,y pairs (triangle fan).
func (g *Game) poly(dst *ebiten.Image, pts []float64, c color.RGBA) {
	n := len(pts) / 2
	vs := make([]ebiten.Vertex, n)
	r, gg, b, a := float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	for i := 0; i < n; i++ {
		vs[i] = ebiten.Vertex{DstX: float32(pts[2*i]), DstY: float32(pts[2*i+1]), SrcX: 1, SrcY: 1,
			ColorR: r * a, ColorG: gg * a, ColorB: b * a, ColorA: a}
	}
	is := make([]uint16, 0, (n-2)*3)
	for i := 1; i < n-1; i++ {
		is = append(is, 0, uint16(i), uint16(i+1))
	}
	dst.DrawTriangles(vs, is, g.white, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

func (g *Game) ellipse(dst *ebiten.Image, cx, cy, rx, ry float64, c color.RGBA) {
	const seg = 14 // low-poly on purpose
	pts := make([]float64, 0, seg*2)
	for i := 0; i < seg; i++ {
		a := float64(i) * 2 * math.Pi / seg
		pts = append(pts, cx+rx*math.Cos(a), cy+ry*math.Sin(a))
	}
	g.poly(dst, pts, c)
}

func (g *Game) thickLine(dst *ebiten.Image, x0, y0, x1, y1, w float64, c color.RGBA) {
	dx, dy := x1-x0, y1-y0
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	nx, ny := -dy/l*w/2, dx/l*w/2
	g.quad(dst, x0+nx, y0+ny, x1+nx, y1+ny, x1-nx, y1-ny, x0-nx, y0-ny, c)
}

func rgb(r, g, b uint8) color.RGBA     { return color.RGBA{r, g, b, 0xff} }
func rgba(r, g, b, a uint8) color.RGBA { return color.RGBA{r, g, b, a} }

// --- background --------------------------------------------------------

// ridge is one parallax layer of low-poly hills, repeating every period.
type ridge struct {
	factor float64 // parallax factor: 0 = static, 1 = moves with the ground
	baseY  float64
	ys     []float64
	col    color.RGBA
}

const ridgePeriod = 1440.0

func newRidge(rng *rand.Rand, factor, baseY, amp float64, c color.RGBA) ridge {
	const n = 16
	ys := make([]float64, n)
	for i := range ys {
		ys[i] = baseY - amp*(0.3+0.7*rng.Float64())
	}
	return ridge{factor: factor, baseY: baseY, ys: ys, col: c}
}

func (r *ridge) draw(dst *ebiten.Image, g *ebiten.Image, scrollX float64) {
	n := len(r.ys)
	step := ridgePeriod / float64(n)
	off := math.Mod(scrollX*r.factor, ridgePeriod)
	if off < 0 {
		off += ridgePeriod
	}
	gm := &Game{white: g}
	for rep := -1; rep <= 1; rep++ {
		for i := 0; i < n; i++ {
			x0 := float64(i)*step + float64(rep)*ridgePeriod - off
			x1 := x0 + step
			if x1 < 0 || x0 > ScreenW {
				continue
			}
			y0, y1 := r.ys[i], r.ys[(i+1)%n]
			// Two facets per segment, the left one lit, the right one shaded.
			mid := (x0 + x1) / 2
			light := scale(r.col, 1.08)
			gm.quad(dst, x0, y0, mid, (y0+y1)/2, mid, ScreenH, x0, ScreenH, light)
			gm.quad(dst, mid, (y0+y1)/2, x1, y1, x1, ScreenH, mid, ScreenH, r.col)
		}
	}
}

func scale(c color.RGBA, f float64) color.RGBA {
	cl := func(v uint8) uint8 { return uint8(math.Min(255, float64(v)*f)) }
	return color.RGBA{cl(c.R), cl(c.G), cl(c.B), c.A}
}

func newSky() *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, 1, ScreenH))
	top, bottom := rgb(0x7f, 0xb3, 0xc9), rgb(0xf3, 0xd9, 0xa4)
	for y := 0; y < ScreenH; y++ {
		t := math.Min(1, float64(y)/float64(ScreenH)*1.6)
		img.SetRGBA(0, y, color.RGBA{
			uint8(float64(top.R) + (float64(bottom.R)-float64(top.R))*t),
			uint8(float64(top.G) + (float64(bottom.G)-float64(top.G))*t),
			uint8(float64(top.B) + (float64(bottom.B)-float64(top.B))*t), 0xff})
	}
	col := ebiten.NewImageFromImage(img)
	sky := ebiten.NewImage(ScreenW, ScreenH)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(ScreenW, 1)
	sky.DrawImage(col, op)
	return sky
}

// newPuff renders a soft white disc used as the particle texture.
func newPuff(size int) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := float64(size-1) / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			d := math.Hypot(float64(x)-c, float64(y)-c) / c
			a := math.Max(0, 1-d)
			a = a * a * (3 - 2*a) // smoothstep
			v := uint8(a * 255)
			img.SetRGBA(x, y, color.RGBA{v, v, v, v}) // premultiplied white
		}
	}
	return ebiten.NewImageFromImage(img)
}

func saveScreenshot(screen *ebiten.Image, path string) error {
	b := screen.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	screen.ReadPixels(pix)
	img := &image.RGBA{Pix: pix, Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
