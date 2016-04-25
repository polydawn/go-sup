package sup

import (
	"polydawn.net/go-sup/latch"
)

type Supervisor struct {
	ctrlChan_spawn    chan msg_spawn   // pass spawn instructions to maintactor
	ctrlChan_winddown chan struct{}    // closes when controller strategy returns
	childBellcord     chan interface{} // gather child completion events

	wards map[Witness]Chaperon

	latch_windingDown latch.Latch // when no new tasks can be accepted
	latch_done        latch.Latch // when all tasks are done
}

func newSupervisor() *Supervisor {
	return &Supervisor{
		ctrlChan_spawn: make(chan msg_spawn),
		childBellcord:  make(chan interface{}),

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
*/
func (svr *Supervisor) run(superFn SupervisonFn) {
	defer func() {
		err := recover()
		if err != nil {
			// TODO uffdah
		}
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
		}
	}
}

type msg_spawn struct {
	fn  Task
	ret chan<- Witness
}
