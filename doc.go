/*Package bite is a toolbox for CLI cobra-based applications.

Example code:
	package main

	var app = &bite.Application{
		Name:    "my-app",
		Version: "0.0.1",
		HelpTemplate: bite.HelpTemplate{
			BuildTime:            "Fr May 01 03:48:42 UTC 2018",
			BuildRevision:        "a0d0c263ec4858fe3e527625b0236584c9f11479",
			ShowGoRuntimeVersion: true,
			// Template: customize template using any custom `fmt.Stringer`,
		},
		Setup: func(cmd *cobra.Command, args []string) error {
			// setup here.
			return nil
		},
		Shutdown: nil,
	}

	func init() {
		// app.AddCommand(aCobraCommand())
	}

	func main() {
		if err := app.Run(os.Stdout, os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

Github: https://github.com/lensesio/bite
*/
package bite
