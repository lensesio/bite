package bite

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type FlagSet = *pflag.FlagSet

type Command struct {
	App                  *Application
	Use                  string
	Aliases              []string
	Short                string
	Example              string
	RunE                 interface{} // Can be Run, PrintObject, PrintInfo or func(flag1 string, flag2 int, flag3 string, flag4 bool, flag5 time.Duration) error { }.
	Run                  func(*Command, []string) error
	PrintObject          func(*Command, []string) (interface{}, error)
	PrintInfo            func(*Command, []string) (string, error)
	CanPrintObject       bool
	CanPrintInfo         bool
	TryLoadFromFile      interface{} // ptr
	Children             []*Command
	ShareFlags           bool
	RequiredRuntimeFlags func() FlagPair
	Flags                func(FlagSet)
	flagset              FlagSet
	parent               *Command

	// after build.
	CobraCommand *cobra.Command
}

func (cmd *Command) canAcceptParentFlags() bool {
	return (cmd.ShareFlags && cmd.parent != nil) || (cmd.parent != nil && cmd.parent.hasRunner() == false)
}

func (cmd *Command) checkRequiredFlags(c *cobra.Command) error {
	if cmd.canAcceptParentFlags() {
		if err := cmd.parent.checkRequiredFlags(c); err != nil {
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

func (cmd *Command) hasRunner() bool {
	return cmd.Run != nil || cmd.PrintObject != nil || cmd.PrintInfo != nil
}

// BuildCommand will build the bite command against an application, it tries to respect its `CobraCommand` values as well.
func BuildCommand(app *Application, cmd *Command) *cobra.Command {
	if cmd.Example == "" {
		cmd.Example = cmd.Name() + " --help"
	}

	if cmd.CobraCommand == nil {
		cmd.CobraCommand = &cobra.Command{
			TraverseChildren: true,
			SilenceErrors:    true,
		}
	}

	cmd.CobraCommand.Use = cmd.Use
	cmd.CobraCommand.Aliases = cmd.Aliases
	cmd.CobraCommand.Short = cmd.Short
	cmd.CobraCommand.Example = app.exampleText(cmd.Example)

	if cmd.hasRunner() {
		cmd.CobraCommand.RunE = func(c *cobra.Command, args []string) error {
			if err := cmd.checkRequiredFlags(c); err != nil {
				return err
			}

			return cmd.run(args)
		}
	}

	cmd.flagset = NewFlagSet("bite-"+cmd.Name(), nil)

	if cmd.CanPrintInfo || cmd.PrintInfo != nil {
		CanBeSilent(cmd.CobraCommand)
	}

	if cmd.CanPrintObject || cmd.PrintObject != nil {
		CanPrintJSON(cmd.CobraCommand)
	}

	// this will do a reclusive flag set too.
	if cmd.canAcceptParentFlags() {
		cmd.flagset.AddFlagSet(cmd.parent.flagset)
	}

	if cmd.Flags != nil {
		cmd.Flags(cmd.flagset)
	}

	cmd.CobraCommand.LocalNonPersistentFlags().AddFlagSet(cmd.flagset)

	if cmd.parent != nil && cmd.parent.TryLoadFromFile != nil && !cmd.parent.hasRunner() {
		cmd.TryLoadFromFile = cmd.parent.TryLoadFromFile
	}

	if cmd.TryLoadFromFile != nil {
		ShouldTryLoadFile(cmd.CobraCommand, cmd.TryLoadFromFile)
	}

	for _, child := range cmd.Children {
		child.parent = cmd
		cmd.CobraCommand.AddCommand(BuildCommand(app, child))
	}

	cmd.Children = nil

	return cmd.CobraCommand
}
