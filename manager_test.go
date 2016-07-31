package sup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestManager(t *testing.T) {
	Convey("Given a Manager", t, func() {
		rootWrit := NewTask()
		rootWrit.Run(func(super Supervisor) {
			mgr := NewManager(super)
			Convey("And some serially executing tasks", func() {
				ch := make(chan string, 2)
				mgr.NewTask("1").Run(ChanWriterAgent("1", ch))
				mgr.NewTask("2").Run(ChanWriterAgent("2", ch))

				Convey("Tasks run and we see their sideeffects", func() {
					So(<-ch, ShouldEqual, "1")
					So(<-ch, ShouldEqual, "2")
				})

				Convey("Manager.Work should gather all the things", func() {
					mgr.Work()
					// maybe not the most useful test
					So(mgr.(*manager).doneFuse.IsBlown(), ShouldBeTrue)
				})
			})
		})
	})
}

func ChanWriterAgent(msg string, ch chan<- string) Agent {
	return func(supvr Supervisor) {
		select {
		case ch <- msg:
		case <-supvr.QuitCh():
		}
	}
}
