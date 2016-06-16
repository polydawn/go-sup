package sup

import (
	"fmt"

	"go.polydawn.net/go-sup/latch"
)

type supervisor struct {
	ctrlChan_spawn    chan msg_spawn   // pass spawn instructions to maintactor
	ctrlChan_winddown chan error       // recieves one message: when director returns
	ctrlChan_quit     latch.Fuse       // signalled to trigger quits, and then move directly to winddown
	childBellcord     chan interface{} // gather child completion events

	wards      map[Witness]*supervisor // currently living children
	tombstones map[Witness]beep        // children which exited with errors

	err        error
	latch_done latch.Latch // used to communicate that the public New method can return
}

// bits of initialization that are in common between
//  spawning a new child and setting up the root supervisor.
func newSupervisor(director Director) (*supervisor, *witness) {
	// Line up all the control structures.
	svr := &supervisor{
		ctrlChan_spawn:    make(chan msg_spawn),
		ctrlChan_winddown: make(chan error, 1),
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
		defer close(svr.ctrlChan_winddown)

		rcvr := recover()

		// No error is the easy route: it's over.  wind'r down; nothing else special.
		if rcvr == nil {
			svr.ctrlChan_winddown <- nil
			return
		}

		// If you panicked with a non-error type, you're a troll.
		var err error
		if cast, ok := rcvr.(error); ok {
			err = cast
		} else {
			err = fmt.Errorf("recovered: %T: %s", rcvr, rcvr)
		}

		// Send the error to the secretary.  It will handle the rest.
		svr.ctrlChan_winddown <- err
	}()

	director(svr)
}

type msg_spawn struct {
	director Director
	ret      chan<- Witness
}
