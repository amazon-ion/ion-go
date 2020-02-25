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

type symbolTableKind uint8

const (
	symbolTableKindSystem symbolTableKind = iota
	symbolTableKindShared
	symbolTableKindLocal
)

// TODO Consider using `maligned` from `golangci-lint` on this struct.

// SymbolTable is the core lookup structure for Tokens from their symbolic ID.
type SymbolTable struct {
	kind    symbolTableKind
	name    string
	version int32
	imports []SymbolTable
	textMap map[string]SymbolToken
	tokens  []SymbolToken
	maxSID  int64
}

func newSymbolTableRaw(kind symbolTableKind, name string, version int32) *SymbolTable {
	table := SymbolTable{
		kind:    kind,
		name:    name,
		version: version,
		textMap: make(map[string]SymbolToken),
	}
	return &table
}

// newSymbolTable constructs a new empty symbol table of the given type.
// The desc and version are applicable for shared/system symbol tables and should be default
// values for local symbol tables.
// The given imports will be loaded into the newly constructed symbol table.
func newSymbolTable(kind symbolTableKind, name string, version int32, imports ...SymbolTable) *SymbolTable {
	table := newSymbolTableRaw(kind, name, version)
	// Copy in the import "references."
	table.imports = append(imports[:0:0], imports...)
	if kind == symbolTableKindLocal {
		// For local symbol table, import system symbol table.
		imports = append([]SymbolTable{systemSymbolTable}, imports...)
	}
	for _, importTable := range imports {
		// TODO Consider if we should not inline the imports or make this delegate to the imports or configurable.
		table.tokens = append(table.tokens, importTable.tokens...)
		for text, newToken := range importTable.textMap {
			if _, exists := table.textMap[text]; !exists {
				table.textMap[text] = newToken
			}
		}
		table.maxSID += importTable.maxSID
	}
	return table
}

// newLocalSymbolTable creates an instance with symbolTableKindLocal.
// The desc is empty and the version is zero. These fields are inapplicable to local symbol tables.
func newLocalSymbolTable(imports ...SymbolTable) *SymbolTable {
	return newSymbolTable(symbolTableKindLocal, "", 0, imports...)
}

// newSharedSymbolTable creates an instance with symbolTableKindShared.
// Returns `nil` if the version is not positive.
func newSharedSymbolTable(name string, version int32, imports ...SymbolTable) *SymbolTable {
	if version <= 0 {
		return nil
	}
	return newSymbolTable(symbolTableKindShared, name, version, imports...)
}

// BySID returns the underlying SymbolToken by local ID.
func (t *SymbolTable) BySID(sid int64) (SymbolToken, bool) {
	if sid <= 0 || sid > t.maxSID {
		return symbolTokenUndefined, false
	}
	return t.tokens[sid-1], true
}

// ByText returns the underlying SymbolToken by text lookup.
func (t *SymbolTable) ByText(text string) (SymbolToken, bool) {
	if tok, exists := t.textMap[text]; exists {
		return tok, true
	}
	return symbolTokenUndefined, false
}

// InternToken adds a symbol to the given table if it does not exist.
func (t *SymbolTable) InternToken(symText string) SymbolToken {
	if tok, exists := t.textMap[symText]; exists {
		return tok
	}

	t.maxSID += 1
	var source *ImportSource = nil
	if t.kind != symbolTableKindLocal {
		// defining a token within a shared/system table makes the token have a source referring to its
		// own table and SID
		source = newSource(t.name, t.maxSID)
	}

	tok := SymbolToken{
		Text:     &symText,
		localSID: t.maxSID,
		Source:   source,
	}
	t.tokens = append(t.tokens, tok)
	t.textMap[symText] = tok
	return tok
}
