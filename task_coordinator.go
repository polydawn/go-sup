package sup

import (
	"polydawn.net/go-sup/latch"
)

type Witness interface {
	WaitSelectably(bellcord chan<- interface{}) // self will be sent when this task is done.  use for selectable fan-in.
	Wait()
	Cancel()
}

type Chaperon interface {
	SelectableQuit() <-chan struct{} // closed when you should die

	// Note the *absense* of a `Done(result interface{})` method.
	// Anything you need to return: do so by writing your actor function to be
	//  closure over a (typed!) var which you set.
	// Your work becomes "done" when it returns.  No functions have to be called
	//  to mark the transition; the scope of your stack is just the whole truth.
}

var (
	_ Witness  = &controller{}
	_ Chaperon = &controller{}
)

// implements both halves of witness and chaperon.
type controller struct {
	quitCh chan struct{}

	doneLatch latch.Latch
}

func newController() *controller {
	ctrlr := &controller{
		quitCh: make(chan struct{}),
	}
	ctrlr.doneLatch = latch.NewWithMessage(ctrlr)
	return ctrlr
}
func (ctrlr *controller) WaitSelectably(bellcord chan<- interface{}) {
	ctrlr.doneLatch.WaitSelectably(bellcord)
}

func (ctrlr *controller) Wait() {
	ctrlr.doneLatch.Wait()
}

func (ctrlr *controller) Cancel() {
	close(ctrlr.quitCh) // TODO this should probably tolerate multiple cancels
}

func (ctrlr *controller) SelectableQuit() <-chan struct{} {
	return ctrlr.quitCh
}
