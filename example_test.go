package sup_test

import (
	"fmt"
	"os"

	"."
)

func ExampleWow() {
	type bid string
	salesFunnel := make(chan bid)
	var salesMinion sup.Agent = sup.Behaviors.Looper(func(super sup.Supervisor) {
		select {
		case salesFunnel <- "sale":
		case <-super.QuitCh():
			return
		}
	})
	var salesDirector sup.Agent = func(super sup.Supervisor) {
		mgr := sup.NewManager(super)
		go mgr.NewTask("sales1").Run(salesMinion)
		go mgr.NewTask("sales2").Run(salesMinion)
		go mgr.NewTask("sales3").Run(salesMinion)
		mgr.Work()
		//sup.Funnel().Gather(mgr.DoneCh()).Await()
	}

	rootWrit := sup.NewTask()
	rootWrit.Run(func(super sup.Supervisor) {
		mgr := sup.NewManager(super)
		go mgr.NewTask("region-a").Run(salesDirector)
		go mgr.NewTask("region-b").Run(salesDirector)
		go mgr.NewTask("region-c").Run(salesDirector)
		salesCnt := 0
		go mgr.NewTask("planner").Run(func(super sup.Supervisor) {
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
