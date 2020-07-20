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
		w.BeginStruct()
		w.EndStruct()

		w.Annotation("foo")
		w.BeginStruct()
		{
			w.FieldName("name")
			w.Annotation("bar")
			w.WriteNull()

			w.FieldName("max_id")
			w.WriteInt(0)
		}
		w.EndStruct()
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
		w.BeginSexp()
		w.EndSexp()

		w.Annotation("foo")
		w.BeginSexp()
		{
			w.Annotation("bar")
			w.WriteNull()

			w.WriteInt(0)
		}
		w.EndSexp()
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
		w.BeginList()
		w.EndList()

		w.Annotation("foo")
		w.BeginList()
		{
			w.Annotation("bar")
			w.WriteNull()

			w.WriteInt(0)
		}
		w.EndList()
	})
}

func TestWriteBinaryBlob(t *testing.T) {
	eval := []byte{
		0xA0,
		0xAB, 'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l', 'd',
	}
	testBinaryWriter(t, eval, func(w Writer) {
		w.WriteBlob([]byte{})
		w.WriteBlob([]byte("Hello World"))
	})
}

func TestWriteLargeBinaryBlob(t *testing.T) {
	eval := make([]byte, 131)
	eval[0] = 0xAE
	eval[1] = 0x01
	eval[2] = 0x80
	testBinaryWriter(t, eval, func(w Writer) {
		w.WriteBlob(make([]byte, 128))
	})
}

func TestWriteBinaryClob(t *testing.T) {
	eval := []byte{
		0x90,
		0x9B, 'H', 'e', 'l', 'l', 'o', ' ', 'W', 'o', 'r', 'l', 'd',
	}
	testBinaryWriter(t, eval, func(w Writer) {
		w.WriteClob([]byte{})
		w.WriteClob([]byte("Hello World"))
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
		w.WriteString("")
		w.WriteString("Hello World")
		w.WriteString("Hello World But Even Longer")
		w.WriteString("\xE0\x01\x00\xEA")
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
		w.WriteSymbol("$ion")
		w.WriteSymbol("name")
		w.WriteSymbol("version")
		w.WriteSymbol("$ion_shared_symbol_table")
		w.WriteSymbol("$4294967295")
	})
}

func TestWriteBinaryTimestamp(t *testing.T) {
	eval := []byte{
		0x67, 0x80, 0x81, 0x81, 0x81, 0x80, 0x80, 0x80, // 0001-01-01T00:00:00Z
		0x6E, 0x8E, // 0x0E-bit timestamp
		0x04, 0xD8, // offset: +600 minutes (+10:00)
		0x0F, 0xE3, // year:   2019
		0x88,                   // month:  8
		0x84,                   // day:    4
		0x88,                   // hour:   8 utc (18 local)
		0x8F,                   // minute: 15
		0xAB,                   // second: 43
		0xC9,                   // exp:    -9
		0x33, 0x77, 0xDF, 0x70, // nsec:   863494000
	}

	nowish, _ := time.Parse(time.RFC3339Nano, "2019-08-04T18:15:43.863494+10:00")

	testBinaryWriter(t, eval, func(w Writer) {
		w.WriteTimestamp(Timestamp{time.Time{}, Second})
		w.WriteTimestamp(Timestamp{nowish, Second})
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
		w.WriteDecimal(MustParseDecimal("0."))
		w.WriteDecimal(MustParseDecimal("0.0"))
		w.WriteDecimal(MustParseDecimal("0.000"))
		w.WriteDecimal(MustParseDecimal("1.000"))
		w.WriteDecimal(MustParseDecimal("-1.000"))
		w.WriteDecimal(MustParseDecimal("1d100"))
		w.WriteDecimal(MustParseDecimal("-1d100"))
	})
}

func TestWriteBinaryFloats(t *testing.T) {
	eval := []byte{
		0x40,                                                 // 0
		0x48, 0x7F, 0xEF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // MaxFloat64
		0x48, 0xFF, 0xEF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // -MaxFloat64
		0x48, 0x7F, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // +inf
		0x48, 0xFF, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // -inf
		0x48, 0x7F, 0xF8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // NaN
	}
	testBinaryWriter(t, eval, func(w Writer) {
		w.WriteFloat(0)
		w.WriteFloat(math.MaxFloat64)
		w.WriteFloat(-math.MaxFloat64)
		w.WriteFloat(math.Inf(1))
		w.WriteFloat(math.Inf(-1))
		w.WriteFloat(math.NaN())
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
		w.WriteBigInt(big.NewInt(0))
		w.WriteBigInt(big.NewInt(0xFF))
		w.WriteBigInt(big.NewInt(-0xFF))
		w.WriteBigInt(new(big.Int).SetBytes([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}))
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
		w.WriteBigInt(i)
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
		w.WriteInt(0)
		w.WriteInt(0xFF)
		w.WriteInt(-0xFF)
		w.WriteInt(0xFFFF)
		w.WriteInt(-0xFFFFFF)
		w.WriteInt(math.MaxInt64)
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
		w.Annotations("name", "version")
		w.WriteBool(false)
	})
}

func TestWriteBinaryBools(t *testing.T) {
	eval := []byte{
		0x10, // false
		0x11, // true
	}

	testBinaryWriter(t, eval, func(w Writer) {
		w.WriteBool(false)
		w.WriteBool(true)
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
		w.WriteNull()
		w.WriteNullType(BoolType)
		w.WriteNullType(IntType)
		w.WriteNullType(FloatType)
		w.WriteNullType(DecimalType)
		w.WriteNullType(TimestampType)
		w.WriteNullType(SymbolType)
		w.WriteNullType(StringType)
		w.WriteNullType(ClobType)
		w.WriteNullType(BlobType)
		w.WriteNullType(ListType)
		w.WriteNullType(SexpType)
		w.WriteNullType(StructType)
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

	if !bytes.Equal(val, eval) {
		t.Errorf("expected %v, got %v", fmtbytes(eval), fmtbytes(val))
	}
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
	bogusSyms := []string{}
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

	if err := w.Finish(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}
