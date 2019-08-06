package ion

import (
	"bytes"
	"math/big"
	"time"
)

func uintLen(v uint64) uint64 {
	len := uint64(1)
	v >>= 8

	for v > 0 {
		len++
		v >>= 8
	}

	return len
}

func packUint(v uint64) []byte {
	var buf [8]byte

	i := 7
	buf[i] = byte(v & 0xFF)
	v >>= 8

	for v > 0 {
		i--
		buf[i] = byte(v & 0xFF)
		v >>= 8
	}

	return buf[i:]
}

func packInt(n int64) []byte {
	if n == 0 {
		return []byte{}
	}

	neg := false
	mag := uint64(n)

	if n < 0 {
		neg = true
		mag = uint64(-n)
	}

	bits := packUint(mag)
	if bits[0]&0x80 != 0 {
		bits = append([]byte{0}, bits...)
	}

	if neg {
		bits[0] ^= 0x80
	}

	return bits
}

func packBigInt(v *big.Int) []byte {
	sign := v.Sign()
	if sign == 0 {
		return []byte{}
	}

	bits := v.Bytes()

	if bits[0]&0x80 != 0 {
		// Need to make room for the sign bit.
		bits = append([]byte{0}, bits...)
	}

	if sign < 0 {
		bits[0] ^= 0x80
	}

	return bits
}

func varUintLen(v uint64) uint64 {
	len := uint64(1)
	v >>= 7

	for v > 0 {
		len++
		v >>= 7
	}

	return len
}

func packVarUint(v uint64) []byte {
	var buf [10]byte

	i := 9
	buf[i] = 0x80 | byte(v&0x7F)
	v >>= 7

	for v > 0 {
		i--
		buf[i] = byte(v & 0x7F)
		v >>= 7
	}

	return buf[i:]
}

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

func packVarInt(v int64) []byte {
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
		return []byte{0x80 | signbit | byte(mag&0x3F)}
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

	return buf[i:]
}

func packTime(t time.Time) []byte {
	_, offset := t.Zone()
	utc := t.In(time.UTC)

	buf := bytes.Buffer{}
	buf.Write(packVarInt(int64(offset / 60)))

	buf.Write(packVarUint(uint64(utc.Year())))
	buf.Write(packVarUint(uint64(utc.Month())))
	buf.Write(packVarUint(uint64(utc.Day())))

	buf.Write(packVarUint(uint64(utc.Hour())))
	buf.Write(packVarUint(uint64(utc.Minute())))
	buf.Write(packVarUint(uint64(utc.Second())))

	ns := utc.Nanosecond()
	if ns > 0 {
		buf.Write(packVarInt(-9))
		buf.Write(packInt(int64(ns)))
	}

	return buf.Bytes()
}
