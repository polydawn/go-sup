package sup

import (
	"go.polydawn.net/go-sup/latch"
)

type writ struct {
	name      WritName
	quitFuse  latch.Fuse
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

func newRootWrit() Writ {
	fuse := latch.NewFuse()
	return &writ{
		quitFuse:  fuse,
		svr:       &supervisor{fuse},
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
