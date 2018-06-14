package bite

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FlagPair is just a map[string]interface{}, see `CheckRequiredFlags` for more.
type FlagPair map[string]interface{}

// CheckRequiredFlags function can be used to manually check for required flags, when the command does not specify a required flag (mostly because of file loading feature).
func CheckRequiredFlags(cmd *cobra.Command, nameValuePairs FlagPair) (err error) {
	if nameValuePairs == nil {
		return nil
	}

	var emptyFlags []string

	for name, value := range nameValuePairs {
		v := reflect.Indirect(reflect.ValueOf(value))

		if v.CanInterface() && v.Type().Comparable() {
			if v.Interface() == reflect.Zero(reflect.TypeOf(value)).Interface() {
				emptyFlags = append(emptyFlags, strconv.Quote(name))
			}
		}
	}

	if n := len(emptyFlags); n > 0 {
		if n == 1 {
			// required flag "flag 1" not set
			err = fmt.Errorf("required flag %s not set", emptyFlags[0])
		} else {
			// required flags "flag 1" and "flag 2" not set
			// required flags "flag 1", "flag 2" and "flag 3" not set
			err = fmt.Errorf("required flags %s and %s not set",
				strings.Join(emptyFlags[0:n-1], ", "), emptyFlags[n-1])
		}

		if len(nameValuePairs) == n {
			// if all required flags are not passed, then show an example in the end.
			err = fmt.Errorf("%s\nexample:\n\t%s", err, cmd.Example)
		}
	}

	return
}

func isFunc(v reflect.Value) bool {
	return !v.IsNil() && v.IsValid() && v.Kind() == reflect.Func
}

var (
	commandTyp = reflect.TypeOf((*Command)(nil)).Elem()
	argsTyp    = reflect.TypeOf([]string{})
	errorTyp   = reflect.TypeOf((*error)(nil)).Elem()
)

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
type runnerValidationResult struct {
	FirstInputAsCommand   bool
	FirstInputAsArguments bool

	LastInputAsArguments bool

	NoInput bool

	FirstOutputAsError  bool
	FirstOutputAsObject bool
	FirstOutputAsString bool

	SecondOutputAsError bool

	NoOutput bool
}

func validateRunnerFn(fn reflect.Value, set *pflag.FlagSet) (runnerValidationResult, error) {
	var r runnerValidationResult
	if !isFunc(fn) {
		return r, fmt.Errorf("runnerFn is not a valid type of Func")
	}

	typ := fn.Type()

	// if func(input) and flags are not match throw error.
	if expected, got := set.NFlag(), typ.NumIn(); expected != got && got > 0 {
		// give a second chance, maybe the first flag is type of our Command.
		if got == expected+1 {
			if ftyp := typ.In(0); ftyp == commandTyp {
				r.FirstInputAsCommand = true
			} else if ftyp == argsTyp {
				r.FirstInputAsArguments = true
			} else {
				if got > 1 && expected > 1 {
					if ltyp := typ.In(got - 1); ltyp == argsTyp {
						r.LastInputAsArguments = true
					}
				} else {
					return r, fmt.Errorf("runnerFn input arguments[%d] and flags[%d] not matched: first input argument expected to be a kind of *Command or []string", got, expected)
				}
			}
		}

		if got > expected+1 {
			if ltyp := typ.In(got - 1); ltyp == argsTyp {
				r.LastInputAsArguments = true
			} else {
				return r, fmt.Errorf("runnerFn input arguments and flags not matched: last input argument expected to be a kind of []string")
			}
		}

		if !r.FirstInputAsArguments && !r.FirstInputAsCommand && !r.LastInputAsArguments {
			return r, fmt.Errorf("runnerFn input arguments and flags not matched, expected %d but got %d input arguments", expected, got)
		}
	} else if got == 0 {
		r.NoInput = true
	}

	numOut := typ.NumOut()
	if numOut == 0 {
		r.NoOutput = true
		return r, nil
	}

	if max, got := 2, numOut; got > max {
		return r, fmt.Errorf("runnerFn output arguments are more than %d (error or string or interface{} and error or string and error)", max)
	}

	// check first output argument.
	if ftyp := typ.Out(0); ftyp == errorTyp {
		r.FirstOutputAsError = true
	} else if ftyp.Kind() == reflect.String {
		r.FirstOutputAsString = true
	} else if ftyp.Kind() == reflect.Interface {
		r.FirstOutputAsObject = true
	}

	if numOut > 1 {
		// check second output argument.
		if ftyp := typ.Out(1); ftyp == errorTyp {
			r.SecondOutputAsError = true
		} else {
			return r, fmt.Errorf("runnerFn second output argument is not a type of error")
		}
	}

	return r, nil
}

func getFlagValue(name string, set *pflag.FlagSet) (flagValue reflect.Value) {
	flag := set.Lookup(name)
	flagName := flag.Name

	switch flag.Value.Type() {
	case "string":
		vv, _ := set.GetString(flagName)
		// don't be lazy, don't try something like 'var flagValue interface{}; ...end... reflect.ValueOf(flagValue)`.
		flagValue = reflect.ValueOf(vv)
	case "bool":
		vv, _ := set.GetBool(flagName)
		flagValue = reflect.ValueOf(vv)

	case "int", "count":
		vv, _ := set.GetInt(flagName)
		flagValue = reflect.ValueOf(vv)
	case "int32":
		vv, _ := set.GetInt32(flagName)
		flagValue = reflect.ValueOf(vv)
	case "int64":
		vv, _ := set.GetInt64(flagName)
		flagValue = reflect.ValueOf(vv)
	case "float32":
		vv, _ := set.GetFloat32(flagName)
		flagValue = reflect.ValueOf(vv)
	case "float64":
		vv, _ := set.GetFloat64(flagName)
		flagValue = reflect.ValueOf(vv)

	case "uint8":
		vv, _ := set.GetUint8(flagName)
		flagValue = reflect.ValueOf(vv)
	case "uint16":
		vv, _ := set.GetUint16(flagName)
		flagValue = reflect.ValueOf(vv)
	case "uint32":
		vv, _ := set.GetUint32(flagName)
		flagValue = reflect.ValueOf(vv)
	case "uint64":
		vv, _ := set.GetUint64(flagName)
		flagValue = reflect.ValueOf(vv)

	case "stringSlice":
		vv, _ := set.GetStringSlice(flagName)
		flagValue = reflect.ValueOf(vv)
	case "intSlice":
		vv, _ := set.GetIntSlice(flagName)
		flagValue = reflect.ValueOf(vv)
	case "uintSlice":
		vv, _ := set.GetUintSlice(flagName)
		flagValue = reflect.ValueOf(vv)
	case "boolSlice":
		vv, _ := set.GetBoolSlice(flagName)
		flagValue = reflect.ValueOf(vv)

	case "ip":
		vv, _ := set.GetIP(flagName)
		flagValue = reflect.ValueOf(vv)
	case "ipMask":
		vv, _ := set.GetIPv4Mask(flagName)
		flagValue = reflect.ValueOf(vv)
	case "ipNet":
		vv, _ := set.GetIPNet(flagName)
		flagValue = reflect.ValueOf(vv)
	}

	return
}

func ackRunnerError(cmd *cobra.Command, out []reflect.Value, rv runnerValidationResult) error {
	if rv.NoOutput || len(out) == 0 {
		return nil
	}

	if rv.FirstOutputAsError || rv.FirstOutputAsString || rv.FirstOutputAsObject {
		val := out[0]
		if val.IsNil() || !val.CanInterface() {
			return nil
		}

		if rv.FirstOutputAsError {
			return val.Interface().(error) // if the error was nil, it never goes here.
		}

		if rv.FirstOutputAsString {
			if err := PrintInfo(cmd, val.Interface().(string)); err != nil {
				return err
			}
		}

		if err := PrintObject(cmd, val.Interface()); err != nil {
			return err
		}
	}

	// has second output.
	if len(out) > 1 && rv.SecondOutputAsError {
		val := out[1]
		if val.IsNil() || !val.CanInterface() {
			return nil
		}

		return val.Interface().(error)
	}

	return nil
}

var emptyIn = []reflect.Value{}

// RunE collects the flags from the dynamic runner, give priority to the func instead of the registered flags, we collect the local flags.
// And returns a cobra-compatible runner function.
func RunE(runnerFn interface{}, set *pflag.FlagSet) (func(*cobra.Command, []string) error, error) {
	if runnerFn == nil {
		return nil, fmt.Errorf("runnerFn is nil")
	}

	fn := reflect.ValueOf(runnerFn)
	rv, err := validateRunnerFn(fn, set)
	if err != nil {
		return nil, err
	}

	if rv.NoInput {
		runner := func(cmd *cobra.Command, args []string) error {
			out := fn.Call(emptyIn)
			return ackRunnerError(cmd, out, rv)
		}

		return runner, nil
	}

	typ := fn.Type()
	n := typ.NumIn()

	in := make([]reflect.Value, n, n)
	i := 0
	if rv.FirstInputAsCommand || rv.FirstInputAsArguments {
		i = 1
	}

	set.Visit(func(flag *pflag.Flag) {
		flagName := flag.Name

		if expected, got := flag.Value.Type(), typ.In(i).Kind().String(); expected != got {
			errMsg := fmt.Sprintf("runnerFn input argument[%d] linked with flag '%s' has invalid kind of type, expected: %T but got %T", i, flagName, expected, got)
			if err != nil {
				errMsg = fmt.Sprintf("%s\n%s", err, errMsg) // catch as many errors as we can now.
			}
			err = fmt.Errorf(errMsg)
			return
		}

		flagValue := getFlagValue(flagName, set)
		in[i] = flagValue
		i++
	})

	if err != nil {
		return nil, err
	}

	runner := func(cmd *cobra.Command, args []string) error {
		if rv.FirstInputAsCommand {
			in[0] = reflect.ValueOf(cmd)
		} else if rv.FirstInputAsArguments {
			in[0] = reflect.ValueOf(args)
		}

		if rv.LastInputAsArguments {
			in[i] = reflect.ValueOf(args)
		}

		return ackRunnerError(cmd, fn.Call(in), rv)
	}

	return runner, nil
}

// Supported custom types underline are: strings, ints and booleans only.
type FlagVar struct {
	value reflect.Value
}

func NewFlagVar(v interface{}) *FlagVar {
	return &FlagVar{reflect.ValueOf(v)}
}

func (f FlagVar) String() string {
	return f.value.Elem().String()
}

func (f FlagVar) Set(v string) error {
	typ := f.value.Elem().Kind()
	switch typ {
	case reflect.String:
		f.value.Elem().SetString(v)
		break
	case reflect.Int:
		intValue, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		f.value.Elem().SetInt(int64(intValue))
		break
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}

		f.value.Elem().SetBool(boolValue)
		break
	}

	return nil
}

func (f FlagVar) Type() string {
	return f.value.Elem().Kind().String() // reflect/type.go#605
}
