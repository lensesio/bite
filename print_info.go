package bite

import (
	"fmt"

	"github.com/spf13/cobra"
)

// if true then commands will not output info messages, like "Processor ___ created".
// Look the `echo` func for more, it's not a global flag but it's a common one, all commands that return info messages
// set that via command flag binding.
//
// Defaults to false.
const silentFlagKey = "silent"

func GetSilentFlag(cmd *cobra.Command) bool {
	b, _ := cmd.Flags().GetBool(silentFlagKey)
	return b
}

func CanBeSilent(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Bool(silentFlagKey, false, "run in silent mode. No printing info messages for CRUD except errors, defaults to false")
	return cmd
}

func PrintInfo(cmd *cobra.Command, format string, a ...interface{}) error {
	if GetSilentFlag(cmd) {
		return nil
	}

	_, err := fmt.Fprintf(cmd.OutOrStdout(), format, a...)
	return err
}
