package latch

import (
	"sync/atomic"
)

type Fuse interface {
	Fire()
	IsBlown() bool
	Selectable() <-chan struct{}
}

type fuse struct {
	ch chan struct{}
	// single CAS field instead of sync.Once or even sync.Mutex, because
	//  we have a very simple application and need precisely nothing more.
	// a defer tacks on about 120ns on a scale where our entire purpose
	//  takes about 65ns.  just ain't gilding we need for an unfailable op.
	done int32
}

func NewFuse() *fuse {
	return &fuse{ch: make(chan struct{})}
}

func (f *fuse) Fire() {
	if !atomic.CompareAndSwapInt32(&f.done, 0, 1) {
		return
	}
	close(f.ch)
	return
}

func (f *fuse) IsBlown() bool {
	return atomic.LoadInt32(&f.done) == 1
}

func (f *fuse) Selectable() <-chan struct{} {
	return f.ch
}
