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
	"github.com/google/go-cmp/cmp"
	"math/rand"
	"reflect"
	"testing"
)

func TestNewSharedSymbolTableBadVersion(t *testing.T) {
	if newSharedSymbolTable("foo", 0) != nil {
		t.Error("Expected version zero to return nil")
	}
	if newSharedSymbolTable("bar", -1) != nil {
		t.Error("Expected negative version to return nil")
	}
}

var exportAll = cmp.Exporter(func(reflect.Type) bool {
	return true
})

func assertSymbolTokenEquals(x, y SymbolToken, t *testing.T) {
	t.Helper()

	diff := cmp.Diff(x, y, exportAll)
	if diff != "" {
		t.Errorf("Tokens are structurally different: %s", diff)
	}
}

func newString(value string) *string {
	return &value
}

type symbolTableCase struct {
	desc    string
	kind    symbolTableKind
	name    string
	version int32
	imports []SymbolTable
	maxSID  int64
	tokens  []SymbolToken
	table   SymbolTable
}

func makeSystemSymbolTableCase() symbolTableCase {
	// construct up all the expected tokens to test against
	tokens := []SymbolToken{
		{
			Text:     newString("$ion"),
			localSID: symbolIDIon,
			Source:   newSource(symbolTextIon, symbolIDIon),
		},
		{
			Text:     newString("$ion_1_0"),
			localSID: symbolIDIon10,
			Source:   newSource(symbolTextIon, symbolIDIon10),
		},
		{
			Text:     newString("$ion_symbol_table"),
			localSID: symbolIDTable,
			Source:   newSource(symbolTextIon, symbolIDTable),
		},
		{
			Text:     newString("desc"),
			localSID: symbolIDName,
			Source:   newSource(symbolTextIon, symbolIDName),
		},
		{
			Text:     newString("version"),
			localSID: symbolIDVersion,
			Source:   newSource(symbolTextIon, symbolIDVersion),
		},
		{
			Text:     newString("imports"),
			localSID: symbolIDImports,
			Source:   newSource(symbolTextIon, symbolIDImports),
		},
		{
			Text:     newString("symbols"),
			localSID: symbolIDSymbols,
			Source:   newSource(symbolTextIon, symbolIDSymbols),
		},
		{
			Text:     newString("max_id"),
			localSID: symbolIDMaxID,
			Source:   newSource(symbolTextIon, symbolIDMaxID),
		},
		{
			Text:     newString("$ion_shared_symbol_table"),
			localSID: symbolIDSharedTable,
			Source:   newSource(symbolTextIon, symbolIDSharedTable),
		},
	}

	// explicitly make a new system table to test against so we don't manipulate the global one
	table := newSystemSymbolTable()
	return symbolTableCase{
		desc:    "systemSymbolTable",
		kind:    symbolTableKindSystem,
		name:    symbolTextIon,
		version: 1,
		maxSID:  9,
		tokens:  tokens,
		table:   *table,
	}
}

func makeSymbolTableCase(
	desc string,
	start int64,
	kind symbolTableKind,
	name string,
	version int32,
	makeToken func(text string, sid int64) SymbolToken,
	newTable func(imports ...SymbolTable) *SymbolTable,
	texts []string,
	imports ...SymbolTable) symbolTableCase {

	// expect that we start at end of system table
	var sid int64 = start
	for _, imp := range imports {
		sid += imp.maxSID
	}

	// manually calculate what the implementation should do
	var tokens []SymbolToken
	for _, text := range texts {
		sid += 1
		tokens = append(tokens, makeToken(text, sid))
	}

	// construct the table to test
	table := newTable(imports...)
	for _, text := range texts {
		table.InternToken(text)
	}

	return symbolTableCase{
		desc:    desc,
		kind:    kind,
		name:    name,
		version: version,
		imports: imports,
		maxSID:  sid,
		tokens:  tokens,
		table:   *table,
	}
}

func makeSharedSymbolTableCase(
	desc string, name string, version int32, texts []string, imports ...SymbolTable) symbolTableCase {
	makeToken := func(text string, sid int64) SymbolToken {
		source := ImportSource{
			Table: name,
			SID:   sid,
		}
		return SymbolToken{
			Text:     newString(text),
			localSID: sid,
			Source:   &source,
		}
	}
	newTable := func(imports ...SymbolTable) *SymbolTable {
		return newSharedSymbolTable(name, version, imports...)
	}
	return makeSymbolTableCase(
		desc, 0, symbolTableKindShared, name, version, makeToken, newTable, texts, imports...)
}

func makeLocalSymbolTableCase(desc string, texts []string, imports ...SymbolTable) symbolTableCase {
	makeToken := func(text string, sid int64) SymbolToken {
		return SymbolToken{
			Text:     newString(text),
			localSID: sid,
			Source:   nil,
		}
	}
	return makeSymbolTableCase(
		desc, 9, symbolTableKindLocal, "", 0, makeToken, newLocalSymbolTable, texts, imports...)
}

// Arbitrary, deterministic seed for our symbol table testing
const testSystemSymbolTableSeed = 0x03CB815D3119751B

func TestSymbolTable(t *testing.T) {
	// TODO test tables with imports
	cases := []symbolTableCase{
		makeSystemSymbolTableCase(),
		makeLocalSymbolTableCase("Empty LST", nil),
		makeLocalSymbolTableCase("Simple LST", []string{"a", "b", "c"}),
		makeSharedSymbolTableCase("Empty SST", "foo", 1, nil),
		makeSharedSymbolTableCase("Simple SST", "bar", 3, []string{"cat", "dog", "moose"}),
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			// TODO Consider using cmp.Transform to make these tests less verbose...
			if c.kind != c.table.kind {
				t.Errorf("Table kind mismatch: %d != %d", c.kind, c.table.kind)
			}
			if c.name != c.table.name {
				t.Errorf("Table desc mismatch: %s != %s", c.name, c.table.name)
			}
			if c.version != c.table.version {
				t.Errorf("Table version mismatch: %d != %d", c.version, c.table.version)
			}
			importDiff := cmp.Diff(c.imports, c.table.imports)
			if importDiff != "" {
				t.Errorf("Table imports mismatch:\n%s", importDiff)
			}
			if c.maxSID != c.table.maxSID {
				t.Errorf("Table maxSID mismatch: %d != %d", c.maxSID, c.table.maxSID)
			}
			for _, badSID := range []int64{0, -10, c.maxSID + 1, c.maxSID + 128} {
				tok, ok := c.table.BySID(badSID)
				if ok {
					t.Errorf("Found a token %v for a non-existent SID %d", tok, badSID)
				}
			}

			// shuffle the tokens to randomize search a bit
			rnd := rand.New(rand.NewSource(testSystemSymbolTableSeed))
			rnd.Shuffle(len(c.tokens), func(i, j int) {
				c.tokens[i], c.tokens[j] = c.tokens[j], c.tokens[i]
			})
			for _, expected := range c.tokens {
				for i := 0; i < 2; i++ {
					actualByID, ok := c.table.BySID(expected.localSID)
					if !ok {
						t.Error("Could not find ", expected)
					} else {
						assertSymbolTokenEquals(expected, actualByID, t)
					}

					actualByText, ok := c.table.ByText(*expected.Text)
					if !ok {
						t.Error("Could not find ", expected)
					} else {
						assertSymbolTokenEquals(expected, actualByText, t)
					}

					// interning the same text should not affect the above
					newTok := c.table.InternToken(*expected.Text)
					if newTok != actualByText {
						t.Errorf("Interned Token should be the same: %v != %v", newTok, actualByText)
					}
				}
			}
		})
	}

}
