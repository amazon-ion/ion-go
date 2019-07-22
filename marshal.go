package ion

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
)

type marshaller struct {
	buf *bytes.Buffer
	w   Writer
}

func newTextMarshaller() *marshaller {
	buf := &bytes.Buffer{}
	return &marshaller{
		buf: buf,
		w:   NewTextWriterOpts(buf, OptQuietFinish),
	}
}

// MarshalText marshals values to text ion.
func MarshalText(v interface{}) ([]byte, error) {
	m := newTextMarshaller()
	if err := m.marshal(v); err != nil {
		return nil, err
	}
	return m.buf.Bytes(), nil
}

func (m *marshaller) marshal(v interface{}) error {
	if err := m.marshalValue(reflect.ValueOf(v)); err != nil {
		return err
	}
	return m.w.Finish()
}

func (m *marshaller) marshalValue(r reflect.Value) error {
	if !r.IsValid() {
		m.w.WriteNull()
		return nil
	}

	t := r.Type()
	switch t.Kind() {
	case reflect.Bool:
		m.w.WriteBool(r.Bool())
		return m.w.Err()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		m.w.WriteInt(r.Int())
		return m.w.Err()

	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		m.w.WriteInt(int64(r.Uint()))
		return m.w.Err()

	case reflect.Uint, reflect.Uint64, reflect.Uintptr:
		i := big.Int{}
		i.SetUint64(r.Uint())
		m.w.WriteBigInt(&i)
		return m.w.Err()

	case reflect.Float32, reflect.Float64:
		m.w.WriteFloat(r.Float())
		return m.w.Err()

		// TODO: Decimal
		// TODO: Time

	case reflect.String:
		m.w.WriteString(r.String())
		return m.w.Err()

	case reflect.Interface, reflect.Ptr:
		return m.marshalInterfaceOrPtr(r)

	case reflect.Struct:
		return m.marshalStruct(r)

	case reflect.Map:
		return m.marshalMap(r)

	case reflect.Slice:
		return m.marshalSlice(r)

	case reflect.Array:
		return m.marshalArray(r)

	default:
		return fmt.Errorf("unsupported type %v", r.Type())
	}
}

func (m *marshaller) marshalInterfaceOrPtr(r reflect.Value) error {
	if r.IsNil() {
		m.w.WriteNull()
		return m.w.Err()
	}
	return m.marshalValue(r.Elem())
}

func (m *marshaller) marshalMap(r reflect.Value) error {
	if r.IsNil() {
		m.w.WriteNull()
		return m.w.Err()
	}

	m.w.BeginStruct()

	for _, key := range getKeys(r) {
		m.w.FieldName(key.s)
		value := r.MapIndex(key.v)
		if err := m.marshalValue(value); err != nil {
			return err
		}
	}

	m.w.EndStruct()
	return m.w.Err()
}

type mapkey struct {
	v reflect.Value
	s string
}

func getKeys(r reflect.Value) []mapkey {
	keys := r.MapKeys()
	res := make([]mapkey, len(keys))

	for i, key := range keys {
		// TODO: Handle other kinds of keys.
		if key.Kind() != reflect.String {
			panic("unexpected map key type")
		}
		res[i] = mapkey{
			v: key,
			s: key.String(),
		}
	}

	sort.Slice(res, func(i, j int) bool { return res[i].s < res[j].s })

	return res
}

func (m *marshaller) marshalSlice(r reflect.Value) error {
	if r.Type().Elem().Kind() == reflect.Uint8 {
		return m.marshalBlob(r)
	}

	if r.IsNil() {
		m.w.WriteNull()
		return m.w.Err()
	}

	return m.marshalArray(r)
}

func (m *marshaller) marshalBlob(r reflect.Value) error {
	if r.IsNil() {
		m.w.WriteNull()
	} else {
		m.w.WriteBlob(r.Bytes())
	}
	return m.w.Err()
}

func (m *marshaller) marshalArray(r reflect.Value) error {
	m.w.BeginList()

	for i := 0; i < r.Len(); i++ {
		if err := m.marshalValue(r.Index(i)); err != nil {
			return err
		}
	}

	m.w.EndList()
	return m.w.Err()
}

func (m *marshaller) marshalStruct(r reflect.Value) error {
	m.w.BeginStruct()

	fields := getFields(r.Type())
	for i := range fields {
		f := &fields[i]
		m.w.FieldName(f.name)
		if err := m.marshalValue(r.Field(f.index)); err != nil {
			return err
		}
	}

	m.w.EndStruct()
	return m.w.Err()
}

type field struct {
	name  string
	typ   reflect.Type
	index int
}

func getFields(t reflect.Type) []field {
	fields := []field{}

	// current := []reflect.Type{}
	// next := []reflect.Type{t}
	// visited := map[reflect.Type]bool{}

	// for len(next) > 0 {
	// 	current, next = next, current[:0]
	// 	for _, c := range current {
	// 		if visited[c] {
	// 			continue
	// 		}
	// 		visited[c] = true

	c := t

	for i := 0; i < c.NumField(); i++ {
		f := c.Field(i)

		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name := parseTag(tag)

		fType := f.Type
		if fType.Name() == "" && fType.Kind() == reflect.Ptr {
			fType = fType.Elem()
		}

		if name == "" && f.Anonymous && fType.Kind() == reflect.Struct {
			// next = append(next, fType)
			continue
		}

		if name == "" {
			name = f.Name
		}

		fields = append(fields, field{
			name:  name,
			typ:   fType,
			index: i,
		})
	}

	// 	}
	// }

	sort.Slice(fields, func(i, j int) bool { return fields[i].index < fields[j].index })

	return fields
}

func parseTag(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		// Ignore additional JSON options, at least for now.
		return tag[:idx]
	}
	return tag
}
