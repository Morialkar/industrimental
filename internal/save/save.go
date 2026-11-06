// Package save stores the game state on disk in versioned, checksummed,
// compressed files, rotated over several slots so that a crash or a corrupted
// write never loses more than one save.
//
// File layout: "IMSV" magic (4 bytes) + SHA-256 of the payload (32 bytes) +
// payload, where payload is the gzip of the JSON envelope.
package save

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CurrentVersion is the schema version written by this build. Bump it and
// register a Migration whenever GameState changes incompatibly.
const CurrentVersion = 1

// Slots is the number of rotating save files.
const Slots = 3

var magic = []byte("IMSV")

// ErrNoSave is returned by Load when no valid save exists.
var ErrNoSave = errors.New("save: no valid save found")

// GameState is everything persisted between sessions.
type GameState struct {
	CurrentEra string               `json:"currentEra"`
	Eras       map[string]*EraState `json:"eras"`
	// LastTick is the wall-clock time of the last simulation step, used to
	// compute offline gains.
	LastTick time.Time `json:"lastTick"`
}

// EraState is the persisted progress of one era.
type EraState struct {
	Resources map[string]float64 `json:"resources"`
	Stations  map[string]string  `json:"stations"` // station ID -> chosen variant ID
	Nodes     []string           `json:"nodes"`    // bought tree nodes
	Quests    []string           `json:"quests"`   // completed quests
}

type envelope struct {
	Version int             `json:"version"`
	SavedAt time.Time       `json:"savedAt"`
	State   json.RawMessage `json:"state"`
}

// Migration upgrades a raw state from version From to From+1.
type Migration struct {
	From  int
	Apply func(state map[string]any) error
}

// Migrations lists the upgrade steps, in order. Empty while at version 1.
var Migrations []Migration

// Encode serializes a state into the on-disk format.
func Encode(s *GameState, savedAt time.Time) ([]byte, error) {
	stateJSON, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return encodeRaw(stateJSON, CurrentVersion, savedAt)
}

func encodeRaw(stateJSON []byte, version int, savedAt time.Time) ([]byte, error) {
	envJSON, err := json.Marshal(envelope{Version: version, SavedAt: savedAt.UTC(), State: stateJSON})
	if err != nil {
		return nil, err
	}
	var payload bytes.Buffer
	zw := gzip.NewWriter(&payload)
	if _, err := zw.Write(envJSON); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(payload.Bytes())
	out := make([]byte, 0, len(magic)+len(sum)+payload.Len())
	out = append(out, magic...)
	out = append(out, sum[:]...)
	return append(out, payload.Bytes()...), nil
}

// Decode parses the on-disk format, verifies its checksum and migrates it to
// CurrentVersion.
func Decode(data []byte) (*GameState, time.Time, error) {
	header := len(magic) + sha256.Size
	if len(data) < header || !bytes.Equal(data[:len(magic)], magic) {
		return nil, time.Time{}, errors.New("save: not a save file")
	}
	payload := data[header:]
	if sum := sha256.Sum256(payload); !bytes.Equal(sum[:], data[len(magic):header]) {
		return nil, time.Time{}, errors.New("save: checksum mismatch")
	}
	zr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return nil, time.Time{}, err
	}
	envJSON, err := io.ReadAll(zr)
	if err != nil {
		return nil, time.Time{}, err
	}
	var env envelope
	if err := json.Unmarshal(envJSON, &env); err != nil {
		return nil, time.Time{}, err
	}
	if env.Version > CurrentVersion {
		return nil, time.Time{}, fmt.Errorf("save: version %d is newer than this build (%d)", env.Version, CurrentVersion)
	}
	stateJSON := []byte(env.State)
	if env.Version < CurrentVersion {
		if stateJSON, err = migrate(stateJSON, env.Version); err != nil {
			return nil, time.Time{}, err
		}
	}
	var s GameState
	if err := json.Unmarshal(stateJSON, &s); err != nil {
		return nil, time.Time{}, err
	}
	return &s, env.SavedAt, nil
}

func migrate(stateJSON []byte, from int) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(stateJSON, &raw); err != nil {
		return nil, err
	}
	for v := from; v < CurrentVersion; v++ {
		found := false
		for _, m := range Migrations {
			if m.From == v {
				if err := m.Apply(raw); err != nil {
					return nil, fmt.Errorf("save: migration from v%d: %w", v, err)
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("save: no migration from v%d", v)
		}
	}
	return json.Marshal(raw)
}

// Store manages the rotating save slots in a directory.
type Store struct {
	Dir string
	Now func() time.Time // injectable clock, defaults to time.Now
}

func (st *Store) now() time.Time {
	if st.Now != nil {
		return st.Now()
	}
	return time.Now()
}

func (st *Store) slotPath(i int) string {
	return filepath.Join(st.Dir, fmt.Sprintf("save-%d.dat", i))
}

// Save writes the state over the oldest (or first invalid) slot, atomically.
func (st *Store) Save(s *GameState) error {
	data, err := Encode(s, st.now())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(st.Dir, 0o755); err != nil {
		return err
	}
	target := st.slotPath(st.oldestSlot())
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

// Load returns the most recent valid save, skipping corrupted slots.
func (st *Store) Load() (*GameState, error) {
	var best *GameState
	var bestAt time.Time
	for i := 0; i < Slots; i++ {
		s, at, ok := st.read(i)
		if ok && (best == nil || at.After(bestAt)) {
			best, bestAt = s, at
		}
	}
	if best == nil {
		return nil, ErrNoSave
	}
	return best, nil
}

func (st *Store) read(i int) (*GameState, time.Time, bool) {
	data, err := os.ReadFile(st.slotPath(i))
	if err != nil {
		return nil, time.Time{}, false
	}
	s, at, err := Decode(data)
	return s, at, err == nil
}

func (st *Store) oldestSlot() int {
	oldest, oldestAt := 0, time.Time{}
	for i := 0; i < Slots; i++ {
		_, at, ok := st.read(i)
		if !ok {
			return i
		}
		if i == 0 || at.Before(oldestAt) {
			oldest, oldestAt = i, at
		}
	}
	return oldest
}
