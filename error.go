package bite

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

type Error interface {
	Code() int
}

type FriendlyErrors map[int]string

func ackError(m FriendlyErrors, err error) error {
	if err == nil {
		return nil
	}

	// catch any errors that should be described by the command that gave that error.
	if resourceErr, ok := err.(Error); ok {
		if m != nil {
			if errMsg, ok := m[resourceErr.Code()]; ok {
				return errors.New(errMsg)
			}
		}
	}

	return err
}

func FriendlyError(cmd *cobra.Command, code int, format string, args ...interface{}) {
	app := Get(cmd)
	if app.FriendlyErrors == nil {
		app.FriendlyErrors = make(FriendlyErrors)
	}

	app.FriendlyErrors[code] = fmt.Sprintf(format, args...)
}
