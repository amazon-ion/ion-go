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

	"github.com/pkg/errors"
)

// This file contains binary parsers for List, SExp, Struct, and Annotation.

// parseBinaryList attempts to read and parse the entirety of the list whether
// it be a List (high == binaryTypeList) or SExp (high == binaryTypeSExp).
func parseBinaryList(ann []Symbol, high byte, lengthByte byte, r io.Reader) (Value, error) {
	if lengthByte == 0 && high == binaryTypeList {
		return List{annotations: ann, values: []Value{}}, nil
	}
	if lengthByte == 0 && high == binaryTypeSExp {
		return SExp{annotations: ann, values: []Value{}}, nil
	}

	numBytes, errLength := determineLength32(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of list")
	}

	data := make([]byte, numBytes)
	if n, err := r.Read(data); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read list - read %d bytes of %d with err: %v", n, numBytes, err)
	}

	var values []Value
	dataReader := bytes.NewReader(data)
	for dataReader.Len() > 0 {
		value, err := parseNextBinaryValue(nil, dataReader)
		if err != nil {
			return nil, errors.WithMessage(err, "unable to parse list")
		}
		values = append(values, value)
	}

	if high == binaryTypeList {
		return List{annotations: ann, values: values}, nil
	}
	return SExp{annotations: ann, values: values}, nil
}

// parseBinaryStruct reads all of the symbol / value pairs and puts them
// into a Struct.
func parseBinaryStruct(ann []Symbol, lengthByte byte, r io.Reader) (Value, error) {
	if lengthByte == 0 {
		return Struct{annotations: ann, fields: []StructField{}}, nil
	}

	var numBytes uint32
	var errLength error
	// "When L is 1, the struct has at least one symbol/value pair, the length
	// field exists, and the field desc integers are sorted in increasing order."
	if lengthByte == 1 {
		numBytes, errLength = readVarUInt32(r)
	} else {
		numBytes, errLength = determineLength32(lengthByte, r)
	}

	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of struct")
	}

	data := make([]byte, numBytes)
	if n, err := r.Read(data); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read struct - read %d bytes of %d with err: %v", n, numBytes, err)
	}

	// Not having any fields isn't the same as being null, so differentiate
	// between the two by ensuring that fields isn't nil even if it's empty.
	fields := []StructField{}
	dataReader := bytes.NewReader(data)
	for dataReader.Len() > 0 {
		symbol, errSymbol := readVarUInt32(dataReader)
		if errSymbol != nil {
			return nil, errors.WithMessage(errSymbol, "unable to read struct field symbol")
		}
		value, errValue := parseNextBinaryValue(nil, dataReader)
		if errValue != nil {
			return nil, errors.WithMessage(errValue, "unable to read struct field value")
		}

		// Ignore padding.
		if value.Type() == TypePadding {
			continue
		}

		fields = append(fields, StructField{
			Symbol: Symbol{id: int32(symbol)},
			Value:  value,
		})
	}

	return Struct{annotations: ann, fields: fields}, nil
}

// parseBinaryAnnotation reads the annotation and the value that it is
// annotating.  If the lengthByte is zero, then this is treated as the
// first byte of a Binary Version Marker.
func parseBinaryAnnotation(lengthByte byte, r io.Reader) (Value, error) {
	// 0xE as the high byte has two potential uses, one for annotations and one for the
	// start of the binary version marker.  We are going to be optimistic and assume that
	// 0xE0 is for the BVM and all other values for the low nibble is for annotations.
	if lengthByte == 0 {
		return parseBinaryVersionMarker(r)
	}

	if lengthByte < 3 {
		return nil, errors.Errorf("length must be at least 3 for an annotation wrapper, found %d", lengthByte)
	}

	numBytes, errLength := determineLength32(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of annotation")
	}

	data := make([]byte, numBytes)
	if n, err := r.Read(data); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read annotation - read %d bytes of %d with err: %v", n, numBytes, err)
	}

	dataReader := bytes.NewReader(data)
	annLen, errAnnLen := readVarUInt16(dataReader)
	if errAnnLen != nil {
		return nil, errors.WithMessage(errAnnLen, "unable to determine annotation symbol length")
	}

	if annLen == 0 || uint32(annLen) >= numBytes {
		return nil, errors.Errorf("invalid lengths for annotation - field length is %d while annotation symbols length is %d", numBytes, annLen)
	}

	annData := make([]byte, annLen)
	// We've already verified lengths and are basically performing a copy to
	// a pre-allocated byte slice.  There is no error to catch.
	_, _ = dataReader.Read(annData)

	annReader := bytes.NewReader(annData)
	var annotations []Symbol
	for annReader.Len() > 0 {
		symbol, errSymbol := readVarUInt32(annReader)
		if errSymbol != nil {
			return nil, errors.WithMessage(errSymbol, "unable to read annotation symbol")
		}
		annotations = append(annotations, Symbol{id: int32(symbol)})
	}

	// Since an annotation is a container for a single value there isn't a need to
	// pre-read the contents so that we know when to stop.
	value, errValue := parseNextBinaryValue(annotations, dataReader)
	if errValue != nil {
		return nil, errors.WithMessage(errValue, "unable to read annotation value")
	}

	if dataReader.Len() > 0 {
		return nil, errors.Errorf("annotation declared %d bytes but there are %d bytes left", numBytes, dataReader.Len())
	}

	if _, ok := value.(padding); ok {
		return nil, errors.New("annotation on padding is not legal")
	}

	return value, nil
}
