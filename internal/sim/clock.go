package sim

import "time"

// Away computes how long the player was gone since the last known time, and
// the new reference time. If the clock jumped backward, the absence counts for
// nothing and the reference stays put, so offline gains stay frozen until the
// real time has caught up. There is no server to ask: it is only a light guard.
func Away(last, now time.Time) (float64, time.Time) {
	if last.IsZero() {
		return 0, now
	}
	if now.Before(last) {
		return 0, last
	}
	return now.Sub(last).Seconds(), now
}
