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
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalBool(t *testing.T) {
	test := func(str string, eval bool) {
		t.Run(str, func(t *testing.T) {
			var val bool
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}

	test("null", false)
	test("true", true)
	test("false", false)
}

func TestUnmarshalBoolPtr(t *testing.T) {
	test := func(str string, eval interface{}) {
		t.Run(str, func(t *testing.T) {
			var bval bool
			val := &bval
			require.NoError(t, UnmarshalString(str, &val))

			if eval == nil {
				assert.Nil(t, val)
			} else {
				switch {
				case val == nil:
					t.Errorf("expected %v, got <nil>", eval)
				case *val != eval.(bool):
					t.Errorf("expected %v, got %v", eval, *val)
				}
			}
		})
	}

	test("null", nil)
	test("null.bool", nil)
	test("false", false)
	test("true", true)
}

func TestUnmarshalInt(t *testing.T) {
	testInt8 := func(str string, eval int8) {
		t.Run(str, func(t *testing.T) {
			var val int8
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testInt8("null", 0)
	testInt8("0", 0)
	testInt8("0x7F", 0x7F)
	testInt8("-0x80", -0x80)

	testInt16 := func(str string, eval int16) {
		t.Run(str, func(t *testing.T) {
			var val int16
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testInt16("0x7F", 0x7F)
	testInt16("-0x80", -0x80)
	testInt16("0x7FFF", 0x7FFF)
	testInt16("-0x8000", -0x8000)

	testInt32 := func(str string, eval int32) {
		t.Run(str, func(t *testing.T) {
			var val int32
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testInt32("0x7FFF", 0x7FFF)
	testInt32("-0x8000", -0x8000)
	testInt32("0x7FFFFFFF", 0x7FFFFFFF)
	testInt32("-0x80000000", -0x80000000)

	testInt := func(str string, eval int) {
		t.Run(str, func(t *testing.T) {
			var val int
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testInt("0x7FFF", 0x7FFF)
	testInt("-0x8000", -0x8000)
	testInt("0x7FFFFFFF", 0x7FFFFFFF)
	testInt("-0x80000000", -0x80000000)

	testInt64 := func(str string, eval int64) {
		t.Run(str, func(t *testing.T) {
			var val int64
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testInt64("0x7FFFFFFF", 0x7FFFFFFF)
	testInt64("-0x80000000", -0x80000000)
	testInt64("0x7FFFFFFFFFFFFFFF", 0x7FFFFFFFFFFFFFFF)
	testInt64("-0x8000000000000000", -0x8000000000000000)
}

func TestUnmarshalUint(t *testing.T) {
	testUint8 := func(str string, eval uint8) {
		t.Run(str, func(t *testing.T) {
			var val uint8
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testUint8("null", 0)
	testUint8("0", 0)
	testUint8("0xFF", 0xFF)

	testUint16 := func(str string, eval uint16) {
		t.Run(str, func(t *testing.T) {
			var val uint16
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testUint16("0xFF", 0xFF)
	testUint16("0xFFFF", 0xFFFF)

	testUint32 := func(str string, eval uint32) {
		t.Run(str, func(t *testing.T) {
			var val uint32
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testUint32("0xFFFF", 0xFFFF)
	testUint32("0xFFFFFFFF", 0xFFFFFFFF)

	testUint := func(str string, eval uint) {
		t.Run(str, func(t *testing.T) {
			var val uint
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testUint("0xFFFF", 0xFFFF)
	testUint("0xFFFFFFFF", 0xFFFFFFFF)

	testUintptr := func(str string, eval uintptr) {
		t.Run(str, func(t *testing.T) {
			var val uintptr
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testUintptr("0xFFFF", 0xFFFF)
	testUintptr("0xFFFFFFFF", 0xFFFFFFFF)

	testUint64 := func(str string, eval uint64) {
		t.Run(str, func(t *testing.T) {
			var val uint64
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}
	testUint64("0xFFFFFFFF", 0xFFFFFFFF)
	testUint64("0xFFFFFFFFFFFFFFFF", 0xFFFFFFFFFFFFFFFF)
}

func TestUnmarshalBigInt(t *testing.T) {
	test := func(str string, eval *big.Int) {
		t.Run(str, func(t *testing.T) {
			var val big.Int
			require.NoError(t, UnmarshalString(str, &val))

			assert.True(t, val.Cmp(eval) == 0, "expected %v, got %v", eval, val)
		})
	}
	test("null", new(big.Int))
	test("1", new(big.Int).SetUint64(1))
	test("-0xFFFFFFFFFFFFFFFF", new(big.Int).Neg(new(big.Int).SetUint64(0xFFFFFFFFFFFFFFFF)))
}

func TestUnmarshalBinary(t *testing.T) {
	test := func(data []byte, val, eval interface{}) {
		t.Run(reflect.TypeOf(val).String(), func(t *testing.T) {
			require.NoError(t, Unmarshal(data, &val))

			res := false
			switch thisValue := val.(type) {
			case *Decimal:
				thisDecimal := ionDecimal{thisValue}
				res = thisDecimal.eq(ionDecimal{eval.(*Decimal)})
			case Timestamp:
				thisTime := ionTimestamp{thisValue}
				res = thisTime.eq(ionTimestamp{eval.(Timestamp)})
			case *SymbolToken:
				thisSymbol := ionSymbol{thisValue}
				res = thisSymbol.eq(ionSymbol{eval.(*SymbolToken)})
			case *interface{}:
				res = reflect.DeepEqual(*thisValue, eval)
			default:
				res = reflect.DeepEqual(val, eval)
			}
			assert.True(t, res, "expected %v, got %v", eval, val)
		})
	}

	var nullVal string
	nullBytes := prefixIVM([]byte{0x0F}) // null
	test(nullBytes, nullVal, nil)

	var boolVal bool
	boolBytes := prefixIVM([]byte{0x11}) // true
	test(boolBytes, boolVal, true)

	var intVal int16
	intBytes := prefixIVM([]byte{0x22, 0x7F, 0xFF}) // 32767
	test(intBytes, intVal, 32767)

	var uintVal uint16
	uintBytes := prefixIVM([]byte{0x32, 0x7F, 0xFF}) // -32767
	test(uintBytes, uintVal, -32767)

	var floatVal float32
	floatBytes := prefixIVM([]byte{0x44, 0x12, 0x12, 0x12, 0x12}) // 4.609175024471393E-28
	test(floatBytes, floatVal, 4.609175024471393e-28)

	var decimalVal Decimal
	decimalBytes := prefixIVM([]byte{0x51, 0xFF}) // 0d-63
	test(decimalBytes, decimalVal, MustParseDecimal("0d-63"))

	var timestampValue Timestamp
	timestampBytes := prefixIVM([]byte{0x67, 0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86}) // 0001-02-03T04:05:06Z
	dateTime := time.Date(1, time.Month(2), 3, 4, 5, 6, 0, time.FixedZone("fixed", 0))
	test(timestampBytes, timestampValue, NewTimestamp(dateTime, TimestampPrecisionSecond, TimezoneUTC))

	var symbolVal string
	symbolBytes := prefixIVM([]byte{0x71, 0x09}) // $9
	test(symbolBytes, symbolVal, &SymbolToken{Text: newString("$ion_shared_symbol_table"), LocalSID: 9})

	var stringVal string
	stringBytes := prefixIVM([]byte{0x83, 'a', 'b', 'c'}) // "abc"
	test(stringBytes, stringVal, newString("abc"))

	var clobVal []byte
	clobBytes := prefixIVM([]byte{0x92, 0x0A, 0x0B})
	test(clobBytes, clobVal, []byte{10, 11})

	var blobVal []byte
	blobBytes := prefixIVM([]byte{0xA3, 'a', 'b', 'c'})
	test(blobBytes, blobVal, []byte{97, 98, 99})

}

func TestUnmarshalStructBinary(t *testing.T) {
	test := func(data []byte, testName string, val, eval interface{}) {
		t.Run(testName, func(t *testing.T) {
			require.NoError(t, Unmarshal(data, &val))

			assert.Equal(t, eval, val)
		})
	}

	eval := map[string]interface{}{}
	eval["name"] = 2
	ionByteValue := prefixIVM([]byte{0xD3, 0x84, 0x21, 0x02}) // {name:2}

	var boolVal interface{}
	test(ionByteValue, "structToInterface", boolVal, eval) // unmarshal IonStruct to an interface

	test(ionByteValue, "structToMap", map[string]string{}, eval) // unmarshal IonStruct to a map

	ionByteValue = prefixIVM([]byte{0xE7, 0x81, 0x83, 0xD4, 0x87, 0xB2, 0x81, 'A', //$10=A
		0xD3, 0x8A, 0x21, 0x02}) // {A:2}
	type foo struct {
		Foo int `ion:"A"`
	}
	test(ionByteValue, "structToStruct", &foo{}, &foo{2}) // unmarshal IonStruct to a Go struct
}

func TestUnmarshalListSexpBinary(t *testing.T) {
	test := func(data []byte, testName string, val, eval interface{}) {
		t.Run("reflect.TypeOf(val).String()", func(t *testing.T) {
			require.NoError(t, Unmarshal(data, &val))

			assert.Equal(t, eval, val)
		})
	}

	ionByteValue := prefixIVM([]byte{0xB6, 0x21, 0x02, 0x21, 0x03, 0x21, 0x04}) // list : [2, 3, 4]

	test(ionByteValue, "listToInterface", &[]interface{}{}, &[]interface{}{2, 3, 4}) // unmarshal IonList to an interface
	test(ionByteValue, "listToSlice", &[]int{}, &[]int{2, 3, 4})                     // unmarshal IonList to Slice of int

	ionByteValue = prefixIVM([]byte{0xC6, 0x21, 0x02, 0x21, 0x03, 0x21, 0x04}) // sexp : (2 3 4)

	test(ionByteValue, "sexpToInterface", &[]interface{}{}, &[]interface{}{2, 3, 4}) // unmarshal IonSexp to an interface
	test(ionByteValue, "sexpToSlice", &[]int{}, &[]int{2, 3, 4})                     // unmarshal IonSexp to Slice of int
}

func TestDecodeFloat(t *testing.T) {
	test32 := func(str string, eval float32) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))

			var val float32
			require.NoError(t, d.DecodeTo(&val))

			assert.Equal(t, eval, val)
		})
	}
	test32("null", 0)
	test32("1e0", 1)
	test32("1e38", 1e38)
	test32("+inf", float32(math.Inf(1)))

	test64 := func(str string, eval float64) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))

			var val float64
			require.NoError(t, d.DecodeTo(&val))

			assert.Equal(t, eval, val)
		})
	}
	test64("1e0", 1)
	test64("1e308", 1e308)
	test64("+inf", math.Inf(1))
}

func TestDecodeDecimal(t *testing.T) {
	test := func(str string, eval *Decimal) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))

			var val *Decimal
			require.NoError(t, d.DecodeTo(&val))

			assert.True(t, val.Equal(eval), "expected %v, got %v", eval, val)
		})
	}

	test("1e10", MustParseDecimal("1d10"))
	test("1.20", MustParseDecimal("1.20"))
}

func TestDecodeTimestampTo(t *testing.T) {
	test := func(str string, eval Timestamp) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))

			var val Timestamp
			require.NoError(t, d.DecodeTo(&val))

			assert.True(t, val.Equal(eval), "expected %v, got %v", eval, val)
		})
	}
	test("null", Timestamp{})
	test("2020T", NewDateTimestamp(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), TimestampPrecisionYear))
}

func TestDecodeStringTo(t *testing.T) {
	test := func(str string, eval string) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))

			var val string
			require.NoError(t, d.DecodeTo(&val))

			assert.Equal(t, eval, val)
		})
	}

	test("null", "")
	test("hello", "hello")
	test("\"hello\"", "hello")
}

func TestDecodeLobTo(t *testing.T) {
	testSlice := func(str string, eval []byte) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))

			var val []byte
			require.NoError(t, d.DecodeTo(&val))

			assert.True(t, bytes.Equal(val, eval), "expected %v, got %v", eval, val)
		})
	}
	testSlice("null", nil)
	testSlice("{{}}", []byte{})
	testSlice("{{aGVsbG8=}}", []byte("hello"))
	testSlice("{{'''hello'''}}", []byte("hello"))

	testArray := func(str string, eval []byte) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))

			var val [8]byte
			require.NoError(t, d.DecodeTo(&val))

			assert.True(t, bytes.Equal(val[:], eval), "expected %v, got %v", eval, val)
		})
	}
	testArray("null", make([]byte, 8))
	testArray("{{aGVsbG8=}}", append([]byte("hello"), []byte{0, 0, 0}...))
}

func TestDecodeStructTo(t *testing.T) {
	test := func(str string, val, eval interface{}) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))
			require.NoError(t, d.DecodeTo(val))

			assert.Equal(t, eval, val)
		})
	}

	type foo struct {
		Foo string
		Baz int `ion:"bar"`
	}

	test("{}", &struct{}{}, &struct{}{})
	test("{bogus:(ignore me)}", &foo{}, &foo{})
	test("{foo:bar}", &foo{}, &foo{"bar", 0})
	test("{bar:42}", &foo{}, &foo{"", 42})
	test("{foo:bar,bar:42,bogus:(ignore me)}", &foo{}, &foo{"bar", 42})

	test("{}", &map[string]string{}, &map[string]string{})
	test("{foo:bar}", &map[string]string{}, &map[string]string{"foo": "bar"})
	test("{a:4,b:2}", &map[string]int{}, &map[string]int{"a": 4, "b": 2})
}

func TestDecodeListTo(t *testing.T) {
	test := func(str string, val, eval interface{}) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderString(str))
			require.NoError(t, d.DecodeTo(val))

			assert.Equal(t, eval, val)
		})
	}

	f := false
	pf := &f
	ppf := &pf

	test("[]", &[]bool{}, &[]bool{})
	test("[]", &[]bool{true}, &[]bool{})

	test("[false]", &[]bool{}, &[]bool{false})
	test("[false]", &[]*bool{}, &[]*bool{pf})
	test("[false,false]", &[]**bool{}, &[]**bool{ppf, ppf})

	test("[true,false]", &[]interface{}{}, &[]interface{}{true, false})

	var i interface{}
	var ei interface{} = []interface{}{true, false}
	test("[true,false]", &i, &ei)
}

func TestDecode(t *testing.T) {
	test := func(data string, eval interface{}) {
		t.Run(data, func(t *testing.T) {
			d := NewDecoder(NewReaderString(data))
			val, err := d.Decode()
			require.NoError(t, err)

			res := false
			switch thisValue := val.(type) {
			case *float64:
				res = cmpFloats(*thisValue, eval)
			case *Timestamp:
				res = cmpTimestamps(*thisValue, eval)
			default:
				res = reflect.DeepEqual(val, eval)
			}
			assert.True(t, res, "expected %v, got %v", eval, val)
		})
	}

	test("null", nil)
	test("null.null", nil)

	test("null.bool", nil)
	test("true", true)
	test("false", false)

	test("null.int", nil)
	test("0", 0)
	test("2147483647", math.MaxInt32)
	test("-2147483648", math.MinInt32)
	test("2147483648", int64(math.MaxInt32)+1)
	test("-2147483649", int64(math.MinInt32)-1)
	test("9223372036854775808", new(big.Int).SetUint64(math.MaxInt64+1))

	test("0e0", 0.0)
	test("1e100", 1e100)

	test("0.", MustParseDecimal("0."))

	test("2020T", NewDateTimestamp(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), TimestampPrecisionYear))

	test("hello", newSimpleSymbolTokenPtr("hello"))
	test("\"hello\"", newString("hello"))

	test("null.blob", nil)
	test("{{}}", []byte{})
	test("{{aGVsbG8=}}", []byte("hello"))

	test("null.clob", nil)
	test("{{''''''}}", []byte{})
	test("{{'''hello'''}}", []byte("hello"))

	test("null.struct", nil)
	test("{}", map[string]interface{}{})
	test("{a:1,b:two}", map[string]interface{}{
		"a": 1,
		"b": newSimpleSymbolTokenPtr("two"),
	})

	test("null.list", nil)
	test("[1, two]", []interface{}{1, newSimpleSymbolTokenPtr("two")})

	test("null.sexp", nil)
	test("(1 + two)", []interface{}{1, newSimpleSymbolTokenPtr("+"), newSimpleSymbolTokenPtr("two")})

	var result []interface{}
	test("()", result)
	test("[]", result)
}

func TestDecodeLotsOfInts(t *testing.T) {
	// Regression test for https://github.com/amzn/ion-go/issues/53
	buf := bytes.Buffer{}
	w := NewBinaryWriter(&buf)
	for i := 0; i < 512; i++ {
		assert.NoError(t, w.WriteInt(1570737066801085))
	}
	assert.NoError(t, w.Finish())
	bs := buf.Bytes()

	// The binary reader wraps a bufio.Reader with an internal 4096-byte
	// buffer. 4 bytes of BVM plus 511 x 8-byte integers (1 byte of tag +
	// 7 bytes of data) leaves 4 bytes left in the buffer and 4 additional
	// bytes in the stream. This test ensures we read all 8 bytes of the
	// final integer, not just the 4 in the buffer.

	dec := NewDecoder(NewReaderBytes(bs))
	for {
		val, err := dec.Decode()
		if err == ErrNoInput {
			break
		}
		require.NoError(t, err)
		require.Equal(t, int64(1570737066801085), val.(int64))
	}
}

func TestUnmarshalWithAnnotation(t *testing.T) {
	type foo struct {
		Value   interface{}
		AnyName []SymbolToken `ion:",annotations"`
	}

	test := func(str, testName string, eval foo) {
		t.Run(testName, func(t *testing.T) {
			var val foo
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}

	test("with::multiple::annotations::null", "null", foo{nil, annotations})
	test("with::multiple::annotations::true", "bool", foo{true, annotations})
	test("with::multiple::annotations::2", "int", foo{2, annotations})
	bi := new(big.Int).Neg(new(big.Int).SetUint64(0xFFFFFFFFFFFFFFFF))
	test("with::multiple::annotations::-18446744073709551615", "big.Int", foo{bi, annotations})
	test("with::multiple::annotations::2.1e1", "float", foo{2.1e1, annotations})
	test("with::multiple::annotations::2.2", "decimal", foo{MustParseDecimal("2.2"), annotations})
	test("with::multiple::annotations::\"abc\"", "string", foo{newString("abc"), annotations})
	timestamp := NewTimestamp(time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC), TimestampPrecisionSecond, TimezoneUTC)
	test("with::multiple::annotations::2000-01-02T03:04:05Z", "timestamp", foo{timestamp, annotations})
	test("with::multiple::annotations::{{'''abc'''}}", "clob", foo{[]byte{97, 98, 99}, annotations})
	test("with::multiple::annotations::{{/w==}}", "blob", foo{[]byte{255}, annotations})
}

func TestUnmarshalContainersWithAnnotation(t *testing.T) {
	type foo struct {
		Value   []int
		AnyName []SymbolToken `ion:",annotations"`
	}

	test := func(str, testName string, eval interface{}) {
		t.Run(testName, func(t *testing.T) {
			var val foo
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}

	test("with::multiple::annotations::[1, 2, 3]", "list", foo{[]int{1, 2, 3}, annotations})
	test("with::multiple::annotations::(1 2 3)", "sexp", foo{[]int{1, 2, 3}, annotations})
}

func TestUnmarshalNestedStructsWithAnnotation(t *testing.T) {
	type nestedInt struct {
		Value           int
		ValueAnnotation []SymbolToken `ion:",annotations"`
	}

	type nestedStruct struct {
		Field2                nestedInt
		InnerStructAnnotation []SymbolToken `ion:",annotations"`
	}

	type topLevelStruct struct {
		Field1             nestedStruct
		TopLevelAnnotation []SymbolToken `ion:",annotations"`
	}

	test := func(str, testName string, eval interface{}) {
		t.Run(testName, func(t *testing.T) {
			var val topLevelStruct
			require.NoError(t, UnmarshalString(str, &val))

			assert.Equal(t, eval, val)
		})
	}

	/*
		foo::{
		  field1: bar::{
		    field2: baz::5
		  }
		}
	*/
	innerStructVal := nestedInt{Value: 5, ValueAnnotation: []SymbolToken{NewSimpleSymbolToken("baz")}}
	mainStructVal := nestedStruct{Field2: innerStructVal, InnerStructAnnotation: []SymbolToken{NewSimpleSymbolToken("bar")}}
	expectedValue := topLevelStruct{Field1: mainStructVal, TopLevelAnnotation: []SymbolToken{NewSimpleSymbolToken("foo")}}

	test("foo::{Field1:bar::{Field2:baz::5}}", "nested structs", expectedValue)
}

var symbolTokenWith = NewSimpleSymbolToken("with")
var symbolTokenMultiple = NewSimpleSymbolToken("multiple")
var symbolTokenAnnotations = NewSimpleSymbolToken("annotations")
var annotations = []SymbolToken{symbolTokenWith, symbolTokenMultiple, symbolTokenAnnotations}
