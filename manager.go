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
		return &writ{nil, 0, nil, nil, nil, nil} // FIXME not a valid thunk anymore
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
