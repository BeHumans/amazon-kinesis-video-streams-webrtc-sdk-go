package signer

import "time"

type DateProvier struct {
	ClockOffset time.Duration
}

// GetDate using config offset
func (d *DateProvier) GetDate() *time.Time {
	dateOffset := time.Now().Add(d.ClockOffset * time.Millisecond)
	return &dateOffset
}
