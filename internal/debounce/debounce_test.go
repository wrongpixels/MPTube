package debounce

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDebouncerOnlyFiresLast(t *testing.T) {
	var n atomic.Int32
	var last atomic.Int32
	d := New(25 * time.Millisecond)
	for i := 1; i <= 5; i++ {
		i := i
		d.Trigger(func() {
			n.Add(1)
			last.Store(int32(i))
		})
		time.Sleep(6 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	if n.Load() != 1 {
		t.Fatalf("fired %d times", n.Load())
	}
	if last.Load() != 5 {
		t.Fatalf("last=%d", last.Load())
	}
}

func TestDebouncerStopPreventsFire(t *testing.T) {
	var n atomic.Int32
	d := New(30 * time.Millisecond)
	d.Trigger(func() { n.Add(1) })
	d.Stop()
	time.Sleep(50 * time.Millisecond)
	if n.Load() != 0 {
		t.Fatal("stopped timer still fired")
	}
}
