/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

package ion

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// This file contains the numeric types Decimal, Float, and Int.

const (
	textNullDecimal = "null.decimal"
	textNullFloat   = "null.float"
	textNullInt     = "null.int"
)

// Decimal is a decimal-encoded real number of arbitrary precision.
// The decimal’s value is coefficient * 10 ^ exponent.
type Decimal struct {
	annotations []Symbol
	value       *decimal.Decimal
	binary      []byte
	text        []byte
	isSet       bool
}

func (d Decimal) Value() decimal.Decimal {
	if d.IsNull() {
		return decimal.New(0, 0)
	}

	if len(d.text) > 0 {
		// decimal uses "e" for exponent while Ion uses "d"
		text := bytes.ReplaceAll(d.text, []byte{'d'}, []byte{'e'})
		text = bytes.ReplaceAll(text, []byte{'D'}, []byte{'E'})
		// decimal doesn't handle underscores
		text = bytes.ReplaceAll(text, []byte{'_'}, []byte{})

		val, err := decimal.NewFromString(string(text))
		if err != nil {
			panic(err)
		}

		d.value = &val
		return val
	}

	if binLen := len(d.binary); binLen > 0 {
		// Bytes are comprised of a VarInt and an Int.
		dataReader := bytes.NewReader(d.binary)
		exponent, errExp := readVarInt64(dataReader)
		if errExp != nil {
			panic(errors.WithMessage(errExp, "unable to read exponent part of decimal"))
		}

		coefficient := d.binary[binLen-dataReader.Len():]
		// SetBytes takes unsigned bytes in big-endian order, so we need to copy the
		// sign of our Int and then erase the traces of that sign.
		isNegative := (coefficient[0] & 0x80) != 0
		coefficient[0] &= 0x7F

		bigInt := &big.Int{}
		bigInt.SetBytes(coefficient)
		if isNegative {
			bigInt.Neg(bigInt)
		}

		val := decimal.NewFromBigInt(bigInt, int32(exponent))
		d.value = &val
		return val
	}
	return decimal.New(0, 0)
}

// Annotations satisfies Value.
func (d Decimal) Annotations() []Symbol {
	return d.annotations
}

// Binary returns the Decimal in binary form.
func (d Decimal) Binary() []byte {
	if d.binary != nil {
		return d.binary
	}

	if d.IsNull() {
		return []byte{}
	}

	// TODO: Turn value into the binary representation.
	_ = d.Value()

	return d.binary
}

// Text returns the Decimal in text form.  If no exponent is set,
// then the text form will not include one.  Otherwise the formatted
// text is in the form: <coefficient>d<exponent>
func (d Decimal) Text() []byte {
	if d.text != nil {
		return d.text
	}

	if d.IsNull() {
		return []byte(textNullDecimal)
	}

	val := d.Value()
	text := ""
	// If the decimal we have is represented by an exact float value, then
	// use that to make the string.  Otherwise we need to turn the big.Int
	// into a string.
	if floatVal, exact := val.Float64(); exact {
		d.text = strconv.AppendFloat(nil, floatVal, 'f', -1, 64)
	} else {
		if val.IsNegative() {
			text = "-"
		}
		text += val.Coefficient().String()
	}

	if exp := val.Exponent(); exp != 0 {
		d.text = append(d.text, []byte(fmt.Sprintf("d%d", exp))...)
	}

	d.text = []byte(text)
	return d.text
}

// IsNull satisfies Value.
func (d Decimal) IsNull() bool {
	return !d.isSet
}

// Type satisfies Value.
func (d Decimal) Type() Type {
	return TypeDecimal
}

// Float is a binary-encoded floating point number (IEEE 64-bit).
type Float struct {
	annotations []Symbol
	value       *float64
	binary      []byte
	text        []byte
	isSet       bool
}

// Value returns the value of this Float, or 0 if the value is null.
func (f Float) Value() float64 {
	if f.value != nil {
		return *f.value
	}

	if f.IsNull() {
		return 0
	}

	// A binary length other than 0 (no value), 4, or 8 is not accepted by
	// the parser.
	if binLen := len(f.binary); binLen == 4 || binLen == 8 {
		if binLen == 4 {
			u32 := (uint32(f.binary[3]) << 24) | (uint32(f.binary[2]) << 16) | (uint32(f.binary[1]) << 8) | uint32(f.binary[0])
			f64 := float64(math.Float32frombits(u32))
			f.value = &f64
		} else {
			u64 := (uint64(f.binary[7]) << 56) | (uint64(f.binary[6]) << 48) | (uint64(f.binary[5]) << 40) | (uint64(f.binary[4]) << 32) |
				(uint64(f.binary[3]) << 24) | (uint64(f.binary[2]) << 16) | (uint64(f.binary[1]) << 8) | uint64(f.binary[0])
			f64 := math.Float64frombits(u64)
			f.value = &f64
		}
		return *f.value
	}

	if len(f.text) > 0 {
		text := string(bytes.ReplaceAll(f.text, []byte{'_'}, []byte{}))
		f64, err := strconv.ParseFloat(text, 64)
		// The float value when the given string is too big is
		// +/- infinity.
		if err != nil {
			numErr, ok := err.(*strconv.NumError)
			if !ok || numErr.Err != strconv.ErrRange {
				panic(err)
			}
		}
		f.value = &f64
		return f64
	}

	// It's possible for a binary float to be set to a zero-length slice, in which case
	// the value is not null but there is no binary or text value to parse.
	return 0
}

// Annotations satisfies Value.
func (f Float) Annotations() []Symbol {
	return f.annotations
}

// Binary returns the Float in binary form.
func (f Float) Binary() []byte {
	if f.binary != nil {
		return f.binary
	}

	if f.IsNull() {
		return []byte{}
	}

	// TODO: Turn value into the binary representation.
	_ = f.Value()

	return f.binary
}

// Text returns the Float in text form.
func (f Float) Text() []byte {
	if f.text != nil {
		return f.text
	}

	if f.IsNull() {
		return []byte(textNullFloat)
	}

	f.text = strconv.AppendFloat(nil, f.Value(), 'f', -1, 64)
	return f.text
}

// IsNull satisfies Value.
func (f Float) IsNull() bool {
	return !f.isSet
}

// Type satisfies Value.
func (f Float) Type() Type {
	return TypeFloat
}

// intBase represents the various bases (binary, decimal, hexadecimal)
// that can be used to represent an integer in text.  The zero value
// is intBase10 which is decimal.
type intBase int

const (
	intBase10 intBase = iota
	intBase2
	intBase16
)

// Int is a signed integer of arbitrary size.
type Int struct {
	annotations []Symbol
	value       *big.Int
	binary      []byte
	text        []byte
	base        intBase
	isNegative  bool
	isSet       bool
}

// Value returns the representation of this Int as a big.Int.
// If this represents null.Int, then nil is returned.
func (i Int) Value() *big.Int {
	if i.IsNull() || i.value != nil {
		return i.value
	}

	if len(i.text) > 0 {
		text := string(bytes.ReplaceAll(i.text, []byte{'_'}, []byte{}))
		i.value = new(big.Int)
		i.value.SetString(text, 0)
		return i.value
	}

	if len(i.binary) > 0 {
		// TODO
	}
	return nil
}

// Annotations satisfies Value.
func (i Int) Annotations() []Symbol {
	return i.annotations
}

// Binary returns the Int in binary form.
func (i Int) Binary() []byte {
	if i.binary != nil {
		return i.binary
	}

	val := i.Value()
	if val == nil {
		return []byte{}
	}

	// TODO: Turn value into the binary representation.
	_ = i.Value()

	return i.binary
}

// Text returns the Int in text form.
func (i Int) Text() []byte {
	if i.text != nil {
		return i.text
	}

	val := i.Value()
	if val == nil {
		return []byte(textNullInt)
	}

	i.text = val.Append(nil, 10)
	return i.text
}

// IsNull satisfies Value.
func (i Int) IsNull() bool {
	return !i.isSet
}

// Type satisfies Value.
func (i Int) Type() Type {
	return TypeInt
}
