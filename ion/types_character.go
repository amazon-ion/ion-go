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

// This file contains the string-like types: String and Symbol.

// String is a unicode text literal of arbitrary length.
type String struct {
	annotations []Symbol
	text        []byte
}

func (s String) Value() string {
	return string(s.text)
}

// Annotations satisfies Value.
func (s String) Annotations() []Symbol {
	return s.annotations
}

// Binary satisfies Value.
func (s String) Binary() []byte {
	// These are always sequences of Unicode characters, encoded as a sequence of UTF-8 octets.
	return s.text
}

// Text returns a string representation of the symbol if a string representation
// has been set.  Otherwise it will be empty.
func (s String) Text() []byte {
	return s.text
}

// IsNull satisfies Value.
func (s String) IsNull() bool {
	return s.text == nil
}

// Type satisfies Value.
func (String) Type() Type {
	return TypeString
}

// Symbol is an interned identifier that is represented as an ID
// and/or text.  If the id is 0 and the text is empty, then this
// represent null.symbol.
type Symbol struct {
	annotations []Symbol
	id          int32
	quoted      bool
	text        []byte
}

// Id returns the ID of the Symbol if it has been set, or SymbolIDUnknown if
// it has not.
func (s Symbol) Id() int32 {
	if s.id == 0 {
		return SymbolIDUnknown
	}
	return s.id
}

func (s Symbol) Value() string {
	// TODO: Things with Symbol tables and looking up the value when we
	//       only have an ID.
	return string(s.text)
}

// Annotations satisfies Value.
func (s Symbol) Annotations() []Symbol {
	return s.annotations
}

// Binary satisfies Value.
func (s Symbol) Binary() []byte {
	// TODO: Return symbol ID.
	return nil
}

// Text returns a string representation of the symbol if a string representation
// has been set.  Otherwise it will be empty.
func (s Symbol) Text() []byte {
	return s.text
}

// IsNull satisfies Value.
func (s Symbol) IsNull() bool {
	return s.id == 0 && len(s.text) == 0
}

// Type satisfies Value.
func (Symbol) Type() Type {
	return TypeSymbol
}
