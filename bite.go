package bite

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/landoop/tableprinter"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type HelpTemplate struct {
	Name				 string	
	BuildTime            string
	BuildRevision        string
	BuildVersion		 string
	ShowGoRuntimeVersion bool

	Template fmt.Stringer
}

func (h HelpTemplate) String() string {
	if tmpl := h.Template; tmpl != nil {
		return tmpl.String()
	}

	buildTitle := ">>>> build" // if we ever want an emoji, there is one: \U0001f4bb
	tab := strings.Repeat(" ", len(buildTitle))

	// unix nanoseconds, as int64, to a human readable time, defaults to time.UnixDate, i.e:
	// Thu Mar 22 02:40:53 UTC 2018
	// but can be changed to something like "Mon, 01 Jan 2006 15:04:05 GMT" if needed.
	n, _ := strconv.ParseInt(h.BuildTime, 10, 64)
	buildTimeStr := time.Unix(n, 0).Format(time.UnixDate)

	tmpl := fmt.Sprintf("%s %s", h.Name, h.BuildVersion) +
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
	Long        string

	HelpTemplate fmt.Stringer
	// ShowSpinner if true(default is false) then
	// it waits via "visual" spinning before each command's job done.
	ShowSpinner                   bool
	DisableOutputFormatController bool
	OutPut                        *string
	PersistentFlags               func(*pflag.FlagSet)

	Setup          CobraRunner
	Shutdown       CobraRunner
	commands       []*cobra.Command // commands should be builded and added on "Build" state or even after it, `AddCommand` will handle this.
	currentCommand *cobra.Command

	FriendlyErrors FriendlyErrors
	Memory         *Memory

	tablePrintersCache                     map[io.Writer]*tableprinter.Printer
	tablePrintersMu                        sync.RWMutex
	TableHeaderBgColor, TableHeaderFgColor string // see `whichColor(string, int) int`

	CobraCommand *cobra.Command // the root command, after "Build" state.

}

func (app *Application) ClearPrintCache() {
	app.tablePrintersCache = make(map[io.Writer]*tableprinter.Printer)
}

func (app *Application) Print(format string, args ...interface{}) error {
	if !strings.HasSuffix(format, "\n") {
		format += "\r\n" // add a new line.
	}

	_, err := fmt.Fprintf(app, format, args...)
	return err
}

func (app *Application) PrintInfo(format string, args ...interface{}) error {
	return PrintInfo(app.currentCommand, format, args...)
}

func (app *Application) PrintObject(v interface{}) error {
	return PrintObject(app.currentCommand, v)
}

// func (app *Application) writeObject(out io.Writer, v interface{}, tableOnlyFilters ...interface{}) error {
// 	machineFriendlyFlagValue := GetMachineFriendlyFlag(app.CobraCommand)
// 	if machineFriendlyFlagValue {
// 		prettyFlagValue := !GetJSONPrettyFlag(app.currentCommand)
// 		jmesQueryPathFlagValue := GetJSONQueryFlag(app.currentCommand)
// 		return WriteJSON(out, v, prettyFlagValue, jmesQueryPathFlagValue)
// 	}
//
// 	tableprinter.Print(out, v, tableOnlyFilters...)
// 	return nil
// }

func PrintObject(cmd *cobra.Command, v interface{}, tableOnlyFilters ...interface{}) error {
	out := cmd.OutOrStdout()
	outputFlagValue := GetOutPutFlag(cmd)
	if strings.ToUpper(outputFlagValue) == "JSON" {
		prettyFlagValue := GetJSONPrettyFlag(cmd)
		jmesQueryPathFlagValue := GetJSONQueryFlag(cmd)
		return WriteJSON(out, v, prettyFlagValue, jmesQueryPathFlagValue)
	} else if strings.ToUpper(outputFlagValue) == "YAML" {
		return WriteYAML(out, v)
	} else {
		app := Get(cmd)
		// normally the io.Writer is one, so the tableprinter; the app(it's io.Writer: cmd -> root command's output -> run(w io.Writer) -> app.Write)
		// but it can be changed manually before this call, so make a check to have only one talbeprinter instance per those writers.
		app.tablePrintersMu.RLock()
		printer, ok := app.tablePrintersCache[out]
		app.tablePrintersMu.RUnlock()
		if !ok {
			// register it.
			printer = tableprinter.New(out)
			app.tablePrintersMu.Lock()
			app.tablePrintersCache[out] = printer
			app.tablePrintersMu.Unlock()
		}

		if v := app.TableHeaderFgColor; v != "" {
			printer.HeaderFgColor = whichColor(v, 30)
		}

		if v := app.TableHeaderBgColor; v != "" {
			printer.HeaderBgColor = whichColor(v, 40)
		}

		// This will try to append a struct-only as a row
		// for same writer (see above) and the headers cache contains this struct's header-tag fields(;printed at least one time before (see StructHeaders[typ])).
		typ := indirectType(reflect.TypeOf(v))
		if typ.Kind() == reflect.Struct {
			if len(tableprinter.StructHeaders[typ]) > 0 {
				row, nums := tableprinter.StructParser.ParseRow(indirectValue(reflect.ValueOf(v)))
				printer.RenderRow(row, nums)
				return nil
			}
		}

		rowsLengthPrinted := printer.Print(v, tableOnlyFilters...)
		if rowsLengthPrinted == -1 { // means we can't print.
			// This makes sure that all content, even if json-only tagged will be printed as table, even if not specified as table-ready,
			// it shouldn't happen but keep it for any case, it's better to show something instead of nothing if there is actually something to be shown here.
			// It should be avoided, manual `header` tagging is required; otherwise the table's cells may be larger than expected due of picking all json-tagged properties.
			//
			// If nothing printed, try load all it as json and print all of its keys(as headers) and rows(values) as a table using the printer's `PrintJSON`.
			prettyFlagValue := GetJSONPrettyFlag(cmd)
			jmesQueryPathFlagValue := GetJSONQueryFlag(cmd)
			rawJSON, err := MarshalJSON(v, prettyFlagValue, jmesQuery(jmesQueryPathFlagValue, v))
			if err != nil {
				return err
			}

			printer.PrintJSON(rawJSON, tableOnlyFilters)
			return nil
		}
	}

	return nil
}

func (app *Application) Write(b []byte) (int, error) {
	if app.CobraCommand == nil {
		return os.Stdout.Write(b)
	}

	return app.CobraCommand.OutOrStdout().Write(b)
}

func (app *Application) AddCommand(cmd *cobra.Command) {
	if app.CobraCommand == nil {
		// not builded yet, add these commands.
		app.commands = append(app.commands, cmd)
	} else {
		// builded, add them directly as cobra commands.
		app.CobraCommand.AddCommand(cmd)
	}
}

func newBashCompletionCommand(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generates bash completion scripts",
		Long: `To load completion run

	. <(lenses-cli completion)

	To configure your bash shell to load completions for each session add to your bashrc

	echo "source <(lenses-cli completion bash)" >> ~/.bashrc
	`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenBashCompletion(os.Stdout);
			
		},	
	}
	return cmd
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

	app.commands = nil

	if app.ShowSpinner {
		return ackError(app.FriendlyErrors, ExecuteWithSpinner(rootCmd))
	}

	app.AddCommand(newBashCompletionCommand(rootCmd))

	return ackError(app.FriendlyErrors, rootCmd.Execute())
}

func (app *Application) exampleText(str string) string {
	return fmt.Sprintf("%s %s", app.Name, str)
}

// keeps track of the Applications, this is the place that built applications are being stored,
// so the `Get` can receive the exact Application that the command belongs to, a good example is the `FriendlyError` function and all the `bite` package-level helpers.
var applications []*Application

func registerApplication(app *Application) {
	for i, a := range applications {
		if a.Name == app.Name {
			// override the existing and exit.
			applications[i] = app
			return
		}
	}

	applications = append(applications, app)
}

func Get(cmd *cobra.Command) *Application {
	if app := GetByName(cmd.Name()); app != nil {
		return app
	}

	if cmd.HasParent() {
		return Get(cmd.Parent())
	}

	return nil
}

func GetByName(applicationName string) *Application {
	for _, app := range applications {
		if app.Name == applicationName {
			return app
		}
	}

	return nil
}

func FindCommand(applicationName string, args []string) (*cobra.Command, []string) {
	app := GetByName(applicationName)
	if app == nil {
		return nil, nil
	}

	c, cArgs, err := app.CobraCommand.Find(args)
	if err != nil {
		return nil, nil
	}

	return c, cArgs
}

func (app *Application) FindCommand(args []string) (*cobra.Command, []string) {
	return FindCommand(app.Name, args)
}

func getCommand(from *cobra.Command, subCommandName string) *cobra.Command {
	for _, c := range from.Commands() {
		if c.Name() == subCommandName {
			return c
		}

		return getCommand(c, subCommandName)
	}

	return nil
}

func GetCommand(applicationName string, commandName string) *cobra.Command {
	app := GetByName(applicationName)
	if app == nil {
		return nil
	}

	return getCommand(app.CobraCommand, commandName)
}

func (app *Application) GetCommand(commandName string) *cobra.Command {
	return GetCommand(app.Name, commandName)
}

func Build(app *Application) *cobra.Command {
	if app.CobraCommand != nil {
		return app.CobraCommand
	}

	if app.FriendlyErrors == nil {
		app.FriendlyErrors = FriendlyErrors{}
	}

	if app.Memory == nil {
		app.Memory = makeMemory()
	}

	app.tablePrintersCache = make(map[io.Writer]*tableprinter.Printer)

	useText := app.Name
	if strings.LastIndexByte(app.Name, '[') < len(strings.Split(app.Name, " ")[0]) {
		useText = fmt.Sprintf("%s [command] [flags]", app.Name)
	}

	// if app.Long == "" {
	// 	app.Long = app.Description
	// }

	rootCmd := &cobra.Command{
		Version:                    app.Version,
		Use:                        useText,
		Short:                      app.Description,
		Long:                       app.Long,
		SilenceUsage:               true,
		SilenceErrors:              true,
		TraverseChildren:           true,
		SuggestionsMinimumDistance: 1,
	}

	fs := rootCmd.PersistentFlags()

	app.OutPut = new(string)

	fmt.Sprintf("HEEEEEERREEEEE flag is [%v] ", app.DisableOutputFormatController)
	if !app.DisableOutputFormatController {
		RegisterOutPutFlagTo(fs, app.OutPut)

		fs.StringVar(&app.TableHeaderFgColor, "header-fgcolor", "", "Table headers forground gcolor=black")
		fs.StringVar(&app.TableHeaderBgColor, "header-bgcolor", "", "Table headers background gcolor=white")
	}

	if app.PersistentFlags != nil {
		app.PersistentFlags(fs)
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		app.currentCommand = cmd // bind current command here.

		if app.Setup != nil {
			return app.Setup(cmd, args)
		}

		return nil
	}

	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if app.Shutdown != nil {
			return app.Shutdown(cmd, args)
		}

		return nil
	}

	if len(app.commands) > 0 {
		for _, cmd := range app.commands {
			rootCmd.AddCommand(cmd)
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

	if versionName := "version"; app.GetCommand(versionName) == nil {
		rootCmd.AddCommand(&cobra.Command{
			Use:           versionName,
			Short:         "Print the current version of " + app.Name,
			Example:       versionName,
			SilenceErrors: true,
			RunE: func(cmd *cobra.Command, _ []string) error {
				// cobra's "tmpl" func is not exported, so we can't use it.
				// Therefore add the --version flag on the root command and let it to its job.
				if def, _ := cmd.Flags().GetBool(versionName); !def {
					rootCmd.SetArgs([]string{"--" + versionName})
					rootCmd.Execute()
				}

				return nil
			},
		})
	}

	app.currentCommand = rootCmd
	app.CobraCommand = rootCmd

	registerApplication(app)
	return rootCmd
}

const outputFlagKey = "output"

func GetOutPutFlagKey() string {
	return outputFlagKey
}

func GetOutPutFlagFrom(set *pflag.FlagSet) string {
	b, _ := set.GetString(outputFlagKey)
	return b
}

func GetOutPutFlag(cmd *cobra.Command) string {
	return GetOutPutFlagFrom(cmd.Flags())
}

func RegisterOutPutFlagTo(set *pflag.FlagSet, ptr *string) {
	set.StringVar(ptr, outputFlagKey, "table", "TABLE, JSON or YAML results and hide all the info messages")
}

func RegisterOutPutFlag(cmd *cobra.Command, ptr *string) {
	RegisterOutPutFlagTo(cmd.Flags(), ptr)
}

type ApplicationBuilder struct {
	app *Application
}

func Name(name string) *ApplicationBuilder {
	return &ApplicationBuilder{
		app: &Application{Name: name},
	}
}

func (b *ApplicationBuilder) Get() *Application {
	return b.app
}

func (b *ApplicationBuilder) Description(description string) *ApplicationBuilder {
	b.app.Description = description
	return b
}

func (b *ApplicationBuilder) Version(version string) *ApplicationBuilder {
	b.app.Version = version
	return b
}

func (b *ApplicationBuilder) Setup(setupFunc CobraRunner) *ApplicationBuilder {
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
