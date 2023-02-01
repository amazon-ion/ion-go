/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package ion

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

var (
	// ErrNoInput is returned when there is no input to decode
	ErrNoInput = errors.New("ion: no input to decode")
)

var typesAcceptableKinds = map[Type][]reflect.Kind{
	IntType: {reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint, reflect.Uint64, reflect.Uintptr},
	FloatType: {reflect.Float32, reflect.Float64},
	BlobType:  {reflect.Slice, reflect.Array},
	ListType:  {reflect.Slice, reflect.Array},
}

// Unmarshaler is the interface implemented by types that can unmarshal themselves to Ion.
type Unmarshaler interface {
	UnmarshalIon(r Reader) error
}

// Unmarshal unmarshals Ion data to the given object.
//
// User must pass the proper object type to the unmarshalled Ion data.
// Below is the mapping between Go native type and Ion types. e.g.,
//
//	    boolBytes := []byte{0xE0, 0x01, 0x00, 0xEA, 0x11}
//	    var boolVal bool
//	    err := Unmarshal(boolBytes, &boolVal)
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	    fmt.Println(boolVal) // prints out: true
//
//		err = UnmarshalString("true", &boolVal)
//		if err != nil {
//		    t.Fatal(err)
//		}
//		fmt.Println(boolVal) // prints out: true
//
// To unmarshal an Ion value with annotations, the object passed to Unmarshal
// must be a Go struct with exactly two fields, where one field's type
// is in accordance with the Ion type which needs to be unmarshalled (list
// of mapping between Go native types and Ion types below); and the other
// field must be of type []string and tagged as `ion:",annotations"`.
//
//	  type foo struct {
//	      Value   int    // or interface{}
//	      AnyName []string `ion:",annotations"`
//	  }
//
//	  var val foo
//	  err := UnmarshalString("age::10", &val)
//	  if err != nil {
//	      t.Fatal(err)
//	  }
//	  fmt.Println(val) // prints out: {10 [age]}
//
//	  Go native type                                  Ion Type
//	--------------------------                     ---------------
//	  nil/interface{}                                 null
//	  bool/interface{}                                bool
//	  Any ints/uints/big.Int/interface{}              int
//	  float32/float64/interface{}                     float
//	  ion.Decimal/interface{}                         decimal
//	  ion.Timestamp/interface{}                       timestamp
//	  string/interface{}                              symbol
//	  string/interface{}                              string
//	  []byte/[]interface{}{}                          clob
//	  []byte/[]interface{}{}                          blob
//	  []interface{}{}                                 list
//	  []interface{}{}                                 sexp
//	  map[string]interface{}{}/struct/interface{}     struct
func Unmarshal(data []byte, v interface{}, ssts ...SharedSymbolTable) error {
	catalog := NewCatalog(ssts...)
	return NewDecoder(NewReaderCat(bytes.NewReader(data), catalog)).DecodeTo(v)
}

// UnmarshalString unmarshals Ion data from a string to the given object.
func UnmarshalString(data string, v interface{}) error {
	return Unmarshal([]byte(data), v)
}

// UnmarshalFrom unmarshal Ion data from a reader to the given object.
func UnmarshalFrom(r Reader, v interface{}) error {
	d := Decoder{
		r: r,
	}
	return d.DecodeTo(v)
}

// A Decoder decodes go values from an Ion reader.
type Decoder struct {
	r Reader
}

// NewDecoder creates a new decoder.
func NewDecoder(r Reader) *Decoder {
	return &Decoder{
		r: r,
	}
}

// NewTextDecoder creates a new text decoder. Well, a decoder that uses a reader with
// no shared symbol tables, it'll work to read binary too if the binary doesn't reference
// any shared symbol tables.
func NewTextDecoder(in io.Reader) *Decoder {
	return NewDecoder(NewReader(in))
}

// Decode decodes a value from the underlying Ion reader without any expectations
// about what it's going to get. Structs become map[string]interface{}s, Lists and
// Sexps become []interface{}s.
func (d *Decoder) Decode() (interface{}, error) {
	if !d.r.Next() {
		if d.r.Err() != nil {
			return nil, d.r.Err()
		}
		return nil, ErrNoInput
	}

	return d.decode()
}

// Helper form of Decode for when you've already called Next.
func (d *Decoder) decode() (interface{}, error) {
	if d.r.IsNull() {
		return nil, nil
	}

	switch d.r.Type() {
	case BoolType:
		val, err := d.r.BoolValue()
		if err != nil {
			return nil, err
		}
		return *val, err

	case IntType:
		return d.decodeInt()

	case FloatType:
		return d.r.FloatValue()

	case DecimalType:
		return d.r.DecimalValue()

	case TimestampType:
		return d.r.TimestampValue()

	case StringType:
		return d.r.StringValue()

	case SymbolType:
		return d.r.SymbolValue()

	case BlobType, ClobType:
		return d.r.ByteValue()

	case StructType:
		return d.decodeMap()

	case ListType, SexpType:
		return d.decodeSlice()

	default:
		panic("cannot recognize the IonType")
	}
}

func (d *Decoder) decodeInt() (interface{}, error) {
	size, err := d.r.IntSize()
	if err != nil {
		return nil, err
	}

	switch size {
	case NullInt:
		return nil, nil
	case Int32:
		val, err := d.r.IntValue()
		if err != nil {
			return nil, err
		}
		return *val, err
	case Int64:
		val, err := d.r.Int64Value()
		if err != nil {
			return nil, err
		}
		return *val, err
	default:
		return d.r.BigIntValue()
	}
}

// DecodeMap decodes an Ion struct to a go map.
func (d *Decoder) decodeMap() (map[string]interface{}, error) {
	if err := d.r.StepIn(); err != nil {
		return nil, err
	}

	result := map[string]interface{}{}

	for d.r.Next() {
		fieldName, err := d.r.FieldName()
		if err != nil {
			return nil, err
		}
		if fieldName != nil && fieldName.Text != nil {
			name := fieldName.Text
			value, err := d.decode()
			if err != nil {
				return nil, err
			}
			result[*name] = value
		}
	}

	if err := d.r.StepOut(); err != nil {
		return nil, err
	}

	return result, nil
}

// DecodeSlice decodes an Ion list or sexp to a go slice.
func (d *Decoder) decodeSlice() ([]interface{}, error) {
	if err := d.r.StepIn(); err != nil {
		return nil, err
	}

	var result []interface{}

	for d.r.Next() {
		value, err := d.decode()
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}

	if err := d.r.StepOut(); err != nil {
		return nil, err
	}

	return result, nil
}

// DecodeTo decodes an Ion value from the underlying Ion reader into the
// value provided.
func (d *Decoder) DecodeTo(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("ion: v must be a pointer")
	}
	if rv.IsNil() {
		return errors.New("ion: v must not be nil")
	}

	if !d.r.Next() {
		if d.r.Err() != nil {
			return d.r.Err()
		}
		return ErrNoInput
	}

	return d.decodeTo(rv)
}

var unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()

func (d *Decoder) decodeTo(v reflect.Value) error {
	if !v.IsValid() {
		// Don't actually have anywhere to put this value; skip it.
		return nil
	}

	isNull := d.r.IsNull()
	v = indirect(v, isNull)
	if isNull {
		v.Set(reflect.Zero(v.Type()))
		if v.Type().Kind() == reflect.Struct {
			return d.attachAnnotations(v)
		}
		return nil
	}

	t := v.Type()
	if t.Kind() != reflect.Ptr && v.CanAddr() && reflect.PtrTo(t).Implements(unmarshalerType) {
		return v.Addr().Interface().(Unmarshaler).UnmarshalIon(d.r)
	}

	switch d.r.Type() {
	case BoolType:
		return d.decodeBoolTo(v)

	case IntType:
		return d.decodeIntTo(v)

	case FloatType:
		return d.decodeFloatTo(v)

	case DecimalType:
		return d.decodeDecimalTo(v)

	case TimestampType:
		return d.decodeTimestampTo(v)

	case StringType:
		return d.decodeStringTo(v)

	case SymbolType:
		return d.decodeSymbolTo(v)

	case BlobType, ClobType:
		return d.decodeLobTo(v)

	case StructType:
		return d.decodeStructTo(v)

	case ListType, SexpType:
		return d.decodeSliceTo(v)

	default:
		panic("cannot recognize the IonType")
	}
}

func (d *Decoder) decodeBoolTo(v reflect.Value) error {
	val, err := d.r.BoolValue()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Bool:
		// Too easy.
		v.SetBool(*val)
		return nil

	case reflect.Struct:
		return d.decodeToStructWithAnnotation(v, reflect.Bool)

	case reflect.Interface:
		if v.NumMethod() == 0 {
			if val != nil {
				v.Set(reflect.ValueOf(*val))
			}
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode bool to %v", v.Type().String())
}

func (d *Decoder) decodeIntTo(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := d.r.Int64Value()
		if err != nil {
			return err
		}
		if v.OverflowInt(*val) {
			return fmt.Errorf("ion: value %v won't fit in type %v", *val, v.Type().String())
		}
		v.SetInt(*val)
		return nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		val, err := d.r.Int64Value()
		if err != nil {
			return err
		}
		if *val < 0 || v.OverflowUint(uint64(*val)) {
			return fmt.Errorf("ion: value %v won't fit in type %v", *val, v.Type().String())
		}
		v.SetUint(uint64(*val))
		return nil

	case reflect.Uint, reflect.Uint64, reflect.Uintptr:
		val, err := d.r.BigIntValue()
		if err != nil {
			return err
		}
		if !val.IsUint64() {
			return fmt.Errorf("ion: value %v won't fit in type %v", val, v.Type().String())
		}
		uiv := val.Uint64()
		if v.OverflowUint(uiv) {
			return fmt.Errorf("ion: value %v won't fit in type %v", val, v.Type().String())
		}
		v.SetUint(uiv)
		return nil

	case reflect.Struct:
		if v.Type() == bigIntType {
			val, err := d.r.BigIntValue()
			if err != nil {
				return err
			}
			if val != nil {
				v.Set(reflect.ValueOf(*val))
			}
			return nil
		}
		return d.decodeToStructWithAnnotation(v, typesAcceptableKinds[IntType]...)

	case reflect.Interface:
		if v.NumMethod() == 0 {
			val, err := d.decodeInt()
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(val))
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode int to %v", v.Type().String())
}

func (d *Decoder) decodeFloatTo(v reflect.Value) error {
	val, err := d.r.FloatValue()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		if v.OverflowFloat(*val) {
			return fmt.Errorf("ion: value %v won't fit in type %v", *val, v.Type().String())
		}
		v.SetFloat(*val)
		return nil

	case reflect.Struct:
		if v.Type() == decimalType {
			flt := strconv.FormatFloat(*val, 'g', -1, 64)
			dec, err := ParseDecimal(strings.Replace(flt, "e", "d", 1))
			if err != nil {
				return err
			}
			if val != nil {
				v.Set(reflect.ValueOf(*dec))
			}
			return d.attachAnnotations(v)
		}
		return d.decodeToStructWithAnnotation(v, typesAcceptableKinds[FloatType]...)

	case reflect.Interface:
		if v.NumMethod() == 0 {
			if val != nil {
				v.Set(reflect.ValueOf(*val))
			}
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode float to %v", v.Type().String())
}

func (d *Decoder) decodeDecimalTo(v reflect.Value) error {
	val, err := d.r.DecimalValue()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Struct:
		if v.Type() == decimalType {
			if val != nil {
				v.Set(reflect.ValueOf(*val))
			}
			return d.attachAnnotations(v)
		}
		return d.decodeToStructWithAnnotation(v, decimalType.Kind())

	case reflect.Interface:
		if v.NumMethod() == 0 {
			if val != nil {
				v.Set(reflect.ValueOf(*val))
			}
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode decimal to %v", v.Type().String())
}

func (d *Decoder) decodeTimestampTo(v reflect.Value) error {
	val, err := d.r.TimestampValue()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Struct:
		switch v.Type() {
		case timestampType:
			if val != nil {
				v.Set(reflect.ValueOf(*val))
			}
			return d.attachAnnotations(v)
		case nativeTimeType:
			if val != nil {
				v.Set(reflect.ValueOf((*val).dateTime))
			}
			return d.attachAnnotations(v)
		default:
			return d.decodeToStructWithAnnotation(v, timestampType.Kind())
		}

	case reflect.Interface:
		if v.NumMethod() == 0 {
			if val != nil {
				v.Set(reflect.ValueOf(*val))
			}
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode timestamp to %v", v.Type().String())
}

func (d *Decoder) decodeSymbolTo(v reflect.Value) error {
	val, err := d.r.SymbolValue()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.String:
		if val != nil {
			v.SetString(*val.Text)
		}
		return nil

	case reflect.Struct:
		if v.Type() == symbolType {
			v.Set(reflect.ValueOf(val))
			return d.attachAnnotations(v)
		}
		return d.decodeToStructWithAnnotation(v, symbolType.Kind())

	case reflect.Interface:
		if v.NumMethod() == 0 {
			v.Set(reflect.ValueOf(val))
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode symbol to %v", v.Type().String())
}

func (d *Decoder) decodeStringTo(v reflect.Value) error {
	val, err := d.r.StringValue()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.String:
		if val != nil {
			v.SetString(*val)
		}
		return nil

	case reflect.Struct:
		return d.decodeToStructWithAnnotation(v, reflect.String)

	case reflect.Interface:
		if v.NumMethod() == 0 {
			v.Set(reflect.ValueOf(*val))
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode string to %v", v.Type().String())
}

func (d *Decoder) decodeLobTo(v reflect.Value) error {
	val, err := d.r.ByteValue()
	if err != nil {
		return err
	}

	switch v.Kind() {
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			v.SetBytes(val)
			return nil
		}

	case reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			i := reflect.Copy(v, reflect.ValueOf(val))
			for ; i < v.Len(); i++ {
				v.Index(i).SetUint(0)
			}
			return nil
		}

	case reflect.Struct:
		return d.decodeToStructWithAnnotation(v, typesAcceptableKinds[BlobType]...)

	case reflect.Interface:
		if v.NumMethod() == 0 {
			v.Set(reflect.ValueOf(val))
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode lob to %v", v.Type().String())
}

func (d *Decoder) decodeStructTo(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		return d.decodeStructToStruct(v)

	case reflect.Map:
		return d.decodeStructToMap(v)

	case reflect.Interface:
		if v.NumMethod() == 0 {
			m, err := d.decodeMap()
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(m))
			return nil
		}
	}
	return fmt.Errorf("ion: cannot decode struct to %v", v.Type().String())
}

func (d *Decoder) decodeStructToStruct(v reflect.Value) error {
	fields := fieldsFor(v.Type())

	err := d.attachAnnotations(v)
	if err != nil {
		return err
	}

	if err := d.r.StepIn(); err != nil {
		return err
	}

	for d.r.Next() {
		fieldName, err := d.r.FieldName()
		if err != nil {
			return err
		}
		if fieldName != nil && fieldName.Text != nil {
			field := findField(fields, *fieldName.Text)
			if field != nil {
				subv, err := findSubvalue(v, field)
				if err != nil {
					return err
				}

				if err := d.decodeTo(subv); err != nil {
					return err
				}
			}
		}
	}

	return d.r.StepOut()
}

func findField(fields []field, name string) *field {
	var f *field
	for i := range fields {
		ff := &fields[i]
		if ff.name == name {
			return ff
		}
		if f == nil && strings.EqualFold(ff.name, name) {
			f = ff
		}
	}
	return f
}

func findSubvalue(v reflect.Value, f *field) (reflect.Value, error) {
	for _, i := range f.path {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				if !v.CanSet() {
					return reflect.Value{}, fmt.Errorf("ion: cannot set embedded pointer to unexported struct: %v", v.Type().Elem())
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v, nil
}

func (d *Decoder) decodeStructToMap(v reflect.Value) error {
	t := v.Type()
	switch t.Key().Kind() {
	case reflect.String:
	default:
		return fmt.Errorf("ion: cannot decode struct to %v", t.String())
	}

	if v.IsNil() {
		v.Set(reflect.MakeMap(t))
	}

	if err := d.r.StepIn(); err != nil {
		return err
	}

	for d.r.Next() {
		subv := reflect.New(t.Elem()).Elem()

		fieldName, err := d.r.FieldName()
		if err != nil {
			return err
		}

		if fieldName != nil && fieldName.Text != nil {
			fieldNameText := *fieldName.Text

			if err := d.decodeTo(subv); err != nil {
				return err
			}

			var kv reflect.Value
			switch t.Key().Kind() {
			case reflect.String:
				kv = reflect.ValueOf(fieldNameText)
			default:
				panic(fmt.Sprintf("the key for map to hold field name must be of type string. Found: %v", t.Key().Kind().String()))
			}

			if kv.IsValid() {
				v.SetMapIndex(kv, subv)
			}
		}
	}

	return d.r.StepOut()
}

func (d *Decoder) decodeSliceTo(v reflect.Value) error {
	k := v.Kind()

	// If all we know is we need an interface{}, decode an []interface{} with
	// types based on the Ion value stream.
	if k == reflect.Interface && v.NumMethod() == 0 {
		s, err := d.decodeSlice()
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(s))
		return nil
	}

	if k == reflect.Struct {
		return d.decodeToStructWithAnnotation(v, typesAcceptableKinds[ListType]...)
	}

	// Only other valid targets are arrays and slices.
	if k != reflect.Array && k != reflect.Slice {
		return fmt.Errorf("ion: cannot unmarshal slice to %v", v.Type().String())
	}

	if err := d.r.StepIn(); err != nil {
		return err
	}

	i := 0

	// Decode values into the array or slice.
	for d.r.Next() {
		if v.Kind() == reflect.Slice {
			// If it's a slice, we can grow it as needed.
			if i >= v.Cap() {
				newcap := v.Cap() + v.Cap()/2
				if newcap < 4 {
					newcap = 4
				}
				newv := reflect.MakeSlice(v.Type(), v.Len(), newcap)
				reflect.Copy(newv, v)
				v.Set(newv)
			}
			if i >= v.Len() {
				v.SetLen(i + 1)
			}
		}

		if i < v.Len() {
			if err := d.decodeTo(v.Index(i)); err != nil {
				return err
			}
		}

		i++
	}

	if err := d.r.StepOut(); err != nil {
		return err
	}

	if i < v.Len() {
		if v.Kind() == reflect.Array {
			// Zero out any additional values.
			z := reflect.Zero(v.Type().Elem())
			for ; i < v.Len(); i++ {
				v.Index(i).Set(z)
			}
		} else {
			v.SetLen(i)
		}
	}

	return nil
}

// Dig in through any pointers to find the actual underlying value that we want
// to set. If wantPtr is false, the algorithm terminates at a non-ptr value (e.g.,
// if passed an *int, it returns the int it points to, allocating such an int if the
// pointer is currently nil). If wantPtr is true, it terminates on a pointer to that
// value (allowing said pointer to be set to nil, generally).
func indirect(v reflect.Value, wantPtr bool) reflect.Value {
	for {
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (!wantPtr || e.Elem().Kind() == reflect.Ptr) {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && wantPtr && v.CanSet() {
			break
		}

		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		v = v.Elem()
	}

	return v
}

func (d *Decoder) decodeToStructWithAnnotation(v reflect.Value, valueAcceptableKinds ...reflect.Kind) error {
	validAnnotatableStruct, err := isValidAnnotatableStruct(v, valueAcceptableKinds)
	if err != nil {
		return err
	}
	if !validAnnotatableStruct {
		return fmt.Errorf("ion: cannot decode %v into %v", v, v.Type().String())
	}

	// populate annotations to the struct
	if err := d.attachAnnotations(v); err != nil {
		return err
	}

	fields := fieldsFor(v.Type())
	for _, field := range fields {
		if !field.annotations {
			field := findField(fields, field.name)
			if field != nil {
				subValue, err := findSubvalue(v, field)
				if err != nil {
					return err
				}

				if err := d.decodeTo(subValue); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (d *Decoder) attachAnnotations(v reflect.Value) error {
	fields := fieldsFor(v.Type())
	for _, field := range fields {
		if field.annotations {
			subValue, err := findSubvalue(v, &field)
			if err != nil {
				return err
			}

			annotations, err := d.r.Annotations()
			if err != nil {
				return err
			}
			subValue.Set(reflect.ValueOf(annotations))
			break
		}
	}
	return nil
}

// expected struct for decoding Ion values must have only 2 fields: one has `ion:",annotation"`
// tag, and the other field must be of a type where Ion value can be decoded to.
func isValidAnnotatableStruct(v reflect.Value, listofkinds []reflect.Kind) (bool, error) {
	fields := fieldsFor(v.Type())
	hasAnnotation := false
	acceptableValueType := false
	if len(fields) == 2 {
		for _, field := range fields {
			subValue, err := findSubvalue(v, &field)
			if err != nil {
				return false, err
			}
			if field.annotations {
				hasAnnotation = true
				continue
			}
			if isAcceptableKind(listofkinds, subValue.Type().Kind()) {
				acceptableValueType = true
			}
		}
	}
	return hasAnnotation && acceptableValueType, nil
}

func isAcceptableKind(valueAcceptableKinds []reflect.Kind, valueKind reflect.Kind) bool {
	for _, kind := range valueAcceptableKinds {
		if kind == valueKind || reflect.Interface == valueKind {
			return true
		}
	}
	return false
}
