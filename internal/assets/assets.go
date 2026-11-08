// Package assets holds the game's images, embedded in the binary.
//
// The era 1 interface kit lives in ui/e01: the ChatGPT originals of
// assets/src/ui/e01, trimmed and downscaled to WebP (the same files as the
// interface mockup in docs/assets/ui).
package assets

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"path"
	"strings"

	"golang.org/x/image/webp"
)

//go:embed ui/e01/*.webp
var files embed.FS

// UI decodes one image of an era's interface kit, by its short name
// ("panneau", "cap_feu"...).
func UI(era, name string) (image.Image, error) {
	data, err := files.ReadFile(path.Join("ui", era, name+".webp"))
	if err != nil {
		return nil, fmt.Errorf("assets: %w", err)
	}
	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("assets: %s/%s: %w", era, name, err)
	}
	return img, nil
}

// UINames lists the images of an era's interface kit.
func UINames(era string) ([]string, error) {
	entries, err := files.ReadDir(path.Join("ui", era))
	if err != nil {
		return nil, fmt.Errorf("assets: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".webp"))
	}
	return names, nil
}
