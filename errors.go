package sup

import (
	"go.polydawn.net/meep"
)

type ErrTaskPanic struct {
	meep.TraitAutodescribing
	meep.TraitCausable

	// The name of the task that panicked.
	// (If this task had a manager, its name is one level up from this one.)
	Task WritName
}
