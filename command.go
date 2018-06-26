package bite

import (
	"strings"

	"github.com/spf13/cobra"
)

type Command struct {
	App     *Application
	Use     string
	Aliases []string
	Short   string
	Long    string
	Example string
	// RunE                 interface{} // Can be Run, PrintObject, PrintInfo or func(flag1 string, flag2 int, flag3 string, flag4 bool, flag5 time.Duration) error { }.
	// Run                  func(*Command, []string) error
	// PrintObject          func(*Command, []string) (interface{}, error)
	// PrintInfo            func(*Command, []string) (string, error)
	Action               interface{}
	CanPrintObject       bool
	CanPrintInfo         bool
	TryLoadFromFile      interface{} // ptr
	Children             []*Command
	ShareFlags           bool
	RequiredRuntimeFlags func() FlagPair
	Flags                func(*Flags)

	parent *Command

	MinArgs int
	MaxArgs int

	// after build.
	CobraCommand *cobra.Command
}

func (cmd *Command) canAcceptParentFlags() bool {
	return (cmd.ShareFlags && cmd.parent != nil) || (cmd.parent != nil && cmd.parent.hasRunner() == false)
}

func (cmd *Command) checkRequiredFlags(c *cobra.Command, _ []string) error {
	if cmd.canAcceptParentFlags() {
		if err := cmd.parent.checkRequiredFlags(c, nil); err != nil {
			return err
		}
	}

	if getFlags := cmd.RequiredRuntimeFlags; getFlags != nil {
		if err := CheckRequiredFlags(c, getFlags()); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *Command) Name() string {
	return strings.Split(cmd.Use, " ")[0]
}

/*
func (cmd *Command) run(args []string) error {
	if cmd.PrintObject != nil {
		v, err := cmd.PrintObject(cmd, args)
		if err != nil {
			return err
		}

		return cmd.App.PrintObject(v)
	}

	if cmd.PrintInfo != nil {
		msg, err := cmd.PrintInfo(cmd, args)
		if err != nil {
			return err
		}

		return cmd.App.PrintInfo(msg)
	}

	if cmd.Run == nil {
		return nil
	}

	return cmd.Run(cmd, args)
}
*/

func (cmd *Command) hasRunner() bool {
	return cmd.Action != nil // cmd.Run != nil || cmd.PrintObject != nil || cmd.PrintInfo != nil
}

// BuildCommand will build the bite command against an application, it tries to respect its `CobraCommand` values as well.
func BuildCommand(app *Application, cmd *Command) *cobra.Command {
	// Build(app) -> if we ever need to use its `CobraCommand` field.

	if cmd.CobraCommand == nil {
		cmd.CobraCommand = &cobra.Command{
			TraverseChildren: true,
			SilenceErrors:    true,
			SilenceUsage:     true,
		}
	}

	cmd.CobraCommand.Use = cmd.Use
	cmd.CobraCommand.Aliases = cmd.Aliases
	cmd.CobraCommand.Short = cmd.Short
	cmd.CobraCommand.Long = cmd.Long
	if cmd.Example != "" {
		cmd.CobraCommand.Example = app.exampleText(cmd.Example)
	}

	flagset := cmd.CobraCommand.LocalNonPersistentFlags()

	if cmd.CanPrintInfo { // || cmd.PrintInfo != nil {
		CanBeSilent(cmd.CobraCommand)
	}

	if cmd.CanPrintObject { //|| cmd.PrintObject != nil {
		CanPrintJSON(cmd.CobraCommand)
	}

	// this will do a reclusive flag set too.
	if cmd.canAcceptParentFlags() {
		flagset.AddFlagSet(cmd.parent.CobraCommand.LocalNonPersistentFlags())
	}

	if cmd.Flags != nil {
		cmd.Flags(flagset)
	}

	if cmd.parent != nil && cmd.parent.TryLoadFromFile != nil && !cmd.parent.hasRunner() {
		cmd.TryLoadFromFile = cmd.parent.TryLoadFromFile
	}

	// if cmd.hasRunner() {
	// after the flags set.
	// run, err := makeAction(cmd.Action, cmd.CobraCommand.Flags())
	// if err != nil {
	// 	panic(err) // panic? I don't like this but keep it as it's so it can break on build time ASAP, should be changed to return (*cobra.Command, error) in the future.
	// }

	Apply(cmd.CobraCommand,
		If(cmd.hasRunner(),
			ArgsRange(cmd.MinArgs, cmd.MaxArgs),
			FileBind(cmd.TryLoadFromFile),
			cmd.checkRequiredFlags,
			// after or before the local flags set, it runs at execute-time.
			Action(cmd.Action),
		))
	// }

	// if cmd.TryLoadFromFile != nil {
	// 	// should be the last .RunE modifier.
	// 	// ShouldTryLoadFile(cmd.CobraCommand, cmd.TryLoadFromFile)
	// }

	for _, child := range cmd.Children {
		child.parent = cmd
		cmd.CobraCommand.AddCommand(BuildCommand(app, child))
	}

	cmd.Children = nil

	return cmd.CobraCommand
}
