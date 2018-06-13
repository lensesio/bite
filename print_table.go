package bite

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	headerTag            = "header"
	headerChildKeyTag    = "-"
	headerAsNumberKeyTag = "number"
	headerAsLenKeyTag    = "len"
)

func getHeaders(typ reflect.Type) (headers []string) {
	for i, n := 0, typ.NumField(); i < n; i++ {
		f := typ.Field(i)

		header := f.Tag.Get(headerTag)
		// embedded structs are acting like headers appended to the existing(s).
		if f.Type.Kind() == reflect.Struct && header == headerChildKeyTag {
			headers = append(headers, getHeaders(f.Type)...)
		} else if header != "" {
			// header is the first part.
			headers = append(headers, strings.Split(header, ",")[0])
		}
	}

	return
}

type tableRowDescription struct {
	valueAsNumber    bool
	valueAsArrayLen  bool
	alternativeValue string
	header           string
}

func getTableRowDescription(f reflect.StructField) (tableRowDescription, bool) {
	var tb tableRowDescription

	headerTagValue := f.Tag.Get(headerTag)
	if headerTagValue == "" {
		return tb, false
	}

	headerValues := strings.Split(headerTagValue, ",")
	switch len(headerValues) {
	case 0, 1:
		tb.header = headerTagValue
		break
	default:
		tb.header = headerValues[0]
		headerValues = headerValues[1:] /* except the first which should be the header value */
		for _, hv := range headerValues {
			switch hv {
			case headerAsNumberKeyTag:
				tb.valueAsNumber = true
				break
			case headerAsLenKeyTag:
				tb.valueAsArrayLen = true
				break
			default:
				tb.alternativeValue = hv
			}
		}
	}

	return tb, true
}

func getRow(val reflect.Value) (rightCells []int, row []string) {
	v := reflect.Indirect(val)
	typ := v.Type()
	j := 0
	for i, n := 0, typ.NumField(); i < n; i++ {
		rowDesc, ok := getTableRowDescription(typ.Field(i))
		if !ok {
			continue
		}

		fieldValue := reflect.Indirect(v.Field(i))

		if fieldValue.CanInterface() {
			s := ""
			vi := fieldValue.Interface()

			switch fieldValue.Kind() {
			case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
				rowDesc.valueAsNumber = true
				break
			case reflect.Float32, reflect.Float64:
				s = fmt.Sprintf("%.2f", vi)
				rightCells = append(rightCells, j)
				break
			case reflect.Bool:
				if vi.(bool) {
					s = "Yes"
				} else {
					s = "No"
				}
				break
			case reflect.Slice, reflect.Array:
				n := fieldValue.Len()
				if rowDesc.valueAsArrayLen {
					s = strconv.Itoa(n)
					rowDesc.valueAsNumber = true
				} else if n == 0 && rowDesc.alternativeValue != "" {
					s = rowDesc.alternativeValue
				} else {
					for fieldSliceIdx, fieldSliceLen := 0, fieldValue.Len(); fieldSliceIdx < fieldSliceLen; fieldSliceIdx++ {
						vf := fieldValue.Index(fieldSliceIdx)
						if vf.CanInterface() {
							s += fmt.Sprintf("%v", vf.Interface())
							if hasMore := fieldSliceIdx+1 > fieldSliceLen; hasMore {
								s += ", "
							}
						}
					}
				}
				break
			default:
				if viTyp := reflect.TypeOf(vi); viTyp.Kind() == reflect.Struct {
					rightEmbeddedSlices, rr := getRow(reflect.ValueOf(vi))
					if len(rr) > 0 {
						row = append(row, rr...)
						for range rightEmbeddedSlices {
							rightCells = append(rightCells, j)
							j++
						}

						continue
					}
				}

				s = fmt.Sprintf("%v", vi)
			}

			if rowDesc.valueAsNumber {
				// rightCells = append(rightCells, j)
				sInt64, err := strconv.ParseInt(fmt.Sprintf("%s", vi), 10, 64)
				if err != nil || sInt64 == 0 {
					s = rowDesc.alternativeValue
					if s == "" {
						s = "0"
					}
				} else {
					s = nearestThousandFormat(float64(sInt64))
				}

				rightCells = append(rightCells, j)
			}

			if s == "" {
				s = rowDesc.alternativeValue
			}

			row = append(row, s)
			j++
		}
	}

	return
}

type rowFilter func(reflect.Value) bool

func canAcceptRow(in reflect.Value, filters []rowFilter) bool {
	acceptRow := true
	for _, filter := range filters {
		if !filter(in) {
			acceptRow = false
			break
		}
	}

	return acceptRow
}

func makeFilters(in reflect.Value, filters []interface{}) (f []rowFilter) {
	for _, filter := range filters {
		filterTyp := reflect.TypeOf(filter)
		// must be a function that accepts one input argument which is the same of the "v".
		if filterTyp.Kind() != reflect.Func || filterTyp.NumIn() != 1 /* not receiver */ || filterTyp.In(0) != in.Type() {
			continue
		}

		// must be a function that returns a single boolean value.
		if filterTyp.NumOut() != 1 || filterTyp.Out(0).Kind() != reflect.Bool {
			continue
		}

		filterValue := reflect.ValueOf(filter)
		func(filterValue reflect.Value) {
			f = append(f, func(in reflect.Value) bool {
				out := filterValue.Call([]reflect.Value{in})
				return out[0].Interface().(bool)
			})
		}(filterValue)
	}

	return

}

// Usage with filters:
// printTable(cmd, topics, func(t Topic) bool { /* or any type */
// 	return t.TopicName == "test" || t.TopicName == "moving_ships"
// })
func PrintTable(cmd *cobra.Command, v interface{}, filters ...interface{}) error {
	return WriteTable(cmd.OutOrStdout(), v, filters...)
}

func WriteTable(w io.Writer, v interface{}, filters ...interface{}) error {
	table := tablewriter.NewWriter(w)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	var (
		headers           []string
		rows              [][]string
		rightAligmentCols []int
	)

	if val := reflect.Indirect(reflect.ValueOf(v)); val.Kind() == reflect.Slice {
		var f []rowFilter
		for i, n := 0, val.Len(); i < n; i++ {
			v := val.Index(i)

			if i == 0 {
				// make filters once instead of each time for each entry, they all have the same v type.
				f = makeFilters(v, filters)
				headers = getHeaders(v.Type())
			}

			if !v.IsValid() {
				rows = append(rows, []string{""})
				continue
			}
			right, row := getRow(v)
			if i == 0 {
				rightAligmentCols = right
			}

			if canAcceptRow(v, f) {
				rows = append(rows, row)
			}
		}
	} else {
		// single.
		headers = getHeaders(val.Type())
		right, row := getRow(val)
		rightAligmentCols = right
		if canAcceptRow(val, makeFilters(val, filters)) {
			rows = append(rows, row)
		}

	}

	if len(headers) == 0 {
		return nil
	}

	// if more than 3 then show the length of results.
	if n := len(rows); n > 3 {
		headers[0] = fmt.Sprintf("%s (%d) ", headers[0], len(rows))
	}

	table.SetHeader(headers)
	table.AppendBulk(rows)

	table.SetAutoFormatHeaders(true)
	table.SetAutoWrapText(true)
	table.SetBorders(tablewriter.Border{Bottom: false, Left: false, Right: false, Top: false})
	table.SetHeaderLine(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(false) /* can be true */
	table.SetColumnSeparator(" ")
	table.SetNewLine("\n")
	table.SetCenterSeparator(" ") /* can be empty */

	columnAlignment := make([]int, len(headers), len(headers))
	for i := range columnAlignment {
		columnAlignment[i] = tablewriter.ALIGN_LEFT

		for _, j := range rightAligmentCols {
			if i == j {
				columnAlignment[i] = tablewriter.ALIGN_RIGHT
				break
			}
		}

	}
	table.SetColumnAlignment(columnAlignment)

	fmt.Fprintln(w)
	table.Render()
	return nil
}
