package debounce

import (
	"sync"
	"time"
)

type Debouncer struct {
	mu    sync.Mutex
	timer *time.Timer
	delay time.Duration
}

func New(delay time.Duration) *Debouncer {
	if delay <= 0 {
		delay = 400 * time.Millisecond
	}
	return &Debouncer{delay: delay}
}

func (d *Debouncer) Trigger(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.delay, fn)
}

func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
