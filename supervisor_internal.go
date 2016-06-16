package sup

import (
	"fmt"

	"go.polydawn.net/go-sup/latch"
)

type supervisor struct {
	ctrlChan_spawn    chan msg_spawn   // pass spawn instructions to maintactor
	ctrlChan_winddown chan beep        // signalled when controller strategy returns
	ctrlChan_quit     latch.Fuse       // signalled to trigger quits, and then move directly to winddown
	childBellcord     chan interface{} // gather child completion events

	wards map[Witness]*supervisor

	err        error
	latch_done latch.Latch // used to communicate that the public New method can return
}

// bits of initialization that are in common between
//  spawning a new child and setting up the root supervisor.
func newSupervisor(director Director) (*supervisor, *witness) {
	// Line up all the control structures.
	svr := &supervisor{
		ctrlChan_spawn:    make(chan msg_spawn),
		ctrlChan_winddown: make(chan beep),
		ctrlChan_quit:     latch.NewFuse(),
		childBellcord:     make(chan interface{}),

		wards: make(map[Witness]*supervisor),
	}
	wit := &witness{
		supervisor: svr,
	}
	svr.latch_done = latch.NewWithMessage(wit)

	// Launch secretary in a goroutine.
	//  (It's all known code on its best behavior: it's not allowed to crash.)
	go svr.supmgr_actor()

	// Launch the director.
	go svr.runDirector(director)

	return svr, wit
}

func (svr *supervisor) NewSupervisor(director Director) Witness {
	retCh := make(chan Witness)
	svr.ctrlChan_spawn <- msg_spawn{director, retCh}
	return <-retCh
}

func (svr *supervisor) SelectableQuit() <-chan struct{} {
	return svr.ctrlChan_quit.Selectable()
}

/*
	Run the controller strategy.

	It will be swaddled in all sorts of error handling, etc; and when
	it returns, we'll put the supervisor into winding down mode.
	This method itself will then return after all of that (but
	the rest of the supervisor and its children may still be running).
*/
func (svr *supervisor) runDirector(director Director) {
	defer func() {
		rcvr := recover()

		// No error is the easy route: it's over.  wind'r down; nothing else special.
		if rcvr == nil {
			svr.ctrlChan_winddown <- beep{}
			return
		}

		// FIXME this crap has to be sent to the secretary too or that'd be a race

		// If you panicked with a non-error type, you're a troll.
		var err error
		if cast, ok := rcvr.(error); ok {
			err = cast
		} else {
			err = fmt.Errorf("recovered: %T: %s", rcvr, rcvr)
		}

		// Save the error.  It should be checked by the parent (or, if it's not
		//  acknowledged, the parent's secretary will continue to propagate it up).
		svr.err = err

		// If there was an error, every child should be cancelled automatically,
		//  because apparently their adult supervison has declared incompetence.
		//  (Of course, this just puts the secretary in "quitting" state; and this
		//  supervisor's Witness still won't return until those wrap-ups are all gathered).
		svr.ctrlChan_quit.Fire()
	}()

	director(svr)
}

type msg_spawn struct {
	director Director
	ret      chan<- Witness
}
