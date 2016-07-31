package sup

import (
	"fmt"
	"sort"
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
					So(mgr.(*manager).wards, ShouldHaveLength, 0)
				})
			})

			Convey("And some parallel executing tasks", func() {
				ch := make(chan string, 2)
				go mgr.NewTask("1").Run(ChanWriterAgent("1", ch))
				go mgr.NewTask("2").Run(ChanWriterAgent("2", ch))

				Convey("Tasks run and we see their sideeffects", func() {
					results := []string{<-ch, <-ch}
					sort.Strings(results)
					So(results[0], ShouldEqual, "1")
					So(results[1], ShouldEqual, "2")
				})

				Convey("Manager.Work should gather all the things", func() {
					mgr.Work()
					// maybe not the most useful test
					So(mgr.(*manager).doneFuse.IsBlown(), ShouldBeTrue)
					So(mgr.(*manager).wards, ShouldHaveLength, 0)
				})
			})

			Convey("And some exploding tasks!", func() {
				ch := make(chan string, 0)
				explo := fmt.Errorf("bang!")
				go mgr.NewTask("1").Run(ChanWriterAgent("1", ch))
				go mgr.NewTask("2").Run(ChanWriterAgent("2", ch))
				go mgr.NewTask("e").Run(ExplosiveAgent(explo))

				Convey("Manager.Work should raise the panic", func() {
					So(mgr.Work, ShouldPanicWith, explo)
					So(mgr.(*manager).doneFuse.IsBlown(), ShouldBeTrue)
					So(mgr.(*manager).wards, ShouldHaveLength, 0)
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

func ExplosiveAgent(err error) Agent {
	return func(Supervisor) {
		panic(err)
	}
}
