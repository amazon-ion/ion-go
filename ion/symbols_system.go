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

const (
	symbolTextIon = "$ion"
	symbolIDIon   = 1

	// Version Identifier for Ion 1.0.
	symbolTextIon10 = "$ion_1_0"
	symbolIDIon10   = 2

	symbolTextTable = "$ion_symbol_table"
	symbolIDTable   = 3

	symbolTextName = "name"
	symbolIDName   = 4

	symbolTextVersion = "version"
	symbolIDVersion   = 5

	symbolTextImports = "imports"
	symbolIDImports   = 6

	symbolTextSymbols = "symbols"
	symbolIDSymbols   = 7

	symbolTextMaxID = "max_id"
	symbolIDMaxID   = 8

	symbolTextSharedTable = "$ion_shared_symbol_table"
	symbolIDSharedTable   = 9
)

func newSystemSymbolTable() *SymbolTable {
	table := newSymbolTableRaw(symbolTableKindSystem, symbolTextIon, 1)
	table.InternToken(symbolTextIon)
	table.InternToken(symbolTextIon10)
	table.InternToken(symbolTextTable)
	table.InternToken(symbolTextName)
	table.InternToken(symbolTextVersion)
	table.InternToken(symbolTextImports)
	table.InternToken(symbolTextSymbols)
	table.InternToken(symbolTextMaxID)
	table.InternToken(symbolTextSharedTable)

	return table
}

var (
	// systemSymbolTable is the implicitly defined Ion 1.0 symbol table that all local symbol tables inherit.
	systemSymbolTable = *newSystemSymbolTable()
)
