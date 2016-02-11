package sup

import (
	"fmt"
	"time"
)

func Example() {
	(&Supervisor{}).NewWard(func(ward Ward) {
		fmt.Printf("whee, i'm an actor!")
		select {
		case <-ward.SelectableQuit():
		case <-time.After(2 * time.Second):
		}
		fmt.Printf("a lazy one!")
		ward.End() <- nil
	})
}

/*
	Nanny is the promise half of a couple.

	A `Nanny` is produced when you ask Supervisor to launch a new function.
	The Supervisor "owns" that Nanny (this is a tree), but others are
	free to witness that Nanny.  (Supervisors will not accept a Nanny that
	they did not create; there is no reparenting.  Reparenting in unix-like
	process trees allows ambiguity to creep in from all directions and
	results in systemic unpleasantness we have no interest in exploring again.)
*/
type Nanny interface {
}

type Ward interface {
	SelectableQuit() <-chan struct{} // closed when you should die
	End() chan<- error               // push up your result
}

type Supervisor struct {
	wards map[*Nanny]struct{}
}

func (s *Supervisor) NewWard(fn func(ward Ward)) {
	// placeholder
}
