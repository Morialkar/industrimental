package sim

import (
	"fmt"
	"math"
)

var suffixes = []string{"", "k", "M", "G", "T"}

// Format writes a number the way the game shows it: suffixes (k, M, G, T), then
// scientific notation beyond 10^15.
func Format(v float64) string {
	if v < 0 {
		return "-" + Format(-v)
	}
	if v < 1000 {
		if v == math.Trunc(v) || v >= 100 {
			return fmt.Sprintf("%.0f", math.Floor(v))
		}
		return fmt.Sprintf("%.1f", v)
	}
	if v >= 1e15 {
		return fmt.Sprintf("%.2e", v)
	}
	i := 0
	for v >= 1000 && i < len(suffixes)-1 {
		v /= 1000
		i++
	}
	if v >= 100 {
		return fmt.Sprintf("%.0f%s", v, suffixes[i])
	}
	return fmt.Sprintf("%.1f%s", v, suffixes[i])
}

// FormatDuration writes a delay briefly: 45 s, 12 min, 3 h 05.
func FormatDuration(seconds float64) string {
	switch {
	case seconds < 60:
		return fmt.Sprintf("%.0f s", math.Max(0, seconds))
	case seconds < 3600:
		return fmt.Sprintf("%.0f min", seconds/60)
	default:
		h := math.Floor(seconds / 3600)
		return fmt.Sprintf("%.0f h %02.0f", h, math.Floor((seconds-h*3600)/60))
	}
}
