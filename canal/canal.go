/*
	Canals are a pattern of using channels where you need to select for ready events,
	but directly using buffered channels isn't desirably because events need to be
	reordered after dropoff; or, because allowing writers to block is simply untenable.

	The read channels produced by a Canal are buffered and shall eventually
	provide one value -- no more; to read again, get another channel.

	Be careful not to create a read request with `Next` and then drop the channel.
	Doing so won't crash or deadlock the program, but it will lose a value.
	Particularly watch out for this in a select used inside a loop;
	naively using `case val := <-canal.Next():` will spawn a read request
	*every* time you enter the select, resulting in too many channels
	and lots of lost reads.  More correct: keep the reference
	and replace it only when you actually follow that select path down.
*/
package canal

import (
	"sync"
)

type T interface{}

type Canal interface {
	Push(T)
	Next() <-chan T
}

func New() Canal {
	return &canal{
		serviceReqs: make(map[chan T]struct{}),
	}
}

type canal struct {
	mu          sync.Mutex
	serviceReqs map[chan T]struct{}
	queue       []T
}

func (db *canal) Push(x T) {
	db.mu.Lock()
	defer db.mu.Unlock()
	req := db.pluck()
	if req == nil {
		db.queue = append(db.queue, x)
	} else {
		req <- x
	}
}

func (db *canal) pluck() chan T {
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
func (db *canal) Next() <-chan T {
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
