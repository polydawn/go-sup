package latch

import (
	"sync"
)

/*
	Sign up for a signal, which can be flipped exactly once.

	Much like a `sync.WaitGroup`, except it uses channels and is thus
	selectable;
	much like a `sync.Condition`, except if someone asks to wait after
	the condition has been fired, they will also immediately act as signalled.

	This is often useful for coordinating things that have "happens-after"
	requirements -- for example modelling state machine transitions
	where we want to act only when "reached ready state (and maybe further)".

	A `Latch` is distinct from a `Fuse` because
	`Fuse.Fire` can always complete immediately and never blocks;
	`Latch.Trigger` sends messages to a channel and thus may block.
	If you can use a `Fuse`, prefer to;
	a `Latch` is necessary if you want to fan-in events to a gathering channel.
	In other words, you can't easily select on $n `Fuse`s because you have $n channels;
	whereas with a `Latch` you can tell $n `Latch`es to trigger just 1 channel and
	selecting for any of those latches to close is easy.
*/
type Latch interface {
	// Block until the latch is flipped.
	Wait()

	// Submit a channel to be signalled as soon as the latch is flipped.
	WaitSelectably(bellcord chan<- interface{})

	// Fire the signal.  If this is called more than once, it will panic (much like closing a closed channel).
	Trigger()

	// Like `Trigger`, but simply no-ops if triggering has already happened.  Use sparingly.
	MaybeTrigger()
}

func New() Latch {
	return &latch{
		bellcords: make([]chan<- interface{}, 0, 1),
	}
}

func NewWithMessage(msg interface{}) Latch {
	return &latch{
		msg:       msg,
		bellcords: make([]chan<- interface{}, 0, 1),
	}
}

type latch struct {
	mu        sync.Mutex
	msg       interface{}
	bellcords []chan<- interface{}
}

func (l *latch) Trigger() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.bellcords == nil {
		panic("cannot repeatedly trigger latch")
	}
	l.trigger()
}

func (l *latch) MaybeTrigger() {
	l.mu.Lock()
	l.trigger()
	l.mu.Unlock()
}

func (l *latch) trigger() {
	for _, bellcord := range l.bellcords {
		bellcord <- l.msg
	}
	l.bellcords = nil
}

func (l *latch) WaitSelectably(bellcord chan<- interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.bellcords == nil {
		// even if we're doing immediate ack, we must do so in a goroutine
		//  in case the caller handed us an unbuffered channel which they expect
		//   to wait on moments after this submission.  Triggers don't come from
		//    the listener-submitting routine; neither should after-trigger acks.
		go func() { bellcord <- l.msg }()
		return
	}
	l.bellcords = append(l.bellcords, bellcord)
}

func (l *latch) Wait() {
	bellcord := make(chan interface{})
	l.WaitSelectably(bellcord)
	<-bellcord
}
