package content

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// SupportedLocales are the languages every text key must exist in.
var SupportedLocales = []string{"fr", "en"}

//go:embed data/eras/*.json data/locales/*.json
var embedded embed.FS

// Bundle is the whole game content: every era plus the locale tables.
type Bundle struct {
	Eras    []Era
	Locales map[string]map[string]string // locale -> key -> text
}

// LoadEmbedded loads and validates the content compiled into the binary.
func LoadEmbedded() (*Bundle, error) {
	sub, err := fs.Sub(embedded, "data")
	if err != nil {
		return nil, err
	}
	return Load(sub)
}

// Load reads eras/*.json and locales/*.json from fsys, then validates them.
func Load(fsys fs.FS) (*Bundle, error) {
	b := &Bundle{Locales: map[string]map[string]string{}}

	eraFiles, err := fs.Glob(fsys, "eras/*.json")
	if err != nil {
		return nil, err
	}
	sort.Strings(eraFiles)
	for _, name := range eraFiles {
		var e Era
		if err := readJSON(fsys, name, &e); err != nil {
			return nil, err
		}
		b.Eras = append(b.Eras, e)
	}
	sort.Slice(b.Eras, func(i, j int) bool { return b.Eras[i].Index < b.Eras[j].Index })

	locFiles, err := fs.Glob(fsys, "locales/*.json")
	if err != nil {
		return nil, err
	}
	for _, name := range locFiles {
		table := map[string]string{}
		if err := readJSON(fsys, name, &table); err != nil {
			return nil, err
		}
		b.Locales[strings.TrimSuffix(path.Base(name), ".json")] = table
	}

	if err := Validate(b); err != nil {
		return nil, err
	}
	return b, nil
}

func readJSON(fsys fs.FS, name string, v any) error {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Era returns the era with the given ID.
func (b *Bundle) Era(id string) (*Era, bool) {
	for i := range b.Eras {
		if b.Eras[i].ID == id {
			return &b.Eras[i], true
		}
	}
	return nil, false
}

// Text resolves a text key in a locale, falling back to the key itself.
func (b *Bundle) Text(locale, key string) string {
	if t, ok := b.Locales[locale][key]; ok {
		return t
	}
	return key
}
