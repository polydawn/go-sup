package sup

// this file contains the state machine functions for the inner workings of manager

/*
	The maintainence actor.

	Your controller strategy code is running in another goroutine.  This one
	is in charge of operations like collecting child status, and is
	purely internal so it can reliably handle its own blocking behavior.
*/
func (mgr *manager) run() {
	log(mgr.reportingTo.Name(), "working", nil, false)
	defer log(mgr.reportingTo.Name(), "all done", nil, false)
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
	During Accepting, new requests for writs will be accepted freely.
	This function gathers childDone signals and waits for quit or winddown instructions.
*/
func (mgr *manager) step_Accepting() mgr_step {
	select {
	case childDone := <-mgr.ctrlChan_childDone:
		mgr.reapChild(childDone)
		return mgr.step_Accepting

	case <-mgr.ctrlChan_quit.Selectable():
		mgr.stopAccepting()
		mgr.cancelAll()
		return mgr.step_Quitting
	case <-mgr.reportingTo.QuitCh():
		mgr.stopAccepting()
		mgr.cancelAll()
		return mgr.step_Quitting

	case <-mgr.ctrlChan_winddown.Selectable():
		mgr.stopAccepting()
		return mgr.step_Winddown
	}
}

/*
	During Winddown, new requests for writs will be handled, but immediately rejected.
	Winddown is reached either by setting the manager to accept no new work.
	Winddown loops until all wards have been gathered,
	then we make the final	transition: to terminated.
	Quits during winddown take us to the Quitting phase, which is mostly
	identical except for obvious reasons doesn't have to keep waiting for
	the potential of a quit signal.
*/
func (mgr *manager) step_Winddown() mgr_step {
	if len(mgr.wards) == 0 {
		return mgr.step_Terminated
	}

	select {
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
	// No other goroutines alive should have reach to this channel, so we can close it.
	close(mgr.ctrlChan_childDone)
	// It's over.  No more step functions to call.
	return nil
}

//// actions

/*
	Release a new writ, appending to wards -- or, if in any state other
	than accepting, return a thunk implement writ but rejecting any work.

	This is the only action that can be called from outside the maint actor.
*/
func (mgr *manager) releaseWrit(name string) Writ {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	// No matter what, we're responding, and it earns a name.
	writName := mgr.reportingTo.Name().New(name)
	// If outside of the accepting states, reject by responding with a doa writ.
	if !mgr.accepting {
		log(mgr.reportingTo.Name(), "manager rejected writ requisition", writName, false)
		// Send back an unusable monad.
		return &writ{
			name:  writName,
			phase: int32(WritPhase_Terminal),
		}
	}
	// Ok, we're doing it: make a new writ to track this upcoming task.
	log(mgr.reportingTo.Name(), "manager releasing writ", writName, false)
	wrt := newWrit(writName)
	// Assign our final report hook to call back home.
	wrt.afterward = func() {
		log(mgr.reportingTo.Name(), "writ turning in", writName, false)
		mgr.ctrlChan_childDone <- wrt
	}
	// Register it.
	mgr.wards[wrt] = wrt.quitFuse.Fire
	// Release it into the wild.
	return wrt
}

func (mgr *manager) stopAccepting() {
	mgr.mu.Lock()
	mgr.accepting = false
	mgr.mu.Unlock()
}

func (mgr *manager) reapChild(childDone Writ) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	log(mgr.reportingTo.Name(), "reaped child", childDone.Name(), false)
	delete(mgr.wards, childDone)
	mgr.tombstones.Push(childDone)
}

func (mgr *manager) cancelAll() {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	log(mgr.reportingTo.Name(), "manager told to cancel all!", nil, false)
	for _, cancelFn := range mgr.wards {
		cancelFn()
	}
}
