package sup

import (
	"go.polydawn.net/meep"
)

type ErrTaskPanic struct {
	meep.TraitAutodescribing
	meep.TraitCausable

	// The first ErrTaskPanic keeps traces, but if a series of them are
	// raised (as the manager hierarchy frequently does), the latter errors
	// will skip collecting their stacks, leaving this trait blank.
	meep.TraitTraceable

	// The name of the task that panicked.
	// (If this task had a manager, its name is one level up from this one.)
	Task WritName
}
