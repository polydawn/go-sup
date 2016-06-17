package sup

type Supervisor interface {
	Fork(Director)
	Join()
	SelectableQuit() <-chan struct{}
}

type Director func(Supervisor)

func NewRootSupervisor() Supervisor {
	return &supervisor{}
}

type supervisor struct{}

func (svr *supervisor) Fork(Director)                   {}
func (svr *supervisor) SelectableQuit() <-chan struct{} { return nil }
func (svr *supervisor) Join()                           {}
