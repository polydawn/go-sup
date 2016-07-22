package sup

/*
	Your functions!
*/
type Agent func(Supervisor)

/*
	The interface workers look up to in order to determine when they can retire.
*/
type Supervisor interface {
	Quit() bool
	QuitCh() <-chan struct{}
	// Name() string // this seems like it should go here because any Agent should be able to ask who it is
}

/*
	Construct a new mindless supervisor who only knows how to tell agents to quit for the day.
	Returns the supervisor and the function you call to trigger the quit.

	This mindless supervisor is useful at the root of a management tree, but otherwise
	you're better off finding someone else to report to.
*/
func NewSupervisor() (Supervisor, func()) {
	return newSupervisor()
}

type Manager interface {
	NewTask() Writ
	Work()
	// TODO i do believe you who initialized this thing ought to be able to cancel it as well.
	// at the same time, no you can't cancel individual supervisors its spawned for agents you've delegated, because wtf is that mate.
}

func NewManager(reportingTo Supervisor) Manager {
	return newManager(reportingTo)
}

/*
	Manufactured when you tell a `Manager` you're about to give it some
	work to supervise.

	The main reason this type exists at all is so that we can capture the
	intention to run the agent function immediately, even if the `Run`
	is kicked to the other side of a new goroutine -- this is necessary
	for making sure we start every task we intended to!
	The `Agent` describing the real work to do is given as a parameter to
	another func to make sure you don't accidentally pass in the agent
	function and then forget to call the real 'go-do-it' method afterwards.
*/
type Writ interface {
	Run(Agent)
}
