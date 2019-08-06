package ion

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"testing"
)

func TestPackUint(t *testing.T) {
	test := func(val uint64, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			len := uintLen(val)
			if len != elen {
				t.Errorf("expected len=%v, got len=%v", elen, len)
			}

			bits := packUint(val)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	test(0, 1, []byte{0})
	test(0xFF, 1, []byte{0xFF})
	test(0x1FF, 2, []byte{0x01, 0xFF})
	test(math.MaxUint64, 8, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
}

func TestPackInt(t *testing.T) {
	test := func(val int64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			bits := packInt(val)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	test(0, []byte{})
	test(0x7F, []byte{0x7F})
	test(-0x7F, []byte{0xFF})

	test(0xFF, []byte{0x00, 0xFF})
	test(-0xFF, []byte{0x80, 0xFF})

	test(0x7FFF, []byte{0x7F, 0xFF})
	test(-0x7FFF, []byte{0xFF, 0xFF})

	test(math.MaxInt64, []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	test(-math.MaxInt64, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	test(math.MinInt64, []byte{0x80, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func TestPackBigInt(t *testing.T) {
	test := func(val *big.Int, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			bits := packBigInt(val)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	test(big.NewInt(0), []byte{})
	test(big.NewInt(0x7F), []byte{0x7F})
	test(big.NewInt(-0x7F), []byte{0xFF})

	test(big.NewInt(0xFF), []byte{0x00, 0xFF})
	test(big.NewInt(-0xFF), []byte{0x80, 0xFF})

	test(big.NewInt(0x7FFF), []byte{0x7F, 0xFF})
	test(big.NewInt(-0x7FFF), []byte{0xFF, 0xFF})
}

func TestPackVarUint(t *testing.T) {
	test := func(val uint64, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			len := varUintLen(val)
			if len != elen {
				t.Errorf("expected len=%v, got len=%v", elen, len)
			}

			bits := packVarUint(val)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	test(0, 1, []byte{0x80})
	test(0x7F, 1, []byte{0xFF})
	test(0xFF, 2, []byte{0x01, 0xFF})
	test(0x1FF, 2, []byte{0x03, 0xFF})
	test(0x3FFF, 2, []byte{0x7F, 0xFF})
	test(0x7FFF, 3, []byte{0x01, 0x7F, 0xFF})
	test(0x7FFFFFFFFFFFFFFF, 9, []byte{0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0xFF})
	test(0xFFFFFFFFFFFFFFFF, 10, []byte{0x01, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0xFF})
}

func TestPackVarInt(t *testing.T) {
	test := func(val int64, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			len := varIntLen(val)
			if len != elen {
				t.Errorf("expected len=%v, got len=%v", elen, len)
			}

			bits := packVarInt(val)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	test(0, 1, []byte{0x80})

	test(0x3F, 1, []byte{0xBF}) // 1011 1111
	test(-0x3F, 1, []byte{0xFF})

	test(0x7F, 2, []byte{0x00, 0xFF})
	test(-0x7F, 2, []byte{0x40, 0xFF})

	test(0x1FFF, 2, []byte{0x3F, 0xFF})
	test(-0x1FFF, 2, []byte{0x7F, 0xFF})

	test(0x3FFF, 3, []byte{0x00, 0x7F, 0xFF})
	test(-0x3FFF, 3, []byte{0x40, 0x7F, 0xFF})

	test(0x3FFFFFFFFFFFFFFF, 9, []byte{0x3F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0xFF})
	test(-0x3FFFFFFFFFFFFFFF, 9, []byte{0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0xFF})

	test(math.MaxInt64, 10, []byte{0x00, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0xFF})
	test(-math.MaxInt64, 10, []byte{0x40, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0xFF})
	test(math.MinInt64, 10, []byte{0x41, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80})
}
