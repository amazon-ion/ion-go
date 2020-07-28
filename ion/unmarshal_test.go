package ion

import (
	"bytes"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"
)

func TestUnmarshalBool(t *testing.T) {
	test := func(str string, eval bool) {
		t.Run(str, func(t *testing.T) {
			var val bool
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
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
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if eval == nil {
				if val != nil {
					t.Errorf("expected <nil>, got %v", *val)
				}
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
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testInt8("null", 0)
	testInt8("0", 0)
	testInt8("0x7F", 0x7F)
	testInt8("-0x80", -0x80)

	testInt16 := func(str string, eval int16) {
		t.Run(str, func(t *testing.T) {
			var val int16
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testInt16("0x7F", 0x7F)
	testInt16("-0x80", -0x80)
	testInt16("0x7FFF", 0x7FFF)
	testInt16("-0x8000", -0x8000)

	testInt32 := func(str string, eval int32) {
		t.Run(str, func(t *testing.T) {
			var val int32
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testInt32("0x7FFF", 0x7FFF)
	testInt32("-0x8000", -0x8000)
	testInt32("0x7FFFFFFF", 0x7FFFFFFF)
	testInt32("-0x80000000", -0x80000000)

	testInt := func(str string, eval int) {
		t.Run(str, func(t *testing.T) {
			var val int
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testInt("0x7FFF", 0x7FFF)
	testInt("-0x8000", -0x8000)
	testInt("0x7FFFFFFF", 0x7FFFFFFF)
	testInt("-0x80000000", -0x80000000)

	testInt64 := func(str string, eval int64) {
		t.Run(str, func(t *testing.T) {
			var val int64
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
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
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testUint8("null", 0)
	testUint8("0", 0)
	testUint8("0xFF", 0xFF)

	testUint16 := func(str string, eval uint16) {
		t.Run(str, func(t *testing.T) {
			var val uint16
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testUint16("0xFF", 0xFF)
	testUint16("0xFFFF", 0xFFFF)

	testUint32 := func(str string, eval uint32) {
		t.Run(str, func(t *testing.T) {
			var val uint32
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testUint32("0xFFFF", 0xFFFF)
	testUint32("0xFFFFFFFF", 0xFFFFFFFF)

	testUint := func(str string, eval uint) {
		t.Run(str, func(t *testing.T) {
			var val uint
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testUint("0xFFFF", 0xFFFF)
	testUint("0xFFFFFFFF", 0xFFFFFFFF)

	testUintptr := func(str string, eval uintptr) {
		t.Run(str, func(t *testing.T) {
			var val uintptr
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testUintptr("0xFFFF", 0xFFFF)
	testUintptr("0xFFFFFFFF", 0xFFFFFFFF)

	testUint64 := func(str string, eval uint64) {
		t.Run(str, func(t *testing.T) {
			var val uint64
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testUint64("0xFFFFFFFF", 0xFFFFFFFF)
	testUint64("0xFFFFFFFFFFFFFFFF", 0xFFFFFFFFFFFFFFFF)
}

func TestUnmarshalBigInt(t *testing.T) {
	test := func(str string, eval *big.Int) {
		t.Run(str, func(t *testing.T) {
			var val big.Int
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if val.Cmp(eval) != 0 {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	test("null", new(big.Int))
	test("1", new(big.Int).SetUint64(1))
	test("-0xFFFFFFFFFFFFFFFF", new(big.Int).Neg(new(big.Int).SetUint64(0xFFFFFFFFFFFFFFFF)))
}

func TestDecodeFloat(t *testing.T) {
	test32 := func(str string, eval float32) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderStr(str))

			var val float32
			err := d.DecodeTo(&val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	test32("null", 0)
	test32("1e0", 1)
	test32("1e38", 1e38)
	test32("+inf", float32(math.Inf(1)))

	test64 := func(str string, eval float64) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderStr(str))

			var val float64
			err := d.DecodeTo(&val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	test64("1e0", 1)
	test64("1e308", 1e308)
	test64("+inf", math.Inf(1))
}

func TestDecodeDecimal(t *testing.T) {
	test := func(str string, eval *Decimal) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderStr(str))

			var val *Decimal
			err := d.DecodeTo(&val)
			if err != nil {
				t.Fatal(err)
			}

			if !val.Equal(eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}

	test("1e10", MustParseDecimal("1d10"))
	test("1.20", MustParseDecimal("1.20"))
}

func TestDecodeTimestampTo(t *testing.T) {
	test := func(str string, eval Timestamp) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderStr(str))

			var val Timestamp
			err := d.DecodeTo(&val)
			if err != nil {
				t.Fatal(err)
			}

			if !val.Equal(eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	test("null", Timestamp{})
	test("2020T", NewDateTimestamp(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), TimestampPrecisionYear))
}

func TestDecodeStringTo(t *testing.T) {
	test := func(str string, eval string) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderStr(str))

			var val string
			err := d.DecodeTo(&val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}

	test("null", "")
	test("hello", "hello")
	test("\"hello\"", "hello")
}

func TestDecodeLobTo(t *testing.T) {
	testSlice := func(str string, eval []byte) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderStr(str))

			var val []byte
			err := d.DecodeTo(&val)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testSlice("null", nil)
	testSlice("{{}}", []byte{})
	testSlice("{{aGVsbG8=}}", []byte("hello"))
	testSlice("{{'''hello'''}}", []byte("hello"))

	testArray := func(str string, eval []byte) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderStr(str))

			var val [8]byte
			err := d.DecodeTo(&val)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(val[:], eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	testArray("null", make([]byte, 8))
	testArray("{{aGVsbG8=}}", append([]byte("hello"), []byte{0, 0, 0}...))
}

func TestDecodeStructTo(t *testing.T) {
	test := func(str string, val, eval interface{}) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewReaderStr(str))
			err := d.DecodeTo(val)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
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
			d := NewDecoder(NewReaderStr(str))
			err := d.DecodeTo(val)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
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
			d := NewDecoder(NewReaderStr(data))
			val, err := d.Decode()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}

	test("null", nil)
	test("null.null", nil)

	test("null.bool", nil)
	test("true", true)
	test("false", false)

	test("null.int", nil)
	test("0", int(0))
	test("2147483647", math.MaxInt32)
	test("-2147483648", math.MinInt32)
	test("2147483648", int64(math.MaxInt32)+1)
	test("-2147483649", int64(math.MinInt32)-1)
	test("9223372036854775808", new(big.Int).SetUint64(math.MaxInt64+1))

	test("0e0", float64(0.0))
	test("1e100", float64(1e100))

	test("0.", MustParseDecimal("0."))

	test("2020T", NewDateTimestamp(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), TimestampPrecisionYear))

	test("hello", "hello")
	test("\"hello\"", "hello")

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
		"b": "two",
	})

	test("null.list", nil)
	test("[]", []interface{}{})
	test("[1, two]", []interface{}{1, "two"})

	test("null.sexp", nil)
	test("()", []interface{}{})
	test("(1 + two)", []interface{}{1, "+", "two"})
}

func TestDecodeLotsOfInts(t *testing.T) {
	// Regression test for https://github.com/amzn/ion-go/issues/53
	buf := bytes.Buffer{}
	w := NewBinaryWriter(&buf)
	for i := 0; i < 512; i++ {
		w.WriteInt(1570737066801085)
	}
	w.Finish()
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
		if err != nil {
			t.Fatal(err)
		}
		if val.(int64) != 1570737066801085 {
			t.Fatalf("expected %v, got %v", 1570737066801085, val)
		}
	}
}

func TestUnmarshalWithAnnotation(t *testing.T) {
	type foo struct {
		Value   interface{}
		AnyName []string `ion:",annotations"`
	}

	test := func(str, testName string, eval foo) {
		t.Run(testName, func(t *testing.T) {
			var val foo
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}

	test("with::multiple::annotations::null", "null", foo{nil, []string{"with", "multiple", "annotations"}})
	test("with::multiple::annotations::true", "bool", foo{true, []string{"with", "multiple", "annotations"}})
	test("with::multiple::annotations::2", "int", foo{2, []string{"with", "multiple", "annotations"}})
	bi := new(big.Int).Neg(new(big.Int).SetUint64(0xFFFFFFFFFFFFFFFF))
	test("with::multiple::annotations::-18446744073709551615", "big.Int", foo{bi, []string{"with", "multiple", "annotations"}})
	test("with::multiple::annotations::2.1e1", "float", foo{2.1e1, []string{"with", "multiple", "annotations"}})
	test("with::multiple::annotations::2.2", "decimal", foo{MustParseDecimal("2.2"), []string{"with", "multiple", "annotations"}})
	test("with::multiple::annotations::\"abc\"", "string", foo{"abc", []string{"with", "multiple", "annotations"}})
	timestamp := time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC)
	test("with::multiple::annotations::2000-01-02T03:04:05Z", "timestamp", foo{timestamp, []string{"with", "multiple", "annotations"}})
	test("with::multiple::annotations::{{'''abc'''}}", "clob", foo{[]byte{97, 98, 99}, []string{"with", "multiple", "annotations"}})
	test("with::multiple::annotations::{{/w==}}", "blob", foo{[]byte{255}, []string{"with", "multiple", "annotations"}})
}

func TestUnmarshalContainersWithAnnotation(t *testing.T) {
	type foo struct {
		Value   []int
		AnyName []string `ion:",annotations"`
	}

	test := func(str, testName string, eval interface{}) {
		t.Run(testName, func(t *testing.T) {
			var val foo
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}

	test("with::multiple::annotations::[1, 2, 3]", "list", foo{[]int{1, 2, 3}, []string{"with", "multiple", "annotations"}})
	test("with::multiple::annotations::(1 2 3)", "sexp", foo{[]int{1, 2, 3}, []string{"with", "multiple", "annotations"}})
}

func TestUnmarshalNestedStructsWithAnnotation(t *testing.T) {
	type nestedInt struct {
		Value           int
		ValueAnnotation []string `ion:",annotations"`
	}

	type nestedStruct struct {
		Field2                nestedInt
		InnerStructAnnotation []string `ion:",annotations"`
	}

	type topLevelStruct struct {
		Field1             nestedStruct
		TopLevelAnnotation []string `ion:",annotations"`
	}

	test := func(str, testName string, eval interface{}) {
		t.Run(testName, func(t *testing.T) {
			var val topLevelStruct
			err := UnmarshalStr(str, &val)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}

	/*
		foo::{
		  field1: bar::{
		    field2: baz::5
		  }
		}
	*/
	innerStructVal := nestedInt{Value: 5, ValueAnnotation: []string{"baz"}}
	mainStructVal := nestedStruct{Field2: innerStructVal, InnerStructAnnotation: []string{"bar"}}
	expectedValue := topLevelStruct{Field1: mainStructVal, TopLevelAnnotation: []string{"foo"}}

	test("foo::{Field1:bar::{Field2:baz::5}}", "nested structs", expectedValue)
}
