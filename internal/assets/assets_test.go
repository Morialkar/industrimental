package assets

import "testing"

func TestEveryKitImageDecodes(t *testing.T) {
	names, err := UINames("e01")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 29 {
		t.Errorf("era 1 kit has %d images, want 29", len(names))
	}
	for _, n := range names {
		img, err := UI("e01", n)
		if err != nil {
			t.Fatal(err)
		}
		if b := img.Bounds(); b.Dx() < 64 || b.Dy() < 64 {
			t.Errorf("%s is only %dx%d", n, b.Dx(), b.Dy())
		}
	}
}
