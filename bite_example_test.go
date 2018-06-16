package bite_test

import (
	"fmt"
	"os"

	"github.com/kataras/bite"
)

func ExampleApplicationBuilder() {
	var name string

	program := bite.
		Name("bite-simple-example").
		Flags(func(flags *bite.Flags) {
			// Register a flag.
			flags.Bool("silent", false, "--silent")
			// Register and bind a flag to a local variable.
			flags.StringVar(&name, "name", "", "--name=")
		})

	// Parse using custom arguments or system's, i.e `os.Args[1:]...`.
	if err := program.Parse("--silent=false", "--name=Joe"); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(-1)
	}

	// Get a flag's value manually.
	silent, _ := program.GetFlags().GetBool("silent")
	if !silent {
		// Use the binded local variable "name".
		fmt.Printf("Hello %s", name)
	}

	// Output: Hello Joe
}
