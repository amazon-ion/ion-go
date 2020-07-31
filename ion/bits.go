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
	"math/big"
)

// uintLen pre-calculates the length, in bytes, of the given uint value.
func uintLen(v uint64) uint64 {
	length := uint64(1)
	v >>= 8

	for v > 0 {
		length++
		v >>= 8
	}

	return length
}

// appendUint appends a uint value to the given slice. The reader is
// expected to know how many bytes the value takes up.
func appendUint(b []byte, v uint64) []byte {
	var buf [8]byte

	i := 7
	buf[i] = byte(v & 0xFF)
	v >>= 8

	for v > 0 {
		i--
		buf[i] = byte(v & 0xFF)
		v >>= 8
	}

	return append(b, buf[i:]...)
}

// intLen pre-calculates the length, in bytes, of the given int value.
func intLen(n int64) uint64 {
	if n == 0 {
		return 0
	}

	mag := uint64(n)
	if n < 0 {
		mag = uint64(-n)
	}

	length := uintLen(mag)

	// If the high bit is a one, we need an extra byte to store the sign bit.
	hb := mag >> ((length - 1) * 8)
	if hb&0x80 != 0 {
		length++
	}

	return length
}

// appendInt appends a (signed) int to the given slice. The reader is
// expected to know how many bytes the value takes up.
func appendInt(b []byte, n int64) []byte {
	if n == 0 {
		return b
	}

	neg := false
	mag := uint64(n)

	if n < 0 {
		neg = true
		mag = uint64(-n)
	}

	var buf [8]byte
	bits := buf[:0]
	bits = appendUint(bits, mag)

	if bits[0]&0x80 == 0 {
		// We've got space we can use for the sign bit.
		if neg {
			bits[0] ^= 0x80
		}
	} else {
		// We need to add more space.
		bit := byte(0)
		if neg {
			bit = 0x80
		}
		b = append(b, bit)
	}

	return append(b, bits...)
}

// bigIntLen pre-calculates the length, in bytes, of the given big.Int value.
func bigIntLen(v *big.Int) uint64 {
	if v.Sign() == 0 {
		return 0
	}

	bitl := v.BitLen()
	bytel := bitl / 8

	// Either bitl is evenly divisible by 8, in which case we need another
	// byte for the sign bit, or its not in which case we need to round up
	// (but will then have room for the sign bit).
	return uint64(bytel) + 1
}

// appendBigInt appends a (signed) big.Int to the given slice. The reader is
// expected to know how many bytes the value takes up.
func appendBigInt(b []byte, v *big.Int) []byte {
	sign := v.Sign()
	if sign == 0 {
		return b
	}

	bits := v.Bytes()

	if bits[0]&0x80 == 0 {
		// We've got space we can use for the sign bit.
		if sign < 0 {
			bits[0] ^= 0x80
		}
	} else {
		// We need to add more space.
		bit := byte(0)
		if sign < 0 {
			bit = 0x80
		}
		b = append(b, bit)
	}

	return append(b, bits...)
}

// varUintLen pre-calculates the length, in bytes, of the given varUint value.
func varUintLen(v uint64) uint64 {
	length := uint64(1)
	v >>= 7

	for v > 0 {
		length++
		v >>= 7
	}

	return length
}

// appendVarUint appends a variable-length-encoded uint to the given slice.
// Each byte stores seven bits of value; the high bit is a flag marking the
// last byte of the value.
func appendVarUint(b []byte, v uint64) []byte {
	var buf [10]byte

	i := 9
	buf[i] = 0x80 | byte(v&0x7F)
	v >>= 7

	for v > 0 {
		i--
		buf[i] = byte(v & 0x7F)
		v >>= 7
	}

	return append(b, buf[i:]...)
}

// varIntLen pre-calculates the length, in bytes, of the given varInt value.
func varIntLen(v int64) uint64 {
	mag := uint64(v)
	if v < 0 {
		mag = uint64(-v)
	}

	// Reserve one extra bit of the first byte for sign.
	length := uint64(1)
	mag >>= 6

	for mag > 0 {
		length++
		mag >>= 7
	}

	return length
}

// appendVarInt appends a variable-length-encoded int to the given slice.
// Most bytes store seven bits of value; the high bit is a flag marking the
// last byte of the value. The first byte additionally stores a sign bit.
func appendVarInt(b []byte, v int64) []byte {
	var buf [10]byte

	signbit := byte(0)
	mag := uint64(v)
	if v < 0 {
		signbit = 0x40
		mag = uint64(-v)
	}

	next := mag >> 6
	if next == 0 {
		// The whole thing fits in one byte.
		return append(b, 0x80|signbit|byte(mag&0x3F))
	}

	i := 9
	buf[i] = 0x80 | byte(mag&0x7F)
	mag >>= 7
	next = mag >> 6

	for next > 0 {
		i--
		buf[i] = byte(mag & 0x7F)
		mag >>= 7
		next = mag >> 6
	}

	i--
	buf[i] = signbit | byte(mag&0x3F)

	return append(b, buf[i:]...)
}

// tagLen pre-calculates the length, in bytes, of a tag.
func tagLen(length uint64) uint64 {
	if length < 0x0E {
		return 1
	}
	return 1 + varUintLen(length)
}

// appendTag appends a code+len tag to the given slice.
func appendTag(b []byte, code byte, length uint64) []byte {
	if length < 0x0E {
		// Short form, with length embedded in the code byte.
		return append(b, code|byte(length))
	}

	// Long form, with separate length.
	b = append(b, code|0x0E)
	return appendVarUint(b, length)
}

// timestampLen pre-calculates the length, in bytes, of the given timestamp value.
func timestampLen(offset int, utc Timestamp) uint64 {
	var ret uint64

	if utc.kind == TimezoneUnspecified {
		ret = 1
	} else {
		ret = varIntLen(int64(offset))
	}

	// We expect at least Year precision.
	ret += varUintLen(uint64(utc.dateTime.Year()))

	// Month, day, hour, minute, and second are all guaranteed to be one byte.
	switch utc.precision {
	case TimestampPrecisionMonth:
		ret++
	case TimestampPrecisionDay:
		ret += 2
	case TimestampPrecisionMinute:
		// Hour and Minute combined
		ret += 4
	case TimestampPrecisionSecond, TimestampPrecisionNanosecond:
		ret += 5
	}

	if utc.precision == TimestampPrecisionNanosecond && utc.numFractionalSeconds > 0 {
		ret++ // For fractional seconds precision indicator

		ns := utc.TruncatedNanoseconds()
		if ns > 0 {
			ret += intLen(int64(ns))
		}
	}

	return ret
}

// appendTimestamp appends a timestamp value
func appendTimestamp(b []byte, offset int, utc Timestamp) []byte {
	if utc.kind == TimezoneUnspecified {
		// Unknown offset
		b = append(b, 0xC0)
	} else {
		b = appendVarInt(b, int64(offset))
	}

	// We expect at least Year precision.
	b = appendVarUint(b, uint64(utc.dateTime.Year()))

	switch utc.precision {
	case TimestampPrecisionMonth:
		b = appendVarUint(b, uint64(utc.dateTime.Month()))
	case TimestampPrecisionDay:
		b = appendVarUint(b, uint64(utc.dateTime.Month()))
		b = appendVarUint(b, uint64(utc.dateTime.Day()))
	case TimestampPrecisionMinute:
		b = appendVarUint(b, uint64(utc.dateTime.Month()))
		b = appendVarUint(b, uint64(utc.dateTime.Day()))

		// The hour and minute is considered as a single component.
		b = appendVarUint(b, uint64(utc.dateTime.Hour()))
		b = appendVarUint(b, uint64(utc.dateTime.Minute()))
	case TimestampPrecisionSecond, TimestampPrecisionNanosecond:
		b = appendVarUint(b, uint64(utc.dateTime.Month()))
		b = appendVarUint(b, uint64(utc.dateTime.Day()))

		// The hour and minute is considered as a single component.
		b = appendVarUint(b, uint64(utc.dateTime.Hour()))
		b = appendVarUint(b, uint64(utc.dateTime.Minute()))
		b = appendVarUint(b, uint64(utc.dateTime.Second()))
	}

	if utc.precision == TimestampPrecisionNanosecond && utc.numFractionalSeconds > 0 {
		b = append(b, utc.numFractionalSeconds|0xC0)

		ns := utc.TruncatedNanoseconds()
		if ns > 0 {
			b = appendInt(b, int64(ns))
		}
	}

	return b
}
