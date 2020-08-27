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

	bits bitstream
	cat  Catalog
	lst  SymbolTable
}

func newBinaryReaderBuf(in *bufio.Reader, cat Catalog) Reader {
	r := &binaryReader{
		cat:    cat,
		reader: reader{fieldNameSymbol: symbolTokenUndefined},
	}
	r.bits.Init(in)
	return r
}

// SymbolTable returns the current symbol table.
func (r *binaryReader) SymbolTable() SymbolTable {
	return r.lst
}

// Next moves the reader to the next value.
func (r *binaryReader) Next() bool {
	if r.eof || r.err != nil {
		return false
	}

	r.Clear()

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
		r.valueType = SymbolType
		if !r.bits.IsNull() {
			id, err := r.bits.ReadSymbolID()
			if err != nil {
				return false, err
			}

			text, ok := r.SymbolTable().FindByID(id)
			if !ok {
				r.value = SymbolToken{LocalSID: (int64)(id)}
			} else {
				r.value = SymbolToken{Text: &text, LocalSID: (int64)(id)}
			}
		}
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

			st, err := readLocalSymbolTable(r, r.cat)
			if err == nil {
				r.lst = st
				return false, nil
			}
			return false, err
		}

		return true, nil
	}
	panic(fmt.Sprintf("invalid bitcode %v", code))
}

func isIonSymbolTable(as []string) bool {
	return len(as) > 0 && as[0] == "$ion_symbol_table"
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

	s, ok := r.lst.FindByID(id)
	if !ok {
		r.fieldNameSymbol.Text = nil
	} else {
		r.fieldNameSymbol.Text = &s
	}
	r.fieldNameSymbol.LocalSID = int64(id)
	return nil
}

// ReadAnnotations reads and resolves a set of annotations.
func (r *binaryReader) readAnnotations() error {
	ids, err := r.bits.ReadAnnotationIDs()
	if err != nil {
		return err
	}

	as := make([]string, len(ids))
	for i, id := range ids {
		as[i] = r.resolve(id)
	}

	r.annotations = as
	return nil
}

// Resolve resolves a symbol ID to a symbol value (possibly ${id} if we're
// missing the appropriate symbol table).
func (r *binaryReader) resolve(id uint64) string {
	s, ok := r.lst.FindByID(id)
	if !ok {
		return fmt.Sprintf("$%v", id)
	}
	return s
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
	r.Clear()
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

	r.Clear()
	r.ctx.pop()
	r.eof = false

	return nil
}

// StringValue returns the current value as a string.
func (r *binaryReader) StringValue() (string, error) {
	if r.valueType != StringType && r.valueType != SymbolType {
		return "", &UsageError{"Reader.StringValue", "value is not a string"}
	}

	if r.value == nil {
		return "", nil
	}

	// check if value is symbol or string.
	st, ok := r.value.(SymbolToken)
	if !ok {
		return r.value.(string), nil
	}
	return r.resolve(uint64(st.LocalSID)), nil
}

// FieldNameSymbol returns the current field name as a symbol token.
func (r *binaryReader) FieldNameSymbol() (SymbolToken, error) {
	if r.fieldNameSymbol.LocalSID == SymbolIDUnknown {
		return symbolTokenUndefined, nil
	}

	text, ok := r.SymbolTable().FindByID(uint64(r.fieldNameSymbol.LocalSID))
	if !ok {
		return SymbolToken{LocalSID: r.fieldNameSymbol.LocalSID}, nil
	}

	return SymbolToken{Text: &text, LocalSID: r.fieldNameSymbol.LocalSID}, nil
}

// SymbolValue returns the current value as a symbol token.
func (r *binaryReader) SymbolValue() (SymbolToken, error) {
	if r.valueType != SymbolType {
		return symbolTokenUndefined, &UsageError{"Reader.SymbolValue", "value is not a symbol"}
	}

	return r.value.(SymbolToken), nil
}
