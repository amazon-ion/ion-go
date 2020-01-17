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
	"io"

	"github.com/pkg/errors"
)

// This file contains binary parsers for Null, Padding, Bool, Symbol, String, Blob, and Clob.

// parseBinaryNull returns the null value for the given type.
func parseBinaryNull(typ byte) (Value, error) {
	if it, ok := binaryTypeMap[typ]; ok {
		return Null{typ: it}, nil
	}
	return nil, errors.Errorf("invalid type value for null: %d", typ)
}

// parseBinaryPadding returns a padding Value while consuming the padding
// value number of bytes.
func parseBinaryPadding(lengthByte byte, r io.Reader) (Value, error) {
	// Special case the "zero" length padding since we don't read anything for it.
	// The zero is in quotes since the marker of this is itself a byte of padding.
	if lengthByte == 0 {
		return padding{}, nil
	}

	numBytes, errLength := determineLength16(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of padding")
	}

	buf := make([]byte, numBytes)
	if n, err := r.Read(buf); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("read %d of expected %d padding bytes with err: %v", n, numBytes, err)
	}
	return padding{binary: buf}, nil
}

// parseBinaryBool returns the Bool Value for the given representation.
// 1 == true and 0 == false.  Null is handled by parseBinaryNull.
func parseBinaryBool(ann []Symbol, rep byte) (Value, error) {
	switch rep {
	case 0:
		return Bool{annotations: ann, isSet: true, value: false}, nil
	case 1:
		return Bool{annotations: ann, isSet: true, value: true}, nil
	default:
		return nil, errors.Errorf("invalid bool representation %#x", rep)
	}
}

// parseBinarySymbol parses an integer Symbol ID.
func parseBinarySymbol(ann []Symbol, lengthByte byte, r io.Reader) (Value, error) {
	if lengthByte == 0 {
		return Symbol{annotations: ann}, nil
	}

	numBytes, errLength := determineLength16(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of symbol")
	}

	// Sanity check the number of bytes expected for the Symbol ID.
	// If it takes more than 4 bytes of UInt, then it won't fit into int32.
	if numBytes > 4 {
		return nil, errors.Errorf("symbol ID length of %d bytes exceeds expected maximum of 4", numBytes)
	}

	buf := make([]byte, numBytes)
	if n, err := r.Read(buf); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read symbol ID - read %d bytes of %d with err: %v", n, numBytes, err)
	}

	var symbolID uint32
	for _, b := range buf {
		symbolID <<= 8
		symbolID |= uint32(b)
	}

	// math.MaxInt32 = 0x7F_FF_FF_FF
	if (symbolID & 0x80000000) != 0 {
		return nil, errors.Errorf("uint32 value %d overflows int32", symbolID)
	}

	return Symbol{annotations: ann, id: int32(symbolID)}, nil
}

// parseBinaryString reads the UTF-8 encoded string.
func parseBinaryString(ann []Symbol, lengthByte byte, r io.Reader) (Value, error) {
	if lengthByte == 0 {
		return String{annotations: ann, text: []byte{}}, nil
	}

	numBytes, errLength := determineLength16(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of string")
	}

	buf := make([]byte, numBytes)
	if n, err := r.Read(buf); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read string - read %d bytes of %d with err: %v", n, numBytes, err)
	}

	return String{annotations: ann, text: buf}, nil
}

// parseBinaryBytes reads the unencoded bytes, whether it be a Blob (high == binaryTypeBlob)
// or Clob (high == binaryTypeClob).
func parseBinaryBytes(ann []Symbol, high byte, lengthByte byte, r io.Reader) (Value, error) {
	numBytes, errLength := determineLength32(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of bytes")
	}

	buf := make([]byte, numBytes)
	if numBytes != 0 {
		if n, err := r.Read(buf); err != nil || n != int(numBytes) {
			return nil, errors.Errorf("unable to read bytes - read %d bytes of %d with err: %v", n, numBytes, err)
		}
	}

	if high == binaryTypeClob {
		return Clob{annotations: ann, text: buf}, nil
	}
	return Blob{annotations: ann, binary: buf}, nil
}
