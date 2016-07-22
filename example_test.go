package sup_test

import (
	"fmt"
	"os"

	"."
)

func ExampleWow() {
	type bid string
	salesFunnel := make(chan bid)
	var salesMinion sup.Agent = sup.Looper(func(super sup.Supervisor) {
		select {
		case salesFunnel <- "sale":
		case <-super.QuitCh():
			return
		}
	})
	var salesDirector sup.Agent = func(super sup.Supervisor) {
		mgr := sup.NewManager(super)
		go mgr.NewTask().Run(salesMinion)
		go mgr.NewTask().Run(salesMinion)
		go mgr.NewTask().Run(salesMinion)
		mgr.Work()
		//sup.Funnel().Gather(mgr.DoneCh()).Await()
	}

	rootWrit := sup.NewWrit()
	rootWrit.Run(func(super sup.Supervisor) {
		mgr := sup.NewManager(super)
		go mgr.NewTask().Run(salesDirector)
		go mgr.NewTask().Run(salesDirector)
		go mgr.NewTask().Run(salesDirector)
		salesCnt := 0
		go mgr.NewTask().Run(func(super sup.Supervisor) {
			for {
				select {
				case sale := <-salesFunnel:
					fmt.Fprintf(os.Stdout, "%s %d!\n", sale, salesCnt)
					salesCnt++
					if salesCnt >= 10 {
						fmt.Fprintf(os.Stderr, "trying to wrap after %s!\n", sale)
						rootWrit.Cancel()
						return
					}
				case <-super.QuitCh():
					return
				}
			}
		})
		mgr.Work()
	})
	go func() { salesFunnel <- "last" }()
	fmt.Printf("%s!\n", <-salesFunnel)

	// Output:
	// sale 0!
	// sale 1!
	// sale 2!
	// sale 3!
	// sale 4!
	// sale 5!
	// sale 6!
	// sale 7!
	// sale 8!
	// sale 9!
	// last!
}
