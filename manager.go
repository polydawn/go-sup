package sup

import (
	"go.polydawn.net/go-sup/canal"
	"go.polydawn.net/go-sup/latch"
)

type manager struct {
	reportingTo Supervisor

	ctrlChan_spawn    chan reqWrit
	ctrlChan_winddown latch.Fuse
	ctrlChan_quit     latch.Fuse
	doneFuse          latch.Fuse

	ctrlChan_childDone chan Writ
	wards              map[Writ]func() // live writs -> cancelfunc
	tombstones         canal.Canal     // of `Writ`s that are done
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

		ctrlChan_spawn:    make(chan reqWrit),
		ctrlChan_winddown: latch.NewFuse(),
		ctrlChan_quit:     latch.NewFuse(),
		doneFuse:          latch.NewFuse(),

		ctrlChan_childDone: make(chan Writ),
		wards:              make(map[Writ]func()),
		tombstones:         canal.New(),
	}
	go mgr.run()
	return mgr
}

func (mgr *manager) NewTask(name string) Writ {
	// REVIEW: if we *could* make this work without queuing into the actor,
	//  it'd... make it *possible* to make the actor optional to run in parallel.
	ret := make(chan Writ, 1)
	mgr.ctrlChan_spawn <- reqWrit{name, ret}
	return <-ret
}

func (mgr *manager) Work() {
	mgr.ctrlChan_winddown.Fire()
	<-mgr.doneFuse.Selectable()
}
