package sup

import (
	"polydawn.net/go-sup/latch"
)

type Supervisor struct {
	ctrlChan_spawn    chan msg_spawn   // pass spawn instructions to maintactor
	ctrlChan_winddown chan struct{}    // signalled when controller strategy returns
	childBellcord     chan interface{} // gather child completion events

	wards map[Witness]Chaperon

	latch_windingDown latch.Latch // when no new tasks can be accepted
	latch_done        latch.Latch // when all tasks are done
}

func newSupervisor() *Supervisor {
	return &Supervisor{
		ctrlChan_spawn:    make(chan msg_spawn),
		ctrlChan_winddown: make(chan struct{}),
		childBellcord:     make(chan interface{}),

		wards: make(map[Witness]Chaperon),

		latch_windingDown: latch.New(),
		latch_done:        latch.New(),
	}
}

type supervisorState byte

const (
	supervisorState_uninitialized supervisorState = iota
	supervisorState_started
	supervisorState_windingDown
	supervisorState_done
)

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
		if err != nil {
			// TODO uffdah
			// So this needs to implode aggressively -- pushing the maint actor into a new (higher priority) state, which then starts firing cancels down.
			// It also needs somewhere to boil out itself.  And we kinda left that behind, somehow.  :(

			// ... Is there something we should do to boil out errors for children that are clearly not being witnessed by the controller that spawned them?
			// ..... Probably?
			// ........ Does that mean we should actually make that *normal*, and you have to intercept explicitly if you want to handle it better?
			//    You can't really do that very well.  It's hard to force the controller to panic.  (This is the interruption-impossibility limit, in fact.)
			//     But we could still certainly push errors *up* by default, so the next level higher supervisory code can react promptly (and then
			//      if your process that didn't handle it wants to be the badly behaved non-cancel-compliant one when that parent shuts down, that's
			//      fine/inevitable, and it'll be correctly reported as such (well, if we can build watchdogs that good, anyway, which is still in the "hope" phase).
		}
		// it's over.  wind'r all down.
		svr.ctrlChan_winddown <- struct{}{}
	}()

	superFn(svr)
}

/*
	The maintainence actor.

	Your controller strategy code is running in another goroutine.  This one
	is in charge of operations like collecting child status, and is
	purely internal so it can reliably handle its own blocking behavior.
*/
func (svr *Supervisor) actor() {
	state := supervisorState_started
	for {
		select {
		case reqSpawn := <-svr.ctrlChan_spawn:
			if state > supervisorState_started {
				panic("supervisor already winding down") // TODO return a witness with an insta error instead?
			}

			ctrlr := newController()
			svr.wards[ctrlr] = ctrlr
			ctrlr.doneLatch.WaitSelectably(svr.childBellcord)
			go func() {
				defer ctrlr.doneLatch.Trigger()
				reqSpawn.fn(ctrlr)
			}()
			reqSpawn.ret <- ctrlr

		case childDone := <-svr.childBellcord:
			delete(svr.wards, childDone.(*controller))
			if state == supervisorState_windingDown && len(svr.wards) == 0 {
				state = supervisorState_done
				svr.latch_done.Trigger()
			}

		case <-svr.ctrlChan_winddown:
			state = supervisorState_windingDown
			svr.latch_windingDown.Trigger()
			if len(svr.wards) == 0 {
				state = supervisorState_done
				svr.latch_done.Trigger()
			}
		}
	}
}

type msg_spawn struct {
	fn  Task
	ret chan<- Witness
}
