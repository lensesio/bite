package bite

import (
	"fmt"
	"reflect"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func isGoodFunc(v reflect.Value) bool {
	return !v.IsNil() && v.IsValid() && v.Kind() == reflect.Func
}

var emptyIn = []reflect.Value{}

// makeAction collects the flags from the dynamic runner, give priority to the func instead of the registered flags, we collect the local flags.
// And returns a cobra-compatible runner function.
func makeAction(action interface{}, set *pflag.FlagSet) (func(*cobra.Command, []string) error, error) {
	fn := reflect.ValueOf(action)

	if !isGoodFunc(fn) {
		return nil, fmt.Errorf("bad type of action func")
	}

	typ := fn.Type()

	inputInfo, err := getInputDescription(typ, set)
	if err != nil {
		return nil, err
	}

	outputInfo, err := getOutputDescription(typ, set)
	if err != nil {
		return nil, err
	}

	if inputInfo.Empty { // return as soon as possible, no need for transfering each flag's value to the input argument of the command func action.
		runner := func(cmd *cobra.Command, args []string) error {
			return handleActionResult(cmd, fn, emptyIn, outputInfo)
		}
		return runner, nil
	}

	in, err := bindActionInputArguments(typ, set, inputInfo)
	if err != nil {
		return nil, err
	}

	runner := func(cmd *cobra.Command, args []string) error {
		if inputInfo.FirstAsCommand {
			in[0] = reflect.ValueOf(cmd)
		} else if inputInfo.FirstAsArguments {
			in[0] = reflect.ValueOf(args)
		}

		if inputInfo.LastAsArguments {
			in[len(in)-1] = reflect.ValueOf(args)
		}

		return handleActionResult(cmd, fn, in, outputInfo)
	}

	return runner, nil
}

var (
	commandTyp = reflect.TypeOf((*cobra.Command)(nil))
	argsTyp    = reflect.TypeOf([]string{})
	errorTyp   = reflect.TypeOf((*error)(nil)).Elem()
)

type (
	// TODO: should be called at build time to reduce any runtime errors on this.
	//
	// func(*Command) error
	// func([]string) error
	// func(*Command, []string) error
	// func(flag1 string, flag2 bool) error
	// func(*Command, []string, flag1 string, flag2 bool) error
	// func(flag1 string, flag2 bool) (interface{}, error)
	// func(flag1 string, flag2 bool) (string, error)
	// func(*Command, []string) (interface{}, error)
	// func(*Command, []string) (string, error)
	// commandFuncDescription struct { // metadata about the command function(runner) itself, useful for building the cobra-compatible runner.
	// 	In  commandFuncDescriptionInput
	// 	Out commandFuncDescriptionOutput
	// }

	actionDescriptionInput struct {
		FirstAsCommand   bool
		FirstAsArguments bool

		LastAsArguments bool

		Empty bool
	}

	actionDescriptionOutput struct {
		FirstAsError  bool
		FirstAsObject bool
		FirstAsString bool

		SecondAsError bool

		Empty bool
	}
)

func (desc actionDescriptionInput) firstPositionIsAllocatable() bool {
	return desc.FirstAsCommand || desc.FirstAsArguments
}

func getInputDescription(typ reflect.Type, set *pflag.FlagSet) (actionDescriptionInput, error) {
	var desc actionDescriptionInput

	// if func(input) and flags are not match throw error.
	if expected, got := set.NFlag(), typ.NumIn(); expected != got && got > 0 {
		if ftyp := typ.In(0); ftyp == commandTyp {
			desc.FirstAsCommand = true
		} else if ftyp == argsTyp {
			desc.FirstAsArguments = true
		}

		if ltyp := typ.In(got - 1); ltyp == argsTyp {
			desc.LastAsArguments = true
		}

		if !desc.FirstAsArguments && !desc.FirstAsCommand && !desc.LastAsArguments {
			return desc, fmt.Errorf("input arguments and flags not matched, expected %d but got %d input arguments", expected, got)
		}
	} else if got == 0 {
		desc.Empty = true
	}

	return desc, nil
}

func getOutputDescription(typ reflect.Type, set *pflag.FlagSet) (actionDescriptionOutput, error) {
	var desc actionDescriptionOutput

	numOut := typ.NumOut()
	if numOut == 0 {
		desc.Empty = true
		return desc, nil
	}

	if max, got := 2, numOut; got > max {
		return desc, fmt.Errorf("output arguments are more than %d (error or string or interface{} and error or string and error)", max)
	}

	// check first output argument.
	if ftyp := typ.Out(0); ftyp == errorTyp {
		desc.FirstAsError = true
	} else if ftyp.Kind() == reflect.String {
		desc.FirstAsString = true
	} else if ftyp.Kind() == reflect.Interface {
		desc.FirstAsObject = true
	}

	if numOut > 1 {
		// check second output argument.
		if ftyp := typ.Out(1); ftyp == errorTyp {
			desc.SecondAsError = true
		} else {
			return desc, fmt.Errorf("second output argument is not a type of error")
		}
	}

	return desc, nil
}

func bindActionInputArguments(typ reflect.Type, set *pflag.FlagSet, inputInfo actionDescriptionInput) ([]reflect.Value, error) {
	var (
		n   = typ.NumIn()
		in  = make([]reflect.Value, n, n)
		pos = 0

		err error
	)

	if inputInfo.firstPositionIsAllocatable() {
		pos = 1
	}

	set.Visit(func(flag *pflag.Flag) {
		flagName := flag.Name

		if expected, got := flag.Value.Type(), typ.In(pos).Kind().String(); expected != got {
			errMsg := fmt.Sprintf("input argument[%d] binded with flag '%s' has invalid kind of type, expected: %T but got %T", pos, flagName, expected, got)
			if err != nil {
				errMsg = fmt.Sprintf("%s\n%s", err, errMsg) // catch as many errors as we can now.
			}
			err = fmt.Errorf(errMsg)
			return
		}

		flagValue := getFlagValue(flagName, set)
		in[pos] = flagValue
		pos++
	})

	return in, err
}

func isGoodValue(val reflect.Value) bool {
	return (val.Kind() == reflect.Ptr && !val.IsNil()) || val.CanInterface()
}

func handleActionResult(cmd *cobra.Command, fn reflect.Value, in []reflect.Value, desc actionDescriptionOutput) error {
	if desc.Empty {
		return nil
	}

	out := fn.Call(in)
	if len(out) == 0 {
		return nil
	}

	if desc.FirstAsError || desc.FirstAsString || desc.FirstAsObject {
		val := out[0]
		if !isGoodValue(val) {
			return nil
		}

		if desc.FirstAsError {
			return val.Interface().(error) // if the error was nil, it never goes here.
		} else if desc.FirstAsString {
			if err := PrintInfo(cmd, val.Interface().(string)); err != nil {
				return err
			}
		} else {
			if err := PrintObject(cmd, val.Interface()); err != nil {
				return err
			}
		}
	}

	// has second output.
	if len(out) > 1 && desc.SecondAsError {
		val := out[1]
		if !isGoodValue(val) {
			return nil
		}

		return val.Interface().(error)
	}

	return nil
}
