// Package game is the Ebitengine entry point shared by the desktop and mobile
// builds: the world full screen, the HUD and the sheets over it, and the link
// to the simulation, the saves and the clock.
package game

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/fx"
	"github.com/morialkar/industrimental/internal/save"
	"github.com/morialkar/industrimental/internal/sim"
	"github.com/morialkar/industrimental/internal/ui"
)

// Logical resolution: portrait, 9:19.5, scaled to the device screen.
const (
	ScreenW = 720
	ScreenH = 1560
	tps     = 60
	dt      = 1.0 / tps

	simEvery     = 6 // ticks of the game per tick of the simulation (10 Hz)
	autosaveTick = 30 * tps
	awayMinimum  = 5.0 // seconds of silence that count as an absence
	currentEra   = "e01"
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
	// DataDir is where saves go. Empty: nothing is saved.
	DataDir string
	// Demo starts from an advanced game, Tab on the given screen: it shows
	// every screen in a capture without playing for two days.
	Demo bool
	Tab  int
}

// DemoScreens names the screens Options.Tab opens, by index.
var DemoScreens = []string{scrWorld, scrMarket, scrTree, scrQuests, scrEra, scrClan, scrShop, scrStation}

// Game implements ebiten.Game.
type Game struct {
	opts   Options
	bundle *content.Bundle
	era    *content.Era
	sim    *sim.Sim
	kit    *ui.Kit
	th     *theme
	locale string

	mu       sync.Mutex // guards the save, which the native shell can request
	store    *save.Store
	lastTick timeState
	started  bool

	frame   int
	t       float64 // seconds since start
	scrollX float64 // panorama offset, in logical pixels
	velX    float64 // inertia after a drag
	dragX   float64
	drag    bool

	screen   string  // what lies over the world, scrWorld for nothing
	openedAt int     // frame at which the screen opened
	sheet    string  // station whose sheet is open
	pick     string  // specialization picked in the station sheet
	sheetT   float64 // opening animation, 0 to 1
	sheetH   float64 // height of the sheet's body at the last pass
	report   *sim.OfflineReport
	toast    string
	toastT   int
	floats   []floatText
	bubbleT  float64
	bubbleN  int

	resScroll, resBase float64 // resource bar, scrolled sideways
	resDrag            bool

	focusQuest   string  // quest highlighted in the journal (from the pinned quest)
	focusPending bool    // the journal still has to scroll to it
	bodyY0       float64 // top of the sheet's body content, this pass

	selNode           string  // node shown in the tree's info panel
	treeBoard         int     // branch board the carousel rests on
	treePos, treeBase float64 // carousel position, in boards
	treeDrag          bool
	sparkFx           *fx.System
	sparks            fx.Emitter

	sky       *ebiten.Image
	hills     [3]ridge
	particles *fx.System
	smoke     fx.Emitter
	emitAcc   float64
	puff      *ebiten.Image // soft round particle texture
	white     *ebiten.Image // 1x1 white source for triangles

	done bool
}

// New loads the content and builds the scene.
func New(opts Options) (*Game, error) {
	b, err := content.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("content: %w", err)
	}
	era, ok := b.Era(currentEra)
	if !ok {
		return nil, fmt.Errorf("content: era %s missing", currentEra)
	}
	kit, err := ui.New()
	if err != nil {
		return nil, fmt.Errorf("ui: %w", err)
	}
	th, err := loadTheme(currentEra)
	if err != nil {
		return nil, err
	}
	g := &Game{
		opts:      opts,
		bundle:    b,
		era:       era,
		kit:       kit,
		th:        th,
		locale:    "fr",
		particles: fx.NewSystem(2000, 1),
		sparkFx:   fx.NewSystem(300, 2),
		sky:       newSky(),
		puff:      newPuff(64),
	}
	base := ebiten.NewImage(3, 3)
	base.Fill(color.White)
	g.white = base.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	rng := rand.New(rand.NewPCG(3, 5))
	g.hills = [3]ridge{
		newRidge(rng, 0.15, groundY-300, 150, rgb(0x8e, 0xa8, 0x8f)),
		newRidge(rng, 0.45, groundY-150, 110, rgb(0xa8, 0x92, 0x5c)),
		newRidge(rng, 1.00, groundY, 50, rgb(0x7b, 0x6a, 0x3e)),
	}
	g.smoke = fx.Emitter{
		Spread: 10, Direction: -math.Pi / 2, Cone: 0.25,
		SpeedMin: 25, SpeedMax: 50, LifeMin: 2.5, LifeMax: 3.5,
		SizeMin: 11, SizeMax: 19, Grow: 14, Gravity: -4,
	}
	g.sparks = fx.Emitter{
		Spread: 6, Direction: -math.Pi / 2, Cone: math.Pi,
		SpeedMin: 120, SpeedMax: 260, LifeMin: 0.35, LifeMax: 0.6,
		SizeMin: 9, SizeMax: 14, Grow: -12, Gravity: 300,
	}
	g.sim = sim.New(era, 1)
	if opts.Demo {
		g.prepareDemo()
		if opts.Tab > 0 && opts.Tab < len(DemoScreens) {
			g.screen = DemoScreens[opts.Tab]
			g.sheet = era.Stations[1].ID
			g.sheetT = 1
		}
	}
	for i := 0; i < int(opts.Warmup*tps); i++ {
		g.step()
	}
	return g, nil
}

// SetDataDir chooses where saves go. The native shells call it at startup,
// before the first frame, once they know their sandbox.
func (g *Game) SetDataDir(dir string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.opts.DataDir = dir
}

// Update runs at a fixed 60 ticks per second.
func (g *Game) Update() error {
	if g.done {
		return ebiten.Termination
	}
	g.kit.Update()
	g.tickWith()
	return nil
}

// tickWith runs one tick with the input the kit already holds.
func (g *Game) tickWith() {
	if !g.started {
		g.start()
	}
	g.catchUp()
	g.frame++
	g.layout(nil)
	if g.frame%simEvery == 0 {
		g.sim.Tick()
	}
	g.step()
	if g.frame%autosaveTick == 0 {
		g.Save()
	}
}

// step advances the scene by one tick.
func (g *Game) step() {
	g.t += dt
	g.moveCamera()
	// Smoke: about 170 particles per second x 3 s of life = ~500 alive.
	g.emitAcc += 170 * dt
	n := int(g.emitAcc)
	g.emitAcc -= float64(n)
	if slot, ok := g.visibleWorkshop(); ok {
		g.smoke.X, g.smoke.Y = slotX(slot)-g.scrollX+120*stationScale, groundY-330*stationScale
		g.particles.Emit(g.smoke, n)
	}
	g.particles.Update(dt)
	g.sparkFx.Update(dt)
}

// moveCamera lets the finger scroll the panorama, with inertia.
func (g *Game) moveCamera() {
	x, _, down := g.kit.Pointer()
	inPanorama := down && g.screen == scrWorld && g.report == nil && g.kit.PressedIn(worldRect) && !g.kit.PressedIn(harvestRect)
	switch {
	case inPanorama && !g.drag:
		g.drag, g.dragX, g.velX = true, x, 0
	case inPanorama && g.drag:
		d := x - g.dragX
		g.scrollX -= d
		g.velX = -d / dt
		g.dragX = x
	case !down && g.drag:
		g.drag = false
	default:
		g.scrollX += g.velX * dt
		g.velX *= math.Pow(0.05, dt) // inertia decays to 5% per second
	}
	limit := slotX(len(g.era.Stations)-1) + 200 - ScreenW
	if g.scrollX < 0 {
		g.scrollX, g.velX = 0, 0
	}
	if g.scrollX > limit {
		g.scrollX, g.velX = limit, 0
	}
}

// Draw renders one frame.
func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawImage(g.sky, nil)
	for i := range g.hills {
		g.hills[i].draw(screen, g.white, g.scrollX)
	}
	g.drawStations(screen)
	g.drawParticles(screen)
	g.drawStationLabels(screen)
	g.layout(screen)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS %.0f  TPS %.0f  P %d", ebiten.ActualFPS(), ebiten.ActualTPS(), g.particles.Len()), 16, 40)

	if g.opts.ScreenshotPath != "" && g.frame >= g.opts.ScreenshotFrame && !g.done {
		g.done = true
		if err := saveScreenshot(screen, g.opts.ScreenshotPath); err != nil {
			log.Println("screenshot:", err)
		}
	}
}

// Layout keeps a fixed logical resolution; Ebitengine scales it to the screen.
func (g *Game) Layout(_, _ int) (int, int) { return ScreenW, ScreenH }

// --- stations of the panorama ---------------------------------------------

const (
	groundY      = 1150.0
	stationScale = 0.62 // the placeholder drawings are made for a scale of 1
	slotSpacing  = 420.0
)

func slotX(i int) float64 { return 200 + float64(i)*slotSpacing }

func (g *Game) visibleWorkshop() (int, bool) {
	for i := range g.era.Stations {
		if i == 0 || !g.sim.Station(g.era.Stations[i].ID).Built {
			continue // the first slot holds the bush: no chimney
		}
		if x := slotX(i) - g.scrollX; x > 40 && x < ScreenW-40 {
			return i, true
		}
	}
	return 0, false
}

func (g *Game) drawStations(dst *ebiten.Image) {
	for i := range g.era.Stations {
		def := &g.era.Stations[i]
		x := slotX(i) - g.scrollX
		if x < -400 || x > ScreenW+400 {
			continue
		}
		st := g.sim.Station(def.ID)
		switch {
		case i == 0:
			g.drawBush(dst, x)
		case st.Built:
			g.drawWorkshop(dst, x, st.Speed)
		default:
			g.drawEmptySlot(dst, x)
		}
	}
}

// at converts a point of a station drawing (made at scale 1, origin at the
// foot of the station) to the screen.
func at(x, dx, dy float64) (float64, float64) {
	return x + dx*penScale, penGround + dy*penScale
}

// The pen draws the stations: in the panorama by default, larger in a
// station's sheet. penAlpha fades a drawing (a station not built yet).
var (
	penScale  = stationScale
	penGround = groundY
	penAlpha  = 1.0
)

// withPen draws with another scale, ground line and opacity.
func withPen(scale, ground, alpha float64, draw func()) {
	s, gr, a := penScale, penGround, penAlpha
	penScale, penGround, penAlpha = scale, ground, alpha
	defer func() { penScale, penGround, penAlpha = s, gr, a }()
	draw()
}

func (g *Game) sQuad(dst *ebiten.Image, x float64, pts [8]float64, c color.RGBA) {
	var out []float64
	for i := 0; i < 8; i += 2 {
		px, py := at(x, pts[i], pts[i+1])
		out = append(out, px, py)
	}
	g.poly(dst, out, c)
}

func (g *Game) sTri(dst *ebiten.Image, x float64, pts [6]float64, c color.RGBA) {
	var out []float64
	for i := 0; i < 6; i += 2 {
		px, py := at(x, pts[i], pts[i+1])
		out = append(out, px, py)
	}
	g.poly(dst, out, c)
}

func (g *Game) sEllipse(dst *ebiten.Image, x, dx, dy, rx, ry float64, c color.RGBA) {
	px, py := at(x, dx, dy)
	g.ellipse(dst, px, py, rx*penScale, ry*penScale, c)
}

func (g *Game) sLine(dst *ebiten.Image, x, x0, y0, x1, y1, w float64, c color.RGBA) {
	ax, ay := at(x, x0, y0)
	bx, by := at(x, x1, y1)
	g.thickLine(dst, ax, ay, bx, by, w*penScale, c)
}

// drawBush is the first station: the bush the player taps at the very start.
func (g *Game) drawBush(dst *ebiten.Image, x float64) {
	sway := 4 * math.Sin(g.t*1.3)
	g.sEllipse(dst, x, 0, 6, 130, 16, rgba(0x30, 0x24, 0x10, 0x55))
	g.sEllipse(dst, x, -60+sway, -70, 80, 70, rgb(0x5f, 0x7f, 0x4a))
	g.sEllipse(dst, x, 60+sway, -65, 85, 65, rgb(0x54, 0x72, 0x42))
	g.sEllipse(dst, x, sway*1.5, -120, 95, 80, rgb(0x6c, 0x8c, 0x55))
	for i, p := range [][2]float64{{-70, -90}, {-20, -140}, {40, -110}, {80, -60}, {-40, -50}, {10, -70}} {
		g.sEllipse(dst, x, p[0]+sway*1.5, p[1]+2*math.Sin(g.t*2+float64(i)), 11, 11, rgb(0xb0, 0x3a, 0x48))
	}
}

func (g *Game) drawEmptySlot(dst *ebiten.Image, x float64) {
	g.sQuad(dst, x, [8]float64{-150, 0, 150, 0, 130, -24, -130, -24}, rgba(0x8d, 0x84, 0x77, 0x80))
	g.sLine(dst, x, -120, -24, -120, -90, 8, rgba(0x5c, 0x40, 0x28, 0x90))
	g.sLine(dst, x, 120, -24, 120, -90, 8, rgba(0x5c, 0x40, 0x28, 0x90))
}

// drawWorkshop draws a flat-shaded placeholder made of separate layers: base,
// chimney, a rotating wheel and an oscillating hammer arm, both slowed down
// when a bottleneck holds the station back.
func (g *Game) drawWorkshop(dst *ebiten.Image, x, speed float64) {
	// Ground shadow (drawn in code, never baked into assets).
	g.sEllipse(dst, x, 0, 6, 230, 22, rgba(0x30, 0x24, 0x10, 0x55))

	// Base: stone platform + walls + two-facet roof.
	g.sQuad(dst, x, [8]float64{-210, 0, 210, 0, 190, -40, -190, -40}, rgb(0x8d, 0x84, 0x77))
	g.sQuad(dst, x, [8]float64{-170, -40, 170, -40, 170, -200, -170, -200}, rgb(0xb0, 0x83, 0x52))
	g.sQuad(dst, x, [8]float64{-170, -200, 0, -200, 0, -40, -170, -40}, rgb(0xa0, 0x74, 0x46)) // shaded half
	g.sTri(dst, x, [6]float64{-200, -200, 0, -310, 0, -200}, rgb(0x9a, 0x5b, 0x3a))
	g.sTri(dst, x, [6]float64{0, -310, 200, -200, 0, -200}, rgb(0x7f, 0x47, 0x2c))
	g.sQuad(dst, x, [8]float64{-40, -40, 40, -40, 40, -130, -40, -130}, rgb(0x4a, 0x35, 0x22)) // door

	// Chimney.
	g.sQuad(dst, x, [8]float64{95, -230, 145, -230, 140, -330, 100, -330}, rgb(0x6e, 0x66, 0x5d))

	// Wheel: rotating part with its own pivot.
	wx, wy := at(x, -245, -120)
	g.wheel(dst, wx, wy, 85*penScale, g.t*1.2*speed)

	// Hammer arm: oscillates around its pivot.
	a := -0.6 + 0.5*math.Sin(g.t*4*speed)
	px, py := 215.0, -150.0
	ex, ey := px+130*math.Cos(a), py+130*math.Sin(a)
	g.sLine(dst, x, px, py, ex, ey, 14, rgb(0x5c, 0x40, 0x28))
	g.sEllipse(dst, x, ex, ey, 26, 20, rgb(0x8a, 0x8a, 0x84))
	g.sEllipse(dst, x, px, py, 10, 10, rgb(0x3b, 0x2a, 0x1a))
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
		g.thickLine(dst, cx, cy, cx+r*0.9*math.Cos(a), cy+r*0.9*math.Sin(a), 8*penScale, rgb(0x5c, 0x40, 0x28))
	}
	g.ellipse(dst, cx, cy, 14*penScale, 14*penScale, rgb(0x3b, 0x2a, 0x1a))
}

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

// drawSparks draws the golden sparks of a harvest or an investment.
func (g *Game) drawSparks(dst *ebiten.Image) {
	size := float64(g.puff.Bounds().Dx())
	g.sparkFx.Each(func(p *fx.Particle) {
		op := &ebiten.DrawImageOptions{}
		s := p.Size / size
		op.GeoM.Translate(-size/2, -size/2)
		op.GeoM.Scale(s, s)
		op.GeoM.Translate(p.X, p.Y)
		a := float32(p.Alpha())
		op.ColorScale.Scale(1*a, 0.8*a, 0.3*a, a)
		op.Blend = ebiten.BlendLighter
		dst.DrawImage(g.puff, op)
	})
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
	r, gg, b, a := float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255*float32(penAlpha)
	for i := 0; i < n; i++ {
		vs[i] = ebiten.Vertex{DstX: float32(pts[2*i]), DstY: float32(pts[2*i+1]), SrcX: 1, SrcY: 1,
			ColorR: r * a, ColorG: gg * a, ColorB: b * a, ColorA: a}
	}
	is := make([]uint16, 0, (n-2)*3)
	for i := 1; i < n-1; i++ {
		is = append(is, 0, uint16(i), uint16(i+1))
	}
	dst.DrawTriangles(vs, is, g.white, nil)
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
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
