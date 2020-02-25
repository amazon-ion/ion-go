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

const (
	// The placeholder for when a symbol token has no symbol ID.
	SymbolIDUnknown = -1
)

// ImportSource is a reference to a SID within a shared symbol table.
type ImportSource struct {
	// The name of the shared symbol table that this token refers to.
	Table string
	// The ID of the interned symbol text within the shared SymbolTable.
	// This must be greater than 1.
	SID int64
}

func newSource(table string, sid int64) *ImportSource {
	value := ImportSource{
		Table: table,
		SID:   sid,
	}
	return &value
}

// A symbolic token for Ion.
// Symbol tokens are the values that annotations, field names, and the textual content of Ion symbol values.
// The `nil` value for SymbolToken is $0.
type SymbolToken struct {
	// The string text of the token or nil if unknown.
	Text *string
	// Local symbol ID associated with the token.
	localSID int64
	// The shared symbol table location that this token came from, or nil if undefined.
	Source *ImportSource
}

var (
	// symbolTokenUndefined is the sentinel for invalid tokens.
	// The `nil` value is actually $0 which is a defined token.
	symbolTokenUndefined = SymbolToken{
		localSID: SymbolIDUnknown,
	}
)

func (t SymbolToken) String() string {
	text := "nil"
	if t.Text != nil {
		text = fmt.Sprintf("%q", *t.Text)
	}

	source := "nil"
	if t.Source != nil {
		source = fmt.Sprintf("{%q %d}", t.Source.Table, t.Source.SID)
	}

	return fmt.Sprintf("{%s %d %s}", text, t.localSID, source)
}
