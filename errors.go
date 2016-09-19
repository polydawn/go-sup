package sup

import (
	"go.polydawn.net/meep"
)

type ErrTaskPanic struct {
	meep.TraitAutodescribing
	meep.TraitCausable
	meep.TraitTraceable
}
