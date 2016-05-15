package sup

import (
	"runtime"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.polydawn.net/go-sup/phist"
)

func init() {
	runtime.GOMAXPROCS(4)
}

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
	Convey("supervisors that crash should have children cancelled", t, func() {
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
		So(results, ShouldHaveLength, 4)
		So(results, phist.ShouldSequence, "child proc started", "child proc recieved quit")
		So(results, phist.ShouldSequence, "supervisor control about to panic", "child proc recieved quit")
	})
}
