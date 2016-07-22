package sup

import (
	"fmt"
	"sync/atomic"

	"go.polydawn.net/go-sup/latch"
)

type writ struct {
	name      WritName
	phase     int32
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

	When `Run(fn)` is called, if the phase is 'Issued', the phase becomes 'InUse';
	if the phase is 'Terminal', it stays 'Terminal' and `fn` will be ignored.

	When `fn` returns, the phase becomes 'Terminal'.

	When `Cancel` is called, the phase is jumped to 'Quitting' if `fn` is still running;
	the phase remains 'Terminal' if `fn` already returned, or if we got there via a previous `Cancel`;
	the phase jumps directly to 'Terminal' if `Run(fn)` has not yet been called.

	Note that if you call `Run(fn)` and `Cancel` in parallel, the `fn` may never run.

	If `Run(fn2)` is called a second time for any reason, a panic is raised.
*/
type WritPhase int32

const (
	WritPhase_Invalid WritPhase = iota
	WritPhase_Issued
	WritPhase_InUse
	WritPhase_Quitting
	WritPhase_Terminal

	writFlag_Used WritPhase = 1 << 8
)

func newRootWrit() Writ {
	quitFuse := latch.NewFuse()
	return &writ{
		name:      WritName{},
		phase:     int32(WritPhase_Issued),
		quitFuse:  quitFuse,
		doneFuse:  latch.NewFuse(),
		svr:       &supervisor{quitFuse},
		afterward: func() {},
	}
}

func (writ *writ) Run(fn Agent) {
	var fly bool
	for {
		fly = false
		ph := WritPhase(atomic.LoadInt32(&writ.phase))
		if ph&writFlag_Used != 0 {
			panic("it is not valid to use a writ more than once")
		}
		var next WritPhase
		switch ph {
		case WritPhase_Issued:
			fly = true
			next = WritPhase_InUse
		case WritPhase_Terminal:
			fly = false
			next = WritPhase_Terminal
		case WritPhase_InUse, WritPhase_Quitting:
			// these statespaces should be unreachable because `writFlag_Used` already covers them.
			fallthrough
		default:
			panic(fmt.Sprintf("invalid writ state %d", ph))
		}
		next = next | writFlag_Used
		if atomic.CompareAndSwapInt32(&writ.phase, int32(ph), int32(next)) {
			break
		}
	}
	if !fly {
		// the writ was cancelled before our goroutine really got started;
		//  we have no choice but to quietly pack it in.
		return
	}
	defer writ.afterward()
	fn(writ.svr)
}

func (writ *writ) Cancel() {
	writ.quitFuse.Fire()
	for {
		ph := WritPhase(atomic.LoadInt32(&writ.phase))
		var next WritPhase
		switch ph {
		case WritPhase_Issued:
			next = WritPhase_Terminal
		case WritPhase_InUse:
			next = WritPhase_Quitting
		case WritPhase_Terminal, WritPhase_Quitting:
			return // we're already wrapping up or full halted: great.
		default:
			panic(fmt.Sprintf("invalid writ state %d", ph))
		}
		next = next | (ph & writFlag_Used)
		if atomic.CompareAndSwapInt32(&writ.phase, int32(ph), int32(next)) {
			break
		}
	}
}
