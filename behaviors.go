package sup

//// Looper

func Looper(agent Agent) Agent {
	return looper{agent}.Work
}

type looper struct{ Agent }

func (x looper) Work(super Supervisor) {
	for !super.Quit() {
		x.Agent(super)
	}
}
