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
	"testing"
	"time"
)

func TestReadTextSymbols(t *testing.T) {
	ionText := `$ion_symbol_table::
				{
					symbols:[ "foo" ]
				}
				$0
				'$4'
				$4
				$10
				foo
				bar
				$ion
				$11`

	r := NewReaderString(ionText)
	_symbolAF(t, r, nil, nil, nil, SymbolToken{Text: nil, LocalSID: 0}, false, false, false)
	_symbol(t, r, newString("$4"), SymbolToken{Text: newString("$4"), LocalSID: SymbolIDUnknown})
	_symbol(t, r, newString("name"), SymbolToken{Text: newString("name"), LocalSID: 4})
	_symbol(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: 10})
	_symbol(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: 10})
	_symbol(t, r, newString("bar"), SymbolToken{Text: newString("bar"), LocalSID: SymbolIDUnknown})
	_symbol(t, r, newString("$ion"), SymbolToken{Text: newString("$ion"), LocalSID: 1})
	_symbolAF(t, r, nil, nil, nil, SymbolToken{}, true, true, true)
}

func TestReadTextFieldNames(t *testing.T) {
	ionText := `$ion_symbol_table::
				{
					symbols:[ "foo" ]
				}
				{
					$0:1,
					'$4':1,
					$4:1,
					$10:1,
					foo:1,
					bar:1,
					$ion:1,
					$11:1
				}`

	r := NewReaderString(ionText)
	r.Next()
	err := r.StepIn()
	if err != nil {
		t.Errorf("stepin returned err: %v", err.Error())
	}
	_nextF(t, r, nil, SymbolToken{Text: nil, LocalSID: 0}, false, false)
	_nextF(t, r, newString("$4"), SymbolToken{Text: newString("$4"), LocalSID: SymbolIDUnknown}, false, false)
	_nextF(t, r, newString("name"), SymbolToken{Text: newString("name"), LocalSID: 4}, false, false)
	_nextF(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: 10}, false, false)
	_nextF(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: 10}, false, false)
	_nextF(t, r, newString("bar"), SymbolToken{Text: newString("bar"), LocalSID: SymbolIDUnknown}, false, false)
	_nextF(t, r, newString("$ion"), SymbolToken{Text: newString("$ion"), LocalSID: 1}, false, false)
	_nextF(t, r, nil, SymbolToken{}, true, true)
}

func TestIgnoreValues(t *testing.T) {
	r := NewReaderString("(skip ++ me / please) {skip: me, please: 0}\n[skip, me, please]\nfoo")

	_next(t, r, SexpType)
	_next(t, r, StructType)
	_next(t, r, ListType)

	_symbol(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown})
	_eof(t, r)
}

func TestReadSexps(t *testing.T) {
	test := func(str string, f containerhandler) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_sexp(t, r, f)
			_eof(t, r)
		})
	}

	test("(\t)", func(t *testing.T, r Reader) {
		if r.Next() {
			t.Errorf("next returned true")
		}
		if r.Err() != nil {
			t.Fatal(r.Err())
		}
	})

	test("(foo)", func(t *testing.T, r Reader) {
		_symbol(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown})
	})

	test("(foo bar baz :: boop)", func(t *testing.T, r Reader) {
		_symbol(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown})
		_symbol(t, r, newString("bar"), SymbolToken{Text: newString("bar"), LocalSID: SymbolIDUnknown})
		_symbolAF(t, r, nil, []string{"baz"}, newString("boop"), SymbolToken{Text: newString("boop"), LocalSID: SymbolIDUnknown}, false, false, false)
	})
}

func TestStructs(t *testing.T) {
	test := func(str string, f containerhandler) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_struct(t, r, f)
			_eof(t, r)
		})
	}

	test("{\r\n}", func(t *testing.T, r Reader) {
		_eof(t, r)
	})

	test("{foo : bar :: baz}", func(t *testing.T, r Reader) {
		foo := "foo"
		_symbolAF(t, r, &foo, []string{"bar"}, newString("baz"), SymbolToken{Text: newString("baz"), LocalSID: SymbolIDUnknown}, false, false, false)
	})

	test("{foo: a, bar: b, baz: c}", func(t *testing.T, r Reader) {
		foo, bar, baz := "foo", "bar", "baz"
		_symbolAF(t, r, &foo, nil, newString("a"), SymbolToken{Text: newString("a"), LocalSID: SymbolIDUnknown}, false, false, false)
		_symbolAF(t, r, &bar, nil, newString("b"), SymbolToken{Text: newString("b"), LocalSID: SymbolIDUnknown}, false, false, false)
		_symbolAF(t, r, &baz, nil, newString("c"), SymbolToken{Text: newString("c"), LocalSID: SymbolIDUnknown}, false, false, false)
	})

	test("{\"\": a}", func(t *testing.T, r Reader) {
		emptyString := ""
		_symbolAF(t, r, &emptyString, nil, newString("a"), SymbolToken{Text: newString("a"), LocalSID: SymbolIDUnknown}, false, false, false)
	})
}

func TestMultipleStructs(t *testing.T) {
	r := NewReaderString("{} {} {}")

	for i := 0; i < 3; i++ {
		_struct(t, r, func(t *testing.T, r Reader) {
			_eof(t, r)
		})
	}

	_eof(t, r)
}

func TestNullStructs(t *testing.T) {
	r := NewReaderString("null.struct 'null'::{foo:bar}")

	_null(t, r, StructType)
	_nextAF(t, r, StructType, nil, []string{"null"})
	_eof(t, r)
}

func TestLists(t *testing.T) {
	test := func(str string, f containerhandler) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_list(t, r, f)
			_eof(t, r)
		})
	}

	test("[    ]", func(t *testing.T, r Reader) {
		_eof(t, r)
	})

	test("[foo]", func(t *testing.T, r Reader) {
		_symbol(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown})
		_eof(t, r)
	})

	test("[foo, bar, baz::boop]", func(t *testing.T, r Reader) {
		_symbol(t, r, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown})
		_symbol(t, r, newString("bar"), SymbolToken{Text: newString("bar"), LocalSID: SymbolIDUnknown})
		_symbolAF(t, r, nil, []string{"baz"}, newString("boop"), SymbolToken{Text: newString("boop"), LocalSID: SymbolIDUnknown}, false, false, false)
		_eof(t, r)
	})
}

func TestReadNestedLists(t *testing.T) {
	empty := func(t *testing.T, r Reader) {
		_eof(t, r)
	}

	r := NewReaderString("[[], [[]]]")

	_list(t, r, func(t *testing.T, r Reader) {
		_list(t, r, empty)

		_list(t, r, func(t *testing.T, r Reader) {
			_list(t, r, empty)
		})

		_eof(t, r)
	})

	_eof(t, r)
}

func TestClobs(t *testing.T) {
	test := func(str string, eval []byte) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_next(t, r, ClobType)

			val, err := r.ByteValue()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}

			_eof(t, r)
		})
	}

	test("{{\"\"}}", []byte{})
	test("{{ \"hello world\" }}", []byte("hello world"))
	test("{{'''hello world'''}}", []byte("hello world"))
	test("{{'''hello'''\n'''world'''}}", []byte("helloworld"))
}

func TestBlobs(t *testing.T) {
	test := func(str string, eval []byte) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_next(t, r, BlobType)

			val, err := r.ByteValue()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(val, eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}

			_eof(t, r)
		})
	}

	test("{{}}", []byte{})
	test("{{AA==}}", []byte{0})
	test("{{  SGVsbG8g\r\nV29ybGQ=  }}", []byte("Hello World"))
}

func TestTimestamps(t *testing.T) {
	testA := func(str string, etas []string, eval Timestamp) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_nextAF(t, r, TimestampType, nil, etas)

			val, err := r.TimestampValue()
			if err != nil {
				t.Fatal(err)
			}

			if !val.Equal(eval) {
				t.Errorf("expected %v, got %v", eval, val)
			}

			_eof(t, r)
		})
	}

	test := func(str string, eval Timestamp) {
		testA(str, nil, eval)
	}

	et := time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
	test("2001T", NewDateTimestamp(et, TimestampPrecisionYear))
	test("2001-01T", NewDateTimestamp(et, TimestampPrecisionMonth))
	test("2001-01-01", NewDateTimestamp(et, TimestampPrecisionDay))
	test("2001-01-01T", NewDateTimestamp(et, TimestampPrecisionDay))
	test("2001-01-01T00:00Z", NewTimestamp(et, TimestampPrecisionMinute, TimezoneUTC))
	test("2001-01-01T00:00:00Z", NewTimestamp(et, TimestampPrecisionSecond, TimezoneUTC))
	test("2001-01-01T00:00:00.000Z", NewTimestampWithFractionalSeconds(et, TimestampPrecisionNanosecond, TimezoneUTC, 3))
	test("2001-01-01T00:00:00.000+00:00", NewTimestampWithFractionalSeconds(et, TimestampPrecisionNanosecond, TimezoneUTC, 3))
	test("2001-01-01T00:00:00.000000Z", NewTimestampWithFractionalSeconds(et, TimestampPrecisionNanosecond, TimezoneUTC, 6))
	test("2001-01-01T00:00:00.000000000Z", NewTimestampWithFractionalSeconds(et, TimestampPrecisionNanosecond, TimezoneUTC, 9))

	et2 := time.Date(2001, time.January, 1, 0, 0, 0, 1, time.UTC)
	test("2001-01-01T00:00:00.000000000999Z", NewTimestampWithFractionalSeconds(et2, TimestampPrecisionNanosecond, TimezoneUTC, 12))

	testA("foo::'bar'::2001-01-01T00:00:00.000Z", []string{"foo", "bar"}, NewTimestampWithFractionalSeconds(et, TimestampPrecisionNanosecond, TimezoneUTC, 3))
}

func TestDecimals(t *testing.T) {
	testA := func(str string, etas []string, eval string) {
		t.Run(str, func(t *testing.T) {
			ee := MustParseDecimal(eval)

			r := NewReaderString(str)
			_nextAF(t, r, DecimalType, nil, etas)

			val, err := r.DecimalValue()
			if err != nil {
				t.Fatal(err)
			}
			if !ee.Equal(val) {
				t.Errorf("expected %v, got %v", ee, val)
			}

			_eof(t, r)
		})
	}

	test := func(str string, eval string) {
		testA(str, nil, eval)
	}

	test("123.", "123")
	test("123.0", "123")
	test("123.456", "123.456")
	test("123d2", "12300")
	test("123d+2", "12300")
	test("123d-2", "1.23")

	testA("  foo :: 'bar' :: 123.  ", []string{"foo", "bar"}, "123")
}

func TestFloats(t *testing.T) {
	testA := func(str string, etas []string, eval float64) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_floatAF(t, r, nil, etas, eval)
			_eof(t, r)
		})
	}

	test := func(str string, eval float64) {
		testA(str, nil, eval)
	}

	test("1e100\n", 1e100)
	test("1.2e+0", 1.2)
	test("-123.456e-78", -123.456e-78)
	test("+inf", math.Inf(1))
	test("-inf", math.Inf(-1))

	testA("foo::'bar'::1e100", []string{"foo", "bar"}, 1e100)
}

func TestInts(t *testing.T) {
	test := func(str string, f func(*testing.T, Reader)) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_next(t, r, IntType)

			f(t, r)

			_eof(t, r)
		})
	}

	test("null.int", func(t *testing.T, r Reader) {
		if !r.IsNull() {
			t.Fatal("expected isnull=true, got false")
		}
	})

	testInt := func(str string, eval int) {
		test(str, func(t *testing.T, r Reader) {
			val, err := r.IntValue()
			if err != nil {
				t.Fatal(err)
			}
			if *val != eval {
				t.Errorf("expected %v, got %v", eval, *val)
			}
		})
	}

	testInt("0", 0)
	testInt("12_345", 12345)
	testInt("-1_2_3_4_5", -12345)
	testInt("0b00_0101", 5)
	testInt("-0b00_0101", -5)
	testInt("0x01_02_0e_0F", 0x01020e0f)
	testInt("-0x0102_0e0F", -0x01020e0f)

	testInt64 := func(str string, eval int64) {
		test(str, func(t *testing.T, r Reader) {
			val, err := r.Int64Value()
			if err != nil {
				t.Fatal(err)
			}
			if *val != eval {
				t.Errorf("expected %v, got %v", eval, *val)
			}
		})
	}

	testInt64("0x123_FFFF_FFFF", 0x123FFFFFFFF)
	testInt64("-0x123_FFFF_FFFF", -0x123FFFFFFFF)

	testBigInt := func(str string, estr string) {
		test(str, func(t *testing.T, r Reader) {
			val, err := r.BigIntValue()
			if err != nil {
				t.Fatal(err)
			}

			eval, _ := (&big.Int{}).SetString(estr, 0)
			if eval.Cmp(val) != 0 {
				t.Errorf("expected %v, got %v", eval, val)
			}
		})
	}

	testBigInt("0xEFFF_FFFF_FFFF_FFFF", "0xEFFFFFFFFFFFFFFF")
	testBigInt("0xFFFF_FFFF_FFFF_FFFF", "0xFFFFFFFFFFFFFFFF")
	testBigInt("-0x1_FFFF_FFFF_FFFF_FFFF", "-0x1FFFFFFFFFFFFFFFF")
}

func TestStrings(t *testing.T) {
	r := NewReaderString(`foo::"bar" "baz" 'a'::'b'::'''beep''' '''boop''' null.string`)

	_stringAF(t, r, nil, []string{"foo"}, newString("bar"))
	_string(t, r, newString("baz"))
	_stringAF(t, r, nil, []string{"a", "b"}, newString("beepboop"))
	_null(t, r, StringType)

	_eof(t, r)
}

func TestSymbols(t *testing.T) {
	r := NewReaderString("'null'::foo bar a::b::'baz' null.symbol")

	_symbolAF(t, r, nil, []string{"null"}, newString("foo"), SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown}, false, false, false)
	_symbol(t, r, newString("bar"), SymbolToken{Text: newString("bar"), LocalSID: SymbolIDUnknown})
	_symbolAF(t, r, nil, []string{"a", "b"}, newString("baz"), SymbolToken{Text: newString("baz"), LocalSID: SymbolIDUnknown}, false, false, false)
	_null(t, r, SymbolType)

	_eof(t, r)
}

func TestSpecialSymbols(t *testing.T) {
	r := NewReaderString("null\nnull.struct\ntrue\nfalse\nnan")

	_null(t, r, NullType)
	_null(t, r, StructType)

	_bool(t, r, true)
	_bool(t, r, false)
	_float(t, r, math.NaN())
	_eof(t, r)
}

func TestOperators(t *testing.T) {
	r := NewReaderString("(a*(b+c))")

	_sexp(t, r, func(t *testing.T, r Reader) {
		_symbol(t, r, newString("a"), SymbolToken{Text: newString("a"), LocalSID: SymbolIDUnknown})
		_symbol(t, r, newString("*"), SymbolToken{Text: newString("*"), LocalSID: SymbolIDUnknown})
		_sexp(t, r, func(t *testing.T, r Reader) {
			_symbol(t, r, newString("b"), SymbolToken{Text: newString("b"), LocalSID: SymbolIDUnknown})
			_symbol(t, r, newString("+"), SymbolToken{Text: newString("+"), LocalSID: SymbolIDUnknown})
			_symbol(t, r, newString("c"), SymbolToken{Text: newString("c"), LocalSID: SymbolIDUnknown})
			_eof(t, r)
		})
		_eof(t, r)
	})
}

func TestTopLevelOperators(t *testing.T) {
	r := NewReaderString("a + b")

	_symbol(t, r, newString("a"), SymbolToken{Text: newString("a"), LocalSID: SymbolIDUnknown})

	if r.Next() {
		t.Errorf("next returned true")
	}
	if r.Err() == nil {
		t.Error("no error")
	}
}

func TestTrsToString(t *testing.T) {
	for i := trsDone; i <= trsAfterValue+1; i++ {
		str := i.String()
		if str == "" {
			t.Errorf("expected a non-empty string for trs %v", uint8(i))
		}
	}
}

func TestInStruct(t *testing.T) {
	r := NewReaderString("[ { a:() } ]")

	r.Next()
	r.StepIn() // In the list, before the struct
	if r.IsInStruct() {
		t.Fatal("IsInStruct returned true before we were in a struct")
	}

	r.Next()
	r.StepIn() // In the struct
	if !r.IsInStruct() {
		t.Fatal("We were in a struct, IsInStruct should have returned true")
	}

	r.Next()
	r.StepIn() // In the Sexp
	if r.IsInStruct() {
		t.Fatal("IsInStruct returned true before we were in a struct")
	}

	r.StepOut() // Out of the Sexp, back in the struct again
	if !r.IsInStruct() {
		t.Fatal("We were in a struct, IsInStruct should have returned true")
	}

	r.StepOut() // out of struct, back in the list again
	if r.IsInStruct() {
		t.Fatal("IsInStruct returned true before we were in a struct")
	}
}

type containerhandler func(t *testing.T, r Reader)

func _sexp(t *testing.T, r Reader, f containerhandler) {
	_sexpAF(t, r, nil, nil, f)
}

func _sexpAF(t *testing.T, r Reader, efn *string, etas []string, f containerhandler) {
	_containerAF(t, r, SexpType, efn, etas, f)
}

func _struct(t *testing.T, r Reader, f containerhandler) {
	_structAF(t, r, nil, nil, f)
}

func _structAF(t *testing.T, r Reader, efn *string, etas []string, f containerhandler) {
	_containerAF(t, r, StructType, efn, etas, f)
}

func _list(t *testing.T, r Reader, f containerhandler) {
	_listAF(t, r, nil, nil, f)
}

func _listAF(t *testing.T, r Reader, efn *string, etas []string, f containerhandler) {
	_containerAF(t, r, ListType, efn, etas, f)
}

func _containerAF(t *testing.T, r Reader, et Type, efn *string, etas []string, f containerhandler) {
	_nextAF(t, r, et, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.%v", et, et)
	}

	if err := r.StepIn(); err != nil {
		t.Fatal(err)
	}

	f(t, r)

	if err := r.StepOut(); err != nil {
		t.Fatal(err)
	}
}

func _int(t *testing.T, r Reader, eval int) {
	_intAF(t, r, nil, nil, eval)
}

func _intAF(t *testing.T, r Reader, efn *string, etas []string, eval int) {
	_nextAF(t, r, IntType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.int", eval)
	}

	size, err := r.IntSize()
	if err != nil {
		t.Fatal(err)
	}
	if size != Int32 {
		t.Errorf("expected size=Int32, got %v", size)
	}

	val, err := r.IntValue()
	if err != nil {
		t.Fatal(err)
	}
	if *val != eval {
		t.Errorf("expected %v, got %v", eval, *val)
	}
}

func _int64(t *testing.T, r Reader, eval int64) {
	_int64AF(t, r, nil, nil, eval)
}

func _int64AF(t *testing.T, r Reader, efn *string, etas []string, eval int64) {
	_nextAF(t, r, IntType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.int", eval)
	}

	size, err := r.IntSize()
	if err != nil {
		t.Fatal(err)
	}
	if size != Int64 {
		t.Errorf("expected size=Int64, got %v", size)
	}

	val, err := r.Int64Value()
	if err != nil {
		t.Fatal(err)
	}
	if *val != eval {
		t.Errorf("expected %v, got %v", eval, *val)
	}
}

func _uint(t *testing.T, r Reader, eval uint64) {
	_uintAF(t, r, nil, nil, eval)
}

func _uintAF(t *testing.T, r Reader, efn *string, etas []string, eval uint64) {
	_nextAF(t, r, IntType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.int", eval)
	}

	size, err := r.IntSize()
	if err != nil {
		t.Fatal(err)
	}
	if size != Uint64 {
		t.Errorf("expected size=Uint, got %v", size)
	}

	val, err := r.Uint64Value()
	if err != nil {
		t.Fatal(err)
	}
	if *val != eval {
		t.Errorf("expected %v, got %v", eval, *val)
	}
}

func _bigInt(t *testing.T, r Reader, eval *big.Int) {
	_bigIntAF(t, r, nil, nil, eval)
}

func _bigIntAF(t *testing.T, r Reader, efn *string, etas []string, eval *big.Int) {
	_nextAF(t, r, IntType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.int", eval)
	}

	size, err := r.IntSize()
	if err != nil {
		t.Fatal(err)
	}
	if size != BigInt {
		t.Errorf("expected size=BigInt, got %v", size)
	}

	val, err := r.BigIntValue()
	if err != nil {
		t.Fatal(err)
	}
	if val.Cmp(eval) != 0 {
		t.Errorf("expected %v, got %v", eval, val)
	}
}

func _float(t *testing.T, r Reader, eval float64) {
	_floatAF(t, r, nil, nil, eval)
}

func _floatAF(t *testing.T, r Reader, efn *string, etas []string, eval float64) {
	_nextAF(t, r, FloatType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.float", eval)
	}

	val, err := r.FloatValue()
	if err != nil {
		t.Fatal(err)
	}

	if math.IsNaN(eval) {
		if !math.IsNaN(*val) {
			t.Errorf("expected %v, got %v", eval, *val)
		}
	} else if eval != *val {
		t.Errorf("expected %v, got %v", eval, *val)
	}
}

func _decimal(t *testing.T, r Reader, eval *Decimal) {
	_decimalAF(t, r, nil, nil, eval)
}

func _decimalAF(t *testing.T, r Reader, efn *string, etas []string, eval *Decimal) {
	_nextAF(t, r, DecimalType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.decimal", eval)
	}

	val, err := r.DecimalValue()
	if err != nil {
		t.Fatal(err)
	}

	if !eval.Equal(val) {
		t.Errorf("expected %v, got %v", eval, val)
	}
}

func _timestamp(t *testing.T, r Reader, eval Timestamp) {
	_timestampAF(t, r, nil, nil, eval)
}

func _timestampAF(t *testing.T, r Reader, efn *string, etas []string, eval Timestamp) {
	_nextAF(t, r, TimestampType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.timestamp", eval)
	}

	val, err := r.TimestampValue()
	if err != nil {
		t.Fatal(err)
	}

	if !eval.Equal(*val) {
		t.Errorf("expected %v, got %v", eval, *val)
	}
}

func _string(t *testing.T, r Reader, eval *string) {
	_stringAF(t, r, nil, nil, eval)
}

func _stringAF(t *testing.T, r Reader, efn *string, etas []string, eval *string) {
	_nextAF(t, r, StringType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.string", eval)
	}

	val, err := r.StringValue()
	if err != nil {
		t.Fatal(err)
	}
	if eval == nil {
		if val != eval {
			t.Errorf("expected %v, got %v", eval, val)
		}
	} else if *val != *eval {
		t.Errorf("expected %v, got %v", *eval, *val)
	}
}

// _symbolAF calls reader.next and asserts the expected symbol value. This function also asserts the value has neither
// annotation or field name.
func _symbol(t *testing.T, r Reader, eval *string, evalst SymbolToken) {
	_symbolAF(t, r, nil, nil, eval, evalst, false, false, false)
}

// _symbolAF calls reader.next and asserts the expected symbol value, annotation, and field name.
func _symbolAF(t *testing.T, r Reader, efn *string, etas []string, eval *string, evalSt SymbolToken, isStringValueError bool, isSymbolValueError bool, isNextError bool) {
	if !isNextError {
		_nextAF(t, r, SymbolType, efn, etas)
	} else {
		r.Next()
	}

	if r.IsNull() {
		t.Fatalf("expected %v, got null.symbol", eval)
	}

	val, err := r.StringValue()
	if isStringValueError {
		if err == nil {
			t.Fatal("stringvalue did not return an error")
		}
	} else {
		if err != nil {
			t.Fatal(err)
		}
		if eval == nil {
			if eval != val {
				t.Errorf("expected %v, got %v", eval, val)
			}
		} else if *val != *eval {
			t.Errorf("expected %v, got %v", *eval, *val)
		}
	}

	symbolVal, err := r.SymbolValue()
	if isSymbolValueError {
		if err == nil {
			t.Fatal("symboltoken did not return an error")
		}
	} else {
		if err != nil {
			t.Fatal(err)
		}
		if !symbolVal.Equal(&evalSt) {
			t.Errorf("expected %v, got %v", &evalSt, symbolVal)
		}
	}
}

func _bool(t *testing.T, r Reader, eval bool) {
	_boolAF(t, r, nil, nil, eval)
}

func _boolAF(t *testing.T, r Reader, efn *string, etas []string, eval bool) {
	_nextAF(t, r, BoolType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.bool", eval)
	}

	val, err := r.BoolValue()
	if err != nil {
		t.Fatal(err)
	}
	if *val != eval {
		t.Errorf("expected %v, got %v", eval, *val)
	}
}

func _clob(t *testing.T, r Reader, eval []byte) {
	_clobAF(t, r, nil, nil, eval)
}

func _clobAF(t *testing.T, r Reader, efn *string, etas []string, eval []byte) {
	_nextAF(t, r, ClobType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.clob", eval)
	}

	val, err := r.ByteValue()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(val, eval) {
		t.Errorf("expected %v, got %v", eval, val)
	}
}

func _blob(t *testing.T, r Reader, eval []byte) {
	_blobAF(t, r, nil, nil, eval)
}

func _blobAF(t *testing.T, r Reader, efn *string, etas []string, eval []byte) {
	_nextAF(t, r, BlobType, efn, etas)
	if r.IsNull() {
		t.Fatalf("expected %v, got null.blob", eval)
	}

	val, err := r.ByteValue()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(val, eval) {
		t.Errorf("expected %v, got %v", eval, val)
	}
}

func _null(t *testing.T, r Reader, et Type) {
	_nullAF(t, r, et, nil, nil)
}

func _nullAF(t *testing.T, r Reader, et Type, efn *string, etas []string) {
	_nextAF(t, r, et, efn, etas)
	if !r.IsNull() {
		t.Error("isnull returned false")
	}
}

func _next(t *testing.T, r Reader, et Type) {
	_nextAF(t, r, et, nil, nil)
}

func _nextAF(t *testing.T, r Reader, et Type, efn *string, etas []string) {
	if !r.Next() {
		t.Fatal(r.Err())
	}
	if r.Type() != et {
		t.Fatalf("expected %v, got %v", et, r.Type())
	}

	fn := r.FieldName()
	if efn != nil && fn != nil && *efn != *fn {
		t.Errorf("expected fieldname=%v, got %v", *efn, *fn)
	}
	if !_strequals(etas, r.Annotations()) {
		t.Errorf("expected type annotations=%v, got %v", etas, r.Annotations())
	}
}

func _nextF(t *testing.T, r Reader, efn *string, efns SymbolToken, isFieldNameSymbolError bool, isNextError bool) {
	if !r.Next() && !isNextError {
		t.Fatal(r.Err())
	}

	fn := r.FieldName()
	if efn != nil && fn != nil && *efn != *fn {
		t.Errorf("expected fieldname=%v, got %v", *efn, *fn)
	}

	fns, err := r.FieldNameSymbol()
	if isFieldNameSymbolError {
		if err == nil {
			t.Fatal("stringvalue did not return an error")
		}
	} else {
		if err != nil {
			t.Errorf("fieldnamesymbol returned an error: %v", err.Error())
		}
		if !efns.Equal(fns) {
			t.Errorf("expected fieldnamesymbol=%v, got %v", efns, fns)
		}
	}
}

func _strequals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func _eof(t *testing.T, r Reader) {
	if r.Next() {
		t.Fatal("next returned true")
	}
	if r.Err() != nil {
		t.Fatal(r.Err())
	}
}
