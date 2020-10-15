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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				null.symbol
				$11`

	r := NewReaderString(ionText)
	_symbolAF(t, r, nil, nil, &SymbolToken{Text: nil, LocalSID: 0}, false, false)
	_symbol(t, r, NewSymbolTokenFromString("'$4'"))
	_symbol(t, r, SymbolToken{Text: newString("name"), LocalSID: 4})
	_symbol(t, r, SymbolToken{Text: newString("foo"), LocalSID: 10})
	_symbol(t, r, SymbolToken{Text: newString("foo"), LocalSID: 10})
	_symbol(t, r, NewSymbolTokenFromString("bar"))
	_symbol(t, r, SymbolToken{Text: newString("$ion"), LocalSID: 1})
	_symbolAF(t, r, nil, nil, nil, false, false)
	_symbolAF(t, r, nil, nil, &SymbolToken{}, true, true)
}

func TestReadTextAnnotations(t *testing.T) {
	ionText := `$ion_symbol_table::
				{
					symbols:[ "foo" ]
				}
				[
					$0::1,
					'$4'::1,
					$4::1,
					$10::1,
					foo::1,
					bar::1,
					$ion::1,
					$11::1
				]`

	r := NewReaderString(ionText)
	assert.True(t, r.Next())
	assert.NoError(t, r.StepIn())

	_nextA(t, r, []SymbolToken{{Text: nil, LocalSID: 0}}, false, false)
	_nextA(t, r, []SymbolToken{NewSymbolTokenFromString("$4")}, false, false)
	_nextA(t, r, []SymbolToken{{Text: newString("name"), LocalSID: 4}}, false, false)
	_nextA(t, r, []SymbolToken{{Text: newString("foo"), LocalSID: 10}}, false, false)
	_nextA(t, r, []SymbolToken{{Text: newString("foo"), LocalSID: 10}}, false, false)
	_nextA(t, r, []SymbolToken{NewSymbolTokenFromString("bar")}, false, false)
	_nextA(t, r, []SymbolToken{{Text: newString("$ion"), LocalSID: 1}}, false, false)
	_nextA(t, r, nil, true, true)
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
					null.symbol:1
					$11:1
				}`

	r := NewReaderString(ionText)
	assert.True(t, r.Next())
	assert.NoError(t, r.StepIn())

	_nextF(t, r, &SymbolToken{Text: nil, LocalSID: 0}, false, false)
	_nextF(t, r, newSymbolTokenPtrFromString("$4"), false, false)
	_nextF(t, r, &SymbolToken{Text: newString("name"), LocalSID: 4}, false, false)
	_nextF(t, r, &SymbolToken{Text: newString("foo"), LocalSID: 10}, false, false)
	_nextF(t, r, &SymbolToken{Text: newString("foo"), LocalSID: 10}, false, false)
	_nextF(t, r, newSymbolTokenPtrFromString("bar"), false, false)
	_nextF(t, r, &SymbolToken{Text: newString("$ion"), LocalSID: 1}, false, false)
	_nextF(t, r, &SymbolToken{}, true, true)
}

func TestReadTextNullFieldName(t *testing.T) {
	ionText := `{
					null.symbol:1
				}`

	r := NewReaderString(ionText)
	assert.True(t, r.Next())
	assert.NoError(t, r.StepIn())
	_nextF(t, r, &SymbolToken{}, true, true)
}

func TestIgnoreValues(t *testing.T) {
	r := NewReaderString("(skip ++ me / please) {skip: me, please: 0}\n[skip, me, please]\nfoo")

	_next(t, r, SexpType)
	_next(t, r, StructType)
	_next(t, r, ListType)

	_symbol(t, r, NewSymbolTokenFromString("foo"))
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
		assert.False(t, r.Next())
		require.NoError(t, r.Err())
	})

	test("(foo)", func(t *testing.T, r Reader) {
		_symbol(t, r, NewSymbolTokenFromString("foo"))
	})

	test("(foo bar baz :: boop)", func(t *testing.T, r Reader) {
		_symbol(t, r, NewSymbolTokenFromString("foo"))
		_symbol(t, r, NewSymbolTokenFromString("bar"))
		_symbolAF(t, r, nil, []SymbolToken{NewSymbolTokenFromString("baz")}, newSymbolTokenPtrFromString("boop"), false, false)
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
		_symbolAF(t, r, newSymbolTokenPtrFromString("foo"), []SymbolToken{NewSymbolTokenFromString("bar")}, newSymbolTokenPtrFromString("baz"), false, false)
	})

	test("{foo: a, bar: b, baz: c}", func(t *testing.T, r Reader) {
		_symbolAF(t, r, newSymbolTokenPtrFromString("foo"), nil, newSymbolTokenPtrFromString("a"), false, false)
		_symbolAF(t, r, newSymbolTokenPtrFromString("bar"), nil, newSymbolTokenPtrFromString("b"), false, false)
		_symbolAF(t, r, newSymbolTokenPtrFromString("baz"), nil, newSymbolTokenPtrFromString("c"), false, false)
	})

	test("{\"\": a}", func(t *testing.T, r Reader) {
		_symbolAF(t, r, newSymbolTokenPtrFromString(""), nil, newSymbolTokenPtrFromString("a"), false, false)
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
	_nextAF(t, r, StructType, nil, []SymbolToken{NewSymbolTokenFromString("null")})
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
		_symbol(t, r, NewSymbolTokenFromString("foo"))
		_eof(t, r)
	})

	test("[foo, bar, baz::boop]", func(t *testing.T, r Reader) {
		_symbol(t, r, NewSymbolTokenFromString("foo"))
		_symbol(t, r, NewSymbolTokenFromString("bar"))
		_symbolAF(t, r, nil, []SymbolToken{NewSymbolTokenFromString("baz")}, newSymbolTokenPtrFromString("boop"), false, false)
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
			require.NoError(t, err)

			assert.True(t, bytes.Equal(val, eval), "expected %v, got %v", eval, val)

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
			require.NoError(t, err)

			assert.True(t, bytes.Equal(val, eval), "expected %v, got %v", eval, val)

			_eof(t, r)
		})
	}

	test("{{}}", []byte{})
	test("{{AA==}}", []byte{0})
	test("{{  SGVsbG8g\r\nV29ybGQ=  }}", []byte("Hello World"))
}

func TestTimestamps(t *testing.T) {
	testA := func(str string, etas []SymbolToken, eval Timestamp) {
		t.Run(str, func(t *testing.T) {
			r := NewReaderString(str)
			_nextAF(t, r, TimestampType, nil, etas)

			val, err := r.TimestampValue()
			require.NoError(t, err)

			assert.True(t, val.Equal(eval), "expected %v, got %v", eval, val)

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

	testA("foo::'bar'::2001-01-01T00:00:00.000Z", []SymbolToken{NewSymbolTokenFromString("foo"), NewSymbolTokenFromString("bar")}, NewTimestampWithFractionalSeconds(et, TimestampPrecisionNanosecond, TimezoneUTC, 3))
}

func TestDecimals(t *testing.T) {
	testA := func(str string, etas []SymbolToken, eval string) {
		t.Run(str, func(t *testing.T) {
			ee := MustParseDecimal(eval)

			r := NewReaderString(str)
			_nextAF(t, r, DecimalType, nil, etas)

			val, err := r.DecimalValue()
			require.NoError(t, err)

			assert.True(t, ee.Equal(val), "expected %v, got %v", ee, val)

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

	testA("  foo :: 'bar' :: 123.  ", []SymbolToken{NewSymbolTokenFromString("foo"), NewSymbolTokenFromString("bar")}, "123")
}

func TestFloats(t *testing.T) {
	testA := func(str string, etas []SymbolToken, eval float64) {
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

	testA("foo::'bar'::1e100", []SymbolToken{NewSymbolTokenFromString("foo"), NewSymbolTokenFromString("bar")}, 1e100)
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
		require.True(t, r.IsNull())
	})

	testInt := func(str string, eval int) {
		test(str, func(t *testing.T, r Reader) {
			val, err := r.IntValue()
			require.NoError(t, err)
			assert.Equal(t, eval, *val)
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
			require.NoError(t, err)
			assert.Equal(t, eval, *val)
		})
	}

	testInt64("0x123_FFFF_FFFF", 0x123FFFFFFFF)
	testInt64("-0x123_FFFF_FFFF", -0x123FFFFFFFF)

	testBigInt := func(str string, estr string) {
		test(str, func(t *testing.T, r Reader) {
			val, err := r.BigIntValue()
			require.NoError(t, err)

			eval, _ := (&big.Int{}).SetString(estr, 0)
			assert.True(t, eval.Cmp(val) == 0, "expected %v, got %v", eval, val)
		})
	}

	testBigInt("0xEFFF_FFFF_FFFF_FFFF", "0xEFFFFFFFFFFFFFFF")
	testBigInt("0xFFFF_FFFF_FFFF_FFFF", "0xFFFFFFFFFFFFFFFF")
	testBigInt("-0x1_FFFF_FFFF_FFFF_FFFF", "-0x1FFFFFFFFFFFFFFFF")
}

func TestStrings(t *testing.T) {
	r := NewReaderString(`foo::"bar" "baz" 'a'::'b'::'''beep''' '''boop''' null.string`)

	_stringAF(t, r, nil, []SymbolToken{NewSymbolTokenFromString("foo")}, newString("bar"))
	_string(t, r, newString("baz"))
	_stringAF(t, r, nil, []SymbolToken{NewSymbolTokenFromString("a"), NewSymbolTokenFromString("b")}, newString("beepboop"))
	_null(t, r, StringType)

	_eof(t, r)
}

func TestSymbols(t *testing.T) {
	r := NewReaderString("'null'::foo bar a::b::'baz' null.symbol")

	_symbolAF(t, r, nil, []SymbolToken{NewSymbolTokenFromString("null")}, newSymbolTokenPtrFromString("foo"), false, false)
	_symbol(t, r, NewSymbolTokenFromString("bar"))
	_symbolAF(t, r, nil, []SymbolToken{NewSymbolTokenFromString("a"), NewSymbolTokenFromString("b")}, newSymbolTokenPtrFromString("baz"), false, false)
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
		_symbol(t, r, NewSymbolTokenFromString("a"))
		_symbol(t, r, NewSymbolTokenFromString("*"))
		_sexp(t, r, func(t *testing.T, r Reader) {
			_symbol(t, r, NewSymbolTokenFromString("b"))
			_symbol(t, r, NewSymbolTokenFromString("+"))
			_symbol(t, r, NewSymbolTokenFromString("c"))
			_eof(t, r)
		})
		_eof(t, r)
	})
}

func TestTopLevelOperators(t *testing.T) {
	r := NewReaderString("a + b")

	_symbol(t, r, NewSymbolTokenFromString("a"))

	assert.False(t, r.Next())
	assert.Error(t, r.Err())
}

func TestTrsToString(t *testing.T) {
	for i := trsDone; i <= trsAfterValue+1; i++ {
		assert.NotEmpty(t, i.String(), "expected a non-empty string for trs %v", uint8(i))
	}
}

func TestInStruct(t *testing.T) {
	r := NewReaderString("[ { a:() } ]")

	assert.True(t, r.Next())
	assert.NoError(t, r.StepIn()) // In the list, before the struct
	require.False(t, r.IsInStruct(), "IsInStruct returned true before we were in a struct")

	assert.True(t, r.Next())
	assert.NoError(t, r.StepIn()) // In the struct
	require.True(t, r.IsInStruct(), "We were in a struct, IsInStruct should have returned true")

	assert.True(t, r.Next())
	assert.NoError(t, r.StepIn()) // In the Sexp
	require.False(t, r.IsInStruct(), "IsInStruct returned true before we were in a struct")

	assert.NoError(t, r.StepOut()) // Out of the Sexp, back in the struct again
	require.True(t, r.IsInStruct(), "We were in a struct, IsInStruct should have returned true")

	assert.NoError(t, r.StepOut()) // out of struct, back in the list again
	require.False(t, r.IsInStruct(), "IsInStruct returned true before we were in a struct")
}

type containerhandler func(t *testing.T, r Reader)

func _sexp(t *testing.T, r Reader, f containerhandler) {
	_sexpAF(t, r, nil, nil, f)
}

func _sexpAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, f containerhandler) {
	_containerAF(t, r, SexpType, efn, etas, f)
}

func _struct(t *testing.T, r Reader, f containerhandler) {
	_structAF(t, r, nil, nil, f)
}

func _structAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, f containerhandler) {
	_containerAF(t, r, StructType, efn, etas, f)
}

func _list(t *testing.T, r Reader, f containerhandler) {
	_listAF(t, r, nil, nil, f)
}

func _listAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, f containerhandler) {
	_containerAF(t, r, ListType, efn, etas, f)
}

func _containerAF(t *testing.T, r Reader, et Type, efn *SymbolToken, etas []SymbolToken, f containerhandler) {
	_nextAF(t, r, et, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.%v", et, et)

	require.NoError(t, r.StepIn())

	f(t, r)

	require.NoError(t, r.StepOut())
}

func _int(t *testing.T, r Reader, eval int) {
	_intAF(t, r, nil, nil, eval)
}

func _intAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval int) {
	_nextAF(t, r, IntType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.int", eval)

	size, err := r.IntSize()
	require.NoError(t, err)
	assert.Equal(t, Int32, size)

	val, err := r.IntValue()
	require.NoError(t, err)
	assert.Equal(t, eval, *val)
}

func _int64(t *testing.T, r Reader, eval int64) {
	_int64AF(t, r, nil, nil, eval)
}

func _int64AF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval int64) {
	_nextAF(t, r, IntType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.int", eval)

	size, err := r.IntSize()
	require.NoError(t, err)
	assert.Equal(t, Int64, size)

	val, err := r.Int64Value()
	require.NoError(t, err)
	assert.Equal(t, eval, *val)
}

func _bigInt(t *testing.T, r Reader, eval *big.Int) {
	_bigIntAF(t, r, nil, nil, eval)
}

func _bigIntAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval *big.Int) {
	_nextAF(t, r, IntType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.int", eval)

	size, err := r.IntSize()
	require.NoError(t, err)
	assert.Equal(t, BigInt, size)

	val, err := r.BigIntValue()
	require.NoError(t, err)
	assert.True(t, val.Cmp(eval) == 0, "expected %v, got %v", eval, val)
}

func _float(t *testing.T, r Reader, eval float64) {
	_floatAF(t, r, nil, nil, eval)
}

func _floatAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval float64) {
	_nextAF(t, r, FloatType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.float", eval)

	val, err := r.FloatValue()
	require.NoError(t, err)

	if math.IsNaN(eval) {
		assert.True(t, math.IsNaN(*val), "expected %v, got %v", eval, val)
	} else {
		assert.Equal(t, eval, *val)
	}
}

func _decimal(t *testing.T, r Reader, eval *Decimal) {
	_decimalAF(t, r, nil, nil, eval)
}

func _decimalAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval *Decimal) {
	_nextAF(t, r, DecimalType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.decimal", eval)

	val, err := r.DecimalValue()
	require.NoError(t, err)

	assert.True(t, eval.Equal(val), "expected %v, got %v", eval, val)
}

func _timestamp(t *testing.T, r Reader, eval Timestamp) {
	_timestampAF(t, r, nil, nil, eval)
}

func _timestampAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval Timestamp) {
	_nextAF(t, r, TimestampType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.timestamp", eval)

	val, err := r.TimestampValue()
	require.NoError(t, err)

	assert.True(t, eval.Equal(*val), "expected %v, got %v", eval, val)
}

func _string(t *testing.T, r Reader, eval *string) {
	_stringAF(t, r, nil, nil, eval)
}

func _stringAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval *string) {
	_nextAF(t, r, StringType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.string", eval)

	val, err := r.StringValue()
	require.NoError(t, err)

	if eval == nil {
		assert.Equal(t, eval, val)
	} else {
		assert.Equal(t, *eval, *val)
	}
}

// _symbolAF calls reader.next and asserts the expected symbol value. This function also asserts the value has neither
// annotation or field name.
func _symbol(t *testing.T, r Reader, evalst SymbolToken) {
	_symbolAF(t, r, nil, nil, &evalst, false, false)
}

// _symbolAF calls reader.next and asserts the expected symbol value, annotation, and field name.
func _symbolAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, evalSt *SymbolToken, isSymbolValueError bool, isNextError bool) {
	if !isNextError {
		_nextAF(t, r, SymbolType, efn, etas)
	} else {
		r.Next()
	}

	if evalSt != nil {
		require.False(t, r.IsNull())
	}

	symbolVal, err := r.SymbolValue()
	if isSymbolValueError {
		require.Error(t, err)
	} else {
		require.NoError(t, err)

		if evalSt == nil {
			assert.True(t, symbolVal == nil, "expected %v, got %v", &evalSt, symbolVal)
		} else {
			assert.True(t, symbolVal.Equal(evalSt), "expected %v, got %v", &evalSt, symbolVal)
		}
	}
}

func _bool(t *testing.T, r Reader, eval bool) {
	_boolAF(t, r, nil, nil, eval)
}

func _boolAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval bool) {
	_nextAF(t, r, BoolType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.bool", eval)

	val, err := r.BoolValue()
	require.NoError(t, err)
	assert.Equal(t, eval, *val, "expected %v, got %v", eval, val)
}

func _clob(t *testing.T, r Reader, eval []byte) {
	_clobAF(t, r, nil, nil, eval)
}

func _clobAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval []byte) {
	_nextAF(t, r, ClobType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.clob", eval)

	val, err := r.ByteValue()
	require.NoError(t, err)
	assert.True(t, bytes.Equal(val, eval), "expected %v, got %v", eval, val)
}

func _blob(t *testing.T, r Reader, eval []byte) {
	_blobAF(t, r, nil, nil, eval)
}

func _blobAF(t *testing.T, r Reader, efn *SymbolToken, etas []SymbolToken, eval []byte) {
	_nextAF(t, r, BlobType, efn, etas)
	require.False(t, r.IsNull(), "expected %v, got null.blob", eval)

	val, err := r.ByteValue()
	require.NoError(t, err)
	assert.True(t, bytes.Equal(val, eval), "expected %v, got %v", eval, val)
}

func _null(t *testing.T, r Reader, et Type) {
	_nullAF(t, r, et, nil, nil)
}

func _nullAF(t *testing.T, r Reader, et Type, efn *SymbolToken, etas []SymbolToken) {
	_nextAF(t, r, et, efn, etas)
	assert.True(t, r.IsNull())
}

func _next(t *testing.T, r Reader, et Type) {
	_nextAF(t, r, et, nil, nil)
}

func _nextAF(t *testing.T, r Reader, et Type, efn *SymbolToken, etas []SymbolToken) {
	require.True(t, r.Next(), "r.Next() failed with error: %v", r.Err())
	require.Equal(t, et, r.Type())

	fn, err := r.FieldName()
	assert.NoError(t, err)

	if efn != nil && fn != nil {
		assert.True(t, efn.Equal(fn), "expected fieldname=%v, got %v", *efn, *fn)
	}

	annotations, err := r.Annotations()
	assert.NoError(t, err)

	assert.True(t, _symbolTokenEquals(etas, annotations), "expected type annotations=%v, got %v", etas, annotations)
}

func _nextA(t *testing.T, r Reader, etas []SymbolToken, isAnnotationError bool, isNextError bool) {
	if !r.Next() && !isNextError {
		t.Fatal(r.Err())
	}

	annotations, err := r.Annotations()

	if isAnnotationError {
		require.Error(t, err)
	} else {
		assert.True(t, _symbolTokenEquals(etas, annotations), "expected type annotations=%v, got %v", etas, annotations)
	}
}

func _nextF(t *testing.T, r Reader, efns *SymbolToken, isFieldNameError bool, isNextError bool) {
	if !r.Next() && !isNextError {
		t.Fatal(r.Err())
	}

	fn, err := r.FieldName()

	if isFieldNameError {
		require.Error(t, err, "fieldName did not return an error")
	} else {
		assert.NoError(t, err)
		if efns != nil {
			assert.True(t, efns.Equal(fn), "expected fieldnamesymbol=%v, got %v", efns, fn)
		} else {
			assert.Nil(t, fn, "expected fieldnamesymbol=%v, got %v", efns, fn)
		}
	}
}

func _symbolTokenEquals(a, b []SymbolToken) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if !a[i].Equal(&b[i]) {
			return false
		}
	}

	return true
}

func _eof(t *testing.T, r Reader) {
	require.False(t, r.Next())
	require.NoError(t, r.Err())
}
