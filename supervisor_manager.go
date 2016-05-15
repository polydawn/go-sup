package sup

/*
	The maintainence actor.

	Your controller strategy code is running in another goroutine.  This one
	is in charge of operations like collecting child status, and is
	purely internal so it can reliably handle its own blocking behavior.
*/
func (svr *Supervisor) supmgr_actor() {
	stepFn := svr.supmgr_stepAccepting
	for {
		if stepFn == nil {
			break
		}
		stepFn = stepFn()
	}
}

/*
	Steps in the state machine of the supervisor's internal maint.

	This pattern is awfully nice:
	  - you can see the transitions by clear name (returns highlight them)
	  - you *don't* see the visual clutter of code for transitions that are
	     not possible for whatever state you're currently looking at
	  - even if things really go poorly, your stack trace clearly indicates
	     exactly which state you were in (it's in the function name after all).
*/
type supmgr_step func() supmgr_step

func (svr *Supervisor) supmgr_stepAccepting() supmgr_step {
	select {
	case reqSpawn := <-svr.ctrlChan_spawn:
		ctrlr := newController()
		svr.wards[ctrlr] = ctrlr
		ctrlr.doneLatch.WaitSelectably(svr.childBellcord)
		go func() {
			defer ctrlr.doneLatch.Trigger()
			reqSpawn.fn(ctrlr)
		}()
		reqSpawn.ret <- ctrlr
		return svr.supmgr_stepAccepting

	case reqFork := <-svr.ctrlChan_fork:
		fork := newSupervisor()
		svr.subsups[fork] = fork
		fork.WaitSelectably(svr.subsupBellcord)
		go fork.supmgr_actor()
		go fork.run(reqFork.fn)
		reqFork.ret <- fork
		return svr.supmgr_stepAccepting

	case childDone := <-svr.childBellcord:
		delete(svr.wards, childDone.(*controller))
		return svr.supmgr_stepAccepting

	case subDone := <-svr.subsupBellcord:
		delete(svr.subsups, subDone.(*Supervisor))
		return svr.supmgr_stepAccepting

	case <-svr.ctrlChan_winddown:
		return svr.supmgr_stepWinddown

	case <-svr.ctrlChan_quit:
		return svr.supmgr_stepQuitting
	}
}

func (svr *Supervisor) supmgr_stepWinddown() supmgr_step {
	if len(svr.wards) == 0 {
		return svr.supmgr_stepTerminated
	}
	select {
	case _ = <-svr.ctrlChan_spawn:
		panic("supervisor already winding down") // TODO return a witness with an insta error instead?
		return svr.supmgr_stepWinddown
	case childDone := <-svr.childBellcord:
		delete(svr.wards, childDone.(*controller))
		return svr.supmgr_stepWinddown
	case subDone := <-svr.subsupBellcord:
		delete(svr.subsups, subDone.(*Supervisor))
		return svr.supmgr_stepWinddown
	case <-svr.ctrlChan_winddown:
		panic("go-sup bug, winddown transition cannot occur twice")
	case <-svr.ctrlChan_quit:
		panic("go-sup bug, cannot transition winddown->quitting") // TODO you most certainly can -- this may necessitate another state
		// also prttty sure we need a sync.Once around the quit chans
	}
}

func (svr *Supervisor) supmgr_stepQuitting() supmgr_step {
	for wit, _ := range svr.wards {
		wit.Cancel()
	}
	for wit, _ := range svr.subsups {
		wit.Cancel()
	}
	return svr.supmgr_stepWinddown
}

func (svr *Supervisor) supmgr_stepTerminated() supmgr_step {
	// can we finally stop selecting?
	// ideally other people shouldn've have *any* writable channels into us
	//  that they could possibly block on at this point.
	svr.latch_done.Trigger()
	return nil
}
