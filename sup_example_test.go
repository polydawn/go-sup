package sup_test

import (
	"fmt"
	"os"
	"time"

	"."
)

func ExampleWow() {
	sup.NewRootSupervisor(func(svr sup.Supervisor) {
		wit := svr.NewSupervisor(func(svr sup.Supervisor) {
			fmt.Printf("whee, i'm an actor!\n")
			select {
			case <-svr.SelectableQuit():
			case <-time.After(200 * time.Millisecond):
				fmt.Printf("a lazy one!\n")
			}
		})
		wit.Wait()
	}).Wait()

	// Output:
	// whee, i'm an actor!
	// a lazy one!
}

func ExampleWowCancel() {
	out := os.Stdout
	sup.NewRootSupervisor(func(svr sup.Supervisor) {
		wit := svr.NewSupervisor(func(svr sup.Supervisor) {
			fmt.Fprintf(out, "whee, i'm an actor!\n")
			select {
			case <-svr.SelectableQuit():
				fmt.Fprintf(out, "cancelled!\n")
			case <-time.After(2 * time.Second):
			}
		})
		wit.Cancel()
		wit.Wait()
	}).Wait()

	// Output:
	// whee, i'm an actor!
	// cancelled!
}

func ExampleMisbehaved() {
	// testing the misbehavior warnings is hard.
	// i'm not sure how to avoid the nature that it's a thing with a wallclock in it.
}

func ExampleTree() {
	sup.NewRootSupervisor(func(svr sup.Supervisor) {
		fmt.Println("sup > .")
		svr.NewSupervisor(func(svr sup.Supervisor) {
			fmt.Println("sup > ..")
			svr.NewSupervisor(func(svr sup.Supervisor) {
				fmt.Println("sup > ...")
				svr.NewSupervisor(func(svr sup.Supervisor) {
					fmt.Println("sup > ....")
					svr.NewSupervisor(func(svr sup.Supervisor) {
						fmt.Println("sup > .....")
						svr.NewSupervisor(func(svr sup.Supervisor) {
							fmt.Println("sup > ......")
						}).Wait()
						fmt.Println("sup < .....")
					}).Wait()
					fmt.Println("sup < ....")
				})
			})
		})
	}).Wait()

	// Output:
	// sup > .
	// sup > ..
	// sup > ...
	// sup > ....
	// sup > .....
	// sup > ......
	// sup < .....
	// sup < ....
}
