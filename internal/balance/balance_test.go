package balance

import (
	"testing"

	"github.com/morialkar/industrimental/internal/content"
)

// The milestone of phase 1: each of the 4 endings of era 1 is reached by a
// regular player in 1 to 2 days, whatever the luck of the contracts.
func TestEveryEndingOfEra1TakesOneToTwoDays(t *testing.T) {
	b, err := content.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	era, ok := b.Era("e01")
	if !ok {
		t.Fatal("era e01 missing")
	}
	for _, strat := range Strategies() {
		for seed := uint64(1); seed <= 5; seed++ {
			r := Play(era, strat, Regular(), seed)
			if !r.Reached {
				t.Errorf("%s (seed %d) not reached: %s", strat.Axis, seed, r.Left)
				continue
			}
			if r.Hours < 24 || r.Hours > 48 {
				t.Errorf("%s (seed %d) took %.1f h, want 24 to 48 h", strat.Axis, seed, r.Hours)
			}
		}
	}
}

func TestPlayIsDeterministic(t *testing.T) {
	b, err := content.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	era, _ := b.Era("e01")
	s := Strategies()[0]
	if a, c := Play(era, s, Regular(), 3), Play(era, s, Regular(), 3); a.Hours != c.Hours || a.Summary != c.Summary {
		t.Errorf("same seed, different games: %v / %v", a.Hours, c.Hours)
	}
}
