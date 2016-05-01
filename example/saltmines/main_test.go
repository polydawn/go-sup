package saltmines

import (
	"fmt"
	"os"
)

func ExampleSaltmines() {
	defer fmt.Printf("Example: complete!")
	Main(os.Stdout)

	// Output:
	// Owner: hello
	// Owner: leaving for cayman
	// Example: complete!
}
