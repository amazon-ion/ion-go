package ion

import (
	"math"
	"math/big"
	"testing"
)

func TestReadBinaryStructs(t *testing.T) {
	r := readBinary([]byte{
		0xD0,                   // {}
		0xDF,                   // null.struct
		0xEA, 0x81, 0xEE, 0xD7, // foo::{
		0x84, 0xE3, 0x81, 0xEF, 0x0F, // name:bar::null,
		0x88, 0x20, // max_id:0
		// }
	})

	_next(t, r, StructType)
	_null(t, r, StructType)
	_nextAF(t, r, StructType, "", []string{"foo"})
	_eof(t, r)
}

func TestReadBinaryDecimals(t *testing.T) {
	r := readBinary([]byte{
		0x50,       // 0.
		0x5F,       // null.decimal
		0x51, 0xC3, // 0.000, aka 0 x 10^-3
		0x53, 0xC3, 0x03, 0xE8, // 1.000, aka 1000 x 10^-3
		0x53, 0xC3, 0x83, 0xE8, // -1.000, aka -1000 x 10^-3
		0x53, 0x00, 0xE4, 0x01, // 1d100, aka 1 * 10^100
		0x53, 0x00, 0xE4, 0x81, // -1d100, aka -1 * 10^100
	})

	_decimal(t, r, MustParseDecimal("0."))
	_null(t, r, DecimalType)
	_decimal(t, r, MustParseDecimal("0.000"))
	_decimal(t, r, MustParseDecimal("1.000"))
	_decimal(t, r, MustParseDecimal("-1.000"))
	_decimal(t, r, MustParseDecimal("1d100"))
	_decimal(t, r, MustParseDecimal("-1d100"))
	_eof(t, r)
}

func TestReadBinaryFloats(t *testing.T) {
	r := readBinary([]byte{
		0x40,                                                 // 0
		0x4F,                                                 // null.float
		0x48, 0x7F, 0xEF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // MaxFloat64
		0x48, 0xFF, 0xEF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // -MaxFloat64
		0x48, 0x7F, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // +inf
		0x48, 0xFF, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // -inf
		0x48, 0x7F, 0xF8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // NaN
	})

	_float(t, r, 0)
	_null(t, r, FloatType)
	_float(t, r, math.MaxFloat64)
	_float(t, r, -math.MaxFloat64)
	_float(t, r, math.Inf(1))
	_float(t, r, math.Inf(-1))
	_float(t, r, math.NaN())
	_eof(t, r)
}

func TestReadBinaryInts(t *testing.T) {
	r := readBinary([]byte{
		0x20,       // 0
		0x2F,       // null.int
		0x21, 0x01, // 1
		0x31, 0x01, // -1
		0x28, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // 0x7FFFFFFFFFFFFFFF
		0x38, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // -0x7FFFFFFFFFFFFFFF
		0x28, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0x8000000000000000
		0x38, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 0x8000000000000000
	})

	_int(t, r, 0)
	_null(t, r, IntType)
	_int(t, r, 1)
	_int(t, r, -1)
	_int64(t, r, math.MaxInt64)
	_int64(t, r, -math.MaxInt64)

	i := new(big.Int).SetUint64(math.MaxInt64 + 1)
	_bigInt(t, r, i)
	_bigInt(t, r, new(big.Int).Neg(i))

	_eof(t, r)
}

func TestReadBinaryBools(t *testing.T) {
	r := readBinary([]byte{
		0x10, // false
		0x11, // true
		0x1F, // null.bool
	})

	_bool(t, r, false)
	_bool(t, r, true)
	_null(t, r, BoolType)
	_eof(t, r)
}

func readBinary(ion []byte) Reader {
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
	return NewBinaryReaderBytes(append(prefix, ion...), nil)
}
