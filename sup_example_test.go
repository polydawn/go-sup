package sup_test

import (
	"fmt"
	"time"

	"."
)

func ExampleWow() {
	svr := sup.NewRootSupervisor()
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
	svr.Wait()

	// Output:
	// whee, i'm an actor!
	// a lazy one!
}

func ExampleWowCancel() {
	svr := sup.NewRootSupervisor()
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
	svr.Wait()

	// Output:
	// whee, i'm an actor!
	// cancelled!
}
