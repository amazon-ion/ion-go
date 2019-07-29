package ion

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"time"
)

// MarshalText marshals values to text ion.
func MarshalText(v interface{}) ([]byte, error) {
	buf := bytes.Buffer{}
	m := Encoder{
		w:        NewTextWriterOpts(&buf, OptQuietFinish),
		sortMaps: true,
	}

	if err := m.Encode(v); err != nil {
		return nil, err
	}
	if err := m.Finish(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// An Encoder writes Ion values to an output stream.
type Encoder struct {
	w        Writer
	sortMaps bool
}

// NewEncoder creates a new encoder.
func NewEncoder(w Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// NewTextEncoder creates a new Encoder that marshals text Ion to the given writer.
func NewTextEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:        NewTextWriter(w),
		sortMaps: true,
	}
}

// Encode marshals the given value to Ion, writing it to the underlying writer.
func (m *Encoder) Encode(v interface{}) error {
	return m.marshalValue(reflect.ValueOf(v))
}

// Finish finishes writing the current Ion datagram.
func (m *Encoder) Finish() error {
	return m.w.Finish()
}

func (m *Encoder) marshalValue(v reflect.Value) error {
	if !v.IsValid() {
		m.w.WriteNull()
		return nil
	}

	t := v.Type()
	switch t.Kind() {
	case reflect.Bool:
		m.w.WriteBool(v.Bool())
		return m.w.Err()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		m.w.WriteInt(v.Int())
		return m.w.Err()

	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		m.w.WriteInt(int64(v.Uint()))
		return m.w.Err()

	case reflect.Uint, reflect.Uint64, reflect.Uintptr:
		i := big.Int{}
		i.SetUint64(v.Uint())
		m.w.WriteBigInt(&i)
		return m.w.Err()

	case reflect.Float32, reflect.Float64:
		m.w.WriteFloat(v.Float())
		return m.w.Err()

	case reflect.String:
		m.w.WriteString(v.String())
		return m.w.Err()

	case reflect.Interface, reflect.Ptr:
		return m.marshalPtr(v)

	case reflect.Struct:
		return m.marshalStruct(v)

	case reflect.Map:
		return m.marshalMap(v)

	case reflect.Slice:
		return m.marshalSlice(v)

	case reflect.Array:
		return m.marshalArray(v)

	default:
		return fmt.Errorf("ion: unsupported type: %v", v.Type().String())
	}
}

func (m *Encoder) marshalPtr(v reflect.Value) error {
	if v.IsNil() {
		m.w.WriteNull()
		return m.w.Err()
	}
	return m.marshalValue(v.Elem())
}

func (m *Encoder) marshalMap(v reflect.Value) error {
	if v.IsNil() {
		m.w.WriteNull()
		return m.w.Err()
	}

	m.w.BeginStruct()

	keys := getKeys(v)
	if m.sortMaps {
		// We do this for text Ion because json.Marshal does, and it's useful for testing.
		// For binary Ion, skip it and write things in whatever order they come back from
		// the map.
		sort.Slice(keys, func(i, j int) bool { return keys[i].s < keys[j].s })
	}

	for _, key := range keys {
		m.w.FieldName(key.s)
		value := v.MapIndex(key.v)
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

func getKeys(v reflect.Value) []mapkey {
	keys := v.MapKeys()
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

	return res
}

func (m *Encoder) marshalSlice(v reflect.Value) error {
	if v.Type().Elem().Kind() == reflect.Uint8 {
		return m.marshalBlob(v)
	}

	if v.IsNil() {
		m.w.WriteNull()
		return m.w.Err()
	}

	return m.marshalArray(v)
}

func (m *Encoder) marshalBlob(v reflect.Value) error {
	if v.IsNil() {
		m.w.WriteNull()
	} else {
		m.w.WriteBlob(v.Bytes())
	}
	return m.w.Err()
}

func (m *Encoder) marshalArray(v reflect.Value) error {
	m.w.BeginList()

	for i := 0; i < v.Len(); i++ {
		if err := m.marshalValue(v.Index(i)); err != nil {
			return err
		}
	}

	m.w.EndList()
	return m.w.Err()
}

var decimalType = reflect.TypeOf(Decimal{})

func (m *Encoder) marshalStruct(v reflect.Value) error {
	t := v.Type()
	if t == timeType {
		return m.marshalTime(v)
	}
	if t == decimalType {
		return m.marshalDecimal(v)
	}

	fields := fieldsFor(v.Type())

	m.w.BeginStruct()

FieldLoop:
	for i := range fields {
		f := &fields[i]

		fv := v
		for _, i := range f.path {
			if fv.Kind() == reflect.Ptr {
				if fv.IsNil() {
					continue FieldLoop
				}
				fv = fv.Elem()
			}
			fv = fv.Field(i)
		}

		if f.omitEmpty && emptyValue(fv) {
			continue
		}

		m.w.FieldName(f.name)
		if err := m.marshalValue(fv); err != nil {
			return err
		}
	}

	m.w.EndStruct()
	return m.w.Err()
}

func (m *Encoder) marshalTime(v reflect.Value) error {
	t := v.Interface().(time.Time)
	m.w.WriteTimestamp(t)
	return m.w.Err()
}

func (m *Encoder) marshalDecimal(v reflect.Value) error {
	d := v.Addr().Interface().(*Decimal)
	m.w.WriteDecimal(d)
	return m.w.Err()
}

func emptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

type field struct {
	name      string
	typ       reflect.Type
	path      []int
	omitEmpty bool
}

type fielder struct {
	fields []field
	index  map[string]bool
}

func fieldsFor(t reflect.Type) []field {
	fldr := fielder{index: map[string]bool{}}
	fldr.inspect(t, nil)
	return fldr.fields
}

func (f *fielder) inspect(t reflect.Type, path []int) {
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !visible(&sf) {
			// Skip non-visible fields.
			continue
		}

		tag := sf.Tag.Get("json")
		if tag == "-" {
			// Skip fields that are explicitly hidden by tag.
			continue
		}
		name, opts := parseTag(tag)

		newpath := make([]int, len(path)+1)
		copy(newpath, path)
		newpath[len(path)] = i

		ft := sf.Type
		if ft.Name() == "" && ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		if name == "" && sf.Anonymous && ft.Kind() == reflect.Struct {
			// Dig in to the embedded struct.
			f.inspect(ft, newpath)
		} else {
			// Add this named field.
			if name == "" {
				name = sf.Name
			}

			if f.index[name] {
				panic(fmt.Sprintf("too many fields named %v", name))
			}
			f.index[name] = true

			f.fields = append(f.fields, field{
				name:      name,
				typ:       ft,
				path:      newpath,
				omitEmpty: omitEmpty(opts),
			})
		}
	}
}

func visible(sf *reflect.StructField) bool {
	exported := sf.PkgPath == ""
	if sf.Anonymous {
		t := sf.Type
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct {
			// Fields of embedded structs are visible even if the struct type itself is not.
			return true
		}
	}
	return exported
}

func parseTag(tag string) (string, string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		// Ignore additional JSON options, at least for now.
		return tag[:idx], tag[idx+1:]
	}
	return tag, ""
}

func omitEmpty(opts string) bool {
	for opts != "" {
		var o string

		i := strings.Index(opts, ",")
		if i >= 0 {
			o, opts = opts[:i], opts[i+1:]
		} else {
			o, opts = opts, ""
		}

		if o == "omitempty" {
			return true
		}
	}
	return false
}
