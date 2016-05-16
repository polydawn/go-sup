package latch

import (
	"sync"
)

type none struct{}

type fuse struct {
	ch chan none
	// mutex and custom instead of sync.Once because we neither care about
	//  a lock-free "fast path" for this application nor need defers.
	// a defer tacks on about 120ns on a scale where our entire purpose
	//  takes about 90ns.  just ain't gilding we need for an unfailable op.
	mu   sync.Mutex
	done bool
}

func NewFuse() *fuse {
	return &fuse{ch: make(chan none)}
}

func (f *fuse) Fire() {
	f.mu.Lock()
	if f.done {
		f.mu.Unlock()
		return
	}
	close(f.ch)
	f.done = true
	f.mu.Unlock()
	return
}

func (f *fuse) Selectable() <-chan none {
	return f.ch
}
