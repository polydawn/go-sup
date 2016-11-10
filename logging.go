package sup

import (
	"fmt"
	"os"
	"strings"
)

type WritName []string

func (wn WritName) String() string {
	if len(wn) == 0 {
		return "[root]"
	}
	return strings.Join(wn, ".")
}

func (wn WritName) Coda() string {
	if len(wn) == 0 {
		return "[root]"
	}
	return wn[len(wn)-1]
}

func (wn WritName) New(segment string) WritName {
	result := make([]string, len(wn)+1)
	copy(result, wn)
	result[len(wn)] = segment
	return result
}

/*
	Called to log lifecycle events inside the supervision system.

	An example event might be

		log(mgr.FullName, "child reaped", writ.Name)

	which one might log as, for example:

		log.debug(evt, {"mgr":name, "regarding":re.Coda()})
		//debug: child reaped -- mgr=root.system.subsys regarding=subproc14

	The `name` and `evt` parameters will always be nonzero; `re` is optional.
	The `important` parameter will be true if this should *really* be printed;
	"important" events are low-volume things and generally warnings, like
	the warnings printed for agents that are not responding to quit signals.
*/
type LogFn func(name WritName, evt string, re WritName, important bool)

var log LogFn = func(name WritName, evt string, re WritName, _ bool) {
	if re == nil {
		fmt.Fprintf(os.Stderr, "mgr=%s: %q\n", name, evt)
	} else {
		fmt.Fprintf(os.Stderr, "mgr=%s: %q re=%s\n", name, evt, re.Coda())
	}
}
