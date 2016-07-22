package sup

/*
	Called to log lifecycle events inside the supervision system.

	An example event might be

		log(mgr.FullName, "child reaped", writ.Name)

	which one might log as, for example:

		log.debug(evt, {"mgr":name, "regarding":param})
		//debug: child reaped -- mgr=root.system.subsys regarding=subproc14
*/
type LogFn func(name string, evt string, param string)
