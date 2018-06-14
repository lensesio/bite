package bite

import (
	"os"
	"reflect"
	"testing"
)

func validationResultShouldBe(t *testing.T, got runnerValidationResult, expected runnerValidationResult) {
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("validation result expected to be:\n%#+v\n but got:\n%#+v", expected, got)
	}
}
func TestValidateRunnerFn(t *testing.T) {
	args := append(os.Args, []string{"--name=name", "--age=12"}...)

	set := NewFlagSet("test.flags", func(flags FlagSet) {
		flags.String("name", "", "--name=name")
		flags.Int("age", 0, "--age=12")
		flags.Bool("acc", false, "--acc")
	})

	tests := []struct {
		args     []string
		fn       interface{}
		expected runnerValidationResult
	}{
		{
			args,
			func(flag1 string, flag2 int) error {
				return nil
			},
			runnerValidationResult{FirstOutputAsError: true},
		},
		{
			append(args, "--acc=true"),
			func(flag1 string, flag2 int, flag3 bool, a []string) (interface{}, error) {
				return nil, nil
			},
			runnerValidationResult{LastInputAsArguments: true, FirstOutputAsObject: true, SecondOutputAsError: true},
		},
	}

	for i, tt := range tests {
		if err := set.Parse(tt.args); err != nil {
			t.Fatalf("[%d] %v", i, err)
		}

		rv, err := validateRunnerFn(reflect.ValueOf(tt.fn), set)
		if err != nil {
			t.Fatalf("[%d] %v", i, err)
		}

		validationResultShouldBe(t, rv, tt.expected)
	}
}
