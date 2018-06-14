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
