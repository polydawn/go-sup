package saltmines

import (
	"fmt"
	"os"
)

/*
	Some drills you could imagine running on our little manufactory:

	- normal operation, until the mines run out and all work is done
	- normal operation, terminated by quit signals at the top
	- a fire in the foundry control office requires us to shut down all foundries
	- an ore washing plant jams and must be replaced
	- the oversight office gets some requests
	- the oversight office is in the process of gathering production reports,
	   when there's an accident in the mines; the report must still go out
	- a fuse blows in the ore washing station and *all* those plants stop for
	   a while.  The mines are backed up, and the foundries waiting.
	   If there's a fire in the foundries at the same time, and we should
	   surely still be able to evacuate everyone in a timely fashion
*/

func ExampleSaltmines() {
	defer fmt.Printf("Example: complete!")
	Main(os.Stdout)

	// Output:
	// Owner: hello
	// Owner: leaving for cayman
	// Example: complete!
}
