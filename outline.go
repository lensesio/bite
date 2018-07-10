package bite

import (
	"reflect"

	"github.com/spf13/cobra"
)

// like reflect.Indirect but reflect.Interface values too.
func indirectValue(val reflect.Value) reflect.Value {
	if kind := val.Kind(); kind == reflect.Interface || kind == reflect.Ptr {
		return val.Elem()
	}

	return val
}

// like reflect.Indirect but for types and reflect.Interface types too.
func indirectType(typ reflect.Type) reflect.Type {
	if kind := typ.Kind(); kind == reflect.Interface || kind == reflect.Ptr {
		return typ.Elem()
	}

	return typ
}

func makeDynamicSingleTableItem(header string, item interface{}) interface{} {
	f := make([]reflect.StructField, 1)
	f[0] = reflect.TypeOf(item).Field(0)
	f[0].Tag = reflect.StructTag(`header:"` + header + `"`)
	withHeaderTyp := reflect.StructOf(f)

	tmp := indirectValue(reflect.ValueOf(item)).Convert(withHeaderTyp)
	return tmp.Interface()
}

// OutlineStringResults accepts a key, i.e "name" and entries i.e ["schema1", "schema2", "schema3"]
// and will convert it to a slice of [{"name":"schema1"},"name":"schema2", "name":"schema3"}] to be able to be printed via `printJSON`.
func OutlineStringResults(cmd *cobra.Command, key string, entries []string) (items []interface{}) { // why not? (items []map[string]string) because jmespath can't work with it, only with []interface.
	// key = strings.Title(key)

	if GetMachineFriendlyFlag(cmd) {
		// prepare as JSON.
		for _, entry := range entries {
			items = append(items, map[string]string{key: entry})
		}

		return
	}

	// prepare as Table.
	for _, entry := range entries {
		item := struct {
			Value string
		}{entry}
		items = append(items, makeDynamicSingleTableItem(key, item))
	}

	return
}

// OutlineIntResults accepts a key, i.e "version" and entries i.e [1, 2, 3]
// and will convert it to a slice of [{"version":3},"version":1, "version":2}] to be able to be printed via `printJSON`.
func OutlineIntResults(cmd *cobra.Command, key string, entries []int) (items []interface{}) {
	// key = strings.Title(key)

	if GetMachineFriendlyFlag(cmd) {
		// prepare as JSON.
		for _, entry := range entries {
			items = append(items, map[string]int{key: entry})
		}

		return
	}

	// prepare as Table.
	for _, entry := range entries {
		item := struct {
			Value int
		}{entry}
		items = append(items, makeDynamicSingleTableItem(key, item))
	}

	return
}
