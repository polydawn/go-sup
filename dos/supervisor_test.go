package sup

/*
	Experiment with some other ways to do things.

	Two things:

	Thing 1: putting the `go` keyword back in the user's hands:

	Pros:

	- having the user call `go` means stacks give useful "created by" info.
	- having the `go` keyword in the user's source means it's highlighted and
	  obvious to anyone, even with no prior knowledge of the sup library, that
	  parallelism has entered play.

	Cons:

	- hard to imagine ever *not* wanting to use the go keyword, so it's a little boilerplatey
	- having the user call `go` makes it that much harder to imagine restart policies.
	  not impossible, but a little funnier looking, certainly.
	  (The answer to this is probably just shrug: spawns with restart policy params
	  will already have a different function sig entirely.)
	- the big one: you can't return a promise.
	  in order to do so, one would have to either hand in a pointer or a chan.
	  either of which is certainly possible, but trends rapidly towards the verbose.

	Thing 2: making a supervisor obj, and calling all methods (including `Join`
	on it manually).

	Red herring:

	- I can't think of an elegant way to do this.  You need a tree; if you're not
	  putting either the parent or the new leaf in the function param, then you're
	  sneaking it in either out of band, or via closure.

	Pros:

	- having an explicit "kay" call means we can panic from there.
	- easier to create a bunch of tasks and toss them into separate groups.
	   (though there's not a lot of evidence that that's very useful.)

	Losses:
	- we're back to explicit waiting being required.
	   (but hell, we could make the starts block until you called that.)

	Coup de grace:

	- you really wouldn't be safe or justified to call the "kay" method from
	   two different groups, without gathering them in another.
	   one panicking alone means the other gets abandoned.
	   preventing this kind of orphaning oversight is much of our intent.
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
