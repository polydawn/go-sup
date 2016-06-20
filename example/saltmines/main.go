package saltmines

import (
	"bufio"
	"fmt"
	"io"

	"go.polydawn.net/go-sup"
)

func Main(stdin io.Reader, stderr io.Writer) {
	sup.NewRootSupervisor(func(svr sup.Supervisor) {
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
		slagPipe := make(chan Slag)
		minePit := &MinePits{
			thePit:   stdin,
			slagPipe: slagPipe,
		}
		minePitWitness := svr.NewSupervisor(minePit.Run)
		minePitWitness.Cancel()

		oreWasher := &OreWashingFacility{
			slagPipe:     slagPipe,
			copperHopper: make(chan OreCopper),
			tinHopper:    make(chan OreTin),
			zincHopper:   make(chan OreZinc),
		}
		oreWasherWitness := svr.NewSupervisor(oreWasher.Run)
		oreWasherWitness.Cancel()

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
	thePit   io.Reader
	slagPipe chan<- Slag
}

func (mp *MinePits) Run(svr sup.Supervisor) {
	scanner := bufio.NewScanner(mp.thePit)
	scanner.Split(bufio.ScanWords)
	for {
		// intentionally evil example.  we need interruptable readers to
		//  be able to shut down truly gracefully.
		select {
		default:
			scanner.Scan()
			// careful.  you have to put nb/cancellable selects for each send, too.
			select {
			case mp.slagPipe <- Slag(scanner.Text()):
			case <-svr.SelectableQuit():
			}
		case <-svr.SelectableQuit():
			return
		}
	}
}

type OreWashingFacility struct {
	slagPipe     <-chan Slag
	copperHopper chan<- OreCopper
	tinHopper    chan<- OreTin
	zincHopper   chan<- OreZinc
}

func (owf *OreWashingFacility) Run(svr sup.Supervisor) {
	// Ore washing is a slow process, and sometimes a batch takes quite
	//  some time; this can strike fairly randomly, so we run a bunch
	//  of processing separately to even things out.
	// That means *we're* a supervisor for all those parallel processors.
	var children []sup.Witness
	for n := 0; n < 4; n++ {
		wit := svr.NewSupervisor(owf.runSingleStation)
		children = append(children, wit)
	}
}

func (owf *OreWashingFacility) runSingleStation(svr sup.Supervisor) {
	for {
		select {
		case slag := <-owf.slagPipe:
			// this looks a little squishy, but keep in mind
			//  the level of contrivance here.  it's quite unlikely
			//   that one would ever write a real typed fanout so trivial as this.
			switch slag {
			case "copper":
				select {
				case owf.copperHopper <- OreCopper(slag):
				case <-svr.SelectableQuit():
				}
			case "tin":
				select {
				case owf.tinHopper <- OreTin(slag):
				case <-svr.SelectableQuit():
				}
			case "zinc":
				select {
				case owf.zincHopper <- OreZinc(slag):
				case <-svr.SelectableQuit():
				}
			default:
				panic("unknown ore type, cannot sort")
			}
		case <-svr.SelectableQuit():
			return
		}
	}
}

type FoundryCoordinator struct {
}

type OversightOffice struct {
}
