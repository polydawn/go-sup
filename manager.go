package sup

import (
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

func (mgr *manager) Work() {
	mgr.ctrlChan_winddown.Fire()
	<-mgr.doneFuse.Selectable()
}
