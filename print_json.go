package bite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jmespath/go-jmespath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewFlagSet(name string, register func(flags *pflag.FlagSet)) *pflag.FlagSet {
	flags := pflag.NewFlagSet(name, pflag.ExitOnError)
	if register != nil {
		register(flags)
	}
	return flags
}

const (
	// if true then noPretty & jmespathQuery are respected, it enables json printing
	// jsonFlagKey = "json"
	// if true then it doesn't prints json result(s) with indent.
	// Defaults to false.
	// It's not a global flag, but it's a common one, all commands that return results
	// use that via command flag binding.
	jsonNoPrettyFlagKey = "no-pretty"
	// jsonQueryFlagKey query to further filter any results, if any.
	// It's not a global flag, but it's a common one, all commands that return results
	// set that via command flag binding.
	jsonQueryFlagKey = "query"
)

// func GetJSONFlag(cmd *cobra.Command) bool {
// 	b, _ := cmd.Flags().GetBool(jsonFlagKey)
// 	return b
// }

func GetJSONNoPrettyFlag(cmd *cobra.Command) bool {
	b, _ := cmd.Flags().GetBool(jsonNoPrettyFlagKey)
	return b
}

func GetJSONQueryFlag(cmd *cobra.Command) string {
	s, _ := cmd.Flags().GetString(jsonQueryFlagKey)
	return s
}

var JSONFlagSet = NewFlagSet("flagset.json", func(flags *pflag.FlagSet) {
	// flags.Bool(jsonFlagKey, false, "enable the JSON output of commands (default false).")
	flags.Bool(jsonNoPrettyFlagKey, false, "disable the pretty format for JSON output of commands (default false).")
	flags.StringP(jsonQueryFlagKey, string(jsonQueryFlagKey[0]), "", "a jmespath query expression. This allows for querying the JSON output of commands")
})

func CanPrintJSON(cmd *cobra.Command) {
	cmd.Flags().AddFlagSet(JSONFlagSet)
}

func PrintJSON(cmd *cobra.Command, v interface{}) error {
	pretty := !GetJSONNoPrettyFlag(cmd)
	jmespathQuery := GetJSONQueryFlag(cmd)
	return WriteJSON(cmd.OutOrStdout(), v, pretty, jmespathQuery)
}

func WriteJSON(w io.Writer, v interface{}, pretty bool, jmespathQuery string) error {
	rawJSON, err := MarshalJSON(v, pretty, jmesQuery(jmespathQuery, v))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, string(rawJSON))
	return err
}

type Transformer func([]byte, bool) ([]byte, error)

func MarshalJSON(v interface{}, pretty bool, transformers ...Transformer) ([]byte, error) {
	var (
		rawJSON []byte
		err     error
	)

	if pretty {
		rawJSON, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
	} else {
		rawJSON, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}

	for _, transformer := range transformers {
		if transformer == nil {
			continue // may give a nil transformer in variadic input.
		}
		b, err := transformer(rawJSON, pretty)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			continue
		}
		rawJSON = b
	}

	// don't escape html.
	rawJSON = bytes.Replace(rawJSON, []byte("\\u003c"), []byte("<"), -1)
	rawJSON = bytes.Replace(rawJSON, []byte("\\u003e"), []byte(">"), -1)
	rawJSON = bytes.Replace(rawJSON, []byte("\\u0026"), []byte("&"), -1)

	return rawJSON, err
}

func jmesQuery(query string, v interface{}) Transformer {
	return func(rawJSON []byte, pretty bool) ([]byte, error) {
		if query == "" || strings.TrimSpace(string(rawJSON)) == "[]" { // if it's empty, exit.
			return nil, nil // don't throw error here, just skip it by returning nil result and nil error.
		}

		result, err := jmespath.Search(query, v)
		if err != nil {
			return nil, err
		}

		return MarshalJSON(result, pretty)
	}
}
