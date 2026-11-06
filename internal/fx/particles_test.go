package fx

import "testing"

var smoke = Emitter{
	X: 100, Y: 100, Spread: 4, Direction: -1.57, Cone: 0.3,
	SpeedMin: 20, SpeedMax: 40, LifeMin: 1, LifeMax: 2,
	SizeMin: 4, SizeMax: 8, Grow: 3,
}

func TestCapacityIsNeverExceeded(t *testing.T) {
	s := NewSystem(100, 1)
	if got := s.Emit(smoke, 150); got != 100 {
		t.Fatalf("created %d, want 100", got)
	}
	if s.Len() != 100 {
		t.Fatalf("len %d, want 100", s.Len())
	}
	if got := s.Emit(smoke, 10); got != 0 {
		t.Fatalf("a full pool must refuse new particles, created %d", got)
	}
}

func TestDeadParticlesAreRemoved(t *testing.T) {
	s := NewSystem(50, 1)
	s.Emit(smoke, 50)
	s.Update(0.5) // nobody dies before LifeMin = 1s
	if s.Len() != 50 {
		t.Fatalf("after 0.5s: %d particles, want 50", s.Len())
	}
	s.Update(2) // everyone is past LifeMax = 2s
	if s.Len() != 0 {
		t.Fatalf("after 2.5s: %d particles, want 0", s.Len())
	}
}

func TestSmokeRisesAndGrows(t *testing.T) {
	s := NewSystem(10, 7)
	s.Emit(smoke, 10)
	s.Update(0.5)
	s.Each(func(p *Particle) {
		if p.Y >= 100 {
			t.Errorf("smoke should rise, y=%v", p.Y)
		}
		if p.Size < 4+3*0.5-1e-9 {
			t.Errorf("smoke should grow, size=%v", p.Size)
		}
		if a := p.Alpha(); a < 0 || a > 1 {
			t.Errorf("alpha out of range: %v", a)
		}
	})
}

func TestDeterministicWithSameSeed(t *testing.T) {
	a, b := NewSystem(20, 42), NewSystem(20, 42)
	a.Emit(smoke, 20)
	b.Emit(smoke, 20)
	var xa, xb []float64
	a.Each(func(p *Particle) { xa = append(xa, p.X) })
	b.Each(func(p *Particle) { xb = append(xb, p.X) })
	for i := range xa {
		if xa[i] != xb[i] {
			t.Fatal("same seed must give the same particles")
		}
	}
}
