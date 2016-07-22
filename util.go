package sup

import (
	"fmt"
)

// Use to compensate for `recover()` returning a wildcard type.
// If you panicked with a non-`error` type, you're a troll.
func coerceToError(rcvr interface{}) error {
	if rcvr == nil {
		return nil
	}
	if cast, ok := rcvr.(error); ok {
		return cast
	}
	return fmt.Errorf("recovered: %T: %s", rcvr, rcvr)
}
