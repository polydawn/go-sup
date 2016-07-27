package sup

// this file contains the state machine functions for the inner workings of manager

/*
	The maintainence actor.

	Your controller strategy code is running in another goroutine.  This one
	is in charge of operations like collecting child status, and is
	purely internal so it can reliably handle its own blocking behavior.
*/
func (mgr *manager) run() {
	log(mgr.reportingTo.Name(), "working", nil)
	defer log(mgr.reportingTo.Name(), "all done", nil)
	stepFn := mgr.step_Accepting
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
type mgr_step func() mgr_step

/*
	During Accepting, new requests for writs will be accepted freely, and
	quickly receive a new writ which is also registered with the manager
	for observation.
*/
func (mgr *manager) step_Accepting() mgr_step {
	select {
	case req := <-mgr.ctrlChan_spawn:
		mgr.releaseWrit(req)
		return mgr.step_Accepting

	case childDone := <-mgr.ctrlChan_childDone:
		mgr.reapChild(childDone)
		return mgr.step_Accepting

	case <-mgr.ctrlChan_quit.Selectable():
		mgr.cancelAll()
		return mgr.step_Quitting
	case <-mgr.reportingTo.QuitCh():
		mgr.cancelAll()
		return mgr.step_Quitting

	case <-mgr.ctrlChan_winddown.Selectable():
		return mgr.step_Winddown
	}
}

/*
	During Winddown, new requests for writs will be handled, but immediately rejected.
	Winddown is reached either by setting the manager to accept no new work;
	or, because a quit was sent (which will have also been forwarded to all children).
	Winddown loops until all wards have been gathered,
	then we make the final	transition: to terminated.
*/
func (mgr *manager) step_Winddown() mgr_step {
	if len(mgr.wards) == 0 {
		return mgr.step_Terminated
	}

	select {
	case req := <-mgr.ctrlChan_spawn:
		mgr.rejectPlea(req)
		return mgr.step_Winddown

	case childDone := <-mgr.ctrlChan_childDone:
		mgr.reapChild(childDone)
		return mgr.step_Winddown

	case <-mgr.ctrlChan_quit.Selectable():
		mgr.cancelAll()
		return mgr.step_Quitting
	case <-mgr.reportingTo.QuitCh():
		mgr.cancelAll()
		return mgr.step_Quitting
	}
}

/*
	During Quitting, behavior is about as per Winddown, but we've also...
	well, quit.
	There's no significant difference to this phase, other than that we no
	long select on either the winddown or quit transitions.
*/
func (mgr *manager) step_Quitting() mgr_step {
	if len(mgr.wards) == 0 {
		return mgr.step_Terminated
	}

	select {
	case req := <-mgr.ctrlChan_spawn:
		mgr.rejectPlea(req)
		return mgr.step_Quitting

	case childDone := <-mgr.ctrlChan_childDone:
		mgr.reapChild(childDone)
		return mgr.step_Quitting
	}
}

/*
	During Termination, we do some final housekeeping real quick, signaling
	our completion and then...
	that's it.
*/
func (mgr *manager) step_Terminated() mgr_step {
	// Let others see us as done.  yayy!
	mgr.doneFuse.Fire()
	// We've finally stopped selecting.  We're done.  We're out.
	// You'd better not have any more writes into here blocking.
	// FIXME : This is wrong.  Closing this is rightly a race.
	//  Someone else can totally keep calling NewWrit after we've stopped.
	//  We actually *need* to hammer that until it works correctly without
	//   a bounce through our actor, at least when we're stopping.
	close(mgr.ctrlChan_spawn)
	close(mgr.ctrlChan_childDone)
	// It's over.  No more step functions to call.
	return nil
}

//// actions

func (mgr *manager) releaseWrit(req reqWrit) {
	// Make a new writ to track this upcoming task.
	writName := mgr.reportingTo.Name().New(req.name)
	log(mgr.reportingTo.Name(), "manager releasing writ", writName)
	wrt := newWrit(writName)
	// Assign our final report hook to call back home.
	wrt.afterward = func() {
		log(mgr.reportingTo.Name(), "writ turning in", writName)
		mgr.ctrlChan_childDone <- wrt
		recover() // FIXME make this more serious again.  Although also REVIEW if maybe we want the writ to do error recovery universally.
	}
	// Register it.
	mgr.wards[wrt] = wrt.quitFuse.Fire
	// Release it into the wild.
	req.ret <- wrt
}

func (mgr *manager) rejectPlea(req reqWrit) {
	// We'll give it a name, anyway.
	writName := mgr.reportingTo.Name().New(req.name)
	log(mgr.reportingTo.Name(), "manager rejected writ requisition", writName)
	// Send back an unusable monad.
	req.ret <- &writ{
		name:  writName,
		phase: int32(WritPhase_Terminal),
	}
}

func (mgr *manager) reapChild(childDone Writ) {
	log(mgr.reportingTo.Name(), "reaped child", nil) // TODO attribution missing, writ doesn't admit own name why?
	delete(mgr.wards, childDone)
	mgr.tombstones.Push(childDone)
}

func (mgr *manager) cancelAll() {
	log(mgr.reportingTo.Name(), "manager told to cancel all!", nil)
	for _, cancelFn := range mgr.wards {
		cancelFn()
	}
}
