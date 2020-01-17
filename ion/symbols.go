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

const (
	SymbolIDUnknown = -1

	SymbolTextIon = "$ion"
	SymbolIDIon   = 1

	// Version Identifier for Ion 1.0.
	SymbolTextIon10 = "$ion_1_0"
	SymbolIDIon10   = 2

	SymbolTextTable = "$ion_symbol_table"
	SymbolIDTable   = 3

	SymbolTextName = "name"
	SymbolIDName   = 4

	SymbolTextVersion = "version"
	SymbolIDVersion   = 5

	SymbolTextImports = "imports"
	SymbolIDImports   = 6

	SymbolTextSymbols = "symbols"
	SymbolIDSymbols   = 7

	SymbolTextMaxID = "max_id"
	SymbolIDMaxID   = 8

	SymbolTextSharedTable = "$ion_shared_symbol_table"
	SymbolIDSharedTable   = 9

	// Maximum ID of the IDs of system symbols defined by Ion 1.0.
	SymbolIDIon10MaxID = 9
)

var (
	systemTableIon10 = SymbolTable{
		mapping: map[string]Symbol{
			SymbolTextIon:         {id: SymbolIDIon, text: []byte(SymbolTextIon)},
			SymbolTextIon10:       {id: SymbolIDIon10, text: []byte(SymbolTextIon10)},
			SymbolTextTable:       {id: SymbolIDTable, text: []byte(SymbolTextTable)},
			SymbolTextName:        {id: SymbolIDName, text: []byte(SymbolTextName)},
			SymbolTextVersion:     {id: SymbolIDVersion, text: []byte(SymbolTextVersion)},
			SymbolTextImports:     {id: SymbolIDImports, text: []byte(SymbolTextImports)},
			SymbolTextSymbols:     {id: SymbolIDSymbols, text: []byte(SymbolTextSymbols)},
			SymbolTextMaxID:       {id: SymbolIDMaxID, text: []byte(SymbolTextMaxID)},
			SymbolTextSharedTable: {id: SymbolIDSharedTable, text: []byte(SymbolTextSharedTable)},
		},
		minID: SymbolIDIon,
		maxID: SymbolIDIon10MaxID,
	}
)

// SymbolTable contains a table of Symbols.  This table will be for a specific
// use, e.g. a table defining all of the Ion system symbols, or a custom table
// supplied to a parser, or a local table that is included in the Ion file itself.
type SymbolTable struct {
	mapping map[string]Symbol
	minID   int
	maxID   int
}

// ByID returns the Symbol in the table that matches the given ID,
// e.g. "name" for ID 4.
func (st SymbolTable) ByID(id int32) (Symbol, bool) {
	for _, sym := range st.mapping {
		if sym.id == id {
			return sym, true
		}
	}
	return Symbol{}, false
}

// ByID returns the Symbol in the table that matches the given text.
func (st SymbolTable) BySymbol(text string) (Symbol, bool) {
	sym, ok := st.mapping[text]
	return sym, ok
}
