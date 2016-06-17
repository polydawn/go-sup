package sup

import (
	"fmt"
)

/*
	The maintainence actor.

	Your controller strategy code is running in another goroutine.  This one
	is in charge of operations like collecting child status, and is
	purely internal so it can reliably handle its own blocking behavior.
*/
func (svr *supervisor) supmgr_actor() {
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

func (svr *supervisor) supmgr_stepAccepting() supmgr_step {
	select {
	case reqSpawn := <-svr.ctrlChan_spawn:
		child, wit := newSupervisor(reqSpawn.director)     // spawn new supervisor
		svr.wards[wit] = child                             // remember it as our child
		child.latch_done.WaitSelectably(svr.childBellcord) // notify ourselves when it's done
		reqSpawn.ret <- wit                                // return a witness to the caller
		return svr.supmgr_stepAccepting

	case childDone := <-svr.childBellcord:
		svr.supmgr_gatherChild(childDone.(*witness))
		return svr.supmgr_stepAccepting

	case result := <-svr.ctrlChan_winddown:
		svr.supmgr_gatherDirector(result)
		return svr.supmgr_stepWinddown

	case <-svr.ctrlChan_quit.Selectable():
		svr.supmgr_cancelAll()
		return svr.supmgr_stepQuitting
	}
}

/*
	Winddown is the only state available after the director has returned.
	We may be in winddown regardless of whether a quit was sent.
	Winddown loops until all wards have been gathered,
	then we make the final	transition: to terminated.
*/
func (svr *supervisor) supmgr_stepWinddown() supmgr_step {
	if len(svr.wards) == 0 {
		return svr.supmgr_stepTerminated
	}
	select {
	case reqSpawn := <-svr.ctrlChan_spawn:
		reqSpawn.ret <- &witnessThunk{err: fmt.Errorf("supervisor already winding down")}
		return svr.supmgr_stepWinddown
	case childDone := <-svr.childBellcord:
		svr.supmgr_gatherChild(childDone.(*witness))
		return svr.supmgr_stepWinddown
	}
}

/*
	Quitting is a state where we've been told to quit,
	so we're denying any requests to spawn more children,
	but the director still hasn't returned, so we're stuck waiting for that,
	and gathering other events same as winddown until then.
*/
func (svr *supervisor) supmgr_stepQuitting() supmgr_step {
	select {
	case reqSpawn := <-svr.ctrlChan_spawn:
		reqSpawn.ret <- &witnessThunk{err: fmt.Errorf("supervisor already winding down")}
		return svr.supmgr_stepQuitting

	case childDone := <-svr.childBellcord:
		svr.supmgr_gatherChild(childDone.(*witness))
		return svr.supmgr_stepQuitting

	case result := <-svr.ctrlChan_winddown:
		svr.supmgr_gatherDirector(result)
		return svr.supmgr_stepWinddown
	}
}

func (svr *supervisor) supmgr_stepTerminated() supmgr_step {
	// Check for tombstones from children which haven't been acknowledged.
	//  The director func has exited -- if they're not ack'd yet, we should
	//  keep raising them.
	// Note that this doesn't include the dead ringer witnesses returned when
	//  a director keeps trying to spawn after it's supposed to quit.
	// TODO : review for useful ways to gather errors, if there are multiple.
	if svr.err == nil {
		for child, _ := range svr.tombstones {
			if !child.(*witness).isHandled() {
				svr.err = fmt.Errorf("unhandled child error: %s", child.Err())
			}
		}
	}

	// Let others see us as done.  yayy!
	svr.latch_done.Trigger()

	// We've finally stopped selecting.  We're done.  We're out.
	// You'd better not have any more writes into here blocking.
	close(svr.ctrlChan_spawn)
	close(svr.childBellcord)

	// It's over.  No more step functions to call.
	return nil
}

func (svr *supervisor) supmgr_gatherDirector(result error) {
	svr.err = result
	if result != nil {
		// cancel all children.
		svr.supmgr_cancelAll()
		// we're about to jump to winddown regardless, but for consistency
		svr.ctrlChan_quit.Fire()
	}
}

func (svr *supervisor) supmgr_gatherChild(childDone *witness) {
	delete(svr.wards, childDone)

	// If the child exited with an error, continue to keep an eye on it.
	//  The error should be checked by the director --
	//  if it's not acknowledged by the time the director exits,
	//  then we'll continue to propagate it up.
	if childDone.Err() != nil {
		svr.tombstones[childDone] = beep{}
	}
}

func (svr *supervisor) supmgr_cancelAll() {
	for wit, _ := range svr.wards {
		wit.Cancel()
	}
}
