package bite

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// if true then commands will not output info messages, like "Processor ___ created".
// Look the `PrintInfo` func for more, it's not a global flag but it's a common one, all commands that return info messages
// set that via command flag binding.
//
// Defaults to false.
const silentFlagKey = "silent"

// GetSilentFlag returns the value(true/false) of the `--silent` flag,
// however if not found it returns false too.
func GetSilentFlag(cmd *cobra.Command) bool {
	b, _ := cmd.Flags().GetBool(silentFlagKey)
	return b
}

// CanBeSilent registeres the `--silent` flag to the "cmd" command.
func CanBeSilent(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Bool(silentFlagKey, false, "run in silent mode. No printing info messages for CRUD except errors, defaults to false")
	return cmd
}

// HasFlag returns true if "flagName" can be found in the "cmd" cobra or its parents, otherwise false.
func HasFlag(cmd *cobra.Command, flagName string) (found bool) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name == flagName {
			found = true
		}
	})

	if !found {
		cmd.VisitParents(func(parentCmd *cobra.Command) {
			if HasFlag(parentCmd, flagName) {
				found = true
			}
		})
	}

	return
}

// HasSilentFlag returns true if the "cmd" command has registered the `--silent` flag.
func HasSilentFlag(cmd *cobra.Command) bool {
	return HasFlag(cmd, silentFlagKey)
}

// PrintInfo prints an info message to the command's standard output.
// If the `--silent`` flag is a REGISTERED flag for that command, then it will check if it's false or not set in order to print, otherwise
// it will check the `--machine-friendly` flag, if true not print.
//
// Useful when you want to have --machine-friendly on but want to print an important info message to the user as well but user can also disable that message via
// a second flag, the --silent one.
func PrintInfo(cmd *cobra.Command, format string, a ...interface{}) error {
	shouldPrint := !GetMachineFriendlyFlag(cmd)
	if HasSilentFlag(cmd) {
		shouldPrint = !GetSilentFlag(cmd)
	}

	if !shouldPrint {
		return nil
	}

	if !strings.HasSuffix(format, "\n") {
		format += "\n" // add a new line.
	}

	_, err := fmt.Fprintf(cmd.OutOrStdout(), format, a...)
	return err
}
