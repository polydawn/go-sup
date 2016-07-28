package sup

/*
	Gathering type to hang methods off of.

	Typical useage is via

		sup.Behaviors.Looper([...])
*/
type Behavior struct{}

var Behaviors Behavior

//// Looper

/*
	Decorates an agent to be invoked in a loop, so long as
	the supervisor hasn't signalled it's time to quit.
*/
func (Behavior) Looper(agent Agent) Agent {
	return looper{agent}.Work
}

type looper struct{ Agent }

func (x looper) Work(super Supervisor) {
	for !super.Quit() {
		x.Agent(super)
	}
}
