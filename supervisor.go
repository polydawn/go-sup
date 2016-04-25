package sup

import (
	"polydawn.net/go-sup/latch"
)

/*
	A `SupervisonFn` is the control code you write to dictate supervisor's behavior.
	This function can spawn tasks, wait around, take orders, spawn more tasks,
	collect task results, etc -- and as long as this function continues, the
	supervisor itself is operational.

	If the `SupervisonFn` panics, the supervisor is in a
	bad state, and all of its children will be killed, and the problem
	reported upwards.
	When the `SupervisonFn` returns, that's the indication that this supervision
	tree will not be assigned any more tasks to babysit, and things will wrap
	up gracefully and the supervisor itself will exit when all children
	have been collected.

	A valid `SupervisonFn` might just spawn one task and return.
	In this case, the supervisor will wait for that child's return, then
	itself return.

	Another valid `SupervisonFn` might spawn a dozen tasks, then select on
	a channel which it responds to by spawning even more tasks.
	In this case, even if all its tasks are done, the supervisor will never
	return until the `SupervisonFn` also returns.  (So, in this scenario,
	you'd probably want to write a "close" channel of sime kind into the
	body of your `SupervisonFn`, so you can tell it when it's time to
	shut down.)

	You should only operate the provided `Supervisor` from within that
	`SupervisonFn` -- there aren't enough mutexes to make that safe, and
	you probably wouldn't like the semantic races and error handling anyway.
	Treat it like another actor: that's what it is.
	(Witnesses are safe to use and pass round anywhere, though.)
*/
type SupervisonFn func(*Supervisor)

func NewSupervisor(superFn SupervisonFn) {
	svr := &Supervisor{
		reqSpawnChan:  make(chan msg_spawn),
		childBellcord: make(chan interface{}),
		wards:         make(map[Witness]Chaperon),
		doneLatch:     latch.New(),
	}
	go svr.actor()
	// TODO more panic-collecting fences around this
	superFn(svr)
	// TODO block for children
	return
}

func (svr *Supervisor) Spawn(fn Task) Witness {
	retCh := make(chan Witness)
	svr.reqSpawnChan <- msg_spawn{fn: fn, ret: retCh}
	return <-retCh
}

func (svr *Supervisor) Wait() {
	// TODO svr.doneLatch.Wait()
}
