/*
	The go-sup package provides a "supervisor" system for easily and safely
	launching concurrent tasks, while also reliably handling errors, and
	simplifying concurrent shutdowns.

	To make use of go-sup, your application functions should implement
	the `Agent` interface: this accepts a `Supervisor`.

	`Supervisor` provides a name to your task (for logging), and channels
	which your application can check to see if this service should quit.
	(This is necessary for orderly shutdown in complex applications.)

	Start your application with `sup.NewTask(yourAgentFn)`.

	Whenever your application needs to split off more worker goroutines,
	create a `Manager`.  Use the `Manager.NewTask` function to set up
	supervision for those tasks, then kick them off in parallel with `go`.
	You can wait for all the tasks, and handle their errors, using the
	manager.

	Build the rest of your application flow with channels as normal.
	go-sup stays out of the way, and provides control channels you can
	use in your application selects if you need to...
	or you can simply ignore them, if you don't need fancy handling.

	That's it.  Error handling made easy; quit signals built in throughout.
*/
package sup

import (
	"go.polydawn.net/go-sup/sluice"
)

/*
	Your functions!
*/
type Agent func(Supervisor)

/*
	A Writ is the authority and the setup for a supervisor -- create one,
	then use it to run your `Agent` function.

	Use `NewTask` to get started at the top of your program.
	After that, any of your agent functions that wants to delegate work
	should create a `Manager`, then launch tasks via `Manager.NewTask`.
*/
type Writ interface {
	/*
		There's a bigger reason for the `Writ` type than vanity naming:
		The main reason so that we can capture the intention to run a
		function immediately, even if the `Run` itself shall be kicked to
		the other side of a new goroutine.

		When using managers, this is necessary in order to make sure we start
		every task we intended to!

		The `Agent` describing the real work to do is given as a parameter to
		`Writ.Run` instead of taken as a parameter at creation time
		to make sure you don't accidentally pass in the agent
		function and then forget to call the real 'go-do-it' method afterwards.
		We need two methods because in the statement `go mgr.NewTask(taskfn)`,
		the `NewTask` call is not evaluated until arbitrarily later, yet
		for system-wide sanity and graceful termination, we need to be able
		to declare "no new tasks accepted" at a manager... and then return,
		but of course only after all already-started tasks are done.
		If task start may arbitrarily delayed, you can see the race: this is
		why we need a register step and a run step in separate methods.
	*/

	/*
		Returns the name assigned to this writ when it was created.
		Names are a list of strings, typically indicating the hierarchy of
		managers the writ was created under.
	*/
	Name() WritName

	/*
		Do the duty: run the given function using the current goroutine.
		Errors will be captured, etc; you're free (advised, even) to
		run this in a new goroutine e.g. `go thewrit.Run(anAgentFunc)`

		Returns self, to enable chaining if desired.
	*/
	Run(Agent) Writ

	/*
		Cancel the writ.  This will cause supervisor handed to a `Run` agent
		to move to its quitting state.

		Returns self, to enable chaining if desired.
	*/
	Cancel() Writ

	/*
		Return the error that was panicked from the running agent, or, wait
		until the agent has returned without error (in which case the result
		is nil).
	*/
	Err() error

	/*
		Return a channel which will be closed when the writ becomes done.
	*/
	DoneCh() <-chan struct{}
}

/*
	Prepare a new task -- it will answer to no one
	and will only be cancelled by your hand.

	Use this to manually supervise over a single Agent.
	If launching multiple Agents, create a `Manager` and use `Manager.NewTask()`.
	A Manager can help monitor, control, and cancel many tasks at once.

	Typically, `sup.NewTask()` is used only once -- at the start of your program.
*/
func NewTask(name ...string) Writ {
	writName := WritName{}
	for _, n := range name {
		writName = writName.New(n)
	}
	return newWrit(writName)
}

/*
	The interface workers look up to in order to determine when they can retire.
*/
type Supervisor interface {
	Name() WritName
	Quit() bool
	QuitCh() <-chan struct{}
}

type Manager interface {
	/*
		Tell the manager there's new stuff to do; get a writ you can use to run
		your new stuff under the manager's supervision.

		The writ may actually be a no-op'er, if the manager is no longer
		accepting new wards.
	*/
	NewTask(name string) Writ

	/*
		Halt accepting new work, and service all existing children.
		Errors raised by any children will cause the manager to cancel all
		other children and raise that first error -- returning (or panicking)
		only after all children have returned.
	*/
	Work()

	/*
		Return a channel that will eventually return a `Writ` of a completed
		child function.

		This is a `go-sup/sluice` channel -- be careful about discarding it;
		the chan will *eventually* soak a value, whether you receive it or not.

		The returned value will be a `Writ` -- the `sluice.T` type is an
		artifact of generics not being a thing.

		You don't need to call this if you find the error handling of `Work`
		to be acceptable.
	*/
	GatherChild() <-chan sluice.T

	// TODO i do believe you who initialized this thing ought to be able to cancel it as well.
	// at the same time, no you can't cancel individual supervisors its spawned for agents you've delegated, because wtf is that mate.
}

func NewManager(reportingTo Supervisor) Manager {
	return newManager(reportingTo)
}

/*
	Sets the log function used for internal debug messages.

	Don't call this in libraries.
	If you do call it in a program, do	so as early as possible;
	if you fail to set this before starting any tasks or supervisors
	(or spinning off any goroutines that will do so), then your logs may be
	of mixed format, and if you have the race detector enabled you will most
	certainly find one.
*/
func SetLogFunction(fn LogFn) {
	log = fn
}
