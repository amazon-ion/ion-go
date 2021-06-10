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
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/big"
	"unicode/utf8"
)

type bss uint8

const (
	bssBeforeValue bss = iota
	bssOnValue
	bssBeforeFieldID
	bssOnFieldID
)

type bitcode uint8

const (
	bitcodeNone bitcode = iota
	bitcodeEOF
	bitcodeBVM
	bitcodeNull
	bitcodeFalse
	bitcodeTrue
	bitcodeInt
	bitcodeNegInt
	bitcodeFloat
	bitcodeDecimal
	bitcodeTimestamp
	bitcodeSymbol
	bitcodeString
	bitcodeClob
	bitcodeBlob
	bitcodeList
	bitcodeSexp
	bitcodeStruct
	bitcodeFieldID
	bitcodeAnnotation
)

func (b bitcode) String() string {
	switch b {
	case bitcodeNone:
		return "none"
	case bitcodeEOF:
		return "eof"
	case bitcodeBVM:
		return "bvm"
	case bitcodeFalse:
		return "false"
	case bitcodeTrue:
		return "true"
	case bitcodeInt:
		return "int"
	case bitcodeNegInt:
		return "negint"
	case bitcodeFloat:
		return "float"
	case bitcodeDecimal:
		return "decimal"
	case bitcodeTimestamp:
		return "timestamp"
	case bitcodeSymbol:
		return "symbol"
	case bitcodeString:
		return "string"
	case bitcodeClob:
		return "clob"
	case bitcodeBlob:
		return "blob"
	case bitcodeList:
		return "list"
	case bitcodeSexp:
		return "sexp"
	case bitcodeStruct:
		return "struct"
	case bitcodeFieldID:
		return "fieldid"
	case bitcodeAnnotation:
		return "annotation"
	default:
		return fmt.Sprintf("<invalid bitcode 0x%2X>", uint8(b))
	}
}

// A bitstream is a low-level parser for binary Ion values.
type bitstream struct {
	in    *bufio.Reader
	pos   uint64
	state bss
	stack bitstack

	code bitcode
	null bool
	len  uint64
}

// Init initializes this stream with the given bufio.Reader.
func (b *bitstream) Init(in *bufio.Reader) {
	b.in = in
}

// InitBytes initializes this stream with the given bytes.
func (b *bitstream) InitBytes(in []byte) {
	b.in = bufio.NewReader(bytes.NewReader(in))
}

// Code returns the type code of the current value.
func (b *bitstream) Code() bitcode {
	return b.code
}

// IsNull returns true if the current value is null.
func (b *bitstream) IsNull() bool {
	return b.null
}

// Pos returns the current position.
func (b *bitstream) Pos() uint64 {
	return b.pos
}

// Len returns the length of the current value.
func (b *bitstream) Len() uint64 {
	return b.len
}

// Next advances the stream to the next value.
func (b *bitstream) Next() error {
	// If we have an unread value, skip over it to get to the next one.
	switch b.state {
	case bssOnValue, bssOnFieldID:
		if err := b.SkipValue(); err != nil {
			return err
		}
	}

	// If we're at the end of the current container, stop and make the user step out.
	if !b.stack.empty() {
		cur := b.stack.peek()
		if b.pos == cur.end {
			b.code = bitcodeEOF
			return nil
		}
	}

	// If it's time to read a field id, do that.
	if b.state == bssBeforeFieldID {
		b.code = bitcodeFieldID
		b.state = bssOnFieldID
		return nil
	}

	// Otherwise it's time to read a value. Read the tag byte.
	c, err := b.read()
	if err != nil {
		return err
	}

	// Found the end of the file.
	if c == -1 {
		b.code = bitcodeEOF
		return nil
	}

	// Parse the tag.
	code, length := parseTag(c)

	// Structs with a length code of 1 are a special case. Their length is always encoded
	// as a VarUInt and their field names appear in ascending symbol ID order.
	if code == bitcodeStruct && length == 1 {
		length, _, err = b.readVarUintLen(b.remaining())
		if err != nil {
			return err
		}
		if length == 0 {
			// Ordered structs must have at least one symbol/value pair.
			return &SyntaxError{"ordered structs cannot be empty", b.pos - 1}
		}
	}

	if code == bitcodeNone {
		return &InvalidTagByteError{byte(c), b.pos - 1}
	}

	b.state = bssOnValue

	if code == bitcodeAnnotation {
		switch length {
		case 0:
			// This value is actually a BVM. It's invalid if we're not at the top level.
			if !b.stack.empty() {
				return &SyntaxError{"invalid BVM in a container", b.pos - 1}
			}
			b.code = bitcodeBVM
			b.len = 3
			return nil

		case 0x0F:
			// No such thing as a null annotation.
			return &InvalidTagByteError{byte(c), b.pos - 1}
		}
	}

	// Booleans are a bit special; the 'length' stores the value.
	if code == bitcodeFalse {
		switch length {
		case 0, 0x0F:
			break
		case 1:
			code = bitcodeTrue
			length = 0
		default:
			// Other forms are invalid.
			return &InvalidTagByteError{byte(c), b.pos - 1}
		}
	}

	if length == 0x0F {
		// This value is actually a null.
		b.code = code
		b.null = true
		return nil
	}

	pos := b.pos
	rem := b.remaining()

	// This value's actual length is encoded as a separate varUint.
	if length == 0x0E {
		var lenghtOfRemaining uint64
		length, lenghtOfRemaining, err = b.readVarUintLen(rem)
		if err != nil {
			return err
		}
		rem -= lenghtOfRemaining
	}

	if length > rem {
		msg := fmt.Sprintf("value overruns its container: %v vs %v", length, rem)
		return &SyntaxError{msg, pos - 1}
	}

	b.code = code
	b.len = length
	return nil
}

// SkipValue skips over the current value.
func (b *bitstream) SkipValue() error {
	switch b.state {
	case bssBeforeFieldID, bssBeforeValue:
		// No current value to skip yet.
		return nil

	case bssOnFieldID:
		if err := b.skipVarUint(); err != nil {
			return err
		}
		b.state = bssBeforeValue

	case bssOnValue:
		if b.len > 0 {
			if err := b.skip(b.len); err != nil {
				return err
			}
		}
		b.state = b.stateAfterValue()

	default:
		panic(fmt.Sprintf("invalid state %v", b.state))
	}

	b.clear()
	return nil
}

// StepIn steps in to a container.
func (b *bitstream) StepIn() {
	switch b.code {
	case bitcodeStruct:
		b.state = bssBeforeFieldID

	case bitcodeList, bitcodeSexp:
		b.state = bssBeforeValue

	default:
		panic(fmt.Sprintf("StepIn called with b.code=%v", b.code))
	}

	b.stack.push(b.code, b.pos+b.len)
	b.clear()
}

// StepOut steps out of a container.
func (b *bitstream) StepOut() error {
	if b.stack.empty() {
		panic("StepOut called at top level")
	}

	cur := b.stack.peek()
	b.stack.pop()

	if cur.end < b.pos {
		panic(fmt.Sprintf("end (%v) greater than b.pos (%v)", cur.end, b.pos))
	}
	diff := cur.end - b.pos

	// Skip over anything left in the container we're stepping out of.
	if diff > 0 {
		if err := b.skip(diff); err != nil {
			return err
		}
	}

	b.state = b.stateAfterValue()
	b.clear()

	return nil
}

// ReadBVM reads a binary version marker, returning its major and minor version.
func (b *bitstream) ReadBVM() (byte, byte, error) {
	if b.code != bitcodeBVM {
		panic("not a BVM")
	}

	major, err := b.read1()
	if err != nil {
		return 0, 0, err
	}

	minor, err := b.read1()
	if err != nil {
		return 0, 0, err
	}

	end, err := b.read1()
	if err != nil {
		return 0, 0, err
	}

	if end != 0xEA {
		msg := fmt.Sprintf("invalid BVM: 0xE0 0x%02X 0x%02X 0x%02X", major, minor, end)
		return 0, 0, &SyntaxError{msg, b.pos - 4}
	}

	b.state = bssBeforeValue
	b.clear()

	return byte(major), byte(minor), nil
}

// ReadFieldID reads a field ID.
func (b *bitstream) ReadFieldID() (uint64, error) {
	if b.code != bitcodeFieldID {
		panic("not a field ID")
	}

	id, err := b.readVarUint()
	if err != nil {
		return 0, err
	}

	b.state = bssBeforeValue
	b.code = bitcodeNone

	return id, nil
}

// ReadAnnotations reads a set of annotation IDs and returns a set of SymbolTokens.
func (b *bitstream) ReadAnnotations(symbolTable SymbolTable) ([]SymbolToken, error) {
	if b.code != bitcodeAnnotation {
		panic("not an annotation")
	}

	annotFieldLength, lengthOfAnnotFieldLength, err := b.readVarUintLen(b.len)
	if err != nil {
		return nil, err
	}

	if annotFieldLength == 0 {
		// An annotation with zero length is illegal because at least one annotation must be present.
		return nil, &SyntaxError{"malformed annotation: at least one annotation must be specified",
			b.pos - lengthOfAnnotFieldLength}
	}

	remainingAnnotationLength := b.len - lengthOfAnnotFieldLength - annotFieldLength

	if remainingAnnotationLength <= 0 {
		// The size of the annotations is larger than the remaining free space inside the
		// annotation container.
		return nil, &SyntaxError{"malformed annotation", b.pos - lengthOfAnnotFieldLength}
	}

	var as []SymbolToken
	for annotFieldLength > 0 {
		id, idlen, err := b.readVarUintLen(annotFieldLength)
		if err != nil {
			return nil, err
		}

		token, err := NewSymbolTokenBySID(symbolTable, int64(id))
		if err != nil {
			return nil, err
		}

		as = append(as, token)

		annotFieldLength -= idlen
	}

	err = b.validateAnnotatedValue(remainingAnnotationLength)
	if err != nil {
		return nil, err
	}

	b.state = bssBeforeValue
	b.clear()

	return as, nil
}

func (b *bitstream) validateAnnotatedValue(remainingLength uint64) error {
	tagByte, err := b.peekAtOffset(0)
	if err != nil {
		return err
	}

	code, length := parseTag(int(tagByte))

	if length == 15 {
		// Anything with length 15 is null and should only require one byte to represent it.
		if remainingLength != 1 {
			return &InvalidTagByteError{tagByte, b.pos}
		}
		return nil
	}

	if code == bitcodeNull {
		// It is illegal for an annotation to wrap a NOP Pad.
		return &SyntaxError{"an annotation cannot wrap a NOP Pad", b.pos}
	} else if code == bitcodeAnnotation {
		// We cannot have an annotation directly wrapping another annotation.
		return &SyntaxError{"an annotation cannot be the enclosed value of another annotation", b.pos}
	}

	// Adjust remainingLength because we just processed the first byte of the annotated data.
	remainingLength--

	// If the above length is 14 or we have an ordered struct (indicated by struct with length 1),
	// then we need to process additional bytes to figure out the full length.
	if length == 0x0E || (code == bitcodeStruct && length == 1) {
		val := uint64(0)
		counter := 1

		for {
			c, err := b.peekAtOffset(counter)
			if err != nil {
				return err
			}

			counter++
			remainingLength--

			val <<= 7
			val ^= uint64(c & 0x7F)

			if (c & 0x80) != 0 {
				length = val
				break
			}
		}
	}

	// Confirm the computed length is consistent with the expected remaining length from the annotation wrapper.
	if length != remainingLength {
		msg := fmt.Sprintf("annotation wrapper indicates the enclosed value's length to be %d "+
			"but the enclosed value claims to have length %d", remainingLength, length)
		return &SyntaxError{msg, b.pos}
	}

	return nil
}

// ReadInt reads an integer value.
func (b *bitstream) ReadInt() (interface{}, error) {
	if b.code != bitcodeInt && b.code != bitcodeNegInt {
		panic("not an integer")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return "", err
	}

	var ret interface{}
	isZero := false
	switch {
	case b.len == 0:
		// Special case for zero.
		ret = int64(0)
		isZero = true

	case b.len < 8, b.len == 8 && bs[0]&0x80 == 0:
		// It'll fit in an int64.
		i := int64(0)
		for _, b := range bs {
			i <<= 8
			i ^= int64(b)
		}
		isZero = i == 0
		if b.code == bitcodeNegInt {
			i = -i
		}
		ret = i

	default:
		// Need to go big.Int.
		i := new(big.Int).SetBytes(bs)
		isZero = i.BitLen() == 0
		if b.code == bitcodeNegInt {
			i = i.Neg(i)
		}
		ret = i
	}

	// Zero is always stored as positive; negative zero is illegal.
	if isZero && b.code == bitcodeNegInt {
		return 0, &SyntaxError{"integer zero cannot be negative", b.pos - b.len}
	}

	b.state = b.stateAfterValue()
	b.clear()

	return ret, nil
}

// ReadFloat reads a float value.
func (b *bitstream) ReadFloat() (float64, error) {
	if b.code != bitcodeFloat {
		panic("not a float")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return 0, err
	}

	var ret float64
	switch len(bs) {
	case 0:
		ret = 0

	case 4:
		ui := binary.BigEndian.Uint32(bs)
		ret = float64(math.Float32frombits(ui))

	case 8:
		ui := binary.BigEndian.Uint64(bs)
		ret = math.Float64frombits(ui)

	default:
		return 0, &SyntaxError{"invalid float size", b.pos - b.len}
	}

	b.state = b.stateAfterValue()
	b.clear()

	return ret, nil
}

// ReadDecimal reads a decimal value.
func (b *bitstream) ReadDecimal() (*Decimal, error) {
	if b.code != bitcodeDecimal {
		panic("not a decimal")
	}

	d, err := b.readDecimal(b.len)
	if err != nil {
		return nil, err
	}

	b.state = b.stateAfterValue()
	b.clear()

	return d, nil
}

// ReadTimestamp reads a timestamp value.
func (b *bitstream) ReadTimestamp() (Timestamp, error) {
	if b.code != bitcodeTimestamp {
		panic("not a timestamp")
	}

	length := b.len

	offset, osign, olength, err := b.readVarIntLen(length)
	if err != nil {
		return Timestamp{}, err
	}
	length -= olength

	ts := []int{1, 1, 1, 0, 0, 0}
	precision := TimestampNoPrecision
	for i := 0; length > 0 && i < 6 && precision < TimestampPrecisionSecond; i++ {
		val, vlength, err := b.readVarUintLen(length)
		if err != nil {
			return Timestamp{}, err
		}
		length -= vlength
		ts[i] = int(val)

		// When i is 3, it means we are setting the hour component. A timestamp with an hour
		// component must also have a minute component. Hence, length cannot be zero at this point.
		if i == 3 {
			if length == 0 {
				return Timestamp{}, &SyntaxError{"invalid timestamp - Hour cannot be present without minute", b.pos}
			}
		} else {
			// Update precision as we read the timestamp.
			// We don't update precision when i is 3 because there is no Hour precision.
			precision++
		}
	}

	nsecs := 0
	overflow := false
	fractionPrecision := uint8(0)

	// Check the fractional seconds part of the timestamp.
	if length > 0 {
		nsecs, overflow, fractionPrecision, err = b.readNsecs(length)
		if err != nil {
			return Timestamp{}, err
		}

		if fractionPrecision > 0 {
			precision = TimestampPrecisionNanosecond
		}
	}

	timestamp, err := tryCreateTimestamp(ts, nsecs, overflow, offset, osign, precision, fractionPrecision)
	if err != nil {
		return Timestamp{}, err
	}

	b.state = b.stateAfterValue()
	b.clear()

	return timestamp, nil
}

// ReadNsecs reads the fraction part of a timestamp and rounds to nanoseconds.
// This function returns the nanoseconds as an int, overflow as a bool, exponent as an uint8, and an error
// if there was a problem executing this function.
func (b *bitstream) readNsecs(length uint64) (int, bool, uint8, error) {
	d, err := b.readDecimal(length)
	if err != nil {
		return 0, false, 0, err
	}

	nsec, err := d.ShiftL(9).trunc()
	if err != nil || nsec < 0 || nsec > 999999999 {
		msg := fmt.Sprintf("invalid timestamp fraction: %v", d)
		return 0, false, 0, &SyntaxError{msg, b.pos}
	}

	nsec, err = d.ShiftL(9).round()
	if err != nil {
		msg := fmt.Sprintf("invalid timestamp fraction: %v", d)
		return 0, false, 0, &SyntaxError{msg, b.pos}
	}

	var exponent uint8

	// check if the scale is negative and coefficient is zero then set exponent value to 0
	// otherwise set exponent value as per the scale value
	if d.scale < 0 && nsec == 0 {
		exponent = uint8(0)
	} else {
		exponent = uint8(d.scale)
	}

	// Overflow to second.
	if nsec == 1000000000 {
		return 0, true, exponent, nil
	}

	return int(nsec), false, exponent, nil
}

// ReadDecimal reads a decimal value of the given length: an exponent encoded as a
// varInt, followed by an integer coefficient taking up the remaining bytes.
func (b *bitstream) readDecimal(length uint64) (*Decimal, error) {
	exp := int64(0)
	coef := new(big.Int)
	negZero := false

	if length > 0 {
		val, _, vlength, err := b.readVarIntLen(length)
		if err != nil {
			return nil, err
		}

		if val > math.MaxInt32 || val < math.MinInt32 {
			msg := fmt.Sprintf("decimal exponent out of range: %v", val)
			return nil, &SyntaxError{msg, b.pos - vlength}
		}

		exp = val
		length -= vlength
	}

	if length > 0 {
		if err := b.readBigInt(length, coef); err != nil {
			return nil, err
		}

		negZero = coef.Sign() == 0
	}

	return NewDecimal(coef, int32(exp), negZero), nil
}

// ReadSymbolID reads a symbol value.
func (b *bitstream) ReadSymbolID() (uint64, error) {
	if b.code != bitcodeSymbol {
		panic("not a symbol")
	}

	if b.len > 8 {
		return 0, &SyntaxError{"symbol id too large", b.pos}
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return 0, err
	}

	b.state = b.stateAfterValue()
	b.clear()

	ret := uint64(0)
	for _, b := range bs {
		ret <<= 8
		ret ^= uint64(b)
	}
	return ret, nil
}

// ReadString reads a string value.
func (b *bitstream) ReadString() (string, error) {
	if b.code != bitcodeString {
		panic("not a string")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return "", err
	}

	b.state = b.stateAfterValue()
	b.clear()

	if utf8.Valid(bs) {
		return string(bs), nil
	}
	return "", &UnexpectedTokenError{"string value contains non-UTF-8 runes", b.pos}
}

// ReadBytes reads a blob or clob value.
func (b *bitstream) ReadBytes() ([]byte, error) {
	if b.code != bitcodeClob && b.code != bitcodeBlob {
		panic("not a lob")
	}

	var bs []byte
	if b.len > 0 {
		var err error
		bs, err = b.readN(b.len)
		if err != nil {
			return nil, err
		}
	} else {
		// A0 and 90 are special cases, denoting an empty blob and an empty clob respectively, with b.length == 0.
		bs = []byte{}
	}

	b.state = b.stateAfterValue()
	b.clear()

	return bs, nil
}

// Clear clears the current code and length.
func (b *bitstream) clear() {
	b.code = bitcodeNone
	b.null = false
	b.len = 0
}

// ReadBigInt reads a fixed-length integer of the given length and stores
// the value in the given big.Int.
func (b *bitstream) readBigInt(length uint64, ret *big.Int) error {
	bs, err := b.readN(length)
	if err != nil {
		return err
	}

	neg := bs[0]&0x80 != 0
	bs[0] &= 0x7F
	if bs[0] == 0 {
		bs = bs[1:]
	}

	ret.SetBytes(bs)
	if neg {
		ret.Neg(ret)
	}

	return nil
}

// ReadVarUint reads a variable-length-encoded uint.
func (b *bitstream) readVarUint() (uint64, error) {
	val, _, err := b.readVarUintLen(b.remaining())
	return val, err
}

// ReadVarUintLen reads a variable-length-encoded uint of at most max bytes,
// returning the value and its actual length in bytes.
func (b *bitstream) readVarUintLen(max uint64) (uint64, uint64, error) {
	if max > 10 {
		max = 10
	}

	val := uint64(0)
	length := uint64(0)

	for {
		if length >= max {
			return 0, 0, &SyntaxError{"varuint too large", b.pos}
		}

		c, err := b.read1()
		if err != nil {
			return 0, 0, err
		}

		val <<= 7
		val ^= uint64(c & 0x7F)
		length++

		if c&0x80 != 0 {
			return val, length, nil
		}
	}
}

// SkipVarUint skips over a variable-length-encoded uint.
func (b *bitstream) skipVarUint() error {
	_, err := b.skipVarUintLen(b.remaining())
	return err
}

// SkipVarUintLen skips over a variable-length-encoded uint of at most max bytes.
func (b *bitstream) skipVarUintLen(max uint64) (uint64, error) {
	if max > 10 {
		max = 10
	}

	length := uint64(0)
	for {
		if length >= max {
			return 0, &SyntaxError{"varuint too large", b.pos - length}
		}

		c, err := b.read1()
		if err != nil {
			return 0, err
		}

		length++

		if c&0x80 != 0 {
			return length, nil
		}
	}
}

// Remaining returns the number of bytes remaining in the current container.
func (b *bitstream) remaining() uint64 {
	if b.stack.empty() {
		return math.MaxUint64
	}

	end := b.stack.peek().end
	if b.pos > end {
		panic(fmt.Sprintf("pos (%v) > end (%v)", b.pos, end))
	}

	return end - b.pos
}

// ReadVarIntLen reads a variable-length-encoded int of at most max bytes,
// returning the value, the sign, and its actual length in bytes
func (b *bitstream) readVarIntLen(max uint64) (int64, int64, uint64, error) {
	if max == 0 {
		return 0, 0, 0, &SyntaxError{"varint too large", b.pos}
	}
	if max > 10 {
		max = 10
	}

	// Read the first byte, which contains the sign bit.
	c, err := b.read1()
	if err != nil {
		return 0, 0, 0, err
	}

	sign := int64(1)
	if c&0x40 != 0 {
		sign = -1
	}

	val := int64(c & 0x3F)
	length := uint64(1)

	// Check if that was the last (only) byte.
	if c&0x80 != 0 {
		return val * sign, sign, length, nil
	}

	for {
		if length >= max {
			return 0, 0, 0, &SyntaxError{"varint too large", b.pos - length}
		}

		c, err := b.read1()
		if err != nil {
			return 0, 0, 0, err
		}

		val <<= 7
		val ^= int64(c & 0x7F)
		length++

		if c&0x80 != 0 {
			return val * sign, sign, length, nil
		}
	}
}

// StateAfterValue returns the state this stream is in after reading a value.
func (b *bitstream) stateAfterValue() bss {
	if b.stack.peek().code == bitcodeStruct {
		return bssBeforeFieldID
	}
	return bssBeforeValue
}

var bitcodes = []bitcode{
	bitcodeNull,       // 0x00
	bitcodeFalse,      // 0x10
	bitcodeInt,        // 0x20
	bitcodeNegInt,     // 0x30
	bitcodeFloat,      // 0x40
	bitcodeDecimal,    // 0x50
	bitcodeTimestamp,  // 0x60
	bitcodeSymbol,     // 0x70
	bitcodeString,     // 0x80
	bitcodeClob,       // 0x90
	bitcodeBlob,       // 0xA0
	bitcodeList,       // 0xB0
	bitcodeSexp,       // 0xC0
	bitcodeStruct,     // 0xD0
	bitcodeAnnotation, // 0xE0
}

// ParseTag parses a tag byte into a type code and a length.
func parseTag(c int) (bitcode, uint64) {
	high := (c >> 4) & 0x0F
	low := c & 0x0F

	code := bitcodeNone
	if high < len(bitcodes) {
		code = bitcodes[high]
	}

	return code, uint64(low)
}

// ReadN reads the next n bytes of input from the underlying stream.
func (b *bitstream) readN(n uint64) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}

	bs := make([]byte, n)
	actual, err := io.ReadFull(b.in, bs)
	b.pos += uint64(actual)

	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil, &UnexpectedEOFError{b.pos}
	}
	if err != nil {
		return nil, &IOError{err}
	}

	return bs, nil
}

// Read1 reads the next byte of input from the underlying stream, returning
// an UnexpectedEOFError if it's an EOF.
func (b *bitstream) read1() (int, error) {
	c, err := b.read()
	if err != nil {
		return 0, err
	}
	if c == -1 {
		return 0, &UnexpectedEOFError{b.pos}
	}
	return c, nil
}

// Read reads the next byte of input from the underlying stream. It returns
// -1 instead of io.EOF if we've hit the end of the stream, because I find
// that easier to reason about.
func (b *bitstream) read() (int, error) {
	c, err := b.in.ReadByte()
	b.pos++

	if err == io.EOF {
		return -1, nil
	}
	if err != nil {
		return 0, &IOError{err}
	}

	return int(c), nil
}

// Skip skips n bytes of input from the underlying stream.
func (b *bitstream) skip(n uint64) error {
	actual, err := b.in.Discard(int(n))
	b.pos += uint64(actual)

	if err == io.EOF {
		return nil
	}
	if err != nil {
		return &IOError{err}
	}

	return nil
}

// PeekAtOffset returns the data at a certain offset without advancing the reader.
func (b *bitstream) peekAtOffset(offset int) (byte, error) {
	data, err := b.in.Peek(offset + 1)
	if err != nil {
		return 0, err
	}

	return data[offset], nil
}

// A bitnode represents a container value, including its type code and
// the offset at which it (supposedly) ends.
type bitnode struct {
	code bitcode
	end  uint64
}

// A stack of bitnodes representing container values that we're currently
// stepped in to.
type bitstack struct {
	arr []bitnode
}

// Empty returns true if this bitstack is empty.
func (b *bitstack) empty() bool {
	return len(b.arr) == 0
}

// Peek peeks at the top bitnode on the stack.
func (b *bitstack) peek() bitnode {
	if len(b.arr) == 0 {
		return bitnode{}
	}
	return b.arr[len(b.arr)-1]
}

// Push pushes a bitnode onto the stack.
func (b *bitstack) push(code bitcode, end uint64) {
	b.arr = append(b.arr, bitnode{code, end})
}

// Pop pops a bitnode from the stack.
func (b *bitstack) pop() {
	if len(b.arr) == 0 {
		panic("pop called on empty bitstack")
	}
	b.arr = b.arr[:len(b.arr)-1]
}
