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
)

// This file contains the container-like types: List, SExp, and Struct.

const (
	textNullList   = "null.list"
	textNullSExp   = "null.sexp"
	textNullStruct = "null.struct"
)

// List is an ordered collections of Values.  The contents of the list are
// heterogeneous, each element can have a different type. Homogeneous lists
// may be imposed by schema validation tools.
type List struct {
	annotations []Symbol
	values      []Value
}

// Value returns the values that this list holds.
func (lst List) Value() []Value {
	return lst.values
}

// Annotations satisfies Value.
func (lst List) Annotations() []Symbol {
	return lst.annotations
}

// Binary satisfies Value.
func (lst List) Binary() []byte {
	// TODO: Figure out how we want to do binary serialization of containers.
	return nil
}

// Text satisfies Value.
func (lst List) Text() []byte {
	if lst.values == nil {
		return []byte(textNullList)
	}

	parts := make([][]byte, len(lst.values))
	for index, value := range lst.values {
		parts[index] = value.Text()
	}

	return bytes.Join(parts, []byte(","))
}

// IsNull satisfies Value.
func (lst List) IsNull() bool {
	return lst.values == nil
}

// Type satisfies Value.
func (List) Type() Type {
	return TypeList
}

// SExp (S-Expression) is an ordered collection of values with application-defined
// semantics.  The contents of the list are
// heterogeneous, each element can have a different type. Homogeneous lists
// may be imposed by schema validation tools.
type SExp struct {
	annotations []Symbol
	values      []Value
}

// Value returns the values held within the s-expression.
func (s SExp) Value() []Value {
	return s.values
}

// Annotations satisfies Value.
func (s SExp) Annotations() []Symbol {
	return s.annotations
}

// Binary satisfies Value.
func (s SExp) Binary() []byte {
	// TODO: Figure out how we want to do binary serialization of containers.
	return nil
}

// Text satisfies Value.
func (s SExp) Text() []byte {
	if s.values == nil {
		return []byte(textNullSExp)
	}

	parts := make([][]byte, len(s.values))
	for index, value := range s.values {
		parts[index] = value.Text()
	}

	return bytes.Join(parts, []byte(" "))
}

// IsNull satisfies Value.
func (s SExp) IsNull() bool {
	return s.values == nil
}

// Type satisfies Value.
func (SExp) Type() Type {
	return TypeSExp
}

// StructField represents the field of a Struct.
type StructField struct {
	Symbol Symbol
	Value  Value
}

// Struct is an unordered collection of tagged values.
// When two fields in the same struct have the same desc we say there
// are “repeated names” or “repeated fields”. All such fields must be
// preserved, any StructField that has a repeated desc must not be discarded.
type Struct struct {
	annotations []Symbol
	fields      []StructField
}

// Value returns the fields that this struct holds.
func (s Struct) Value() []StructField {
	return s.fields
}

// Annotations satisfies Value.
func (s Struct) Annotations() []Symbol {
	return s.annotations
}

// Binary satisfies Value.
func (s Struct) Binary() []byte {
	// TODO: Figure out how we want to do binary serialization of containers.
	return nil
}

// Text satisfies Value.
func (s Struct) Text() []byte {
	if s.fields == nil {
		return []byte(textNullStruct)
	}

	parts := make([][]byte, len(s.fields))
	for index, fld := range s.fields {
		line := append(fld.Symbol.Text(), ':')
		parts[index] = append(line, fld.Value.Text()...)
	}

	return bytes.Join(parts, []byte(","))
}

// IsNull satisfies Value.
func (s Struct) IsNull() bool {
	return s.fields == nil
}

// Type satisfies Value.
func (Struct) Type() Type {
	return TypeStruct
}
