package bite

import (
	"fmt"
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
	Flags                func(FlagSet)

	parent *Command

	MinArgs int
	MaxArgs int

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

	if cmd.hasRunner() {
		// after the flags set.
		run, err := makeAction(cmd.Action, cmd.CobraCommand.Flags())
		if err != nil {
			panic(err) // panic? I don't like this but keep it as it's so it can break on build time ASAP, should be changed to return (*cobra.Command, error) in the future.
		}
		cmd.CobraCommand.RunE = func(c *cobra.Command, args []string) error {
			min, max, got := cmd.MinArgs, cmd.MaxArgs, len(args)
			gotStr := "nothing"
			if got > 0 {
				gotStr = fmt.Sprintf("%d", got)
			}

			if min+max > 0 && min == max && got != min {
				if min == 1 {
					return fmt.Errorf("%s command expected only one argument but got %s", c.Name(), gotStr)
				}
				return fmt.Errorf("%s command expected exactly %d arguments but got %s", c.Name(), min, gotStr)
			}

			if min > 0 && got < min {
				if max <= 0 {
					if min == 1 {
						return fmt.Errorf("%s command expected a single argument but got %s", c.Name(), gotStr)
					}

					return fmt.Errorf("%s command expected %d arguments but got %s", c.Name(), min, gotStr)
				}

				if min == 1 {
					return fmt.Errorf("%s command expected at least one argument", c.Name())
				}
				return fmt.Errorf("%s command expected at least %d arguments but got %s", c.Name(), min, gotStr)
			}

			if max > 0 && got > max {
				return fmt.Errorf("%s command can not accept more than %d arguments", c.Name(), max)
			}

			if err := cmd.checkRequiredFlags(c); err != nil {
				return err
			}

			return run(c, args)
		}
	}

	if cmd.TryLoadFromFile != nil {
		// should be the last .RunE modifier.
		ShouldTryLoadFile(cmd.CobraCommand, cmd.TryLoadFromFile)
	}

	for _, child := range cmd.Children {
		child.parent = cmd
		cmd.CobraCommand.AddCommand(BuildCommand(app, child))
	}

	cmd.Children = nil

	return cmd.CobraCommand
}
