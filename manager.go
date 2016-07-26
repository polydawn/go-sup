package sup

import (
	"fmt"
	"sync"

	"go.polydawn.net/go-sup/latch"
)

type manager struct {
	reportingTo   Supervisor
	ctrlChan_quit latch.Fuse

	mu      sync.Mutex
	stop    bool
	wards   map[Writ]func() // supervisor -> cancelfunc
	results chan (error)
}

func newManager(reportingTo Supervisor) Manager {
	return &manager{
		reportingTo:   reportingTo,
		ctrlChan_quit: latch.NewFuse(),

		wards:   make(map[Writ]func()),
		results: make(chan error),
	}
}

func (mgr *manager) NewTask(name string) Writ {
	// Make a new writ to track this upcoming task.
	writName := mgr.reportingTo.Name().New(name)
	svr := &supervisor{writName, mgr.ctrlChan_quit}
	wrt := &writ{
		// FIXME partial initialization
		name:     writName,
		phase:    int32(WritPhase_Issued),
		doneFuse: latch.NewFuse(),
		svr:      svr,
	}
	// Register it.  Or bail if we have to stop now.
	if halt := func() bool {
		mgr.mu.Lock()
		defer mgr.mu.Unlock()

		if mgr.stop {
			return true
		}
		mgr.wards[wrt] = svr.ctrlChan_quit.Fire
		return false
	}(); halt {
		log(mgr.reportingTo.Name(), "manager rejected writ requisition", writName)
		return &writ{nil, 0, nil, nil, nil, nil} // FIXME not a valid thunk anymore
	} else {
		log(mgr.reportingTo.Name(), "manager releasing writ", writName)
	}

	// Fill in rest of writ now that we we've decided we're serious.
	// FIXME this is an insane amount of race, plz stop
	wrt.afterward = func() {
		mgr.mu.Lock()
		delete(mgr.wards, wrt)
		err := coerceToError(recover())
		mgr.mu.Unlock()
		mgr.results <- err
	}
	return wrt
}

func (mgr *manager) step() (halt bool) {
	/*
		We can ALMOST entirely get away without an actor for manager state.
		Creating writs: caller has power, can mutex and update things.
		Returning writs: have some power, can mutex and update things.
		Awaiting final return: has power, can yada yada.
		Selecting on an error chan: doesn't need power, is filled by returning writs.
		Proxying down quit from supervisor: shit.
		This gives me Feels, because sometimes it's shrug to do that, and other times horrid:
		In the daemonspawner example, sure, we already have a selecting actor, adding the quit shuttle is easy.
		In the simpler example... `mgr.Work(repeatedSupervRef.QuitCh())`.  Could be worse I guess.
		Other reasons to add a worker: it can do timeout checks reliably.

		So after a lot of thought: yes -- use a secretary routine.
		It's fundamentally incorrect to make the manager's understanding of the world be causally entangled with your actor (if you have one) except in very confined and polite opt-in ways.
		In other words, a `case myactor.ctrlChan <- reallySlowFunc():` should not be capable of altering how rapidly the real sequence of child events can be logged.
		The only thing we're really worried about conserving here is the amount of noise you get if you dump *all* stacks.  Which is not a normal operation; and, pretty easily to filter if you really must.

		The last generation of supervisor_internal code is mostly correct, but we can simplify it significantly.
		For example, I don't think we care anymore if the thing was quit or not.  Just whether it will serve more writs or not.
		If you call quit repeately, I don't really care; we'll proxy that call (and the fuses will noop redundant things out); and the quit just moves us to not-accepting.
	*/
	select {
	case <-mgr.reportingTo.QuitCh():
		// fixme this overreceives because you need a statemachine here and you know it
		log(mgr.reportingTo.Name(), "received quit from its supervisor", nil)
		mgr.mu.Lock()
		mgr.stop = true
		for _, cancelFn := range mgr.wards {
			cancelFn()
		}
		mgr.mu.Unlock()
	case err := <-mgr.results: // TODO Plz don't eat these errors...
		log(mgr.reportingTo.Name(), "gathered child", nil) // TODO attribution missing
		if err != nil {
			log(mgr.reportingTo.Name(), fmt.Sprintf("picked up unrecovered error: %s", err), nil) // TODO attribution missing, and err serialization janky
		}
	}
	// FIXME this concept of wrapup is badly fucked by the fact delegate calls you've already made may not have landed yet.
	// because goroutines.  they weren't ours.  eeeeeiyh.  we moved on before being able to register the intention at all.  this might be unavoidable.
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if len(mgr.wards) == 0 {
		mgr.stop = true
	}
	return len(mgr.wards) == 0
}
func (mgr *manager) Work() {
	log(mgr.reportingTo.Name(), "working", nil)
	for {
		if halt := mgr.step(); halt {
			log(mgr.reportingTo.Name(), "all done", nil)
			return
		}
	}
}
