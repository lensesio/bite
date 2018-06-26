package bite

import (
	"fmt"

	"github.com/spf13/cobra"
)

func ArgsRange(min int, max int) CobraRunner {
	return func(c *cobra.Command, args []string) error {
		got := len(args)

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

		return nil
	}
}
