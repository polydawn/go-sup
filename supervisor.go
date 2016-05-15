package sup

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

/*
	Start a new supervisor.  Put the given function in charge of it.
	When the controller function returns, the supervisor will start
	winding down (it won't accept any new tasks to be launched), and
	when all outstanding tasks have completed, the supervisor will become
	done.

	This method blocks for the duration -- it will return when the
	supervisor has become done.
*/
func NewSupervisor(superFn SupervisonFn) {
	svr := newSupervisor()
	go svr.supmgr_actor()
	svr.run(superFn)
	svr.latch_done.Wait()
}

func (svr *Supervisor) Spawn(fn Task) Witness {
	retCh := make(chan Witness)
	svr.ctrlChan_spawn <- msg_spawn{fn: fn, ret: retCh}
	return <-retCh
}

func (svr *Supervisor) Fork(superFn SupervisonFn) Witness {
	retCh := make(chan Witness)
	svr.ctrlChan_fork <- msg_fork{fn: superFn, ret: retCh}
	return <-retCh
}

func (svr *Supervisor) WaitSelectably(bellcord chan<- interface{}) {
	svr.latch_done.WaitSelectably(bellcord)
}

func (svr *Supervisor) Wait() {
	svr.latch_done.Wait()
}

func (svr *Supervisor) Cancel() {
	svr.ctrlChan_quit <- beep{} // TODO this should probably tolerate multiple cancels
}
