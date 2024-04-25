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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalText(t *testing.T) {
	test := func(v interface{}, eval string) {
		t.Run(eval, func(t *testing.T) {
			val, err := MarshalText(v)
			require.NoError(t, err)
			assert.Equal(t, eval, string(val))
		})
	}

	test(nil, "null")
	test(true, "true")
	test(false, "false")

	test(byte(42), "42")
	test(-42, "-42")
	test(uint64(math.MaxUint64), "18446744073709551615")
	bigIntPos, _ := new(big.Int).SetString("123456789012345678901234567890", 10)
	test(bigIntPos, "123456789012345678901234567890")
	bigIntNeg, _ := new(big.Int).SetString("-123456789012345678901234567890", 10)
	test(bigIntNeg, "-123456789012345678901234567890")
	test(math.MinInt64, "-9223372036854775808")

	test(42.0, "4.2e+1")
	test(math.Inf(1), "+inf")
	test(math.Inf(-1), "-inf")
	test(math.NaN(), "nan")

	test(MustParseDecimal("1.20"), "1.20")
	test(NewTimestamp(time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), TimestampPrecisionSecond, TimezoneUTC),
		"2010-01-01T00:00:00Z")
	test(time.Date(2010, 1, 1, 0, 0, 0, 1, time.UTC), "2010-01-01T00:00:00.000000001Z")
	test(time.Date(2010, 1, 1, 0, 0, 0, 770000000, time.UTC), "2010-01-01T00:00:00.770000000Z")
	loc, _ := time.LoadLocation("EST")
	test(time.Date(2010, 1, 1, 0, 0, 0, 0, loc), "2010-01-01T00:00:00.000000000-05:00")
	loc = time.FixedZone("UTC+8", 8*60*60)
	test(time.Date(2010, 1, 1, 0, 0, 0, 0, loc), "2010-01-01T00:00:00.000000000+08:00")

	test("hello\tworld", "\"hello\\tworld\"")

	test(struct{ A, B int }{42, 0}, "{A:42,B:0}")
	test(struct {
		A int `ion:"val,ignoreme"`
		B int `ion:"-"`
		C int `ion:",omitempty"`
		d int
	}{42, 0, 0, 0}, "{val:42}")

	test(struct{ V interface{} }{}, "{V:null}")
	test(struct{ V interface{} }{"42"}, "{V:\"42\"}")

	fortyTwo := 42

	test(struct{ V *int }{}, "{V:null}")
	test(struct{ V *int }{&fortyTwo}, "{V:42}")

	test(map[string]int{"b": 2, "a": 1}, "{a:1,b:2}")

	test(struct{ V []int }{}, "{V:null}")
	test(struct{ V []int }{[]int{4, 2}}, "{V:[4,2]}")

	test(struct{ V []byte }{}, "{V:null}")
	test(struct{ V []byte }{[]byte{4, 2}}, "{V:{{BAI=}}}")

	test(struct{ V [2]byte }{[2]byte{4, 2}}, "{V:[4,2]}")
}

func TestMarshalBinary(t *testing.T) {
	test := func(v interface{}, name string, eval []byte) {
		t.Run(name, func(t *testing.T) {
			val, err := MarshalBinary(v)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(val, eval), "expected '%v', got '%v'", fmtbytes(eval), fmtbytes(val))
		})
	}

	// Null.
	test(nil, "null", prefixIVM([]byte{0x0F}))

	// Boolean.
	test(true, "boolean", prefixIVM([]byte{0x11})) // true

	// Positive Integer.
	test(32767, "integer", prefixIVM([]byte{0x22, 0x7F, 0xFF})) // 32767

	// Negative Integer.
	test(-32767, "negative integer", prefixIVM([]byte{0x32, 0x7F, 0xFF})) // -32767

	// Float32 valid type. Go treats floats as float64 by default, unless specified.
	// Explicitly cast number to be of float32 and ensure type is handled. This should not be an unknown type.
	test(float32(math.MaxFloat32), "float32 valid type", prefixIVM([]byte{0x44, 0x7F, 0x7F, 0xFF, 0xFF})) // 3.40282346638528859811704183484516925440e+38

	// Float32. Ensure number can be represented losslessly as a float32 by testing that byte length is 5.
	// This should not be represented as a float64.
	test(math.MaxFloat32, "float32", prefixIVM([]byte{0x44, 0x7F, 0x7F, 0xFF, 0xFF})) // 3.40282346638528859811704183484516925440e+38

	// Float 64.
	test(math.MaxFloat64, "float64", prefixIVM([]byte{0x48, 0x7F, 0xEF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})) // 1.797693134862315708145274237317043567981e+308

	// Decimal.
	test(MustParseDecimal("0d-63"), "decimal", prefixIVM([]byte{0x51, 0xFF})) // 0d-63

	// String.
	test("abc", "string", prefixIVM([]byte{0x83, 'a', 'b', 'c'})) // "abc"

	// Blob.
	test([]byte{97, 98, 99}, "blob", prefixIVM([]byte{0xA3, 'a', 'b', 'c'}))

	// List.
	test([]int{2, 3, 4}, "list", prefixIVM([]byte{0xB6, 0x21, 0x02, 0x21, 0x03, 0x21, 0x04}))

	// Struct.
	test(struct{ A, B int }{42, 0}, "{A:42,B:0}", prefixIVM([]byte{
		0xE9, 0x81, 0x83, 0xD6, 0x87, 0xB4, 0x81, 'A', 0x81, 'B',
		0xD5,
		0x8A, 0x21, 0x2A,
		0x8B, 0x20,
	}))

	test(time.Date(2010, 1, 1, 0, 0, 0, 1, time.UTC), "time with 1 nanosecond", prefixIVM([]byte{
		0x6A, 0x80, 0x0F, 0xDA, 0x81, 0x81, 0x80, 0x80, 0x80,
		0xC9, // exponent: 9
		0x01, // coefficient: 1
	}))

	test(time.Date(2010, 1, 1, 0, 0, 0, 5000, time.UTC), "time with 5000 nanoseconds", prefixIVM([]byte{
		0x6B, 0x80, 0x0F, 0xDA, 0x81, 0x81, 0x80, 0x80, 0x80,
		0xC9,       // exponent: 9
		0x13, 0x88, // coefficient: 5000
	}))
}

func prefixIVM(data []byte) []byte {
	prefix := []byte{0xE0, 0x01, 0x00, 0xEA} // $ion_1_0
	return append(prefix, data...)
}

func TestMarshalBinaryLST(t *testing.T) {
	lsta := NewLocalSymbolTable(nil, nil)
	lstb := NewLocalSymbolTable(nil, []string{
		"A", "B",
	})

	test := func(v interface{}, name string, lst SymbolTable, eval []byte) {
		t.Run(name, func(t *testing.T) {
			val, err := MarshalBinaryLST(v, lst)
			require.NoError(t, err)
			assert.True(t, bytes.Equal(val, eval), "expected '%v', got '%v'", fmtbytes(eval), fmtbytes(val))
		})
	}

	test(nil, "null", lsta, []byte{0xE0, 0x01, 0x00, 0xEA, 0x0F})
	test(struct{ A, B int }{42, 0}, "{A:42,B:0}", lstb, []byte{
		0xE0, 0x01, 0x00, 0xEA,
		0xE9, 0x81, 0x83, 0xD6, 0x87, 0xB4, 0x81, 'A', 0x81, 'B',
		0xD5,
		0x8A, 0x21, 0x2A,
		0x8B, 0x20,
	})
}

func TestMarshalNestedStructs(t *testing.T) {
	type gp struct {
		A int `ion:"a"`
	}

	type gp2 struct {
		B int `ion:"b"`
	}

	type parent struct {
		gp
		*gp2
		C int `ion:"c"`
	}

	type root struct {
		parent
		D int `ion:"d"`
	}

	v := root{
		parent: parent{
			gp: gp{
				A: 1,
			},
			gp2: &gp2{
				B: 2,
			},
			C: 3,
		},
		D: 4,
	}

	val, err := MarshalText(v)
	require.NoError(t, err)

	eval := "{a:1,b:2,c:3,d:4}"
	assert.Equal(t, eval, string(val))
}

func TestMarshalHints(t *testing.T) {
	type hints struct {
		String  string            `ion:"str,omitempty,string"`
		Symbol  string            `ion:"sym,omitempty,symbol"`
		Strings []string          `ion:"strs,string"`
		Symbols []string          `ion:"syms,symbol"`
		StrMap  map[string]string `ion:"strm"`
		SymMap  map[string]string `ion:"symm,symbol"`
		Blob    []byte            `ion:"bl,blob,omitempty"`
		Clob    []byte            `ion:"cl,clob,omitempty"`
		Sexp    []int             `ion:"sx,sexp"`
	}

	v := hints{
		String:  "string",
		Symbol:  "symbol",
		Strings: []string{"a", "b"},
		Symbols: []string{"c", "d"},
		StrMap:  map[string]string{"a": "b"},
		SymMap:  map[string]string{"c": "d"},
		Blob:    []byte("blob"),
		Clob:    []byte("clob"),
		Sexp:    []int{1, 2, 3},
	}

	val, err := MarshalText(v)
	require.NoError(t, err)

	eval := `{` +
		`str:"string",` +
		`sym:symbol,` +
		`strs:["a","b"],` +
		`syms:[c,d],` +
		`strm:{a:"b"},` +
		`symm:{c:d},` +
		`bl:{{YmxvYg==}},` +
		`cl:{{"clob"}},` +
		`sx:(1 2 3)` +
		`}`

	assert.Equal(t, eval, string(val))
}

type marshalMe uint8

var _ Marshaler = marshalMe(0)

const (
	one marshalMe = iota
	two
	three
	four
)

func (m marshalMe) String() string {
	switch m {
	case one:
		return "ONE"
	case two:
		return "TWO"
	case three:
		return "THREE"
	case four:
		return "FOUR"
	default:
		panic("unexpected value")
	}
}

func (m marshalMe) MarshalIon(w Writer) error {
	return w.WriteSymbolFromString(m.String())
}

func TestMarshalCustomMarshaler(t *testing.T) {
	buf := strings.Builder{}
	enc := NewTextEncoder(&buf)

	require.NoError(t, enc.Encode(one))
	require.NoError(t, enc.EncodeAs([]marshalMe{two, three}, SexpType))

	v := struct {
		Num marshalMe `ion:"num"`
	}{four}
	require.NoError(t, enc.Encode(v))
	require.NoError(t, enc.Finish())

	val := buf.String()
	eval := "ONE\n(TWO THREE)\n{num:FOUR}\n"

	assert.Equal(t, eval, val)
}

func TestMarshalValuesWithAnnotation(t *testing.T) {
	test := func(v interface{}, testName, eval string) {
		t.Run(testName, func(t *testing.T) {
			val, err := MarshalText(v)
			require.NoError(t, err)
			assert.Equal(t, eval, string(val))
		})
	}

	type foo struct {
		Value   interface{}
		AnyName []SymbolToken `ion:",annotations"`
	}

	buildValue := func(val interface{}) foo {
		return foo{val, []SymbolToken{NewSymbolTokenFromString("symbols or string"), NewSymbolTokenFromString("annotations")}}
	}

	test(buildValue(nil), "null", "'symbols or string'::annotations::null")
	test(buildValue(true), "bool", "'symbols or string'::annotations::true")
	test(buildValue(5), "int", "'symbols or string'::annotations::5")
	test(buildValue(float32(math.MaxFloat32)), "float", "'symbols or string'::annotations::3.4028234663852886e+38")
	test(buildValue(MustParseDecimal("1.2")), "decimal", "'symbols or string'::annotations::1.2")
	test(buildValue(NewTimestamp(time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC), TimestampPrecisionSecond, TimezoneUTC)),
		"timestamp", "'symbols or string'::annotations::2000-01-02T03:04:05Z")
	test(buildValue("stringValue"), "string", "'symbols or string'::annotations::\"stringValue\"")
	test(buildValue([]byte{4, 2}), "blob", "'symbols or string'::annotations::{{BAI=}}")
	test(buildValue([]int{3, 5, 7}), "list", "'symbols or string'::annotations::[3,5,7]")
	test(buildValue(map[string]int{"b": 2, "a": 1}), "struct", "'symbols or string'::annotations::{a:1,b:2}")
}
