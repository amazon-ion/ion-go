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

import "fmt"

// A Type represents the type of an Ion Value.
type Type uint8

const (
	// NoType is returned by a Reader that is not currently pointing at a value.
	NoType Type = iota

	// NullType is the type of the (unqualified) Ion null value.
	NullType

	// BoolType is the type of an Ion boolean, true or false.
	BoolType

	// IntType is the type of a signed Ion integer of arbitrary size.
	IntType

	// FloatType is the type of a fixed-precision Ion floating-point value.
	FloatType

	// DecimalType is the type of an arbitrary-precision Ion decimal value.
	DecimalType

	// TimestampType is the type of an arbitrary-precision Ion timestamp.
	TimestampType

	// SymbolType is the type of an Ion symbol, mapped to an integer ID by a SymbolTable
	// to (potentially) save space.
	SymbolType

	// StringType is the type of a non-symbol Unicode string, represented directly.
	StringType

	// ClobType is the type of a character large object. Like a BlobType, it stores an
	// arbitrary sequence of bytes, but it represents them in text form as an escaped-ASCII
	// string rather than a base64-encoded string.
	ClobType

	// BlobType is the type of a binary large object; a sequence of arbitrary bytes.
	BlobType

	// ListType is the type of a list, recursively containing zero or more Ion values.
	ListType

	// SexpType is the type of an s-expression. Like a ListType, it contains a sequence
	// of zero or more Ion values, but with a lisp-like syntax when encoded as text.
	SexpType

	// StructType is the type of a structure, recursively containing a sequence of named
	// (by an Ion symbol) Ion values.
	StructType
)

// String implements fmt.Stringer for Type.
func (t Type) String() string {
	switch t {
	case NoType:
		return "<no type>"
	case NullType:
		return "null"
	case BoolType:
		return "bool"
	case IntType:
		return "int"
	case FloatType:
		return "float"
	case DecimalType:
		return "decimal"
	case TimestampType:
		return "timestamp"
	case StringType:
		return "string"
	case SymbolType:
		return "symbol"
	case BlobType:
		return "blob"
	case ClobType:
		return "clob"
	case StructType:
		return "struct"
	case ListType:
		return "list"
	case SexpType:
		return "sexp"
	default:
		return fmt.Sprintf("<unknown type %v>", uint8(t))
	}
}

// IsScalar determines if the type is a scalar type
func IsScalar(t Type) bool {
	return NullType <= t && t <= BlobType
}

// IsContainer determines if the type is a container type
func IsContainer(t Type) bool {
	return ListType <= t && t <= StructType
}

// IntSize represents the size of an integer.
type IntSize uint8

const (
	// NullInt is the size of null.int and other things that aren't actually ints.
	NullInt IntSize = iota
	// Int32 is the size of an Ion integer that can be losslessly stored in an int32.
	Int32
	// Int64 is the size of an Ion integer that can be losslessly stored in an int64.
	Int64
	// BigInt is the size of an Ion integer that can only be losslessly stored in a big.Int.
	BigInt
)

// String implements fmt.Stringer for IntSize.
func (i IntSize) String() string {
	switch i {
	case NullInt:
		return "null.int"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case BigInt:
		return "big.Int"
	default:
		return fmt.Sprintf("<unknown size %v>", uint8(i))
	}
}
