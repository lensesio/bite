package bite

import (
	"errors"
)

type Error interface {
	Code() int
	Message() string
	Error() string
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
