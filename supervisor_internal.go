package sup

import (
	"go.polydawn.net/go-sup/latch"
)

type supervisor struct {
	ctrlChan_spawn    chan msg_spawn   // pass spawn instructions to maintactor
	ctrlChan_winddown chan beep        // signalled when controller strategy returns
	ctrlChan_quit     latch.Fuse       // signalled to trigger quits, and then move directly to winddown
	childBellcord     chan interface{} // gather child completion events

	wards map[Witness]*supervisor

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
		err := recover()
		// no error is the easy route: it's over.  wind'r down.
		if err == nil {
			svr.ctrlChan_winddown <- beep{}
			return
		}
		// if there was an error, every child should be cancelled automatically,
		//  because apparently their adult supervison has declared incompetence.
		svr.ctrlChan_quit.Fire()
		// TODO It also needs somewhere to boil out itself.  And we kinda left that behind, somehow.  :(

		// ... Is there something we should do to boil out errors for children that are clearly not being witnessed by the controller that spawned them?
		// ..... Probably?
		// ........ Does that mean we should actually make that *normal*, and you have to intercept explicitly if you want to handle it better?
		//    You can't really do that very well.  It's hard to force the controller to panic.  (This is the interruption-impossibility limit, in fact.)
		//     But we could still certainly push errors *up* by default, so the next level higher supervisory code can react promptly (and then
		//      if your process that didn't handle it wants to be the badly behaved non-cancel-compliant one when that parent shuts down, that's
		//      fine/inevitable, and it'll be correctly reported as such (well, if we can build watchdogs that good, anyway, which is still in the "hope" phase).
	}()

	director(svr)
}

type msg_spawn struct {
	director Director
	ret      chan<- Witness
}
