package sup

import (
	"sync/atomic"
)

/*
	Witness implementation paired to a supervisor.
*/
type witness struct {
	supervisor *supervisor
	handled    int32
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

func (wit *witness) Err() error {
	wit.Wait()
	return wit.supervisor.err
}

func (wit *witness) Handled() {
	atomic.CompareAndSwapInt32(&wit.handled, 0, 1)
}

func (wit *witness) isHandled() bool {
	return atomic.LoadInt32(&wit.handled) == 1
}

/*
	Witness implementation which instantly ends as errored.

	TODO ayy maybe a method for getting errors
*/
type witnessThunk struct {
	err     error
	handled int32
}

func (wit *witnessThunk) WaitSelectably(bellcord chan<- interface{}) {
	bellcord <- wit
}

func (wit *witnessThunk) Wait() {
	// return immediately.
}

func (wit *witnessThunk) Cancel() {
	// nothing to cancel.
}

func (wit *witnessThunk) Err() error {
	return wit.err
}

func (wit *witnessThunk) Handled() {
	atomic.CompareAndSwapInt32(&wit.handled, 0, 1)
}
