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

var UnknownSid int64 = -1

// A SymbolToken providing both the symbol text and the assigned symbol ID.
// Symbol tokens may be interned into a SymbolTable.
// A text=nil or sid=-1 value might indicate that such field is unknown in the contextual symbol table.
type SymbolToken struct {
	text           *string
	sid            int64
	importLocation *ImportLocation
}

// Gets the ID of this symbol token.
func (st *SymbolToken) Sid() int64 {
	return st.sid
}

// Gets the text of this symbol token.
func (st *SymbolToken) Text() *string {
	return st.text
}

func (st *SymbolToken) Equal(o *SymbolToken) bool {
	return *st.text == *o.text && st.sid == o.sid
}

// NewSymbolToken creates a SymbolToken struct.
func NewSymbolToken(text *string, sid int64, importLocation *ImportLocation) *SymbolToken {
	return &SymbolToken{text: text, sid: sid, importLocation: importLocation}
}
