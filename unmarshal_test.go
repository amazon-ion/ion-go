package ion

import (
	"bytes"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"
)

func TestDecodeBool(t *testing.T) {
	test := func(str string, eval bool) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewTextReaderString(str))

			var val bool
			err := d.DecodeTo(&val)
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
func TestDecodeBoolPtr(t *testing.T) {
	test := func(str string, eval interface{}) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewTextReaderString(str))

			var bval bool
			val := &bval
			err := d.DecodeTo(&val)
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

func TestDecodeInt(t *testing.T) {
	testInt8 := func(str string, eval int8) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewTextReaderString(str))

			var val int8
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val int16
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val int32
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val int
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val int64
			err := d.DecodeTo(&val)
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

func TestDecodeUint(t *testing.T) {
	testUint8 := func(str string, eval uint8) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewTextReaderString(str))

			var val uint8
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val uint16
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val uint32
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val uint
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val uintptr
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

			var val uint64
			err := d.DecodeTo(&val)
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

func TestDecodeBigInt(t *testing.T) {
	test := func(str string, eval *big.Int) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewTextReaderString(str))

			var val big.Int
			err := d.DecodeTo(&val)
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
			d := NewDecoder(NewTextReaderString(str))

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
			d := NewDecoder(NewTextReaderString(str))

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
			d := NewDecoder(NewTextReaderString(str))

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

func TestDecodeTimeTo(t *testing.T) {
	test := func(str string, eval time.Time) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewTextReaderString(str))

			var val time.Time
			err := d.DecodeTo(&val)
			if err != nil {
				t.Fatal(err)
			}

			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}
	test("null", time.Time{})
	test("2020T", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
}

func TestDecodeStringTo(t *testing.T) {
	test := func(str string, eval string) {
		t.Run(str, func(t *testing.T) {
			d := NewDecoder(NewTextReaderString(str))

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
			d := NewDecoder(NewTextReaderString(str))

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
			d := NewDecoder(NewTextReaderString(str))

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
			d := NewDecoder(NewTextReaderString(str))
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
		Baz int `json:"bar"`
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
			d := NewDecoder(NewTextReaderString(str))
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
			d := NewDecoder(NewTextReaderString(data))
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

	test("2020T", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))

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
