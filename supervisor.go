package sup

type Supervisor interface {
	/*
		Spawns a new supervisor which is a child of the current supervisor,
		immediately starting the `SupervisionFn` therein,
		and returns immediately with a `Witness` that can be watched to see
		when that supervisor returns gracefully.
	*/
	NewSupervisor(Director) Witness

	/*
		Returns a channel that will be closed when you should gracefully terminate.
		Your `Director` should select on this channel during all blocking operations:
		if it is selected, the `Director` should wind things up and return as soon as possible.
	*/
	SelectableQuit() <-chan struct{}
}

type Witness interface {
	// Subscribe a channel to be signalled when this witness is done.
	WaitSelectably(chan<- interface{})

	// Block until this witness is done.
	Wait()

	// Returns the error, if the task died from a panic.
	// `Wait()`s first.
	Err() error

	// Acknowledge an error as handled, so that it won't be reported to parents.
	// It's inadvisable to call this outside of the `Director` that spawned the task,
	// since it directly impacts the way errors propagate up its supervisor tree.
	Handled()

	// Send a cancellation signal to the witnessed system.
	Cancel()
}

/*
	A `Director` is the control code you write to dictate supervisor's behavior.
	This function can spawn tasks, wait around, take orders, spawn more tasks,
	collect task results, etc -- and as long as this function continues, the
	supervisor itself is operational.

	If the `Director` panics, the supervisor is in a
	bad state, and all of its children will be killed, and the problem
	reported upwards.
	When the `Director` returns, that's the indication that this supervision
	tree will not be assigned any more tasks to babysit, and things will wrap
	up gracefully and the supervisor itself will exit when all children
	have been collected.

	A valid `Director` might just spawn one task and return.
	In this case, the supervisor will wait for that child's return, then
	itself return.

	Another valid `Director` might spawn a dozen tasks, then select on
	a channel which it responds to by spawning even more tasks.
	In this case, even if all its tasks are done, the supervisor will never
	return until the `Director` also returns.  (So, in this scenario,
	you'd probably want to write a "close" channel of sime kind into the
	body of your `Director`, so you can tell it when it's time to
	shut down.)

	You should only operate the provided `Supervisor` from within that
	`Director` -- there aren't enough mutexes to make other usage safe, and
	you probably wouldn't like the semantic races and error handling anyway.
	Treat it like another actor: that's what it is.
	(Witnesses are safe to use and pass round anywhere, though.)
*/
type Director func(Supervisor)

/*
	Zonk is an empty type, used to mark channels which never carry a signal
	-- not a single message will ever be sent -- other than their `close`.
*/
type Zonk struct{}

/*
	Start a new supervisor.  Put the given function in charge of it.
	When the controller function returns, the supervisor will start
	winding down (it won't accept any new tasks to be launched), and
	when all outstanding tasks have completed, the supervisor will become
	done.

	This method blocks for the duration -- it will return when the
	supervisor has become done.
*/
func NewRootSupervisor(director Director) {
	_, wit := newSupervisor(director)
	wit.Wait()
}
