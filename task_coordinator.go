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
	Done(result interface{})         // push up your result.  failure to return shortly after calling this will result in warnings.
}

var (
	_ Witness  = &controller{}
	_ Chaperon = &controller{}
)

// implements both halves of witness and chaperon.
type controller struct {
	quitCh chan struct{}

	doneLatch latch.Latch

	result interface{} // fenced by doneLatch
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

func (ctrlr *controller) Done(result interface{}) {
	ctrlr.result = result
	// does NOT trigger the 'done' latch.  actual control return does so.
}
