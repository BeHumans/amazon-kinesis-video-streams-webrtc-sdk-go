package signer

import "time"

type DateProvier struct {
	ClockOffsetMs time.Duration
}

// Get Date using config offset
func (d *DateProvier) GetDate() *time.Time {
	dateOffset := time.Now().Add(d.ClockOffsetMs * time.Millisecond)
	return &dateOffset
}
