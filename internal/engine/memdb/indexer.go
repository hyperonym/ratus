package memdb

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	"github.com/hyperonym/ratus"
)

// StateFieldIndex is used to extract a task state field from an object using
// reflection and builds an index on that field.
type StateFieldIndex struct {
	Field  string
	Filter ratus.TaskState
}

// FromObject implements the memdb.SingleIndexer interface.
func (i *StateFieldIndex) FromObject(obj interface{}) (bool, []byte, error) {

	// Extract and validate the value.
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v)
	v = v.FieldByName(i.Field)
	v = reflect.Indirect(v)
	if !v.IsValid() {
		return false, nil, nil
	}

	// Check the type of the value.
	if v.Kind() != reflect.Int32 {
		return false, nil, fmt.Errorf("field %q is of type %v; want a ratus.TaskState", i.Field, v.Kind())
	}

	// Check if the index should include the value.
	s := ratus.TaskState(v.Int())
	if s != i.Filter {
		return false, nil, nil
	}

	return true, []byte{uint8(s)}, nil
}

// FromArgs implements the memdb.Indexer interface.
func (i *StateFieldIndex) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}

	// Extract and validate the value.
	v := reflect.ValueOf(args[0])
	v = reflect.Indirect(v)
	if !v.IsValid() {
		return nil, fmt.Errorf("%#v is invalid", args[0])
	}

	// Check the type of the value.
	if v.Kind() != reflect.Int32 {
		return nil, fmt.Errorf("arg is of type %v; want a ratus.TaskState", v.Kind())
	}

	// Check if the index should include the value.
	s := ratus.TaskState(v.Int())
	if s != i.Filter {
		return nil, fmt.Errorf("value %v is not included in the index", s)
	}

	return []byte{uint8(s)}, nil
}

// TimeFieldIndex is used to extract a time field from an object using
// reflection and builds an index on that field.
type TimeFieldIndex struct {
	Field string
}

// FromObject implements the memdb.SingleIndexer interface.
func (i *TimeFieldIndex) FromObject(obj interface{}) (bool, []byte, error) {

	// Extract and validate the value.
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v)
	v = v.FieldByName(i.Field)
	v = reflect.Indirect(v)
	if !v.IsValid() {
		return false, nil, nil
	}

	// Check the type of the value.
	t, ok := v.Interface().(time.Time)
	if !ok {
		return false, nil, fmt.Errorf("field %q is of type %v; want a time.Time", i.Field, v.Kind())
	}

	return true, i.encodeInt64(t.UnixMilli()), nil
}

// FromArgs implements the memdb.Indexer interface.
func (i *TimeFieldIndex) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}

	// Extract and validate the value.
	v := reflect.ValueOf(args[0])
	v = reflect.Indirect(v)
	if !v.IsValid() {
		return nil, fmt.Errorf("%#v is invalid", args[0])
	}

	// Check the type of the value.
	t, ok := v.Interface().(time.Time)
	if !ok {
		return nil, fmt.Errorf("arg is of type %v; want a time.Time", v.Kind())
	}

	return i.encodeInt64(t.UnixMilli()), nil
}

func (i *TimeFieldIndex) encodeInt64(v int64) []byte {

	// This bit flips the sign bit on any sized signed twos-complement integer,
	// which when truncated to a uint of the same size will bias the value such
	// that the maximum negative int becomes 0, and the maximum positive int
	// becomes the maximum positive uint.
	v = v ^ int64(-1<<63)

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
