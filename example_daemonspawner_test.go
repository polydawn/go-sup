package sup_test

import (
	"fmt"
	"time"

	"."
)

func ExampleDaemonSpawner() {
	type bigTask string

	var workerFactory = func(task bigTask) sup.Agent {
		return func(super sup.Supervisor) {
			// has lots to do, i'm sure
			time.Sleep(time.Millisecond * 100)
		}
	}

	firehose := make(chan bigTask)
	cntWorkFound := 0
	var workFinder sup.Agent = sup.Looper(func(super sup.Supervisor) {
		select {
		case firehose <- "work":
			cntWorkFound++
			if cntWorkFound >= 12 {
				close(firehose)
				panic(nil)
			}
		case <-super.QuitCh():
			return
		}
	})

	var daemonMaster sup.Agent = func(super sup.Supervisor) {
		mgr := sup.NewManager(super)
		for {
			select {
			case <-super.QuitCh():
				goto procede
			case todo, ok := <-firehose:
				if !ok {
					goto procede
				}
				taskName := string(todo)
				go mgr.NewTask(taskName).Run(workerFactory(todo))
			}
			// do i need to 'step' periodically here?  how important is manager state maint?
			// or check if it has any errors to raise, even if we're not done accepting?  yes that's important.
		}
	procede:
		fmt.Printf("daemonMaster wrapping up\n")
		defer fmt.Printf("daemonMaster returned\n")
		mgr.Work()
	}

	rootWrit := sup.NewWrit()
	rootWrit.Run(func(super sup.Supervisor) {
		mgr := sup.NewManager(super)
		go mgr.NewTask("workFinder").Run(workFinder)
		go mgr.NewTask("daemonMaster").Run(daemonMaster)
		mgr.Work()
	})

	// Output:
	// daemonMaster wrapping up
	// daemonMaster returned
}
