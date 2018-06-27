package bite

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/spf13/cobra"
)

// Memory describes a temporary storage for each application, useful to store different kind of custom values.
type Memory struct {
	tmp map[uint8]interface{} // why not string and uint8 as key? With uint8(0-255) we set a kind of limit of the elements stored without any further complications.
	mu  sync.RWMutex
}

// m.Set(MyKey, MyValue{}) == existed and replaced, safe for concurrent access.
func (m *Memory) Set(key uint8, value interface{}) (replacement bool) {
	if m.tmp == nil {
		return
	}

	replacement = m.Has(key)

	m.mu.Lock()
	m.tmp[key] = value
	m.mu.Unlock()

	return
}

// SetOnce will store a value based on its key if it's not there already, do not confuse its action with immutability.
func (m *Memory) SetOnce(key uint8, value interface{}) bool {
	if m.tmp == nil {
		return false
	}

	if len(m.tmp) > 0 && m.Has(key) {
		return false
	}

	m.mu.Lock()
	m.tmp[key] = value
	m.mu.Unlock()

	return true
}

var (
	emptyIn  []reflect.Value
	errorTyp = reflect.TypeOf((*error)(nil)).Elem()
)

func (m *Memory) SetOnceFunc(key uint8, receiverFunc interface{}) error {
	if m.tmp == nil {
		return nil
	}

	if len(m.tmp) > 0 && m.Has(key) {
		return nil
	}

	// execute the receiverFunc to get its value.
	fn := reflect.Indirect(reflect.ValueOf(receiverFunc))
	if fn.Kind() == reflect.Interface {
		fn = fn.Elem()
	}

	if fn.Kind() != reflect.Func {
		return fmt.Errorf("mem: receiverFunc not type of func")
	}

	fnTyp := fn.Type()
	if fnTyp.NumIn() != 0 {
		return fmt.Errorf("mem: receiverFunc should not accept any input argument")
	}

	fnOut := fnTyp.NumOut()
	if fnOut < 1 || fnOut > 2 {
		return fmt.Errorf("mem: receiverFunc should return one(value) or two (value, error) but returns %d values", fnOut)
	}

	// check if second output value is error, if so we will return that to the caller.
	if fnOut == 2 && fnTyp.Out(1) != errorTyp {
		return fmt.Errorf("mem: receiverFunc second output value should be type of error")
	}

	out := fn.Call(emptyIn)

	// first is the value we must set.
	if got := out[0]; got.CanInterface() {
		m.Set(key, got.Interface())
	}
	// }else if fnOut == 1{
	// 	return fmt.Errorf("mem: nothing to set")
	// }

	if fnOut == 2 {
		// second value was error, return that.
		if got := out[1]; got.CanInterface() {
			if err, sure := got.Interface().(error); sure {
				return err
			}
		}
	}

	return nil
}

// m.Unset(MyKey) == removed, safe for concurrent access.
func (m *Memory) Unset(key uint8) (removed bool) {
	if len(m.tmp) == 0 {
		return
	}

	if m.Has(key) {
		m.mu.Lock()
		delete(m.tmp, key)
		m.mu.Unlock()
		removed = true
	}

	return
}

// m.Has(MyKey) == exists, safe for concurrent access.
func (m *Memory) Has(key uint8) bool {
	if len(m.tmp) == 0 {
		return false
	}

	m.mu.RLock()
	_, exists := m.tmp[key]
	m.mu.RUnlock()

	return exists
}

// value, found := m.Get(MyKey), safe for concurrent access.
func (m *Memory) Get(key uint8) (value interface{}, found bool) {
	if len(m.tmp) == 0 {
		return
	}

	m.mu.Lock()
	value, found = m.tmp[key]
	m.mu.Unlock()

	return
}

func (m *Memory) MustGet(key uint8) interface{} {
	v, ok := m.Get(key)
	if !ok {
		panic(fmt.Sprintf("mem: key for %d missing", key))
	}

	if v == nil {
		panic(fmt.Sprintf("mem: value for key %d is nil", key))
	}

	return v
}

// GetAll returns a clone of all the stored values, safe for concurrent access.
func (m *Memory) GetAll() map[uint8]interface{} {
	if len(m.tmp) == 0 {
		return make(map[uint8]interface{})
	}

	clone := make(map[uint8]interface{}, len(m.tmp))

	m.mu.RLock()
	for key, value := range m.tmp {
		clone[key] = value
	}
	m.mu.RUnlock()

	return clone
}

// m.Visit(MyKey, func(value MyValue) { do something with the value if found }) == visitor function was compatible and executed successfully.
// Note that it doesn't lock until the function ends, this is because the function may access other memory's function which locks and that will result on a deadlock.
func (m *Memory) Visit(key uint8, visitorFunc interface{}) bool {
	if visitorFunc == nil || len(m.tmp) == 0 {
		return false
	}

	value, ok := m.Get(key)
	if !ok || value == nil { // we can't work with a nil value, even if it's there (it's possible if dev do it), so return with false.
		return false
	}

	fn := reflect.Indirect(reflect.ValueOf(visitorFunc))
	if fn.Kind() == reflect.Interface {
		fn = fn.Elem()
	}

	if fn.Kind() != reflect.Func {
		return false
	}

	fnTyp := fn.Type()
	if fnTyp.NumIn() != 1 {
		return false
	}

	v := reflect.ValueOf(value)

	if fnTyp.In(0) != v.Type() {
		return false
	}

	fn.Call([]reflect.Value{v})
	return true
}

// Clear removes all the stored elements and returns the total length of the elements removed, safe for concurrent access.
func (m *Memory) Clear() int {
	n := len(m.tmp)
	if n == 0 {
		return 0
	}

	m.mu.Lock()
	for key := range m.tmp {
		delete(m.tmp, key)
	}
	m.mu.Unlock()

	return n
}

func makeMemory() *Memory {
	return &Memory{tmp: make(map[uint8]interface{})}
}

func GetMemory(cmd *cobra.Command) *Memory {
	if cmd == nil {
		return nil
	}

	app := Get(cmd)
	if app == nil ||
		// do not initialize if memory is nil,
		// package-level function with *cobra.Command as a receiver means that it should be used after the `Build/Run` state.
		app.Memory == nil {

		return nil
	}

	return app.Memory
}
