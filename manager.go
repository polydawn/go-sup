package sup

import (
	"fmt"
	"sync"

	"go.polydawn.net/go-sup/latch"
	"go.polydawn.net/go-sup/sluice"
)

type manager struct {
	reportingTo Supervisor // configured at start

	ctrlChan_winddown latch.Fuse // set at init.  fired by external event.
	ctrlChan_quit     latch.Fuse // set at init.  fired by external event.
	doneFuse          latch.Fuse // set at init.  fired to announce internal state change.

	mu                 sync.Mutex      // must hold while touching wards
	accepting          bool            // must hold `mu`.  if false, may no longer append to wards.
	wards              map[Writ]func() // live writs -> cancelfunc
	ctrlChan_childDone chan Writ       // writs report here when done
	tombstones         sluice.Sluice   // of `Writ`s that are done and not yet externally ack'd.  no sync needed.
}

type (
	reqWrit struct {
		name string
		ret  chan<- Writ
	}
)

func newManager(reportingTo Supervisor) Manager {
	mgr := &manager{
		reportingTo: reportingTo,

		ctrlChan_winddown: latch.NewFuse(),
		ctrlChan_quit:     latch.NewFuse(),
		doneFuse:          latch.NewFuse(),

		accepting:          true,
		wards:              make(map[Writ]func()),
		ctrlChan_childDone: make(chan Writ),
		tombstones:         sluice.New(),
	}
	go mgr.run()
	return mgr
}

func (mgr *manager) NewTask(name string) Writ {
	return mgr.releaseWrit(name)
}

/*
	"probably what you want" to do after launching all tasks to get your
	management tree to wind up nice.

	  - Moves the manager to winddown mode (no more new tasks will be accepted).
	  - Starts gathering child statuses...
	  - If any are errors...
	    - Moves the manager to quit mode (quit signals are sent to all other children).
		- Saves that error
		- Keeps waiting
	  - When all children are done...
	  - Panic up the first error we gathered.  (The rest are lost.)
*/
func (mgr *manager) Work() {
	mgr.ctrlChan_winddown.Fire()
	var devastation error
PreDoneLoop:
	for {
		// note: if we had a true fire-drill exit mode, we'd probably
		//  have a select over `<-mgr.reportingTo.QuitCh()` here as well.
		//  but we don't really believe in that: cleanup is important.
		select {
		case rcv := <-mgr.tombstones.Next():
			writ := (rcv).(*writ)
			if writ.err != nil {
				devastation = writ.err
				mgr.ctrlChan_quit.Fire()
				break PreDoneLoop
			}
		case <-mgr.doneFuse.Selectable():
			break PreDoneLoop
		}
	}
	<-mgr.doneFuse.Selectable()
FinalizeLoop:
	for {
		select {
		case rcv := <-mgr.tombstones.Next():
			writ := (rcv).(*writ)
			if writ.err != nil {
				if devastation != nil {
					msg := fmt.Sprintf("manager gathered additional errors while shutting down: %s", writ.err)
					log(mgr.reportingTo.Name(), msg, writ.name)
					continue
				}
				devastation = writ.err
			}
		default:
			// no new tombstones should be coming, so the first time
			//  polling it blocks, we're done: leave.
			break FinalizeLoop
		}
	}
	if devastation != nil {
		panic(devastation)
	}
}
