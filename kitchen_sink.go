package sup

import (
	"fmt"
	"os"
	"sync"

	"go.polydawn.net/go-sup/latch"
)

type Agent func(Supervisor)

////

/*
	Manufactured when you tell a `Manager` you're about to give it some
	work to supervise.

	The main reason this type exists at all is so that we can capture the
	intention to run the agent function immediately, even if the `Run`
	is kicked to the other side of a new goroutine -- this is necessary
	for making sure we start every task we intended to!
	The `Agent` describing the real work to do is given as a parameter to
	another func to make sure you don't accidentally pass in the agent
	function and then forget to call the real 'go-do-it' method afterwards.
*/
type Writ struct {
	svr       Supervisor
	afterward func()
}

func (writ *Writ) Run(fn Agent) {
	if writ.svr == nil {
		// the manager started winding down before our goroutine really got started;
		// we have no choice but to quietly pack it in, because there's no one to watch us.
		return
	}
	defer writ.afterward()
	fn(writ.svr)
}

////

/*
	The interface workers look up to in order to determine when they can retire.
*/
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
	NewTask() *Writ
	Work()
	// TODO i do believe you who initialized this thing ought to be able to cancel it as well.
	// at the same time, no you can't cancel individual supervisors its spawned for agents you've delegated, because wtf is that mate.
}

type manager struct {
	reportingTo   Supervisor
	ctrlChan_quit latch.Fuse

	mu      sync.Mutex
	stop    bool
	wards   map[*Writ]func() // supervisor -> cancelfunc
	results chan (error)
}

func (mgr *manager) NewTask() *Writ {
	// Make a new writ to track this upcoming task.
	svr := &supervisor{mgr.ctrlChan_quit}
	writ := &Writ{svr: svr}
	// Register it.  Or bail if we have to stop now.
	if halt := func() bool {
		mgr.mu.Lock()
		defer mgr.mu.Unlock()

		if mgr.stop {
			return true
		}
		mgr.wards[writ] = svr.ctrlChan_quit.Fire
		return false
	}(); halt {
		return &Writ{nil, nil}
	}

	// Fill in rest of writ now that we we've decided we're serious.
	// FIXME this is an insane amount of race, plz stop
	writ.afterward = func() {
		mgr.mu.Lock()
		delete(mgr.wards, writ)
		err := coerceToError(recover())
		mgr.mu.Unlock()
		mgr.results <- err
	}
	return writ
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

		wards:   make(map[*Writ]func()),
		results: make(chan error),
	}
}
