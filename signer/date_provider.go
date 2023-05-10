package signer

import "time"

type DateProvier struct {
	ClockOffset time.Duration
}

// Get Date using config offset
func (d *DateProvier) GetDate() *time.Time {
	dateOffset := time.Now().Add(d.ClockOffset * time.Millisecond)
	return &dateOffset
}
