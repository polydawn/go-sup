package saltmines

import (
	"fmt"
	"io"

	"go.polydawn.net/go-sup"
)

func Main(stderr io.Writer) {
	sup.NewSupervisor(func(svr *sup.Supervisor) {
		// I'm a very lazy controller, and I mostly delegate work to others.
		// I don't actually even wake up if something goes wrong!
		// I assume things are going according to plan unless something
		//  really terrible gets so far that it sets my office on fire around me.
		// Even then: honestly, I'm already in the Cayman Islands.  There's
		//  nobody in the head office anymore.  Really, all the other workers
		//  need is the idea that there's someone they *could* complain to.
		// If somebody actually *does* file a report, the mail will make
		//  it to the corporate franchise office, somehow.  (Maybe the dutiful
		//  secretary I left behind will actually do the maint work for me,
		//  even though I've nipped off.)
		fmt.Fprintf(stderr, "Owner: hello\n")

		// There are four major operations going on under my domain:
		//   - The mining pits -- these produce a steady stream of "slag"
		//   - The ore washing plants -- these do some basic processing, and route out several different kinds of "ore"
		//   - The foundries -- there's several different kinds of these, they take only specific kinds of "ore"
		//   - The shipping wharf -- this station packages up all the ingots into crates for sale
		// Keep an eye on the ore washing plants.  Sometimes they get jammed,
		//  and we have to take one out of service, scrap it for parts, and
		//  just install a whole new one without any of the wear-n-tear.
		// There's also a fivth operation: the oversight office.
		//  The oversight office can sometimes get letters from other parts
		//  of Her Majesty's Goverment, to which the office is required to
		//  respond in a timely fashion.  Sometimes this requires the
		//  oversight office to commission a team to gather a report.  Such
		//  teams tend to be short-lived, but they may ask questions about
		//  (or sometimes give odd orders to) the other three major operational
		//  centers of our production pipeline.

		fmt.Fprintf(stderr, "Owner: leaving for cayman\n")
	})
}

type (
	Slag string

	OreCopper string
	OreTin    string
	OreZinc   string

	IngotCopper string
	IngotTin    string
	IngotZinc   string

	Crate struct{ ingots []string }
)

type MinePits struct {
}

func (mp *MinePits) Run() {

}

type OreWashingFacility struct {
}

type FoundryCoordinator struct {
}

type OversightOffice struct {
}
