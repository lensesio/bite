package bite

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestActionDescription(t *testing.T) {
	args := append(os.Args, []string{"--name=name", "--age=12"}...)

	set := NewFlagSet("test.flags", func(flags FlagSet) {
		flags.String("name", "", "--name=name")
		flags.Int("age", 0, "--age=12")
		flags.Bool("acc", false, "--acc")
	})

	tests := []struct {
		args []string
		fn   interface{}
		in   actionDescriptionInput
		out  actionDescriptionOutput
	}{
		{
			args: args,
			fn: func(string, int, bool) error { // strict, may "Strict" option for that(?).
				return nil
			},
			out: actionDescriptionOutput{FirstAsError: true},
		},
		{
			args: append(args, "--acc=true"),
			fn: func(string, int, bool, []string) (interface{}, error) {
				return nil, nil
			},
			in:  actionDescriptionInput{LastAsArguments: true},
			out: actionDescriptionOutput{FirstAsObject: true, SecondAsError: true},
		},
		{
			fn: func() (interface{}, error) {
				return nil, nil
			},
			in:  actionDescriptionInput{Empty: true},
			out: actionDescriptionOutput{FirstAsObject: true, SecondAsError: true},
		},
		{
			fn: func(*cobra.Command, []string) (interface{}, error) {
				return nil, nil
			},
			in:  actionDescriptionInput{FirstAsCommand: true, LastAsArguments: true},
			out: actionDescriptionOutput{FirstAsObject: true, SecondAsError: true},
		},
	}

	for i, tt := range tests {
		if len(tt.args) > 0 {
			if err := set.Parse(tt.args); err != nil {
				t.Fatalf("[%d] %v", i, err)
			}
		}

		typ := reflect.TypeOf(tt.fn)

		in, err := getInputDescription(typ, set)
		if err != nil {
			t.Fatalf("[%d] input description resolver failure: %v", i, err)
		}
		out, err := getOutputDescription(typ, set)
		if err != nil {
			t.Fatalf("[%d] output description resolver failure: %v", i, err)
		}

		if !reflect.DeepEqual(in, tt.in) {
			t.Fatalf("[%d] input description expected to be:\n%#+v\n but got:\n%#+v", i, tt.in, in)
		}

		if !reflect.DeepEqual(out, tt.out) {
			t.Fatalf("[%d] output description expected to be:\n%#+v\n but got:\n%#+v", i, tt.out, out)
		}
	}
}

type noOpWriter struct{}

func (w noOpWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func TestAction(t *testing.T) {
	app := &Application{
		Name:        "my-app",
		Version:     "0.0.1",
		Description: "My awesome app",
	}

	cmd := &Command{
		Use:     "echo",
		Example: "echo World!",
		Action: func(args []string) string {
			return fmt.Sprintf("Hello %s", strings.Join(args, " "))
		},
	}

	cobraCommand := BuildCommand(app, cmd)
	w := new(bytes.Buffer)
	cobraCommand.SetOutput(w)

	if err := cobraCommand.RunE(cobraCommand, []string{"World!"}); err != nil {
		t.Fatal(err)
	}

	if expected, got := "Hello World!", w.String(); expected != got {
		t.Fatalf("expected output: '%s' but got: '%s'", expected, got)
	}

	w.Reset()
}
