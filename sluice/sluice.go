/*
	`sluice` provides a way to use channels where you need to select for ready events,
	but directly using buffered channels isn't desirable because events need to be
	reordered after dropoff; or, because allowing writers to block is simply untenable.

	The read channels produced by a Sluice are buffered and shall eventually
	provide one value -- no more; to read again, get another channel.

	A Sluice will internally buffer without limit.
	That means if the input volume is unlimited, and the consumers are
	slower than the producers, you will eventually run out of memory!
	If backpressure is important, a sluice is *not* the right choice;
	prefer a regular buffered channel instead.

	Be careful not to create a read request with `Next` and then drop the channel.
	Doing so won't crash or deadlock the program, but it will lose a value.
	Particularly watch out for this in a select used inside a loop;
	naively using `case val := <-sluice.Next():` will spawn a read request
	*every* time you enter the select, resulting in too many channels
	and lots of lost reads.  More correct: keep the reference
	and replace it only when you actually follow that select path down.
*/
package sluice

import (
	"sync"
)

type T interface{}

type Sluice interface {
	Push(T)
	Next() <-chan T
}

func New() Sluice {
	return &sluice{
		serviceReqs: make(map[chan T]struct{}),
	}
}

type sluice struct {
	mu          sync.Mutex
	serviceReqs map[chan T]struct{}
	queue       []T
}

func (db *sluice) Push(x T) {
	db.mu.Lock()
	defer db.mu.Unlock()
	req := db.pluck()
	if req == nil {
		db.queue = append(db.queue, x)
	} else {
		req <- x
	}
}

func (db *sluice) pluck() chan T {
	for req, _ := range db.serviceReqs {
		delete(db.serviceReqs, req)
		return req
	}
	return nil
}

/*
	Request a pull of what's next; a channel for the future result is returned.
	One value will eventually be sent on the channel.
*/
func (db *sluice) Next() <-chan T {
	respCh := make(chan T, 1)
	db.mu.Lock()
	defer db.mu.Unlock()
	if len(db.queue) > 0 {
		var pop T
		pop, db.queue = db.queue[0], db.queue[1:]
		respCh <- pop
	} else {
		db.serviceReqs[respCh] = struct{}{}
	}
	return respCh
}
