package util

import (
	"sync"
	"time"
)

// Debouncer delays function execution until after a quiet period.
type Debouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	duration time.Duration
}

// NewDebouncer creates a new Debouncer with the given delay.
func NewDebouncer(d time.Duration) *Debouncer {
	return &Debouncer{duration: d}
}

// Do resets the timer and schedules fn to be called after the quiet period.
func (d *Debouncer) Do(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.duration, fn)
}
