package bite

import (
	"fmt"
	"io"
	"gopkg.in/yaml.v2"
)

func WriteYAML(w io.Writer, v interface{}) error {
	y, err := yaml.Marshal(v)

	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, string(y))
	return err
}