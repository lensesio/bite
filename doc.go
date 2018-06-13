/*Package bite provides a toolbox for CLI cobra-based applications.

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
		app.AddCommand(newMyCommand())
	}

	var names = []string{"name1", "name2"}

	func newMyCommand() *bite.Command {
		var list bool
		return &bite.Command{
			Use:     "name",
			Short: "Print the last name or the available names",
			Example:   "name --list Print the list of the available names (as JSON if --machine-friendly)",
			Flags: func(set *bite.FlagSet) {
				set.BoolVar(&list, "--list", "","--list Print the list of the available names (as JSON if --machine-friendly)")
			},
			Run: func(cmd *bite.Command, args []string) error {
				if list {
					return cmd.App.PrintObject(names)
				}
				name := names[len(names)-1], nil
				return cmd.App.Print(name)
			},
		}
	}

	func main() {
		// my-app name [--list [--machine-friendly]]
		if err := app.Run(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

Github: https://github.com/kataras/bite
*/
package bite
