package sup

import (
	"go.polydawn.net/go-sup/latch"
)

type Supervisor struct {
	ctrlChan_spawn    chan msg_spawn   // pass spawn instructions to maintactor
	ctrlChan_fork     chan msg_fork    // pass fork instructions -- differs from spawn (this is node, that's leaf)
	ctrlChan_winddown chan beep        // signalled when controller strategy returns
	ctrlChan_quit     chan beep        // signalled to trigger quits, and then move directly to winddown
	subsupBellcord    chan interface{} // gather child completion events
	childBellcord     chan interface{} // gather child completion events

	subsups map[Witness]*Supervisor
	wards   map[Witness]Chaperon

	latch_done latch.Latch // used to communicate that the public New method can return
}

func newSupervisor() *Supervisor {
	return &Supervisor{
		ctrlChan_spawn:    make(chan msg_spawn),
		ctrlChan_fork:     make(chan msg_fork),
		ctrlChan_winddown: make(chan beep),
		ctrlChan_quit:     make(chan beep),
		subsupBellcord:    make(chan interface{}),
		childBellcord:     make(chan interface{}),

		subsups: make(map[Witness]*Supervisor),
		wards:   make(map[Witness]Chaperon),

		latch_done: latch.New(),
	}
}

/*
	Run the controller strategy.

	It will be swaddled in all sorts of error handling, etc; and when
	it returns, we'll put the supervisor into winding down mode.
	This method itself will then return after all of that (but
	the rest of the supervisor and its children may still be running).
*/
func (svr *Supervisor) run(superFn SupervisonFn) {
	defer func() {
		err := recover()
		// no error is the easy route: it's over.  wind'r down.
		if err == nil {
			svr.ctrlChan_winddown <- beep{}
			return
		}
		// if there was an error, every child should be cancelled automatically,
		//  because apparently their adult supervison has declared incompetence.
		svr.ctrlChan_quit <- beep{}
		// TODO It also needs somewhere to boil out itself.  And we kinda left that behind, somehow.  :(

		// ... Is there something we should do to boil out errors for children that are clearly not being witnessed by the controller that spawned them?
		// ..... Probably?
		// ........ Does that mean we should actually make that *normal*, and you have to intercept explicitly if you want to handle it better?
		//    You can't really do that very well.  It's hard to force the controller to panic.  (This is the interruption-impossibility limit, in fact.)
		//     But we could still certainly push errors *up* by default, so the next level higher supervisory code can react promptly (and then
		//      if your process that didn't handle it wants to be the badly behaved non-cancel-compliant one when that parent shuts down, that's
		//      fine/inevitable, and it'll be correctly reported as such (well, if we can build watchdogs that good, anyway, which is still in the "hope" phase).
	}()

	superFn(svr)
}

type msg_spawn struct {
	fn  Task
	ret chan<- Witness
}

type msg_fork struct {
	fn  SupervisonFn
	ret chan<- Witness
}
