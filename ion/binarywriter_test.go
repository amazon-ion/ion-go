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
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteBinaryStruct(t *testing.T) {
	eval := []byte{
		0xD0,                   // {}
		0xEA, 0x81, 0xEE, 0xD7, // foo::{
		0x84, 0xE3, 0x81, 0xEF, 0x0F, // name:bar::null,
		0x88, 0x20, // max_id:0
		// }
	}
	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.BeginStruct())
		assert.NoError(t, w.EndStruct())

		assert.NoError(t, w.Annotation(newSimpleSymbolToken("foo")))
		assert.NoError(t, w.BeginStruct())
		{
			assert.NoError(t, w.FieldName(newSimpleSymbolToken("name")))
			assert.NoError(t, w.Annotation(newSimpleSymbolToken("bar")))
			assert.NoError(t, w.WriteNull())

			assert.NoError(t, w.FieldName(newSimpleSymbolToken("max_id")))
			assert.NoError(t, w.WriteInt(0))
		}
		assert.NoError(t, w.EndStruct())
	})
}

func TestWriteBinarySexp(t *testing.T) {
	eval := []byte{
		0xC0,                   // ()
		0xE8, 0x81, 0xEE, 0xC5, // foo::(
		0xE3, 0x81, 0xEF, 0x0F, // bar::null,
		0x20, // 0
		// )
	}
	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.EndSexp())

		assert.NoError(t, w.Annotation(newSimpleSymbolToken("foo")))
		assert.NoError(t, w.BeginSexp())
		{
			assert.NoError(t, w.Annotation(newSimpleSymbolToken("bar")))
			assert.NoError(t, w.WriteNull())

			assert.NoError(t, w.WriteInt(0))
		}
		assert.NoError(t, w.EndSexp())
	})
}

func TestWriteBinaryList(t *testing.T) {
	eval := []byte{
		0xB0,                   // []
		0xE8, 0x81, 0xEE, 0xB5, // foo::[
		0xE3, 0x81, 0xEF, 0x0F, // bar::null,
		0x20, // 0
		// ]
	}
	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.BeginList())
		assert.NoError(t, w.EndList())

		assert.NoError(t, w.Annotation(newSimpleSymbolToken("foo")))
		assert.NoError(t, w.BeginList())
		{
			assert.NoError(t, w.Annotation(newSimpleSymbolToken("bar")))
			assert.NoError(t, w.WriteNull())

			assert.NoError(t, w.WriteInt(0))
		}
		assert.NoError(t, w.EndList())
	})
}

func TestWriteBinaryBlob(t *testing.T) {
	eval := []byte{
		0xA0,
		0xAB, 'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l', 'd',
	}
	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteBlob([]byte{}))
		assert.NoError(t, w.WriteBlob([]byte("Hello World")))
	})
}

func TestWriteLargeBinaryBlob(t *testing.T) {
	eval := make([]byte, 131)
	eval[0] = 0xAE
	eval[1] = 0x01
	eval[2] = 0x80
	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteBlob(make([]byte, 128)))
	})
}

func TestWriteBinaryClob(t *testing.T) {
	eval := []byte{
		0x90,
		0x9B, 'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l', 'd',
	}
	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteClob([]byte{}))
		assert.NoError(t, w.WriteClob([]byte("Hello World")))
	})
}

func TestWriteBinaryString(t *testing.T) {
	eval := []byte{
		0x80, // ""
		0x8B, 'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l', 'd',
		0x8E, 0x9B, 'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l', 'd',
		' ', 'B', 'u', 't', ' ', 'E', 'v', 'e', 'n', ' ', 'L', 'o', 'n', 'g', 'e', 'r',
		0x84, 0xE0, 0x01, 0x00, 0xEA,
	}
	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteString(""))
		assert.NoError(t, w.WriteString("Hello World"))
		assert.NoError(t, w.WriteString("Hello World But Even Longer"))
		assert.NoError(t, w.WriteString("\xE0\x01\x00\xEA"))
	})
}

func TestWriteBinarySymbol(t *testing.T) {
	eval := []byte{
		0x71, 0x01, // $ion
		0x71, 0x04, // name
		0x71, 0x05, // version
		0x71, 0x09, // $ion_shared_symbol_table
		0x74, 0xFF, 0xFF, 0xFF, 0xFF, // $4294967295
	}
	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteSymbolFromString("$ion"))
		assert.NoError(t, w.WriteSymbolFromString("name"))
		assert.NoError(t, w.WriteSymbolFromString("version"))
		assert.NoError(t, w.WriteSymbolFromString("$ion_shared_symbol_table"))
		assert.NoError(t, w.WriteSymbolFromString("$4294967295"))
	})
}

func TestWriteBinaryTimestamp(t *testing.T) {
	eval := []byte{
		0x67, 0x80, 0x81, 0x81, 0x81, 0x80, 0x80, 0x80, // 0001-01-01T00:00:00Z
		0x6D,       // 0x0D-byte timestamp
		0x04, 0xD8, // offset: +600 minutes (+10:00)
		0x0F, 0xE3, // year:   2019
		0x88,             // month:  8
		0x84,             // day:    4
		0x88,             // hour:   8 utc (18 local)
		0x8F,             // minute: 15
		0xAB,             // second: 43
		0xC6,             // exp:    6 precision units
		0x0D, 0x2D, 0x06, // nsec:   863494
	}

	nowish, _ := NewTimestampFromStr("2019-08-04T18:15:43.863494+10:00", TimestampPrecisionNanosecond, TimezoneLocal)

	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteTimestamp(NewTimestamp(time.Time{}, TimestampPrecisionNanosecond, TimezoneUTC)))
		assert.NoError(t, w.WriteTimestamp(nowish))
	})
}

func TestWriteBinaryDecimal(t *testing.T) {
	eval := []byte{
		0x50,       // 0.
		0x51, 0xC1, // 0.0, aka 0 x 10^-1
		0x51, 0xC3, // 0.000, aka 0 x 10^-3
		0x53, 0xC3, 0x03, 0xE8, // 1.000, aka 1000 x 10^-3
		0x53, 0xC3, 0x83, 0xE8, // -1.000, aka -1000 x 10^-3
		0x53, 0x00, 0xE4, 0x01, // 1d100, aka 1 * 10^100
		0x53, 0x00, 0xE4, 0x81, // -1d100, aka -1 * 10^100
	}

	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("0.")))
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("0.0")))
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("0.000")))
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("1.000")))
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("-1.000")))
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("1d100")))
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("-1d100")))
	})
}

func TestWriteBinaryFloats(t *testing.T) {
	eval := []byte{
		0x40,                                                 // 0
		0x48, 0x7F, 0xEF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // MaxFloat64
		0x48, 0xFF, 0xEF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // -MaxFloat64
		0x44, 0x7F, 0x80, 0x00, 0x00, // +inf (float32)
		0x44, 0xFF, 0x80, 0x00, 0x00, // -inf (float32)
		0x44, 0x7F, 0xC0, 0x00, 0x00, // NaN
	}

	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteFloat(0))
		assert.NoError(t, w.WriteFloat(math.MaxFloat64))
		assert.NoError(t, w.WriteFloat(-math.MaxFloat64))
		assert.NoError(t, w.WriteFloat(math.Inf(1)))
		assert.NoError(t, w.WriteFloat(math.Inf(-1)))
		assert.NoError(t, w.WriteFloat(math.NaN()))
	})
}

func TestWriteBinaryBigInts(t *testing.T) {
	eval := []byte{
		0x20,       // 0
		0x21, 0xFF, //  0xFF
		0x31, 0xFF, // -0xFF
		0x2E, 0x90, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // a really big integer
	}

	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteBigInt(big.NewInt(0)))
		assert.NoError(t, w.WriteBigInt(big.NewInt(0xFF)))
		assert.NoError(t, w.WriteBigInt(big.NewInt(-0xFF)))
		assert.NoError(t, w.WriteBigInt(new(big.Int).SetBytes([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})))
	})
}

func TestWriteBinaryReallyBigInts(t *testing.T) {
	eval := []byte{
		0x2E, 0x01, 0x80, // 128-byte positive integer
		0x80, // high bit set
	}
	eval = append(eval, make([]byte, 127)...)
	testBinaryWriter(t, eval, func(w Writer) {
		i := new(big.Int)
		i = i.SetBit(i, 1023, 1)
		assert.NoError(t, w.WriteBigInt(i))
	})
}

func TestWriteBinaryInts(t *testing.T) {
	eval := []byte{
		0x20,       // 0
		0x21, 0xFF, //  0xFF
		0x31, 0xFF, // -0xFF
		0x22, 0xFF, 0xFF, // 0xFFFF
		0x33, 0xFF, 0xFF, 0xFF, // -0xFFFFFF
		0x28, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // math.MaxInt64
	}

	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteInt(0))
		assert.NoError(t, w.WriteInt(0xFF))
		assert.NoError(t, w.WriteInt(-0xFF))
		assert.NoError(t, w.WriteInt(0xFFFF))
		assert.NoError(t, w.WriteInt(-0xFFFFFF))
		assert.NoError(t, w.WriteInt(math.MaxInt64))
	})
}

func TestWriteBinaryBoolAnnotated(t *testing.T) {
	eval := []byte{
		0xE4, // 4-byte annotated value
		0x82, // 2 bytes of annotations
		0x84, // $4 (name)
		0x85, // $5 (version)
		0x10, // false
	}

	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.Annotations(SymbolToken{Text: newString("name"), LocalSID: 4}, SymbolToken{Text: newString("version"), LocalSID: 5}))
		assert.NoError(t, w.WriteBool(false))
	})
}

func TestWriteBinaryBools(t *testing.T) {
	eval := []byte{
		0x10, // false
		0x11, // true
	}

	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteBool(false))
		assert.NoError(t, w.WriteBool(true))
	})
}

func TestWriteBinaryNulls(t *testing.T) {
	eval := []byte{
		0x0F,
		0x1F,
		0x2F,
		// 0x3F, // negative integer, not actually valid
		0x4F,
		0x5F,
		0x6F,
		0x7F,
		0x8F,
		0x9F,
		0xAF,
		0xBF,
		0xCF,
		0xDF,
	}

	testBinaryWriter(t, eval, func(w Writer) {
		assert.NoError(t, w.WriteNull())
		assert.NoError(t, w.WriteNullType(BoolType))
		assert.NoError(t, w.WriteNullType(IntType))
		assert.NoError(t, w.WriteNullType(FloatType))
		assert.NoError(t, w.WriteNullType(DecimalType))
		assert.NoError(t, w.WriteNullType(TimestampType))
		assert.NoError(t, w.WriteNullType(SymbolType))
		assert.NoError(t, w.WriteNullType(StringType))
		assert.NoError(t, w.WriteNullType(ClobType))
		assert.NoError(t, w.WriteNullType(BlobType))
		assert.NoError(t, w.WriteNullType(ListType))
		assert.NoError(t, w.WriteNullType(SexpType))
		assert.NoError(t, w.WriteNullType(StructType))
	})
}

func testBinaryWriter(t *testing.T, eval []byte, f func(w Writer)) {
	val := writeBinary(t, f)

	prefix := []byte{
		0xE0, 0x01, 0x00, 0xEA, // $ion_1_0
		0xEE, 0x9F, 0x81, 0x83, 0xDE, 0x9B, // $ion_symbol_table::{
		0x86, 0xBE, 0x8E, // imports:[
		0xDD,                                // {
		0x84, 0x85, 'b', 'o', 'g', 'u', 's', // name: "bogus"
		0x85, 0x21, 0x2A, // version: 42
		0x88, 0x21, 0x64, // max_id: 100
		// }]
		0x87, 0xB8, // symbols: [
		0x83, 'f', 'o', 'o', // "foo"
		0x83, 'b', 'a', 'r', // "bar"
		// ]
		// }
	}
	eval = append(prefix, eval...)

	assert.True(t, bytes.Equal(val, eval), "expected %v, got %v", fmtbytes(eval), fmtbytes(val))
}

func fmtbytes(bs []byte) string {
	buf := strings.Builder{}
	buf.WriteByte('[')
	for i, b := range bs {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(hex.EncodeToString([]byte{b}))
	}
	buf.WriteByte(']')
	return buf.String()
}

func writeBinary(t *testing.T, f func(w Writer)) []byte {
	var bogusSyms []string
	for i := 0; i < 100; i++ {
		bogusSyms = append(bogusSyms, fmt.Sprintf("bogus_sym_%v", i))
	}

	bogus := []SharedSymbolTable{
		NewSharedSymbolTable("bogus", 42, bogusSyms),
	}

	buf := bytes.Buffer{}
	w := NewBinaryWriterLST(&buf, NewLocalSymbolTable(bogus, []string{
		"foo",
		"bar",
	}))

	f(w)

	require.NoError(t, w.Finish())

	return buf.Bytes()
}
