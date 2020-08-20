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

var (
	// symbolTokenUndefined is the sentinel for invalid tokens.
	// The `nil` value is actually SID `$0` which is a defined token.
	symbolTokenUndefined = SymbolToken{
		LocalSID: SymbolIDUnknown,
	}
)

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

func getSystemSymbolMapping(symbolTable SymbolTable, symbolName string) string {
	// If we have a symbol name of the form '$n' for some integer n,
	// then we want to use the corresponding system symbol name.
	if len(symbolName) > 1 && symbolName[0] == '$' {
		if id, err := strconv.Atoi(symbolName[1:]); err == nil {
			if systemSymbolName, ok := symbolTable.FindByID(uint64(id)); ok {
				return systemSymbolName
			}
		}
	}

	return ""
}

// NewSymbolToken will check and return a symbol token if it exists in a symbol table,
// otherwise return a new symbol token.
func NewSymbolToken(symbolTable SymbolTable, text string) SymbolToken {
	systemSymbolName := getSystemSymbolMapping(symbolTable, text)
	if systemSymbolName != "" {
		return SymbolToken{Text: &systemSymbolName, LocalSID: SymbolIDUnknown}
	}

	token := symbolTable.Find(text)
	if token != nil {
		return *token
	}

	return SymbolToken{Text: &text, LocalSID: SymbolIDUnknown}
}

// NewSymbolTokens will check and return a list of symbol tokens if they exists in a symbol table,
// otherwise return a list of new symbol tokens.
func NewSymbolTokens(symbolTable SymbolTable, textVals []string) []SymbolToken {
	var tokens []SymbolToken
	for _, text := range textVals {
		tokens = append(tokens, NewSymbolToken(symbolTable, text))
	}

	return tokens
}
