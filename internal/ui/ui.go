// Package ui is the home-grown interface kit: text, buttons with long press,
// and scrolling lists with inertia, driven by touch or mouse.
//
// It is immediate mode. The same layout function runs twice: in Update with a
// nil destination, where widgets read the input and report clicks, and in Draw
// with the screen, where they only paint. Ebitengine may skip or repeat Draw,
// so no decision is ever taken there.
package ui

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	longPressTicks = 27   // 450 ms at 60 ticks per second
	dragSlop       = 14.0 // finger travel that turns a press into a scroll
	friction       = 0.05 // share of a list's speed left after one second
	tickSeconds    = 1.0 / 60
)

// Align is the horizontal alignment of a text.
type Align int

const (
	Left Align = iota
	Center
	Right
)

// Rect is a rectangle in logical pixels.
type Rect struct{ X, Y, W, H float64 }

// Contains tells whether a point lies inside.
func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

// Inset shrinks a rectangle on every side.
func (r Rect) Inset(d float64) Rect { return Rect{r.X + d, r.Y + d, r.W - 2*d, r.H - 2*d} }

// Kit holds the input state and the font faces.
type Kit struct {
	regular, bold *text.GoTextFaceSource

	x, y           float64
	down           bool
	pressed        bool
	released       bool
	pressX, pressY float64
	held           int
	dragged        bool
	horizontal     bool // the current drag runs mostly sideways
	longFired      bool
	pressOwner     string // the list that captured the current press
	touch          bool   // the last press came from a finger

	scroll map[string]*scrollState
	clip   *Rect // current list area, during a pass
	wheel  float64

	recorded map[string]Rect // enabled buttons of the last pass, for tests
}

type scrollState struct {
	offset, vel, lastY float64
	dragging           bool
}

// New builds a kit.
func New() (*Kit, error) {
	reg, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, err
	}
	bo, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		return nil, err
	}
	return &Kit{regular: reg, bold: bo, scroll: map[string]*scrollState{}}, nil
}

// Update samples the input. Call it once per tick, before the layout pass.
func (k *Kit) Update() {
	x, y := ebiten.CursorPosition()
	fx, fy := float64(x), float64(y)
	down := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if ids := ebiten.AppendTouchIDs(nil); len(ids) > 0 {
		x, y = ebiten.TouchPosition(ids[0])
		fx, fy = float64(x), float64(y)
		down = true
		k.touch = true
	} else if k.touch && !down {
		// The finger just left the glass: a phone has no cursor, so the lift
		// happens where the finger last was, not at the mouse's (0, 0).
		fx, fy = k.x, k.y
		k.touch = false
	}
	_, k.wheel = ebiten.Wheel()
	k.Feed(fx, fy, down)
}

// Feed gives the kit the pointer's state for this tick. Update does it from
// the devices; tests do it directly to play the real interface.
func (k *Kit) Feed(x, y float64, down bool) {
	k.pressed, k.released = false, false
	k.x, k.y = x, y
	if k.recorded != nil {
		clear(k.recorded)
	}
	if down && !k.down {
		k.pressed = true
		k.pressX, k.pressY = k.x, k.y
		k.held, k.dragged, k.longFired, k.pressOwner = 0, false, false, ""
	}
	if !down && k.down {
		k.released = true
	}
	k.down = down
	if down {
		k.held++
		if !k.dragged && math.Hypot(k.x-k.pressX, k.y-k.pressY) > dragSlop {
			k.dragged = true
			k.horizontal = math.Abs(k.x-k.pressX) > math.Abs(k.y-k.pressY)
		}
	}
}

// Drag reports the current drag: how far the pointer moved since the press,
// and whether it runs sideways. ok is false when no drag is under way.
func (k *Kit) Drag() (dx, dy float64, horizontal, ok bool) {
	if !k.down || !k.dragged {
		return 0, 0, false, false
	}
	return k.x - k.pressX, k.y - k.pressY, k.horizontal, true
}

// Released tells whether the pointer was just lifted, and whether that ended
// a drag that started inside a rectangle.
func (k *Kit) Released() (released, dragged bool, pressX, pressY float64) {
	return k.released, k.dragged, k.pressX, k.pressY
}

// Pointer returns the pointer position and whether it is down.
func (k *Kit) Pointer() (x, y float64, down bool) { return k.x, k.y, k.down }

// Tapped tells whether a short press was just released inside a rectangle.
func (k *Kit) Tapped(r Rect) bool {
	return k.released && !k.dragged && !k.longFired &&
		r.Contains(k.pressX, k.pressY) && k.visible(k.pressX, k.pressY)
}

// PressedIn tells whether the current press started inside a rectangle.
func (k *Kit) PressedIn(r Rect) bool { return k.down && r.Contains(k.pressX, k.pressY) }

func (k *Kit) visible(x, y float64) bool { return k.clip == nil || k.clip.Contains(x, y) }

// --- text -----------------------------------------------------------------

func (k *Kit) face(size float64, bold bool) *text.GoTextFace {
	src := k.regular
	if bold {
		src = k.bold
	}
	return &text.GoTextFace{Source: src, Size: size}
}

// Text draws a single line. A nil destination draws nothing.
func (k *Kit) Text(dst *ebiten.Image, s string, x, y, size float64, c color.Color, a Align, bold bool) {
	if dst == nil || s == "" {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(Straight(c))
	switch a {
	case Center:
		op.PrimaryAlign = text.AlignCenter
	case Right:
		op.PrimaryAlign = text.AlignEnd
	}
	text.Draw(dst, s, k.face(size, bold), op)
}

// Shadowed draws a line with a dark drop shadow, for text laid on wood.
func (k *Kit) Shadowed(dst *ebiten.Image, s string, x, y, size float64, c color.Color, a Align, bold bool) {
	k.Text(dst, s, x, y+size*0.08, size, color.RGBA{0x28, 0x12, 0x04, 0xd8}, a, bold)
	k.Text(dst, s, x, y, size, c, a, bold)
}

// ShadowedParagraph draws wrapped shadowed text and returns its height.
func (k *Kit) ShadowedParagraph(dst *ebiten.Image, s string, x, y, size, width float64, c color.Color, bold bool) float64 {
	lines := k.Wrap(s, size, width, bold)
	lh := size * 1.3
	for i, l := range lines {
		k.Shadowed(dst, l, x, y+float64(i)*lh, size, c, Left, bold)
	}
	return float64(len(lines)) * lh
}

// Fit shortens a line with an ellipsis until it fits a width.
func (k *Kit) Fit(s string, size, width float64, bold bool) string {
	if k.Width(s, size, bold) <= width {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && k.Width(string(r)+"…", size, bold) > width {
		r = r[:len(r)-1]
	}
	return strings.TrimRight(string(r), " ") + "…"
}

// Width measures a line of text.
func (k *Kit) Width(s string, size float64, bold bool) float64 {
	w, _ := text.Measure(s, k.face(size, bold), 0)
	return w
}

// Wrap splits a text into lines that fit a width.
func (k *Kit) Wrap(s string, size, width float64, bold bool) []string {
	var lines []string
	line := ""
	word := ""
	flush := func() {
		if word == "" {
			return
		}
		next := word
		if line != "" {
			next = line + " " + word
		}
		if line != "" && k.Width(next, size, bold) > width {
			lines = append(lines, line)
			line = word
		} else {
			line = next
		}
		word = ""
	}
	for _, r := range s {
		if r == ' ' {
			flush()
		} else {
			word += string(r)
		}
	}
	flush()
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// Paragraph draws wrapped text and returns the height it took.
func (k *Kit) Paragraph(dst *ebiten.Image, s string, x, y, size, width float64, c color.Color) float64 {
	lines := k.Wrap(s, size, width, false)
	lh := size * 1.3
	for i, l := range lines {
		k.Text(dst, l, x, y+float64(i)*lh, size, c, Left, false)
	}
	return float64(len(lines)) * lh
}

// --- shapes and buttons ---------------------------------------------------

// Fill paints a rectangle.
func (k *Kit) Fill(dst *ebiten.Image, r Rect, c color.Color) {
	if dst == nil {
		return
	}
	vector.FillRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), Straight(c), false)
}

// Straight reads a color.RGBA as straight (not premultiplied) alpha, the way
// the game writes its translucent colors, and converts it for drawing.
func Straight(c color.Color) color.Color {
	if rc, ok := c.(color.RGBA); ok && rc.A < 0xff {
		return color.NRGBA(rc)
	}
	return c
}

// Style is the look of a button.
type Style struct {
	Fill, Disabled, Text, TextDisabled color.Color
	Size                               float64
}

// Click is the outcome of a button during the Update pass.
type Click struct {
	Tap  bool // a short press
	Long bool // the finger stayed down (fires once)
}

// Control is the input side of a button whose look the caller paints. In the
// Update pass (dst nil) it reports presses; in the Draw pass it tells whether
// the button shows as held down. The label names it for tests.
func (k *Kit) Control(dst *ebiten.Image, r Rect, label string, enabled bool) (out Click, held bool) {
	if dst != nil {
		return out, enabled && k.down && !k.dragged && r.Contains(k.pressX, k.pressY) && k.visible(k.pressX, k.pressY)
	}
	if !enabled {
		return out, false
	}
	if k.recorded != nil && k.visible(r.X+r.W/2, r.Y+r.H/2) {
		if _, dup := k.recorded[label]; !dup {
			k.recorded[label] = r
		}
	}
	out.Tap = k.Tapped(r)
	if k.down && !k.dragged && !k.longFired && k.held >= longPressTicks &&
		r.Contains(k.pressX, k.pressY) && r.Contains(k.x, k.y) && k.visible(k.pressX, k.pressY) {
		k.longFired = true
		out.Long = true
	}
	return out, false
}

// Button draws a flat button and, during the Update pass, reports presses.
func (k *Kit) Button(dst *ebiten.Image, r Rect, label string, enabled bool, st Style) Click {
	out, held := k.Control(dst, r, label, enabled)
	if dst == nil {
		return out
	}
	fill, txt := st.Fill, st.Text
	if !enabled {
		fill, txt = st.Disabled, st.TextDisabled
	}
	k.Fill(dst, r, fill)
	if held {
		k.Fill(dst, r, color.RGBA{0, 0, 0, 0x30})
	}
	k.Text(dst, label, r.X+r.W/2, r.Y+(r.H-st.Size*1.2)/2, st.Size, txt, Center, true)
	return out
}

// Record makes the kit list the enabled buttons of each Update pass.
func (k *Kit) Record() { k.recorded = map[string]Rect{} }

// Recorded returns where a recorded button is.
func (k *Kit) Recorded(label string) (Rect, bool) {
	r, ok := k.recorded[label]
	return r, ok
}

// RecordedPrefix returns the first recorded label with a prefix, in reading order.
func (k *Kit) RecordedPrefix(prefix string) (string, bool) {
	best, bestY := "", math.Inf(1)
	for label, r := range k.recorded {
		if strings.HasPrefix(label, prefix) && r.Y < bestY {
			best, bestY = label, r.Y
		}
	}
	return best, best != ""
}

// RecordedLabels lists the recorded buttons.
func (k *Kit) RecordedLabels() []string {
	labels := make([]string, 0, len(k.recorded))
	for label := range k.recorded {
		labels = append(labels, label)
	}
	slices.Sort(labels)
	return labels
}

// ScrollOffset is how far a list has scrolled.
func (k *Kit) ScrollOffset(id string) float64 {
	if st := k.scroll[id]; st != nil {
		return st.offset
	}
	return 0
}

// --- images ---------------------------------------------------------------

// Frame is an image cut in 9 pieces: the corners keep their size, the edges
// and the center stretch. Insets are in source pixels.
type Frame struct {
	Img        *ebiten.Image
	L, T, R, B int
}

// NewFrame cuts an image with insets given as shares of its width (l, r) and
// height (t, b), like CSS border-image-slice percentages.
func NewFrame(img *ebiten.Image, t, r, b, l float64) Frame {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	return Frame{Img: img, L: int(l * float64(w)), T: int(t * float64(h)), R: int(r * float64(w)), B: int(b * float64(h))}
}

// Draw paints the frame over a rectangle with the given border widths, in
// destination pixels. Alpha multiplies the opacity.
func (f Frame) Draw(dst *ebiten.Image, r Rect, bt, br, bb, bl float64, alpha float32) {
	if dst == nil || f.Img == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	b := f.Img.Bounds()
	sx := [4]int{b.Min.X, b.Min.X + f.L, b.Max.X - f.R, b.Max.X}
	sy := [4]int{b.Min.Y, b.Min.Y + f.T, b.Max.Y - f.B, b.Max.Y}
	bl, br = math.Min(bl, r.W/2), math.Min(br, r.W/2)
	bt, bb = math.Min(bt, r.H/2), math.Min(bb, r.H/2)
	dx := [4]float64{r.X, r.X + bl, r.X + r.W - br, r.X + r.W}
	dy := [4]float64{r.Y, r.Y + bt, r.Y + r.H - bb, r.Y + r.H}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			sw, sh := sx[i+1]-sx[i], sy[j+1]-sy[j]
			dw, dh := dx[i+1]-dx[i], dy[j+1]-dy[j]
			if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
				continue
			}
			op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			op.GeoM.Scale(dw/float64(sw), dh/float64(sh))
			op.GeoM.Translate(dx[i], dy[j])
			op.ColorScale.ScaleAlpha(alpha)
			dst.DrawImage(f.Img.SubImage(image.Rect(sx[i], sy[j], sx[i+1], sy[j+1])).(*ebiten.Image), op)
		}
	}
}

// Image draws a whole image stretched over a rectangle.
func Image(dst, img *ebiten.Image, r Rect, cs ebiten.ColorScale) {
	if dst == nil || img == nil {
		return
	}
	b := img.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(r.W/float64(b.Dx()), r.H/float64(b.Dy()))
	op.GeoM.Translate(r.X, r.Y)
	op.ColorScale = cs
	dst.DrawImage(img, op)
}

// --- scrolling list -------------------------------------------------------

// List is a vertically scrolling area: it clips what is drawn through Target
// and shifts the layout by Y.
type List struct {
	Rect Rect
	k    *Kit
	st   *scrollState
	sub  *ebiten.Image
}

// BeginList opens a scrolling area of the given content height. In the Update
// pass it also runs the drag and the inertia.
func (k *Kit) BeginList(dst *ebiten.Image, id string, r Rect, contentH float64) *List {
	st := k.scroll[id]
	if st == nil {
		st = &scrollState{}
		k.scroll[id] = st
	}
	if dst == nil {
		k.scrollUpdate(st, r, contentH)
	}
	k.clip = &r
	l := &List{Rect: r, k: k, st: st}
	if dst != nil {
		l.sub = dst.SubImage(image.Rect(int(r.X), int(r.Y), int(r.X+r.W), int(r.Y+r.H))).(*ebiten.Image)
	}
	return l
}

func (k *Kit) scrollUpdate(st *scrollState, r Rect, contentH float64) {
	maxOff := math.Max(0, contentH-r.H)
	if r.Contains(k.x, k.y) && k.wheel != 0 {
		st.offset -= k.wheel * 30
	}
	switch {
	case k.down && r.Contains(k.pressX, k.pressY) && k.dragged && !k.horizontal:
		if st.dragging {
			d := k.y - st.lastY
			st.offset -= d
			st.vel = -d / tickSeconds
		}
		st.dragging = true
		st.lastY = k.y
	default:
		st.dragging = false
		st.offset += st.vel * tickSeconds
		st.vel *= math.Pow(friction, tickSeconds)
		if math.Abs(st.vel) < 2 {
			st.vel = 0
		}
	}
	if st.offset < 0 {
		st.offset, st.vel = 0, 0
	}
	if st.offset > maxOff {
		st.offset, st.vel = maxOff, 0
	}
}

// Target is where the list's content must be drawn: a clipped view of the
// screen, or nil during the Update pass.
func (l *List) Target() *ebiten.Image { return l.sub }

// Y returns the top of the content in screen coordinates.
func (l *List) Y() float64 { return l.Rect.Y - l.st.offset }

// Offset is how far the list has scrolled.
func (l *List) Offset() float64 { return l.st.offset }

// End closes the scrolling area.
func (l *List) End() { l.k.clip = nil }

// ScrollTo moves a list to an offset; the next pass clamps it to the content.
func (k *Kit) ScrollTo(id string, offset float64) {
	st := k.scroll[id]
	if st == nil {
		st = &scrollState{}
		k.scroll[id] = st
	}
	st.offset, st.vel = offset, 0
}

// Reset brings a list back to its top.
func (k *Kit) Reset(id string) {
	if st := k.scroll[id]; st != nil {
		st.offset, st.vel = 0, 0
	}
}
