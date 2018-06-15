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

	"github.com/spf13/cobra"
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
	ShowSpinner  bool

	MachineFriendly *bool

	Setup    func(*cobra.Command, []string) error
	Shutdown func(*cobra.Command, []string) error

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

func PrintObject(cmd *cobra.Command, v interface{}) error {
	machineFriendlyFlagValue := GetMachineFriendlyFlag(cmd)
	if machineFriendlyFlagValue {
		prettyFlagValue := !GetJSONNoPrettyFlag(cmd)
		jmesQueryPathFlagValue := GetJSONQueryFlag(cmd)
		return WriteJSON(cmd.Root().OutOrStdout(), v, prettyFlagValue, jmesQueryPathFlagValue)
	}

	return WriteTable(cmd.Root().OutOrStdout(), v)
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

func (app *Application) Run(output io.Writer) error {
	rand.Seed(time.Now().UTC().UnixNano()) // <3

	if output == nil {
		output = os.Stdout
	}

	rootCmd := Build(app)
	rootCmd.SetOutput(output)

	if app.ShowSpinner {
		ackError(app.FriendlyErrors, ExecuteWithSpinner(rootCmd))
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
		SilenceUsage:               true,
		SilenceErrors:              true,
		TraverseChildren:           true,
		SuggestionsMinimumDistance: 1,
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

	if helpTmpl := app.HelpTemplate.String(); helpTmpl != "" {
		rootCmd.SetVersionTemplate(helpTmpl)
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		app.currentCommand = cmd // bind current command here.

		if app.Setup != nil {
			return app.Setup(cmd, args)
		}

		return nil
	}

	if app.Shutdown != nil {
		rootCmd.PersistentPostRunE = app.Shutdown
	}

	rootCmd.PersistentFlags().BoolVar(app.MachineFriendly, machineFriendlyFlagKey, false, "--"+machineFriendlyFlagKey+" to output JSON results and hide all the info messages")

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
