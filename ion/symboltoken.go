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

// UnknownSid is the default SID, which is unknown.
const UnknownSid = -1

// A SymbolToken providing both the symbol text and the assigned symbol ID.
// Symbol tokens may be interned into a SymbolTable.
// Text = nil or SID = -1 value might indicate that such field is unknown in the contextual symbol table.
type SymbolToken struct {
	Text           *string
	SID            int64
	importLocation ImportLocation
}

// Equal figures out if two symbol tokens are equal for each component.
func (st *SymbolToken) Equal(o *SymbolToken) bool {
	if st.Text == nil || o.Text == nil {
		if st.Text == nil && o.Text == nil && st.SID == o.SID {
			return true
		} else {
			return false
		}
	} else {
		return *st.Text == *o.Text && st.SID == o.SID
	}
}
