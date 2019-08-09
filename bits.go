package ion

import (
	"math/big"
	"time"
)

// uintLen pre-calculates the length, in bytes, of the given uint value.
func uintLen(v uint64) uint64 {
	len := uint64(1)
	v >>= 8

	for v > 0 {
		len++
		v >>= 8
	}

	return len
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

	len := uintLen(mag)

	// If the high bit is a one, we need an extra byte to store the sign bit.
	hb := mag >> ((len - 1) * 8)
	if hb&0x80 != 0 {
		len++
	}

	return len
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

	// Either bitl is evenly divisibly by 8, in which case we need another
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
	len := uint64(1)
	v >>= 7

	for v > 0 {
		len++
		v >>= 7
	}

	return len
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
	len := uint64(1)
	mag >>= 6

	for mag > 0 {
		len++
		mag >>= 7
	}

	return len
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
func tagLen(len uint64) uint64 {
	if len < 0x0E {
		return 1
	}
	return 1 + varUintLen(len)
}

// appendTag appends a code+len tag to the given slice.
func appendTag(b []byte, code byte, len uint64) []byte {
	if len < 0x0E {
		// Short form, with length embedded in the code byte.
		return append(b, code|byte(len))
	}

	// Long form, with separate length.
	b = append(b, code|0x0E)
	return appendVarUint(b, len)
}

// timeLen pre-calculates the length, in bytes, of the given time value.
func timeLen(offset int, utc time.Time) uint64 {
	ret := varIntLen(int64(offset))

	// Almost certainly two but let's be safe.
	ret += varUintLen(uint64(utc.Year()))

	// Month, day, hour, minute, and second are all guaranteed to be one byte.
	ret += 5

	ns := utc.Nanosecond()
	if ns > 0 {
		ret++ // varIntLen(-9)
		ret += intLen(int64(ns))
	}

	return ret
}

// appendTime appends a timestamp value
func appendTime(b []byte, offset int, utc time.Time) []byte {
	b = appendVarInt(b, int64(offset))

	b = appendVarUint(b, uint64(utc.Year()))
	b = appendVarUint(b, uint64(utc.Month()))
	b = appendVarUint(b, uint64(utc.Day()))

	b = appendVarUint(b, uint64(utc.Hour()))
	b = appendVarUint(b, uint64(utc.Minute()))
	b = appendVarUint(b, uint64(utc.Second()))

	ns := utc.Nanosecond()
	if ns > 0 {
		b = appendVarInt(b, -9)
		b = appendInt(b, int64(ns))
	}

	return b
}
