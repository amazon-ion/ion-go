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
	"fmt"
	"strconv"
)

const (
	// SymbolIDUnknown is placeholder for when a symbol token has no symbol ID.
	SymbolIDUnknown = -1
)

// ImportSource is a reference to a SID within a shared symbol table.
type ImportSource struct {
	// The name of the shared symbol table that this token refers to.
	Table string

	// The ID of the interned symbol text within the shared SymbolTable.
	// This must be greater or equal to 1.
	SID int64
}

func newSource(table string, sid int64) *ImportSource {
	value := ImportSource{
		Table: table,
		SID:   sid,
	}
	return &value
}

// Equal figures out if two import sources are equal for each component.
func (is *ImportSource) Equal(o *ImportSource) bool {
	return is.Table == o.Table && is.SID == o.SID
}

// SymbolToken is the representation for annotations, field names, and the textual content of Ion symbol values.
// The `nil` value for SymbolToken is the SID `$0`.
type SymbolToken struct {
	// The string text of the token or nil if unknown.
	Text *string
	// Local symbol ID associated with the token.
	LocalSID int64
	// The shared symbol table location that this token came from, or nil if undefined.
	Source *ImportSource
}

func (st *SymbolToken) String() string {
	text := "nil"
	if st.Text != nil {
		text = fmt.Sprintf("%q", *st.Text)
	}

	source := "nil"
	if st.Source != nil {
		source = fmt.Sprintf("{%q %d}", st.Source.Table, st.Source.SID)
	}

	return fmt.Sprintf("{%s %d %s}", text, st.LocalSID, source)
}

// Equal figures out if two symbol tokens are equivalent.
func (st *SymbolToken) Equal(o *SymbolToken) bool {
	if st.Text == nil && o.Text == nil {
		if st.Source == nil && o.Source == nil {
			return true
		}
		if st.Source != nil && o.Source != nil {
			return st.Source.Equal(o.Source)
		}
		return false
	}

	if st.Text != nil && o.Text != nil {
		return *st.Text == *o.Text
	}
	return false
}

// Parses text of the form '$n' for some integer n.
func symbolIdentifier(symbolText string) (int64, bool) {
	if len(symbolText) > 1 && symbolText[0] == '$' {
		if sid, err := strconv.Atoi(symbolText[1:]); err == nil {
			return int64(sid), true
		}
	}

	return SymbolIDUnknown, false
}

// NewSymbolTokenFromString returns a Symbol Token with the given text value and undefined SID and Source.
func NewSymbolTokenFromString(text string) SymbolToken {
	return SymbolToken{Text: &text, LocalSID: SymbolIDUnknown}
}

func newSymbolTokenPtrFromString(text string) *SymbolToken {
	return &SymbolToken{Text: &text, LocalSID: SymbolIDUnknown}
}

// NewSymbolTokenBySID will check and return a symbol token if the given id exists in a symbol table,
// otherwise return a new symbol token.
func NewSymbolTokenBySID(symbolTable SymbolTable, sid int64) (SymbolToken, error) {
	if sid < 0 || uint64(sid) > symbolTable.MaxID() {
		return SymbolToken{}, fmt.Errorf("ion: Symbol token not found for SID '%v' in symbol table %v", sid, symbolTable)
	}

	text, ok := symbolTable.FindByID(uint64(sid))
	if !ok {
		return SymbolToken{LocalSID: sid}, nil
	}

	return SymbolToken{Text: &text, LocalSID: sid}, nil
}

// NewSymbolToken will check and return a symbol token if it exists in a symbol table,
// otherwise return a new symbol token.
func NewSymbolToken(symbolTable SymbolTable, text string) (SymbolToken, error) {
	if symbolTable == nil {
		return SymbolToken{}, fmt.Errorf("ion: invalid symbol table")
	}

	sid, ok := symbolTable.FindByName(text)
	if !ok {
		return SymbolToken{Text: &text, LocalSID: SymbolIDUnknown}, nil
	}

	return SymbolToken{Text: &text, LocalSID: int64(sid)}, nil
}

// NewSymbolTokens will check and return a list of symbol tokens if they exists in a symbol table,
// otherwise return a list of new symbol tokens.
func NewSymbolTokens(symbolTable SymbolTable, textVals []string) ([]SymbolToken, error) {
	var tokens []SymbolToken
	for _, text := range textVals {
		token, err := NewSymbolToken(symbolTable, text)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}

func newSymbolToken(symbolTable SymbolTable, text string) (SymbolToken, error) {
	if sid, ok := symbolIdentifier(text); ok {
		return NewSymbolTokenBySID(symbolTable, sid)
	}
	return NewSymbolToken(symbolTable, text)
}
