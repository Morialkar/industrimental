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
	longFired      bool
	pressOwner     string // the list that captured the current press

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
	down := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if ids := ebiten.AppendTouchIDs(nil); len(ids) > 0 {
		x, y = ebiten.TouchPosition(ids[0])
		down = true
	}
	_, k.wheel = ebiten.Wheel()
	k.Feed(float64(x), float64(y), down)
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
		if math.Hypot(k.x-k.pressX, k.y-k.pressY) > dragSlop {
			k.dragged = true
		}
	}
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
	op.ColorScale.ScaleWithColor(c)
	switch a {
	case Center:
		op.PrimaryAlign = text.AlignCenter
	case Right:
		op.PrimaryAlign = text.AlignEnd
	}
	text.Draw(dst, s, k.face(size, bold), op)
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
	vector.FillRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), c, false)
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

// Button draws a button and, during the Update pass, reports presses.
func (k *Kit) Button(dst *ebiten.Image, r Rect, label string, enabled bool, st Style) Click {
	var out Click
	if dst == nil && enabled && k.recorded != nil && k.visible(r.X+r.W/2, r.Y+r.H/2) {
		if _, dup := k.recorded[label]; !dup {
			k.recorded[label] = r
		}
	}
	if dst == nil && enabled {
		out.Tap = k.Tapped(r)
		if k.down && !k.dragged && !k.longFired && k.held >= longPressTicks &&
			r.Contains(k.pressX, k.pressY) && r.Contains(k.x, k.y) && k.visible(k.pressX, k.pressY) {
			k.longFired = true
			out.Long = true
		}
		return out
	}
	if dst == nil {
		return out
	}
	fill, txt := st.Fill, st.Text
	if !enabled {
		fill, txt = st.Disabled, st.TextDisabled
	}
	k.Fill(dst, r, fill)
	if enabled && k.down && !k.dragged && r.Contains(k.pressX, k.pressY) && k.visible(k.pressX, k.pressY) {
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
	case k.down && r.Contains(k.pressX, k.pressY) && k.dragged:
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

// Reset brings a list back to its top.
func (k *Kit) Reset(id string) {
	if st := k.scroll[id]; st != nil {
		st.offset, st.vel = 0, 0
	}
}
