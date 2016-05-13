package sup

import "testing"

type blackbox chan string

func newBlackbox() blackbox {
	return make(chan string, 100)
}

func (bb blackbox) drain() (lst []string) {
	close(bb)
	for s := range bb {
		lst = append(lst, s)
	}
	return
}

func TestSupervisorCrashcancels(t *testing.T) {
	blackbox := newBlackbox()
	NewSupervisor(func(svr *Supervisor) {
		blackbox <- "supervisor control started"
		svr.Spawn(func(chap Chaperon) {
			blackbox <- "child proc started"
			<-chap.SelectableQuit()
			blackbox <- "child proc recieved quit"
		})
		blackbox <- "supervisor control about to panic"
		panic("whoa")
	})
	results := blackbox.drain()
	t.Log(results[0])
	t.Log(results[1])
	t.Log(results[2])
	t.Log(results[3])
}
