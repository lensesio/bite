package bite

// OutlineStringResults accepts a key, i.e "name" and entries i.e ["schema1", "schema2", "schema3"]
// and will convert it to a slice of [{"name":"schema1"},"name":"schema2", "name":"schema3"}] to be able to be printed via `printJSON`.
func OutlineStringResults(key string, entries []string) (items []interface{}) { // why not? (items []map[string]string) because jmespath can't work with it, only with []interface.
	// key = strings.Title(key)
	for _, entry := range entries {
		items = append(items, map[string]string{key: entry})
	}

	return
}

// OutlineIntResults accepts a key, i.e "version" and entries i.e [1, 2, 3]
// and will convert it to a slice of [{"version":3},"version":1, "version":2}] to be able to be printed via `printJSON`.
func OutlineIntResults(key string, entries []int) (items []interface{}) {
	// key = strings.Title(key)
	for _, entry := range entries {
		items = append(items, map[string]int{key: entry})
	}

	return
}
