package sup

import (
	"polydawn.net/go-sup/latch"
)

type Supervisor struct {
	reqSpawnChan  chan msg_spawn
	childBellcord chan interface{}

	noAdmittance bool
	wards        map[Witness]Chaperon

	doneLatch latch.Latch
}

func NewRootSupervisor() *Supervisor {
	svr := &Supervisor{
		reqSpawnChan:  make(chan msg_spawn),
		childBellcord: make(chan interface{}),
		wards:         make(map[Witness]Chaperon),
		doneLatch:     latch.New(),
	}
	go svr.actor()
	return svr
}

func NewReportingSupervisor(upsub Chaperon) *Supervisor {
	return &Supervisor{} // TODO
}

func (svr *Supervisor) Spawn(fn Task) Witness {
	retCh := make(chan Witness)
	svr.reqSpawnChan <- msg_spawn{fn: fn, ret: retCh}
	return <-retCh
}

func (svr *Supervisor) Wait() {
	// TODO svr.doneLatch.Wait()
}

type supervisorState byte

const (
	supervisorState_uninitialized supervisorState = iota
	supervisorState_started                       // properly initialized, ready to spawn tasks
	supervisorState_awaited                       // has been `Await`'d,
)

type msg_spawn struct {
	fn  Task
	ret chan<- Witness
}

func (svr *Supervisor) actor() {
	for {
		select {
		case reqSpawn := <-svr.reqSpawnChan:
			ctrlr := newController()
			svr.wards[ctrlr] = ctrlr
			ctrlr.doneLatch.WaitSelectably(svr.childBellcord)
			go func() {
				defer ctrlr.doneLatch.Trigger()
				reqSpawn.fn(ctrlr)
			}()
			reqSpawn.ret <- ctrlr
		case childDone := <-svr.childBellcord:
			delete(svr.wards, childDone.(*controller))
		}
	}
}
