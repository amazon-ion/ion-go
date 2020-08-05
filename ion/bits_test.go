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
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"
)

func TestAppendUint(t *testing.T) {
	test := func(val uint64, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			length := uintLen(val)
			if length != elen {
				t.Errorf("expected length=%v, got length=%v", elen, length)
			}

			bits := appendUint(nil, val)
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

func TestAppendInt(t *testing.T) {
	test := func(val int64, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			length := intLen(val)
			if length != elen {
				t.Errorf("expected length=%v, got length=%v", elen, length)
			}

			bits := appendInt(nil, val)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	test(0, 0, []byte{})
	test(0x7F, 1, []byte{0x7F})
	test(-0x7F, 1, []byte{0xFF})

	test(0xFF, 2, []byte{0x00, 0xFF})
	test(-0xFF, 2, []byte{0x80, 0xFF})

	test(0x7FFF, 2, []byte{0x7F, 0xFF})
	test(-0x7FFF, 2, []byte{0xFF, 0xFF})

	test(math.MaxInt64, 8, []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	test(-math.MaxInt64, 8, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	test(math.MinInt64, 9, []byte{0x80, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func TestAppendBigInt(t *testing.T) {
	test := func(val *big.Int, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			length := bigIntLen(val)
			if length != elen {
				t.Errorf("expected length=%v, got length=%v", elen, length)
			}

			bits := appendBigInt(nil, val)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	test(big.NewInt(0), 0, []byte{})
	test(big.NewInt(0x7F), 1, []byte{0x7F})
	test(big.NewInt(-0x7F), 1, []byte{0xFF})

	test(big.NewInt(0xFF), 2, []byte{0x00, 0xFF})
	test(big.NewInt(-0xFF), 2, []byte{0x80, 0xFF})

	test(big.NewInt(0x7FFF), 2, []byte{0x7F, 0xFF})
	test(big.NewInt(-0x7FFF), 2, []byte{0xFF, 0xFF})
}

func TestAppendVarUint(t *testing.T) {
	test := func(val uint64, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			length := varUintLen(val)
			if length != elen {
				t.Errorf("expected length=%v, got length=%v", elen, length)
			}

			bits := appendVarUint(nil, val)
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

func TestAppendVarInt(t *testing.T) {
	test := func(val int64, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val), func(t *testing.T) {
			length := varIntLen(val)
			if length != elen {
				t.Errorf("expected length=%v, got length=%v", elen, length)
			}

			bits := appendVarInt(nil, val)
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

func TestAppendTag(t *testing.T) {
	test := func(code byte, vlen uint64, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("(%x,%v)", code, vlen), func(t *testing.T) {
			length := tagLen(vlen)
			if length != elen {
				t.Errorf("expected length=%v, got length=%v", elen, length)
			}

			bits := appendTag(nil, code, vlen)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	test(0x20, 1, 1, []byte{0x21})
	test(0x30, 0x0D, 1, []byte{0x3D})
	test(0x40, 0x0E, 2, []byte{0x4E, 0x8E})
	test(0x50, math.MaxInt64, 10, []byte{0x5E, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0x7F, 0xFF})
}

func TestAppendTimestamp(t *testing.T) {
	test := func(val Timestamp, elen uint64, ebits []byte) {
		t.Run(fmt.Sprintf("%x", val.dateTime), func(t *testing.T) {
			_, offset := val.dateTime.Zone()
			offset /= 60
			val.dateTime = val.dateTime.In(time.UTC)

			length := timestampLen(offset, val)
			if length != elen {
				t.Errorf("expected length=%v, got length=%v", elen, length)
			}

			bits := appendTimestamp(nil, offset, val)
			if !bytes.Equal(bits, ebits) {
				t.Errorf("expected %v, got %v", fmtbytes(ebits), fmtbytes(bits))
			}
		})
	}

	nowish, _ := NewTimestampFromStr("2019-08-04T18:15:43.863494+10:00", TimestampPrecisionNanosecond, TimezoneLocal)

	test(NewDateTimestamp(time.Time{}, TimestampPrecisionSecond), 7, []byte{0xC0, 0x81, 0x81, 0x81, 0x80, 0x80, 0x80})
	test(nowish, 13, []byte{
		0x04, 0xD8, // offset: +600 minutes (+10:00)
		0x0F, 0xE3, // year:   2019
		0x88,             // month:  8
		0x84,             // day:    4
		0x88,             // hour:   8 utc (18 local)
		0x8F,             // minute: 15
		0xAB,             // second: 43
		0xC6,             // exp:    6 precision units
		0x0D, 0x2D, 0x06, // nsec:   863494
	})
}
