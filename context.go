package sup

import (
	"time"
)

/*
	Copy of the stdlib (or `x/net`, in older go versions) `Context` interface.

	Cloned here simply so we can assert that we implement it (without requiring
	either that package import or go1.7 for build) -- and not exported because
	if you care about this, use those packages to refer to it.

	Upstream: https://tip.golang.org/src/context/context.go
*/
type context interface {
	// Returns a deadline, if there is one.  (go-sup doesn't use these.)
	Deadline() (deadline time.Time, ok bool)

	// Select on this to know when you should return.
	Done() <-chan struct{}

	// Name and type somewhat confusing: this is a nilipotent checker for if `Done` already happened
	Err() error

	// Bag for values.  (go-sup doesn't use these.)  https://blog.golang.org/context has suggested usage.
	Value(key interface{}) interface{}
}
