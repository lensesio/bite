package bite

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/tableprinter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type HelpTemplate struct {
	BuildTime            string
	BuildRevision        string
	ShowGoRuntimeVersion bool

	Template fmt.Stringer
}

func (h HelpTemplate) String() string {
	if tmpl := h.Template.String(); tmpl != "" {
		return tmpl
	}
	buildTitle := ">>>> build" // if we ever want an emoji, there is one: \U0001f4bb
	tab := strings.Repeat(" ", len(buildTitle))

	// unix nanoseconds, as int64, to a human readable time, defaults to time.UnixDate, i.e:
	// Thu Mar 22 02:40:53 UTC 2018
	// but can be changed to something like "Mon, 01 Jan 2006 15:04:05 GMT" if needed.
	n, _ := strconv.ParseInt(h.BuildTime, 10, 64)
	buildTimeStr := time.Unix(n, 0).Format(time.UnixDate)

	tmpl := `{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}` +
		fmt.Sprintf("\n%s\n", buildTitle) +
		fmt.Sprintf("%s revision %s\n", tab, h.BuildRevision) +
		fmt.Sprintf("%s datetime %s\n", tab, buildTimeStr)
	if h.ShowGoRuntimeVersion {
		tmpl += fmt.Sprintf("%s go       %s\n", tab, runtime.Version())
	}

	return tmpl
}

type Application struct {
	Name        string
	Version     string
	Description string

	HelpTemplate fmt.Stringer
	// ShowSpinner if true(default is false) and machine-friendly is false(default is true) then
	// it waits via "visual" spinning before each command's job done.
	ShowSpinner bool
	// if true then the --machine-friendly flag will be added to the application and PrintObject will check for that.
	DisableOutputFormatController bool
	MachineFriendly               *bool
	PersistentFlags               func(*pflag.FlagSet)

	Setup    interface{} // func(*cobra.Command, []string) error
	Shutdown interface{} // func(*cobra.Command, []string) error

	commands       []*Command // commands should be builded and added on "Build" state or even after it, `AddCommand` will handle this.
	currentCommand *cobra.Command

	FriendlyErrors FriendlyErrors

	CobraCommand *cobra.Command // the root command, after "Build" state.
}

func (app *Application) Print(format string, args ...interface{}) error {
	if !strings.HasSuffix(format, "\n") {
		format += "\r\n" // add a new line.
	}

	_, err := fmt.Fprintf(app, format, args...)
	return err
}

func (app *Application) PrintInfo(format string, args ...interface{}) error {
	if *app.MachineFriendly || GetSilentFlag(app.CobraCommand) {
		// check both --machine-friendly and --silent(optional flag,
		// but can be used side by side without machine friendly to disable info messages on user-friendly state)
		return nil
	}

	return app.Print(format, args...)
}

func (app *Application) PrintObject(v interface{}) error {
	return PrintObject(app.CobraCommand, v)
}

// func (app *Application) writeObject(w io.Writer, v interface{}) error {
// 	if *app.MachineFriendly {
// 		prettyFlagValue := !GetJSONNoPrettyFlag(app.CobraCommand)
// 		jmesQueryPathFlagValue := GetJSONQueryFlag(app.CobraCommand)
// 		return WriteJSON(w, v, prettyFlagValue, jmesQueryPathFlagValue)
// 	}

// 	return WriteTable(w, v)
// }

func PrintObject(cmd *cobra.Command, v interface{}, tableOnlyFilters ...interface{}) error {
	out := cmd.Root().OutOrStdout()
	machineFriendlyFlagValue := GetMachineFriendlyFlag(cmd)
	if machineFriendlyFlagValue {
		prettyFlagValue := !GetJSONNoPrettyFlag(cmd)
		jmesQueryPathFlagValue := GetJSONQueryFlag(cmd)
		return WriteJSON(out, v, prettyFlagValue, jmesQueryPathFlagValue)
	}

	tableprinter.Print(out, v, tableOnlyFilters...)
	return nil
}

func (app *Application) Write(b []byte) (int, error) {
	if app.CobraCommand == nil {
		return os.Stdout.Write(b)
	}

	return app.CobraCommand.OutOrStdout().Write(b)
}

func (app *Application) AddCommand(cmd *Command) {
	if app.CobraCommand == nil {
		// not builded yet, add these commands.
		app.commands = append(app.commands, cmd)
	} else {
		// builded, add them directly as cobra commands.
		app.CobraCommand.AddCommand(BuildCommand(app, cmd))
	}
}

func (app *Application) Run(output io.Writer, args []string) error {
	rand.Seed(time.Now().UTC().UnixNano()) // <3

	if output == nil {
		output = os.Stdout
	}

	rootCmd := Build(app)
	rootCmd.SetOutput(output)
	if len(args) == 0 && len(os.Args) > 0 {
		args = os.Args[1:]
	}

	if !rootCmd.DisableFlagParsing {
		rootCmd.ParseFlags(args)
	}

	if app.ShowSpinner && !*app.MachineFriendly {
		return ackError(app.FriendlyErrors, ExecuteWithSpinner(rootCmd))
	}

	return ackError(app.FriendlyErrors, rootCmd.Execute())
}

func (app *Application) exampleText(str string) string {
	return fmt.Sprintf("%s %s", app.Name, str)
}

func Build(app *Application) *cobra.Command {
	if app.CobraCommand != nil {
		return app.CobraCommand
	}

	if app.FriendlyErrors == nil {
		app.FriendlyErrors = FriendlyErrors{}
	}

	rootCmd := &cobra.Command{
		Version:                    app.Version,
		Use:                        fmt.Sprintf("%s [command] [flags]", app.Name),
		Short:                      app.Description,
		Long:                       app.Description,
		SilenceUsage:               true,
		SilenceErrors:              true,
		TraverseChildren:           true,
		SuggestionsMinimumDistance: 1,
	}

	app.MachineFriendly = new(bool)
	if !app.DisableOutputFormatController {
		rootCmd.PersistentFlags().BoolVar(app.MachineFriendly, machineFriendlyFlagKey, false, "--"+machineFriendlyFlagKey+" to output JSON results and hide all the info messages")
	}

	fs := rootCmd.Flags()
	if app.PersistentFlags != nil {
		app.PersistentFlags(fs)
	}

	var shutdown func(*cobra.Command, []string) error

	// DON'T DO IT HERE <- we dont have the arguments needed for parsing the "fs" yet.
	// if app.Setup != nil {
	// 	fn, err := makeAction(app.Setup, fs)
	// 	if err != nil {
	// 		panic(err) // TODO: remove panic but keep check before run.
	// 	}

	// 	setup = fn
	// }

	if app.Shutdown != nil {
		fn, err := makeAction(app.Shutdown, rootCmd.PersistentFlags())
		if err != nil {
			panic(err) // TODO: remove panic but keep check before run.
		}

		shutdown = fn
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		app.currentCommand = cmd // bind current command here.

		if app.Setup != nil {
			fn, err := makeAction(app.Setup, fs)
			if err != nil {
				return fmt.Errorf("%s: %v", cmd.Name(), err) // dev-use.
			}

			return fn(cmd, args)
		}

		return nil
	}

	// if setup == nil {
	// 	rootCmd.RunE = func(*cobra.Command, []string) error { return nil }
	// }

	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if shutdown != nil {
			return shutdown(cmd, args)
		}

		return nil
	}

	if len(app.commands) > 0 {
		for _, cmd := range app.commands {
			rootCmd.AddCommand(BuildCommand(app, cmd))
		}

		// clear mem.
		app.commands = nil
	}

	if rootCmd.HasAvailableSubCommands() {
		exampleText := rootCmd.Commands()[0].Example
		rootCmd.Example = exampleText
	}

	if app.HelpTemplate != nil {
		if helpTmpl := app.HelpTemplate.String(); helpTmpl != "" {
			rootCmd.SetVersionTemplate(helpTmpl)
		}
	}

	app.currentCommand = rootCmd
	app.CobraCommand = rootCmd
	return rootCmd
}

const machineFriendlyFlagKey = "machine-friendly"

func GetMachineFriendlyFlag(cmd *cobra.Command) bool {
	b, _ := cmd.Flags().GetBool(machineFriendlyFlagKey)
	return b
}

/* no...
type ApplicationBuilder struct {
	app *Application
}

func NewApplication(name, version string) *ApplicationBuilder {
	return &ApplicationBuilder{
		app: &Application{
			Name:    name,
			Version: version,
		},
	}
}

func (b *ApplicationBuilder) BuildNumbers(buildTime, buildRevision string) *ApplicationBuilder {
	oldHelpTmpl := b.app.HelpTemplate
	b.app.HelpTemplate = HelpTemplate{buildTime, buildRevision, true, oldHelpTmpl}

	return b
}

func (b *ApplicationBuilder) Spinner(enable bool) *ApplicationBuilder {
	b.app.ShowSpinner = enable

	return b
}

func (b *ApplicationBuilder) WithFriendlyError(code int, message string) *ApplicationBuilder {
	if b.app.FriendlyErrors == nil {
		b.app.FriendlyErrors = FriendlyErrors{}
	}

	b.app.FriendlyErrors[code] = message

	return b
}

func (b *ApplicationBuilder) AddCommand() *ApplicationBuilder {
	return b
}

func (b *ApplicationBuilder) BuildAndRun(output io.Writer) *Application {
	b.app.Run(output)
}
*/

type ApplicationBuilder struct {
	app *Application
}

func Name(name string) *ApplicationBuilder {
	return &ApplicationBuilder{
		app: &Application{Name: name},
	}
}

func (b *ApplicationBuilder) Description(description string) *ApplicationBuilder {
	b.app.Description = description
	return b
}

func (b *ApplicationBuilder) Version(version string) *ApplicationBuilder {
	b.app.Version = version
	return b
}

func (b *ApplicationBuilder) Setup(setupFunc interface{}) *ApplicationBuilder {
	b.app.Setup = setupFunc
	return b
}

func (b *ApplicationBuilder) Flags(fn func(*Flags)) *ApplicationBuilder {
	b.app.PersistentFlags = fn
	return b
}

func (b *ApplicationBuilder) GetFlags() *Flags {
	return b.app.CobraCommand.Flags()
}

func (b *ApplicationBuilder) Parse(args ...string) error {
	rootCmd := Build(b.app)
	return rootCmd.ParseFlags(args)
}

func (b *ApplicationBuilder) Run(w io.Writer, args []string) error {
	return b.app.Run(w, args)
}
