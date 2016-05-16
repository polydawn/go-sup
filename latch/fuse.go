package latch

import (
	"sync"
)

type none struct{}

type fuse struct {
	ch   chan none
	once sync.Once
}

func NewFuse() *fuse {
	return &fuse{ch: make(chan none)}
}

func (f *fuse) Fire() {
	f.once.Do(f.snap)
}

// never call this directly, is only to be handed to the once'ing
func (f *fuse) snap() {
	close(f.ch)
}

func (f *fuse) Selectable() <-chan none {
	return f.ch
}
