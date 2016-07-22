package sup

import (
	"fmt"
	"os"
	"sync"

	"go.polydawn.net/go-sup/latch"
)

type Agent func(Supervisor)

////

type Supervisor interface {
	Quit() bool
	QuitCh() <-chan struct{}
}

type supervisor struct {
	ctrlChan_quit latch.Fuse // typically a copy of the one from the manager.  the supervisor is all receiving end.
}

func (super *supervisor) QuitCh() <-chan struct{} {
	return super.ctrlChan_quit.Selectable()
}

func (super *supervisor) Quit() bool {
	return super.ctrlChan_quit.IsBlown()
}

/*
	Construct a new mindless supervisor who only knows how to tell agents to quit for the day.
	Returns the supervisor and the function you call to trigger the quit.

	This mindless supervisor is useful at the root of a management tree, but otherwise
	you're better off finding someone else to report to.
*/
func NewSupervisor() (Supervisor, func()) {
	svr := &supervisor{
		ctrlChan_quit: latch.NewFuse(),
	}
	return svr, svr.ctrlChan_quit.Fire
}

////

type Manager interface {
	Delegate(Agent)
	Work()
	// TODO i do believe you who initialized this thing ought to be able to cancel it as well.
	// at the same time, no you can't cancel individual supervisors its spawned for agents you've delegated, because wtf is that mate.
}

type manager struct {
	reportingTo   Supervisor
	ctrlChan_quit latch.Fuse

	mu      sync.Mutex
	stop    bool
	wards   map[Supervisor]func() // supervisor -> cancelfunc
	results chan (error)
}

func (mgr *manager) Delegate(agent Agent) {
	println("delebate!!!")
	// Make a new supervisor for this agent to report to.
	svr := &supervisor{mgr.ctrlChan_quit}
	// Register it.
	if halt := func() bool {
		mgr.mu.Lock()
		defer mgr.mu.Unlock()

		if mgr.stop {
			return true
		}
		mgr.wards[svr] = svr.ctrlChan_quit.Fire
		return false
	}(); halt {
		return
	}

	go func() {
		// Make sure the manager will eventually hear about it, even if the agent walks out.
		defer func() {
			mgr.mu.Lock()
			delete(mgr.wards, svr)
			err := coerceToError(recover())
			mgr.mu.Unlock()
			mgr.results <- err
		}()
		// Give the agent their time in the spotlight.
		agent(svr)
		// TODO consider making this block until `Work` is called so you're less likely to accidentally orphan a manager.
	}()
}

func (mgr *manager) step() (halt bool) {
	select {
	case <-mgr.reportingTo.QuitCh():
		// fixme this overreceives because you need a statemachine here and you know it
		fmt.Fprintf(os.Stderr, "manager received quit from its supervisor\n")
		mgr.mu.Lock()
		mgr.stop = true
		for _, cancelFn := range mgr.wards {
			cancelFn()
		}
		mgr.mu.Unlock()
	case err := <-mgr.results: // TODO Plz don't eat these errors...
		fmt.Fprintf(os.Stderr, "unrecovered error: %s\n", err)
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
	println("lol!!!")
	for {
		if halt := mgr.step(); halt {
			println("mgr halting!!!")
			return
		}
	}
}

func NewManager(reportingTo Supervisor) Manager {
	return &manager{
		reportingTo:   reportingTo,
		ctrlChan_quit: latch.NewFuse(),

		wards:   make(map[Supervisor]func()),
		results: make(chan error),
	}
}
