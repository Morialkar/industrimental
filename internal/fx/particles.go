// Package fx simulates particles (smoke, sparks, dust...). It is pure logic
// with no rendering dependency, so it can be unit-tested and budgeted: the
// system never holds more particles than its capacity.
package fx

import (
	"math"
	"math/rand/v2"
)

// Particle is one live particle. Angles are in radians, distances in logical
// pixels, times in seconds.
type Particle struct {
	X, Y, VX, VY  float64
	Size, Grow    float64 // current size and growth per second
	Angle, Spin   float64
	Life, MaxLife float64 // remaining and initial lifetime
	Kind          int     // free tag for the renderer (texture, tint...)
}

// Alpha fades a particle in over the first 10% of its life and out after.
func (p *Particle) Alpha() float64 {
	t := 1 - p.Life/p.MaxLife // 0 at birth, 1 at death
	if t < 0.1 {
		return t / 0.1
	}
	return 1 - (t-0.1)/0.9
}

// Emitter describes how new particles are born.
type Emitter struct {
	X, Y               float64
	Spread             float64 // radius of the spawn area
	Direction, Cone    float64 // mean direction and half-angle of the velocity
	SpeedMin, SpeedMax float64
	LifeMin, LifeMax   float64
	SizeMin, SizeMax   float64
	Grow               float64
	Gravity            float64 // added to VY every second (positive = down)
	Kind               int
}

// System owns a fixed-capacity pool of particles.
type System struct {
	parts   []Particle
	gravity []float64
	rng     *rand.Rand
}

// NewSystem creates a pool that never exceeds capacity live particles.
func NewSystem(capacity int, seed uint64) *System {
	return &System{
		parts:   make([]Particle, 0, capacity),
		gravity: make([]float64, 0, capacity),
		rng:     rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15)),
	}
}

// Len is the number of live particles.
func (s *System) Len() int { return len(s.parts) }

// Cap is the maximum number of live particles.
func (s *System) Cap() int { return cap(s.parts) }

// Emit spawns up to n particles; it returns how many were actually created
// (fewer when the pool is full).
func (s *System) Emit(e Emitter, n int) int {
	created := 0
	for ; created < n && len(s.parts) < cap(s.parts); created++ {
		r := e.Spread * math.Sqrt(s.rng.Float64())
		a := s.rng.Float64() * 2 * math.Pi
		dir := e.Direction + (s.rng.Float64()*2-1)*e.Cone
		speed := lerp(e.SpeedMin, e.SpeedMax, s.rng.Float64())
		life := lerp(e.LifeMin, e.LifeMax, s.rng.Float64())
		s.parts = append(s.parts, Particle{
			X: e.X + r*math.Cos(a), Y: e.Y + r*math.Sin(a),
			VX: speed * math.Cos(dir), VY: speed * math.Sin(dir),
			Size: lerp(e.SizeMin, e.SizeMax, s.rng.Float64()), Grow: e.Grow,
			Angle: a, Spin: (s.rng.Float64()*2 - 1) * 1.5,
			Life: life, MaxLife: life, Kind: e.Kind,
		})
		s.gravity = append(s.gravity, e.Gravity)
	}
	return created
}

// Update advances every particle by dt seconds and removes dead ones.
func (s *System) Update(dt float64) {
	for i := 0; i < len(s.parts); {
		p := &s.parts[i]
		p.Life -= dt
		if p.Life <= 0 {
			last := len(s.parts) - 1
			s.parts[i], s.gravity[i] = s.parts[last], s.gravity[last]
			s.parts, s.gravity = s.parts[:last], s.gravity[:last]
			continue
		}
		p.VY += s.gravity[i] * dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Size += p.Grow * dt
		p.Angle += p.Spin * dt
		i++
	}
}

// Each calls fn for every live particle, in no particular order.
func (s *System) Each(fn func(p *Particle)) {
	for i := range s.parts {
		fn(&s.parts[i])
	}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }
