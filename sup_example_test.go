package sup_test

import (
	"fmt"
	"time"

	"."
)

func ExampleWow() {
	sup.NewSupervisor(func(svr *sup.Supervisor) {
		wit := svr.Spawn(func(chap sup.Chaperon) {
			fmt.Printf("whee, i'm an actor!\n")
			select {
			case <-chap.SelectableQuit():
			case <-time.After(200 * time.Millisecond):
				fmt.Printf("a lazy one!\n")
			}
			chap.Done("result!\n")
		})
		wit.Wait()
	})

	// Output:
	// whee, i'm an actor!
	// a lazy one!
}

func ExampleWowCancel() {
	sup.NewSupervisor(func(svr *sup.Supervisor) {
		wit := svr.Spawn(func(chap sup.Chaperon) {
			fmt.Printf("whee, i'm an actor!\n")
			select {
			case <-chap.SelectableQuit():
				fmt.Printf("cancelled!\n")
				chap.Done("cancelled!\n")
			case <-time.After(2 * time.Second):
			}
			chap.Done("result!\n")
		})
		wit.Cancel()
		wit.Wait()
	})

	// Output:
	// whee, i'm an actor!
	// cancelled!
}

func ExampleMisbehaved() {
	// testing the misbehavior warnings is hard.
	// i'm not sure how to avoid the nature that it's a thing with a wallclock in it.
}

func ExampleTree() {
	sup.NewSupervisor(func(svr *sup.Supervisor) {
		svr.Spawn(func(chap sup.Chaperon) {
			sup.NewSupervisor(func(svr *sup.Supervisor) {
				svr.Spawn(func(chap sup.Chaperon) {
					sup.NewSupervisor(func(svr *sup.Supervisor) {
						svr.Spawn(func(chap sup.Chaperon) {
							chap.Done("t3\n")
						})
					})
				})
			})
		})
	})

	// skip // Output:
	// t3
}
