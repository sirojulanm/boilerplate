package helper

import "time"

// DurationMS converts d to milliseconds with sub-millisecond precision.
func DurationMS(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}
