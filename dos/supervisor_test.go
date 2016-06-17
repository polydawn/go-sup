package sup

/*
	Experiment with some other ways to do things.

	- having the user call `go` means stacks give useful "created by" info.
	- easier to create a bunch of tasks and toss them into separate groups.
	   (though there's not a lot of evidence that that's very useful.)
	- having an explicit "kay" call means we can panic from there.
	- maybe if careful can not require secretaries all the way down?

	Losses:

	- having the user call `go` makes it that much harder to imagine restart policies.
	   not impossible, but a little funnier looking, certainly.
	- we're back to explicit waiting being required.
	   (but hell, we could make the starts block until you called that.)

	Coup de grace:

	- you really wouldn't be safe or justified to call the "kay" method from
	   two different groups, without gathering them in another.
	   one panicking alone means the other gets abandoned.
	   preventing this kind of orphaning oversight is much of our intent.
	   nah, that's a red herring though.
*/

func ExampleWow() {
	groupHandle := NewRootSupervisor()
	go groupHandle.Fork(func(svr Supervisor) {
		go svr.Fork(func(Supervisor) {})
		go svr.Fork(func(Supervisor) {})
		go svr.Fork(func(Supervisor) {})
	})
	go groupHandle.Fork(func(Supervisor) {})
	go groupHandle.Fork(func(Supervisor) {})
	groupHandle.Join()

	/*
		   OR:

		groupHandle := NewRootSupervisor()
		go groupHandle.Run(func() {
			subgroup := groupHandle.Fork()
			go subgroup.Run(func() {})
			go subgroup.Run(func() {})
			go subgroup.Run(func() {})
		})
		go groupHandle.Run(func() {})
		go groupHandle.Run(func() {})
		groupHandle.Join()

		... but this wouldn't look nearly as graceful if you weren't spamming closures.
	*/

	// Output:
}
