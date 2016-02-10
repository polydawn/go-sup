package latch

import (
	"sync"
)

func NewLatch() *Latch {
	return &Latch{
		bellcords: make([]chan<- struct{}, 0, 1),
	}
}

/*
	Sign up for a signal, which can be flipped exactly once.

	Much like a `sync.WaitGroup`, except it uses channels and is thus
	selectable;
	much like a `sync.Condition`, except if someone asks to wait after
	the condition has been fired, they will also immediately act as signalled.

	This is often useful for coordinating things that have "happens-after"
	requirements -- for example modelling state machine transitions
	where we want to act only when "reached ready state (and maybe further)".
*/
type Latch struct {
	sync.Mutex
	bellcords []chan<- struct{}
}

func (s *Latch) Trigger() {
	s.Lock()
	defer s.Unlock()
	if s.bellcords == nil {
		panic("cannot repeatedly drop signal")
	}
	for _, bellcord := range s.bellcords {
		bellcord <- struct{}{}
	}
	s.bellcords = nil
}

func (s *Latch) Wait(bellcord chan<- struct{}) {
	s.Lock()
	defer s.Unlock()
	if s.bellcords == nil {
		bellcord <- struct{}{}
		return
	}
	s.bellcords = append(s.bellcords, bellcord)
}
