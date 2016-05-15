package sup

import (
	"runtime"
	"strings"
	"testing"
	"time"

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

func logResults(results []string) {
	Print("\nseq >>> " + strings.Join(results, "\n      > ") + "\n      ----\n")
}

func TestSupervisorCrashcancels(t *testing.T) {
	Convey("supervisors that crash should have children cancelled", t, FailureContinues, func() {
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
		logResults(results)
		So(results, ShouldHaveLength, 4)
		So(results, phist.ShouldSequence, "child proc started", "child proc recieved quit")
		So(results, phist.ShouldSequence, "supervisor control about to panic", "child proc recieved quit")
	})

	Convey("supervisory trees that crash at the top should have (grand)children cancelled", t, FailureContinues, func() {
		blackbox := newBlackbox()
		NewSupervisor(func(svr *Supervisor) {
			grandchildHello := make(chan beep)
			blackbox <- "supervisor control started"
			svr.Fork(func(svr *Supervisor) {
				blackbox <- "supervisor L2 control started"
				svr.Spawn(func(chap Chaperon) {
					blackbox <- "leaf proc started"
					grandchildHello <- beep{}
					blackbox <- "leaf proc beeped"
					<-chap.SelectableQuit()
					blackbox <- "leaf proc recieved quit"
				})
				blackbox <- "supervisor L2 control returning"
			})
			<-grandchildHello
			blackbox <- "supervisor control about to panic"
			panic("whoa")
		})
		time.Sleep(1 * time.Second)
		results := blackbox.drain()
		logResults(results)
		So(results, ShouldHaveLength, 7)
		// unremarkably, initial control should flow downtree:
		So(results, phist.ShouldSequence, "supervisor control started", "supervisor L2 control started")
		So(results, phist.ShouldSequence, "supervisor L2 control started", "leaf proc started")
		// we're enforcing the order that the grandchild is definitely started, before we panic up near the root:
		So(results, phist.ShouldSequence, "leaf proc beeped", "supervisor control about to panic")
		// the leaf should get a quit, and it should follow the supervisor control func panicking
		So(results, phist.ShouldSequence, "leaf proc started", "leaf proc recieved quit")
		So(results, phist.ShouldSequence, "supervisor control about to panic", "leaf proc recieved quit")
	})
}
