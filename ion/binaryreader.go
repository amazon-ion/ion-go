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
	"fmt"
)

// A binaryReader reads binary Ion.
type binaryReader struct {
	reader

	bits     bitstream
	cat      Catalog
	resetPos uint64
}

func newBinaryReaderBuf(in *bufio.Reader, cat Catalog) Reader {
	r := &binaryReader{
		cat: cat,
	}
	r.bits.Init(in)
	return r
}

// Reset causes the binary reader to start reading from the given input bytes
// while skipping most of the initialization steps needed to prepare the
// reader. Reset is most commonly called with the same bytes as the reader
// was originally created with (e.g. via NewReaderBytes) as an optimization
// when the same data needs to be read multiple times.
//
// While it is possible to call Reset with different input bytes, the Reader
// will only work correctly if the new bytes contain the exact same binary
// version marker and local symbols as the original input. If there are any
// doubts whether this is the case, it is instead recommended to create a
// new Reader using NewReaderBytes (or NewReaderCat) instead. Attempting to
// reuse a binaryReader with inconsistent input bytes will cause the reader
// to return errors, misappropriate values to unrelated or non-existent
// attributes, or incorrectly parse data values.
//
// This API is experimental and should be considered unstable.
// See https://github.com/amazon-ion/ion-go/pull/196
func (r *binaryReader) Reset(in []byte) error {
	if r.resetPos == invalidReset {
		return &UsageError{"binaryReader.Reset", "cannot reset when multiple local symbol tables found"}
	}
	r.annotations = nil
	r.valueType = NoType
	r.value = nil
	r.err = nil
	r.eof = false
	r.bits = bitstream{}
	r.bits.InitBytes(in[r.resetPos:])
	return nil
}

const invalidReset uint64 = 1<<64 - 1

// Next moves the reader to the next value.
func (r *binaryReader) Next() bool {
	if r.eof || r.err != nil {
		return false
	}

	r.clear()

	done := false
	for !done {
		done, r.err = r.next()
		if r.err != nil {
			return false
		}
	}

	return !r.eof
}

// Next consumes the next raw value from the stream, returning true if it
// represents a user-facing value and false if it does not.
func (r *binaryReader) next() (bool, error) {
	if err := r.bits.Next(); err != nil {
		return false, err
	}

	code := r.bits.Code()
	switch code {
	case bitcodeEOF:
		r.eof = true
		return true, nil

	case bitcodeBVM:
		err := r.readBVM()
		return false, err

	case bitcodeFieldID:
		err := r.readFieldName()
		return false, err

	case bitcodeAnnotation:
		err := r.readAnnotations()
		return false, err

	case bitcodeNull:
		if !r.bits.IsNull() {
			// NOP padding; skip it and keep going.
			err := r.bits.SkipValue()
			return false, err
		}
		r.valueType = NullType
		return true, nil

	case bitcodeFalse, bitcodeTrue:
		r.valueType = BoolType
		if !r.bits.IsNull() {
			r.value = r.bits.Code() == bitcodeTrue
		}
		return true, nil

	case bitcodeInt, bitcodeNegInt:
		r.valueType = IntType
		if !r.bits.IsNull() {
			val, err := r.bits.ReadInt()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeFloat:
		r.valueType = FloatType
		if !r.bits.IsNull() {
			val, err := r.bits.ReadFloat()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeDecimal:
		r.valueType = DecimalType
		if !r.bits.IsNull() {
			val, err := r.bits.ReadDecimal()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeTimestamp:
		r.valueType = TimestampType
		if !r.bits.IsNull() {
			val, err := r.bits.ReadTimestamp()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeSymbol:
		if !r.bits.IsNull() {
			id, err := r.bits.ReadSymbolID()
			if err != nil {
				return false, err
			}
			st, err := NewSymbolTokenBySID(r.SymbolTable(), int64(id))
			if err != nil {
				return false, err
			}
			r.value = &st
		}
		r.valueType = SymbolType
		return true, nil

	case bitcodeString:
		r.valueType = StringType
		if !r.bits.IsNull() {
			val, err := r.bits.ReadString()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeClob:
		r.valueType = ClobType
		if !r.bits.IsNull() {
			val, err := r.bits.ReadBytes()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeBlob:
		r.valueType = BlobType
		if !r.bits.IsNull() {
			val, err := r.bits.ReadBytes()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeList:
		r.valueType = ListType
		if !r.bits.IsNull() {
			r.value = ListType
		}
		return true, nil

	case bitcodeSexp:
		r.valueType = SexpType
		if !r.bits.IsNull() {
			r.value = SexpType
		}
		return true, nil

	case bitcodeStruct:
		r.valueType = StructType
		if !r.bits.IsNull() {
			r.value = StructType
		}

		// If it's a local symbol table, install it and keep going.
		if r.ctx.peek() == ctxAtTopLevel && isIonSymbolTable(r.annotations) {
			if r.IsNull() {
				r.clear()
				r.lst = V1SystemSymbolTable
				return false, nil
			}
			st, err := readLocalSymbolTable(r, r.cat)
			if err == nil {
				r.lst = st
				if r.resetPos == 0 {
					r.resetPos = r.bits.pos
				} else {
					r.resetPos = invalidReset
				}
				return false, nil
			}
			return false, err
		}

		return true, nil
	}
	panic(fmt.Sprintf("invalid bitcode %v", code))
}

func isIonSymbolTable(as []SymbolToken) bool {
	return len(as) > 0 && as[0].Text != nil && *as[0].Text == "$ion_symbol_table"
}

// ReadBVM reads a BVM, validates it, and resets the local symbol table.
func (r *binaryReader) readBVM() error {
	major, minor, err := r.bits.ReadBVM()
	if err != nil {
		return err
	}

	switch major {
	case 1:
		switch minor {
		case 0:
			r.lst = V1SystemSymbolTable
			return nil
		}
	}

	return &UnsupportedVersionError{
		int(major),
		int(minor),
		r.bits.Pos() - 4,
	}
}

// ReadFieldName reads and resolves a field name.
func (r *binaryReader) readFieldName() error {
	id, err := r.bits.ReadFieldID()
	if err != nil {
		return err
	}

	st, err := NewSymbolTokenBySID(r.SymbolTable(), int64(id))
	if err != nil {
		return err
	}

	r.fieldName = &st
	return nil
}

// ReadAnnotations reads and resolves a set of annotations.
func (r *binaryReader) readAnnotations() error {
	as, err := r.bits.ReadAnnotations(r.SymbolTable())
	if err != nil {
		return err
	}

	r.annotations = as

	return nil
}

// StepIn steps in to a container-type value
func (r *binaryReader) StepIn() error {
	if r.err != nil {
		return r.err
	}

	if r.valueType != ListType && r.valueType != SexpType && r.valueType != StructType {
		return &UsageError{"Reader.StepIn", fmt.Sprintf("cannot step in to a %v", r.valueType)}
	}
	if r.value == nil {
		return &UsageError{"Reader.StepIn", "cannot step in to a null container"}
	}

	r.ctx.push(containerTypeToCtx(r.valueType))
	r.clear()
	r.bits.StepIn()

	return nil
}

// StepOut steps out of a container-type value.
func (r *binaryReader) StepOut() error {
	if r.err != nil {
		return r.err
	}
	if r.ctx.peek() == ctxAtTopLevel {
		return &UsageError{"Reader.StepOut", "cannot step out of top-level datagram"}
	}

	if err := r.bits.StepOut(); err != nil {
		return err
	}

	r.clear()
	r.ctx.pop()
	r.eof = false

	return nil
}
