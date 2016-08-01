package sup

import (
	"fmt"
	"sync"
	"time"

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

	// While we're in the winddown state --
	//  Passively collecting results, and jump ourselves to quit in case of errors.
PreDoneLoop:
	for {
		// note: if we had a true fire-drill exit mode, we'd probably
		//  have a select over `<-mgr.reportingTo.QuitCh()` here as well.
		//  but we don't really believe in that: cleanup is important.
		select {
		case rcv := <-mgr.tombstones.Next():
			writ := (rcv).(*writ)
			if writ.err != nil {
				msg := fmt.Sprintf("manager autoquitting becomes of error child error: %s", writ.err)
				log(mgr.reportingTo.Name(), msg, writ.name)
				devastation = writ.err
				mgr.ctrlChan_quit.Fire()
				break PreDoneLoop
			}
		case <-mgr.doneFuse.Selectable():
			break PreDoneLoop
		}
	}

	// If we need to keep waiting for alldone, we also tick during it, so
	//  we can warn you about children not responding to quit reasonably quickly.
	quitTime := time.Now()
	tick := time.NewTicker(2 * time.Second)
YUNoDoneLoop:
	for {
		select {
		case <-tick.C:
			mgr.mu.Lock()
			var names []string
			for ward, _ := range mgr.wards {
				names = append(names, ward.Name().Coda())
			}
			msg := fmt.Sprintf("quit %d ago, still waiting for children: %d remaining [%s]",
				int(time.Now().Sub(quitTime).Seconds()),
				len(mgr.wards),
				names,
			)
			mgr.mu.Unlock()
			log(mgr.reportingTo.Name(), msg, nil)
		case <-mgr.doneFuse.Selectable():
			break YUNoDoneLoop
		}
	}
	tick.Stop()

	// Now that we're fully done: range over all the child tombstones, so that
	//  any errors can be raised upstream (or if we already have a little
	//   bundle of joy, at least make brief mention of others in the log).
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
				msg := fmt.Sprintf("manager gathered an error while shutting down: %s", writ.err)
				log(mgr.reportingTo.Name(), msg, writ.name)
				devastation = writ.err
			}
		default:
			// no new tombstones should be coming, so the first time
			//  polling it blocks, we're done: leave.
			break FinalizeLoop
		}
	}

	// If we collected a child error at any point, raise it now.
	if devastation != nil {
		panic(devastation)
	}
}
