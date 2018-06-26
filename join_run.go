package bite

import (
	"github.com/spf13/cobra"
)

type CobraRunner func(*cobra.Command, []string) error

func Join(runners ...CobraRunner) CobraRunner {
	return func(cmd *cobra.Command, args []string) error {
		// err := first(cmd, args)
		// if err != nil {
		// 	return err
		// }

		for _, r := range runners {
			if r == nil {
				continue
			}

			if err := r(cmd, args); err != nil {
				return err // first on fail error of course.
			}
		}

		return nil
	}
}

var emptyRunner = func(*cobra.Command, []string) error { return nil }

func If(condStatic bool, runners ...CobraRunner) CobraRunner {
	if !condStatic {
		return emptyRunner
	}

	return Join(runners...)
}

// // See Append as well.
// func ApplyRunners(cmd *cobra.Command, runners ...CobraRunner) {
// 	if len(runners) == 0 {
// 		return
// 	}

// 	cmd.RunE = Join(runners...)
// } <- no, let's keep the `Append -> Apply` only.

func Apply(cmd *cobra.Command, runners ...CobraRunner) {
	if len(runners) == 0 {
		return
	}

	if oldRunner := cmd.RunE; oldRunner != nil {
		runners = append([]CobraRunner{oldRunner}, runners...)
	}

	cmd.RunE = Join(runners...)
}
