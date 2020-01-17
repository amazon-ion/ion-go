/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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
	"io"
	"time"

	"github.com/pkg/errors"
)

const (
	binaryTypePadding    = 0
	binaryTypeBool       = 1
	binaryTypeInt        = 2
	binaryTypeNegInt     = 3
	binaryTypeFloat      = 4
	binaryTypeDecimal    = 5
	binaryTypeTimestamp  = 6
	binaryTypeSymbol     = 7
	binaryTypeString     = 8
	binaryTypeClob       = 9
	binaryTypeBlob       = 0xA
	binaryTypeList       = 0xB
	binaryTypeSExp       = 0xC
	binaryTypeStruct     = 0xD
	binaryTypeAnnotation = 0xE
)

var (
	// From http://amzn.github.io/ion-docs/docs/binary.html#value-streams :
	// The only valid BVM, identifying Ion 1.0, is 0xE0 0x01 0x00 0xEA
	ion10BVM = []byte{0xE0, 0x01, 0x00, 0xEA}

	// Map of the type portion (high nibble) of a value header byte to
	// the corresponding Ion Type.  Padding itself does not have a type mapping,
	// but it shares a high nibble with Null.  That means that the type
	// is determined by the low nibble when the high nibble is 0.
	binaryTypeMap = map[byte]Type{
		binaryTypePadding:    TypeNull,
		binaryTypeBool:       TypeBool,
		binaryTypeInt:        TypeInt,
		binaryTypeNegInt:     TypeInt,
		binaryTypeFloat:      TypeFloat,
		binaryTypeDecimal:    TypeDecimal,
		binaryTypeTimestamp:  TypeTimestamp,
		binaryTypeSymbol:     TypeSymbol,
		binaryTypeString:     TypeString,
		binaryTypeClob:       TypeClob,
		binaryTypeBlob:       TypeBlob,
		binaryTypeList:       TypeList,
		binaryTypeSExp:       TypeSExp,
		binaryTypeStruct:     TypeStruct,
		binaryTypeAnnotation: TypeAnnotation,
	}
)

// parseBinaryBlob decodes a single blob of bytes into a Digest.
func parseBinaryBlob(blob []byte) (*Digest, error) {
	// Read a single item from the binary stream.  Timeout after
	// five seconds because it shouldn't take that long to parse
	// a blob that is already loaded into memory.
	ch := parseBinaryStream(bytes.NewReader(blob))
	select {
	case out := <-ch:
		return out.Digest, out.Error
	case <-time.After(5 * time.Second):
		return nil, errors.New("timed out waiting for parser to finish")
	}
}

type streamItem struct {
	Digest *Digest
	Error  error
}

// parseBinaryStream reads from the given reader until either an error
// is encountered or the reader returns (0, io.EOF), at which point
// the returned channel is closed.  If an error occurred then the error
// is sent on the channel before it is closed.  Reading from the stream
// is not buffered.
func parseBinaryStream(r io.Reader) <-chan streamItem {
	itemChannel := make(chan streamItem)

	go func() {
		// First four bytes of the stream must be the version marker.
		if err := verifyByteVersionMarker(r); err != nil {
			itemChannel <- streamItem{Error: err}
			return
		}

		var values []Value
		for {
			switch val, err := parseNextBinaryValue(nil, r); {
			case err == io.EOF:
				itemChannel <- streamItem{Digest: &Digest{values: values}}
				// Signal that there isn't any more data coming.
				close(itemChannel)
				return
			case err != nil:
				itemChannel <- streamItem{Error: err}
				return
			case val != nil:
				values = append(values, val)
			default:
				itemChannel <- streamItem{Digest: &Digest{values: values}}
				values = nil
			}
		}
	}()

	return itemChannel
}

// parseNextBinaryValue parses the next binary value from the stream.  It returns
// io.EOF as the error if the first read shows that the end of the stream has been
// reached.  It returns a nil value and nil error if a new ByteVersionMarker has
// been reached. ann is an optional list of annotations to associate with the next
// value that is parsed.
func parseNextBinaryValue(ann []Symbol, r io.Reader) (Value, error) {
	switch high, low, err := readNibblesHighAndLow(r); {
	case err != nil:
		return nil, err
	case low == 0xF:
		return parseBinaryNull(high)
	case high == binaryTypePadding:
		return parseBinaryPadding(low, r)
	case high == binaryTypeBool:
		return parseBinaryBool(ann, low)
	case high == binaryTypeInt || high == binaryTypeNegInt:
		// 2 = positive integer, 3 = negative integer.
		return parseBinaryInt(ann, high == binaryTypeNegInt, low, r)
	case high == binaryTypeFloat:
		return parseBinaryFloat(ann, low, r)
	case high == binaryTypeDecimal:
		return parseBinaryDecimal(ann, low, r)
	case high == binaryTypeTimestamp:
		return parseBinaryTimestamp(ann, low, r)
	case high == binaryTypeSymbol:
		return parseBinarySymbol(ann, low, r)
	case high == binaryTypeString:
		return parseBinaryString(ann, low, r)
	case high == binaryTypeBlob || high == binaryTypeClob:
		return parseBinaryBytes(ann, high, low, r)
	case high == binaryTypeList || high == binaryTypeSExp:
		return parseBinaryList(ann, high, low, r)
	case high == binaryTypeStruct:
		return parseBinaryStruct(ann, low, r)
	case high == binaryTypeAnnotation:
		if len(ann) != 0 {
			return nil, errors.New("nesting annotations is not legal")
		}
		return parseBinaryAnnotation(low, r)
	default:
		return nil, errors.Errorf("invalid header combination - high: %d low: %d", high, low)
	}
}

// parseBinaryVersionMarker verifies that what is read next is a valid BVM.  If it is
// then a nil Value and error are returned.  It is assumed that the first byte has already
// been read and that it's value is 0xE0.
func parseBinaryVersionMarker(r io.Reader) (Value, error) {
	numBytes := len(ion10BVM)
	bvm := make([]byte, numBytes)
	bvm[0] = ion10BVM[0]
	if n, err := r.Read(bvm[1:]); err != nil || n != numBytes-1 {
		return nil, errors.Errorf("unable to read binary version marker - read %d bytes of %d with err: %v", n, numBytes-1, err)
	}

	if err := verifyByteVersionMarker(bytes.NewReader(bvm)); err != nil {
		return nil, err
	}

	return nil, nil
}

// verifyByteVersionMarker reads the BVM from the stream and ensures that it matches
// what is expected for Ion 1.0.
func verifyByteVersionMarker(r io.Reader) error {
	buf := make([]byte, 4)
	// First four bytes must be the version marker.
	if n, err := r.Read(buf); err != nil || n != 4 {
		return errors.Errorf("read %d bytes of binary version marker with err: %v", n, err)
	}
	if bytes.Compare(buf, ion10BVM) != 0 {
		return errors.Errorf("invalid binary version marker: %0 #x", buf)
	}
	return nil
}

// determineLength16 takes in the length nibble from the header byte and determines
// whether or not there is a Length portion to the value.  If there is, it then
// reads the length portion.
func determineLength16(lengthByte byte, r io.Reader) (uint16, error) {
	// "If the representation is at least 14 bytes long, then L is set to 14, and
	// the length field is set to the representation length, encoded as a VarUInt field."
	if lengthByte != 14 {
		return uint16(lengthByte), nil
	}
	return readVarUInt16(r)
}

// determineLength32 takes in the length nibble from the header byte and determines
// whether or not there is a Length portion to the value.  If there is, it then
// reads the length portion.
func determineLength32(lengthByte byte, r io.Reader) (uint32, error) {
	// "If the representation is at least 14 bytes long, then L is set to 14, and
	// the length field is set to the representation length, encoded as a VarUInt field."
	if lengthByte != 14 {
		return uint32(lengthByte), nil
	}
	return readVarUInt32(r)
}

// readVarUInt8 reads a variable-length number, but assumes that variable-length number is
// only one byte.  It then converts that byte into an uint8 for return.
func readVarUInt8(r io.Reader) (uint8, error) {
	bits, err := readVarNumber(1, r)
	if err != nil {
		return 0, err
	}

	// Ignore the stop bit.
	return bits[0] & 0x7F, nil
}

// readVarUInt16 reads until the high bit is set, which signals the end of the
// variable-length number, or we hit the maximum number of bytes for uint16.
// We compress the number it into a uint16.  When used to express a length in
// bytes, the max value would signal a size of  63KB.
func readVarUInt16(r io.Reader) (uint16, error) {
	// Since we are being given seven bits per byte, we can fit 2 1/4 bytes
	// of input into our two bytes of value, so don't read more than 3 bytes.
	bits, err := readVarNumber(3, r)
	if err != nil {
		return 0, err
	}

	// 0xFC == 0b1111_1100.
	if (len(bits) == 3) && (bits[2]&0xFC != 0) {
		return 0, errors.Errorf("number is too big to fit into uint16: % #x", bits)
	}

	var ret uint16
	// Compact all of the bits into a uint16, ignoring the stop bit.
	// Turn [0111 0001] [1110 0001] into [0011 1000] [1110 0001].
	for _, b := range bits {
		ret <<= 7
		ret |= uint16(b & 0x7F)
	}

	return ret, nil
}

// readVarUInt32 reads until the high bit is set, which signals the end of the
// variable-length number, or we hit the maximum number of bytes for uint32.
// We compress the number into a uint32.  When used to express a length in
// bytes, the max value would signal a size of 3GB.
func readVarUInt32(r io.Reader) (uint32, error) {
	// Since we are being given seven bits per byte, we can fit 4 1/2 bytes
	// of input into our four bytes of value, so don't read more than 5 bytes.
	bits, err := readVarNumber(5, r)
	if err != nil {
		return 0, err
	}

	// 0xF0 == 0b1111_0000.
	if (len(bits) == 5) && (bits[4]&0xF0 != 0) {
		return 0, errors.Errorf("number is too big to fit into uint32: % #x", bits)
	}

	var ret uint32
	// Compact all of the bits into a uint32, ignoring the stop bit.
	// Turn [0111 1111] [1110 1111] into [0011 1111] [1110 1111].
	for _, b := range bits {
		ret <<= 7
		ret |= uint32(b & 0x7F)
	}

	return ret, nil
}

// readVarInt64 reads until the high bit is set, which signals the end of the
// variable-length number, or we hit the maximum number of bytes for int64.
// We compress the number into an int64.
func readVarInt64(r io.Reader) (int64, error) {
	// Since we are being given seven bits per byte, we can fit 9 1/8 bytes
	// of input into our eight bytes of value, so don't read more than 10 bytes.
	bits, err := readVarNumber(10, r)
	if err != nil {
		return 0, err
	}

	// 0xFE == 0b1111_1110.
	if (len(bits) == 10) && (bits[9]&0xFE != 0) {
		return 0, errors.Errorf("number is too big to fit into int64: % #x", bits)
	}

	var ret int64
	// Compact all of the bits into an int64, ignoring the stop bit.
	// Turn [0111 1111] [1110 1111] into [0011 1111] [1110 1111].
	for i, b := range bits {
		ret <<= 7
		// Need to ignore the sign bit.  We add the sign later.
		if i == 0 {
			ret |= int64(b & 0x3F)
		} else {
			ret |= int64(b & 0x7F)
		}
	}

	// The second bit of the number is the sign bit.
	if bits[0]&0x40 != 0 {
		ret *= -1
	}

	return ret, nil
}

// readVarNumber reads until the high bit is set, which signals the end of the
// variable-length number, or maxBytes is hit.  If maxBytes is reached without
// the number being terminated, then an error is returned.  The bits are not modified.
func readVarNumber(maxBytes uint16, r io.Reader) ([]byte, error) {
	buf := make([]byte, 1)
	var bits []byte
	for {
		if n, err := r.Read(buf); err != nil || n != 1 {
			return nil, errors.Errorf("read %d bytes (wanted one) of number with err: %v", n, err)
		}
		bits = append(bits, buf[0])
		if (buf[0] & 0x80) != 0 {
			break
		}
		if uint16(len(bits)) >= maxBytes {
			return nil, errors.Errorf("number not terminated after %d bytes", maxBytes)
		}
	}

	return bits, nil
}

// readNibblesHighAndLow reads one byte from the given reader then returns
// the high nibble and the low nibble of that byte.  If read encounters the
// error io.EOF, then that error is returned.
func readNibblesHighAndLow(r io.Reader) (byte, byte, error) {
	buf := make([]byte, 1)
	if n, err := r.Read(buf); err != nil || n != 1 {
		if err == io.EOF {
			return 0, 0, err
		}
		return 0, 0, errors.Wrapf(err, "read %d bytes when wanted to read the one byte header", n)
	}
	return highNibble(buf[0]), lowNibble(buf[0]), nil
}

// highNibble returns a byte representation of the high-order nibble
// (half a byte) of the given byte.
func highNibble(b byte) byte {
	return (b >> 4) & 0x0F
}

// lowNibble returns a byte representation of the low-order nibble
// (half a byte) of the given byte.
func lowNibble(b byte) byte {
	return b & 0x0F
}
