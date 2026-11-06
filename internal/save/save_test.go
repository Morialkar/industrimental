package save

import (
	"errors"
	"os"
	"testing"
	"time"
)

func sample(n float64) *GameState {
	return &GameState{
		CurrentEra: "e01",
		Eras: map[string]*EraState{"e01": {
			Resources: map[string]float64{"baies": n},
			Stations:  map[string]string{"cueillette": "juteuses"},
		}},
		LastTick: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC),
	}
}

func TestRoundTrip(t *testing.T) {
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	data, err := Encode(sample(42), at)
	if err != nil {
		t.Fatal(err)
	}
	s, savedAt, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Eras["e01"].Resources["baies"]; got != 42 {
		t.Errorf("baies = %v, want 42", got)
	}
	if !savedAt.Equal(at) {
		t.Errorf("savedAt = %v, want %v", savedAt, at)
	}
}

func TestCorruptionIsDetected(t *testing.T) {
	data, _ := Encode(sample(1), time.Now())
	data[len(data)-1] ^= 0xFF
	if _, _, err := Decode(data); err == nil {
		t.Fatal("corrupted save should not decode")
	}
}

func TestStoreRotatesAndSkipsCorruptSlots(t *testing.T) {
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	st := &Store{Dir: t.TempDir(), Now: func() time.Time { clock = clock.Add(time.Minute); return clock }}

	if _, err := st.Load(); !errors.Is(err, ErrNoSave) {
		t.Fatalf("empty store: want ErrNoSave, got %v", err)
	}
	for i := 1; i <= 5; i++ { // more saves than slots: rotation must kick in
		if err := st.Save(sample(float64(i))); err != nil {
			t.Fatal(err)
		}
	}
	s, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Eras["e01"].Resources["baies"]; got != 5 {
		t.Fatalf("latest save should hold 5, got %v", got)
	}

	// Corrupt the newest slot: Load must fall back to the previous save.
	for i := 0; i < Slots; i++ {
		if _, at, ok := st.read(i); ok && at.Equal(clock) {
			if err := os.WriteFile(st.slotPath(i), []byte("garbage"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	s, err = st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Eras["e01"].Resources["baies"]; got != 4 {
		t.Fatalf("fallback save should hold 4, got %v", got)
	}
}

func TestNewerVersionIsRejected(t *testing.T) {
	data, err := encodeRaw([]byte(`{"currentEra":"e01"}`), CurrentVersion+1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Decode(data); err == nil {
		t.Fatal("a save from a newer build must be rejected")
	}
}

func TestMigrationFromOlderVersion(t *testing.T) {
	saved := Migrations
	defer func() { Migrations = saved }()
	// Pretend version CurrentVersion-1 stored the era under "era".
	Migrations = []Migration{{From: CurrentVersion - 1, Apply: func(s map[string]any) error {
		s["currentEra"] = s["era"]
		delete(s, "era")
		return nil
	}}}
	data, err := encodeRaw([]byte(`{"era":"e02"}`), CurrentVersion-1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	s, _, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if s.CurrentEra != "e02" {
		t.Fatalf("migrated currentEra = %q, want e02", s.CurrentEra)
	}
}
