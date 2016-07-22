package sup

import (
	"go.polydawn.net/go-sup/latch"
)

type writ struct {
	name      WritName
	phase     WritPhase
	quitFuse  latch.Fuse // fire this to move to quitting
	doneFuse  latch.Fuse // we'll fire this when moving to done
	svr       Supervisor
	afterward func()
}

/*
	Phases:

		- Issued
		- InUse
		- Quitting
		- Terminal

	When first created, the phase is 'Issued'.

	When `Run(fn)` is called, the phase becomes 'InUse'.

	When `fn` returns, the phase becomes 'Terminal'.

	When `Cancel` is called, the phase is jumped to `Quitting`.

	Note that if you call `Run(fn)` and `Cancel` in parallel, the `fn` may never run.

	If `Run(fn2)` is called a second time, a panic is raised.
*/
type WritPhase int32

const (
	WritPhase_Invalid WritPhase = iota
	WritPhase_Issued
	WritPhase_InUse
	WritPhase_Quitting
	WritPhase_Terminal
	writFlag_Used int32 = 1 << 8
)

func newRootWrit() Writ {
	quitFuse := latch.NewFuse()
	return &writ{
		name:      WritName{},
		phase:     WritPhase_Issued,
		quitFuse:  quitFuse,
		doneFuse:  latch.NewFuse(),
		svr:       &supervisor{quitFuse},
		afterward: func() {},
	}
}

func (writ *writ) Run(fn Agent) {
	if writ.svr == nil {
		// the manager started winding down before our goroutine really got started;
		// we have no choice but to quietly pack it in, because there's no one to watch us.
		return
	}
	defer writ.afterward()
	fn(writ.svr)
}

func (writ *writ) Cancel() {
	writ.quitFuse.Fire()
}
