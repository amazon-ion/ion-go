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
	"math/big"
	"time"
)

type Event interface {
	// Returns the depth into the Ion value that this Reader has traversed. At top level the depth is 0, and
	// it increases by one on each call to StepIn().
	Depth() int

	// Returns the symbol table that is applicable to the current value.
	SymbolTable() SymbolTable

	// Returns the type of the current value, or null if there is no current value.
	Type() Type

	// Return the annotations of the current value as an array of strings.
	TypeAnnotations() []string

	// Returns the current value's annotations as symbol tokens (text + ID).
	TypeAnnotationSymbols() []SymbolToken

	// Determines if the current value contains such annotation.
	HasAnnotation(annotation string) bool

	// Return the field name of the current value, or null if there is no current value.
	FieldName() string

	// Gets the current value's field name as a symbol token (text + ID).
	FieldNameSymbol() SymbolToken

	// Determines whether the current value is a null Ion value of any type.
	IsNullValue() bool

	// Returns the current value as a bool. It returns an error if the current value is not an Ion Bool.
	BoolValue() (bool, error)

	// IntSize returns the size of integer needed to losslessly represent the current value. It returns an
	// error if the current value is not an Ion Int.
	IntSize() (IntSize, error)

	// Returns the current value as a 32-bit integer. It returns an error if the current value is not an Ion Integer
	// or requires more than 32 bits to represent losslessly.
	IntValue() (int, error)

	// Returns the current value as a 64-bit integer. It returns an error if the current value is not an Ion Integer
	// or requires more than 64 bits to represent losslessly.
	Int64Value() (int64, error)

	// Returns the current value as a big.Integer. It returns an error if the current value is not an Ion Integer.
	BigIntValue() (*big.Int, error)

	// Returns the current value as a 64-bit floating point number. It returns an error if the current value is
	// not an Ion Float.
	FloatValue() (float64, error)

	// Returns the current value as an arbitrary-precision Decimal. It returns an error if the current value is
	// not an Ion Decimal.
	DecimalValue() (*Decimal, error)

	// Returns the current value as a timestamp. It returns an error if the current value is not an Ion Timestamp.
	TimeValue() (time.Time, error)

	// Returns the current value as a string. It returns an error if the current value is not an
	// Ion Symbol or an Ion String.
	StringValue() (string, error)

	// Returns the current value as a SymbolToken. It returns an error if the current value is not an Ion Symbol.
	SymbolValue() (SymbolToken, error)

	// Returns the current value as a byte slice. It returns an error if the current value is not an Ion Clob
	// or an Ion Blob.
	ByteValue() ([]byte, error)

	EventType() container
}

// Holds the commonalities between binary and text readers.
type event struct {
	containerStack 			containerStack
	symbolTable				SymbolTable
	valueType				Type
	value					interface{}
}

// Returns the current value's depth.
func (e event) Depth() int {
	return e.containerStack.len()
}

// Returns the current value's symbol table.
func (e event) SymbolTable() SymbolTable {
	return e.symbolTable
}

// Returns the current value's type.
func (e event) Type() Type {
	return e.valueType
}

// Returns true if the current value is null.
func (e event) IsNullValue() bool {
	return e.value == nil
}

// Returns the current value as a bool.
func (e event) BoolValue() (bool, error) {
	return e.value.(bool), nil
}

// Returns the size of the current int value.
func (e event) IntSize() (IntSize, error) {
	return e.value.(IntSize), nil
}

// Returns the current value as an int.
func (e event) IntValue() (int, error) {
	return e.value.(int), nil
}

// Returns the current value as an int64.
func (e event) Int64Value() (int64, error) {
	return e.value.(int64), nil
}

// Returns the current value as a big int.
func (e event) BigIntValue() (*big.Int, error) {
	return e.value.(*big.Int), nil
}

// Returns the current value as a float.
func (e event) FloatValue() (float64, error) {
	return e.value.(float64), nil
}

// Returns the current value as a Decimal.
func (e event) DecimalValue() (*Decimal, error) {
	return e.value.(*Decimal), nil
}

// Returns the current value as a time.
func (e event) TimeValue() (time.Time, error) {
	return e.value.(time.Time), nil
}

// Returns the current value as a string.
func (e event) StringValue() (string, error) {
	return e.value.(string), nil
}

// Returns the current value as a byte slice.
func (e event) ByteValue() ([]byte, error) {
	return e.value.([]byte), nil
}

// Returns the current value as a symbol token.
func (e event) SymbolValue() (SymbolToken, error) {
	return e.value.(SymbolToken), nil
}

// Returns the current value's event type.
func (e event) EventType() container {
	return e.containerStack.peek()
}
