package sup

/*
	Witness implementation paired to a supervisor.
*/
type witness struct {
	supervisor *supervisor
}

func (wit *witness) WaitSelectably(bellcord chan<- interface{}) {
	wit.supervisor.latch_done.WaitSelectably(bellcord)
}

func (wit *witness) Wait() {
	wit.supervisor.latch_done.Wait()
}

func (wit *witness) Cancel() {
	wit.supervisor.ctrlChan_quit.Fire()
}
