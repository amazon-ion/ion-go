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

type Type int

const (
	TypeNull Type = iota
	TypeAnnotation
	TypeBlob
	TypeBool
	TypeClob
	TypeDecimal
	TypeFloat
	TypeInt
	TypeList
	TypeLongString
	TypePadding
	TypeSExp
	TypeString
	TypeStruct
	TypeSymbol
	TypeTimestamp
)

var typeNameMap = map[Type]string{
	TypeNull: "Null", TypeAnnotation: "Annotation", TypeBlob: "Blob", TypeClob: "Clob", TypeDecimal: "Decimal",
	TypeFloat: "Float", TypeInt: "Int", TypeList: "List", TypeLongString: "LongString", TypePadding: "Padding",
	TypeSExp: "S-Expression", TypeString: "String", TypeStruct: "Struct", TypeSymbol: "Symbol", TypeTimestamp: "Timestamp",
}

// String satisfies Stringer.
func (t Type) String() string {
	if s, ok := typeNameMap[t]; ok {
		return s
	}
	return "Unknown"
}

const (
	textBoolFalse     = "false"
	textBoolTrue      = "true"
	textNullBool      = "null.bool"
	textNullTimestamp = "null.timestamp"
)

// Digest is a top-level Ion container of Ion values.  It is also the
// granularity of binary encoding Ion content.  It is not a defined
// type in the Ion spec, but is used as a container of types and Symbols.
type Digest struct {
	values []Value
}

// Value returns the Values that make up this Digest.
func (d Digest) Value() []Value {
	return d.values
}

// Value is a basic interface for all Ion types.
// http://amzn.github.io/ion-docs/docs/spec.html
type Value interface {
	// Annotations returns any annotations that have been set for this Value.
	Annotations() []Symbol
	// Binary returns the binary representation of the Value.
	// http://amzn.github.io/ion-docs/docs/binary.html
	Binary() []byte
	// Text returns the text representation of the Value.
	// http://amzn.github.io/ion-docs/docs/text.html
	Text() []byte
	// IsNull returns whether or not this instance of the value represents a
	// null value for a given type.
	// TODO: Determine if we want to use IsNull or use the Null struct.
	IsNull() bool
	// Type returns the Type of the Value.
	Type() Type

	// Note: There is no general "Value" function to retrieve the Go version of
	//       the underlying value because we would need to define it to return
	//       interface{}.  This decision may be revisited after playing with
	//       the library a bit.
}

// Padding represents no-op padding in a binary stream.
type padding struct {
	// Note that the name "length" is a little bit of a misnomer since a
	// padding of length n pads n+1 bytes.
	binary []byte
}

// Annotations satisfies Value.
func (p padding) Annotations() []Symbol {
	return nil
}

// Binary satisfies Value.
func (p padding) Binary() []byte {
	return p.binary
}

// Text satisfies Value.
func (p padding) Text() []byte {
	// Text padding isn't a thing.
	return nil
}

// IsNull satisfies Value.
func (p padding) IsNull() bool {
	return false
}

// Type satisfies Value.
func (p padding) Type() Type {
	return TypePadding
}

// Bool is the boolean type.
type Bool struct {
	annotations []Symbol
	isSet       bool
	value       bool
}

// Value returns the boolean value.  This will be false if it has not been set.
func (b Bool) Value() bool {
	if !b.isSet {
		return false
	}
	return b.value
}

// Annotations satisfies Value.
func (b Bool) Annotations() []Symbol {
	return b.annotations
}

// Binary satisfies Value.
func (b Bool) Binary() []byte {
	return nil
}

// Text satisfies Value.
func (b Bool) Text() []byte {
	if !b.isSet {
		return []byte(textNullBool)
	}
	if b.value {
		return []byte(textBoolTrue)
	}
	return []byte(textBoolFalse)
}

// IsNull satisfies Value.
func (b Bool) IsNull() bool {
	return !b.isSet
}

// Type satisfies Value.
func (b Bool) Type() Type {
	return TypeBool
}

// Null represents Null values and is able to take on the guise of
// any of the null-able types.
type Null struct {
	annotations []Symbol
	typ         Type
}

// Annotations satisfies Value.
func (n Null) Annotations() []Symbol {
	return n.annotations
}

// Binary satisfies Value.
func (n Null) Binary() []byte {
	// TODO: Implement returning a byte based on the null type.
	return nil
}

// Text satisfies Value.
func (n Null) Text() []byte {
	switch n.typ {
	case TypeBlob:
		return []byte("null.blob")
	case TypeBool:
		return []byte("null.bool")
	case TypeClob:
		return []byte("null.clob")
	case TypeDecimal:
		return []byte("null.decimal")
	case TypeFloat:
		return []byte("null.float")
	case TypeInt:
		return []byte("null.int")
	case TypeList:
		return []byte("null.list")
	case TypeLongString:
		return []byte("null.string")
	case TypeNull:
		return []byte("null.null")
	case TypeSExp:
		return []byte("null.sexp")
	case TypeString:
		return []byte("null.string")
	case TypeStruct:
		return []byte("null.struct")
	case TypeSymbol:
		return []byte("null.symbol")
	case TypeTimestamp:
		return []byte("null.timestamp")
	default:
		return []byte("null")
	}
}

// IsNull satisfies Value.
func (n Null) IsNull() bool {
	return true
}

// Type satisfies Value.
func (n Null) Type() Type {
	return n.typ
}
