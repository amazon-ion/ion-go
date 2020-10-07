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
	"strings"
)

// A SymbolTable maps binary-representation symbol IDs to
// text-representation strings and vice versa.
type SymbolTable interface {
	// Imports returns the symbol tables this table imports.
	Imports() []SharedSymbolTable
	// Symbols returns the symbols this symbol table defines.
	Symbols() []string
	// MaxID returns the maximum ID this symbol table defines.
	MaxID() uint64
	// Find finds the SymbolToken by its name.
	Find(symbol string) *SymbolToken
	// FindByName finds the ID of a symbol by its name.
	FindByName(symbol string) (uint64, bool)
	// FindByID finds the name of a symbol given its ID.
	FindByID(id uint64) (string, bool)
	// WriteTo serializes the symbol table to an ion.Writer.
	WriteTo(w Writer) error
	// String returns an ion text representation of the symbol table.
	String() string
}

// A SharedSymbolTable is distributed out-of-band and referenced from
// a local SymbolTable to save space.
type SharedSymbolTable interface {
	SymbolTable

	// Name returns the name of this shared symbol table.
	Name() string
	// Version returns the version of this shared symbol table.
	Version() int
	// Adjust returns a new shared symbol table limited or extended to the given max ID.
	Adjust(maxID uint64) SharedSymbolTable
}

type sst struct {
	name    string
	version int
	symbols []string
	index   map[string]uint64
	maxID   uint64
}

// NewSharedSymbolTable creates a new shared symbol table.
func NewSharedSymbolTable(name string, version int, symbols []string) SharedSymbolTable {
	syms := make([]string, len(symbols))
	copy(syms, symbols)

	index := buildIndex(syms, 1)

	return &sst{
		name:    name,
		version: version,
		symbols: syms,
		index:   index,
		maxID:   uint64(len(syms)),
	}
}

func (s *sst) Name() string {
	return s.name
}

func (s *sst) Version() int {
	return s.version
}

func (s *sst) Imports() []SharedSymbolTable {
	return nil
}

func (s *sst) Symbols() []string {
	syms := make([]string, s.maxID)
	copy(syms, s.symbols)
	return syms
}

func (s *sst) MaxID() uint64 {
	return s.maxID
}

func (s *sst) Adjust(maxID uint64) SharedSymbolTable {
	if maxID == s.maxID {
		// Nothing needs to change.
		return s
	}

	if maxID > uint64(len(s.symbols)) {
		// Old index will work fine, just adjust the maxID.
		return &sst{
			name:    s.name,
			version: s.version,
			symbols: s.symbols,
			index:   s.index,
			maxID:   maxID,
		}
	}

	// Slice the symbols down to size and reindex.
	symbols := s.symbols[:maxID]
	index := buildIndex(symbols, 1)

	return &sst{
		name:    s.name,
		version: s.version,
		symbols: symbols,
		index:   index,
		maxID:   maxID,
	}
}

func (s *sst) Find(sym string) *SymbolToken {
	id, ok := s.FindByName(sym)
	if !ok {
		return nil
	}

	text, ok := s.FindByID(id)
	if !ok {
		return nil
	}

	return &SymbolToken{Text: &text, LocalSID: SymbolIDUnknown}
}

func (s *sst) FindByName(sym string) (uint64, bool) {
	id, ok := s.index[sym]
	return id, ok
}

func (s *sst) FindByID(id uint64) (string, bool) {
	if id <= 0 || id > uint64(len(s.symbols)) {
		return "", false
	}
	return s.symbols[id-1], true
}

func (s *sst) WriteTo(w Writer) error {
	ionSharedSymbolTableText := "$ion_shared_symbol_table"
	if err := w.Annotation(SymbolToken{Text: &ionSharedSymbolTableText, LocalSID: 9}); err != nil {
		return err
	}
	if err := w.BeginStruct(); err != nil {
		return err
	}
	{
		st, err := NewSymbolToken(s, "name")
		if err != nil {
			return err
		}
		if err := w.FieldName(st); err != nil {
			return err
		}
		if err := w.WriteString(s.name); err != nil {
			return err
		}

		st, err = NewSymbolToken(s, "version")
		if err != nil {
			return err
		}
		if err := w.FieldName(st); err != nil {
			return err
		}
		if err := w.WriteInt(int64(s.version)); err != nil {
			return err
		}

		st, err = NewSymbolToken(s, "symbols")
		if err != nil {
			return err
		}
		if err := w.FieldName(st); err != nil {
			return err
		}
		if err := w.BeginList(); err != nil {
			return err
		}
		{
			for _, sym := range s.symbols {
				if err := w.WriteString(sym); err != nil {
					return err
				}
			}
		}
		if err := w.EndList(); err != nil {
			return err
		}
	}
	return w.EndStruct()
}

func (s *sst) String() string {
	buf := strings.Builder{}

	w := NewTextWriter(&buf)
	_ = s.WriteTo(w)

	return buf.String()
}

// V1SystemSymbolTable is the (implied) system symbol table for Ion v1.0.
var V1SystemSymbolTable = NewSharedSymbolTable("$ion", 1, []string{
	"$ion",
	"$ion_1_0",
	"$ion_symbol_table",
	"name",
	"version",
	"imports",
	"symbols",
	"max_id",
	"$ion_shared_symbol_table",
})

// A BogusSST represents an SST imported by an LST that cannot be found in the
// local catalog. It exists to reserve some part of the symbol ID space so other
// symbol tables get mapped to the right IDs.
type bogusSST struct {
	name    string
	version int
	maxID   uint64
}

var _ SharedSymbolTable = &bogusSST{}

func (s *bogusSST) Name() string {
	return s.name
}

func (s *bogusSST) Version() int {
	return s.version
}

func (s *bogusSST) Imports() []SharedSymbolTable {
	return nil
}

func (s *bogusSST) Symbols() []string {
	return nil
}

func (s *bogusSST) MaxID() uint64 {
	return s.maxID
}

func (s *bogusSST) Adjust(maxID uint64) SharedSymbolTable {
	return &bogusSST{
		name:    s.name,
		version: s.version,
		maxID:   maxID,
	}
}

func (s *bogusSST) Find(sym string) *SymbolToken {
	return nil
}

func (s *bogusSST) FindByName(sym string) (uint64, bool) {
	return 0, false
}

func (s *bogusSST) FindByID(id uint64) (string, bool) {
	return "", false
}

func (s *bogusSST) WriteTo(w Writer) error {
	return &UsageError{"SharedSymbolTable.WriteTo", "bogus symbol table should never be written"}
}

func (s *bogusSST) String() string {
	ionSharedSymbolTableText := "$ion_shared_symbol_table"
	bogusText := "bogus"

	buf := strings.Builder{}
	w := NewTextWriter(&buf)
	_ = w.Annotations(SymbolToken{Text: &ionSharedSymbolTableText, LocalSID: 9}, SymbolToken{Text: &bogusText, LocalSID: SymbolIDUnknown})
	_ = w.BeginStruct()

	st, _ := NewSymbolToken(s, "name")
	_ = w.FieldName(st)
	_ = w.WriteString(s.name)

	st, _ = NewSymbolToken(s, "version")
	_ = w.FieldName(st)
	_ = w.WriteInt(int64(s.version))

	st, _ = NewSymbolToken(s, "max_id")
	_ = w.FieldName(st)
	_ = w.WriteUint(s.maxID)

	_ = w.EndStruct()
	return buf.String()
}

// A LocalSymbolTable is transmitted in-band along with the binary data
// it describes. It may include SharedSymbolTables by reference.
type lst struct {
	imports     []SharedSymbolTable
	offsets     []uint64
	maxImportID uint64

	symbols []string
	index   map[string]uint64
}

// NewLocalSymbolTable creates a new local symbol table.
func NewLocalSymbolTable(imports []SharedSymbolTable, symbols []string) SymbolTable {
	imps, offsets, maxID := processImports(imports)
	syms := make([]string, len(symbols))
	copy(syms, symbols)

	index := buildIndex(syms, maxID+1)

	return &lst{
		imports:     imps,
		offsets:     offsets,
		maxImportID: maxID,
		symbols:     syms,
		index:       index,
	}
}

func (t *lst) Imports() []SharedSymbolTable {
	imps := make([]SharedSymbolTable, len(t.imports))
	copy(imps, t.imports)
	return imps
}

func (t *lst) Symbols() []string {
	syms := make([]string, len(t.symbols))
	copy(syms, t.symbols)
	return syms
}

func (t *lst) MaxID() uint64 {
	return t.maxImportID + uint64(len(t.symbols))
}

func (t *lst) Find(s string) *SymbolToken {
	// Check import
	for _, imp := range t.imports {
		symbolToken := imp.Find(s)
		if symbolToken != nil {
			return symbolToken
		}
	}

	// Check local
	if _, ok := t.index[s]; ok {
		return &SymbolToken{Text: &s, LocalSID: SymbolIDUnknown}
	}

	return nil
}

func (t *lst) FindByName(s string) (uint64, bool) {
	for i, imp := range t.imports {
		if id, ok := imp.FindByName(s); ok {
			return t.offsets[i] + id, true
		}
	}

	if id, ok := t.index[s]; ok {
		return id, true
	}

	return 0, false
}

func (t *lst) FindByID(id uint64) (string, bool) {
	if id <= 0 {
		return "", false
	}
	if id <= t.maxImportID {
		return t.findByIDInImports(id)
	}

	// Local to this symbol table.
	idx := id - t.maxImportID - 1
	if idx < uint64(len(t.symbols)) {
		return t.symbols[idx], true
	}

	return "", false
}

func (t *lst) findByIDInImports(id uint64) (string, bool) {
	i := 1
	off := uint64(0)

	for ; i < len(t.imports); i++ {
		if id <= t.offsets[i] {
			break
		}
		off = t.offsets[i]
	}

	return t.imports[i-1].FindByID(id - off)
}

func (t *lst) WriteTo(w Writer) error {
	if len(t.imports) == 1 && len(t.symbols) == 0 {
		return nil
	}

	ionSymbolTableText := "$ion_symbol_table"
	if err := w.Annotation(SymbolToken{Text: &ionSymbolTableText, LocalSID: 3}); err != nil {
		return err
	}
	if err := w.BeginStruct(); err != nil {
		return err
	}

	if len(t.imports) > 1 {
		st, err := NewSymbolToken(t, "imports")
		if err != nil {
			return err
		}
		if err := w.FieldName(st); err != nil {
			return err
		}
		if err := w.BeginList(); err != nil {
			return err
		}
		for i := 1; i < len(t.imports); i++ {
			imp := t.imports[i]
			if err := w.BeginStruct(); err != nil {
				return err
			}

			st, err := NewSymbolToken(t, "name")
			if err != nil {
				return err
			}
			if err := w.FieldName(st); err != nil {
				return err
			}
			if err := w.WriteString(imp.Name()); err != nil {
				return err
			}

			st, err = NewSymbolToken(t, "version")
			if err != nil {
				return err
			}
			if err := w.FieldName(st); err != nil {
				return err
			}
			if err := w.WriteInt(int64(imp.Version())); err != nil {
				return err
			}

			st, err = NewSymbolToken(t, "max_id")
			if err != nil {
				return err
			}
			if err := w.FieldName(st); err != nil {
				return err
			}
			if err := w.WriteUint(imp.MaxID()); err != nil {
				return err
			}

			if err := w.EndStruct(); err != nil {
				return err
			}
		}
		if err := w.EndList(); err != nil {
			return err
		}
	}

	if len(t.symbols) > 0 {
		st, err := NewSymbolToken(t, "symbols")
		if err != nil {
			return err
		}
		if err := w.FieldName(st); err != nil {
			return err
		}

		if err := w.BeginList(); err != nil {
			return err
		}
		for _, sym := range t.symbols {
			if err := w.WriteString(sym); err != nil {
				return err
			}
		}
		if err := w.EndList(); err != nil {
			return err
		}
	}

	return w.EndStruct()
}

func (t *lst) String() string {
	buf := strings.Builder{}

	w := NewTextWriter(&buf)
	_ = t.WriteTo(w)

	return buf.String()
}

// A SymbolTableBuilder helps you iteratively build a local symbol table.
type SymbolTableBuilder interface {
	SymbolTable

	// Add adds a symbol to this symbol table.
	Add(symbol string) (uint64, bool)
	// Build creates an immutable local symbol table.
	Build() SymbolTable
}

type symbolTableBuilder struct {
	lst
}

// NewSymbolTableBuilder creates a new symbol table builder with the given imports.
func NewSymbolTableBuilder(imports ...SharedSymbolTable) SymbolTableBuilder {
	imps, offsets, maxID := processImports(imports)
	return &symbolTableBuilder{
		lst{
			imports:     imps,
			offsets:     offsets,
			maxImportID: maxID,
			index:       make(map[string]uint64),
		},
	}
}

func (b *symbolTableBuilder) Add(symbol string) (uint64, bool) {
	if id, ok := b.FindByName(symbol); ok {
		return id, false
	}

	b.symbols = append(b.symbols, symbol)
	id := b.maxImportID + uint64(len(b.symbols))
	b.index[symbol] = id

	return id, true
}

func (b *symbolTableBuilder) Build() SymbolTable {
	symbols := append([]string{}, b.symbols...)
	index := make(map[string]uint64)
	for s, i := range b.index {
		index[s] = i
	}

	return &lst{
		imports:     b.imports,
		offsets:     b.offsets,
		maxImportID: b.maxImportID,
		symbols:     symbols,
		index:       index,
	}
}

// ProcessImports processes a slice of imports, returning an (augmented) copy, a set of
// offsets for each import, and the overall max ID.
func processImports(imports []SharedSymbolTable) ([]SharedSymbolTable, []uint64, uint64) {
	// Add in V1SystemSymbolTable at the head of the list if it's not already included.
	var imps []SharedSymbolTable
	if len(imports) > 0 && imports[0].Name() == "$ion" {
		imps = make([]SharedSymbolTable, len(imports))
		copy(imps, imports)
	} else {
		imps = make([]SharedSymbolTable, len(imports)+1)
		imps[0] = V1SystemSymbolTable
		copy(imps[1:], imports)
	}

	// Calculate offsets.
	maxID := uint64(0)
	offsets := make([]uint64, len(imps))
	for i, imp := range imps {
		offsets[i] = maxID
		maxID += imp.MaxID()
	}

	return imps, offsets, maxID
}

// BuildIndex builds an index from symbol name to symbol ID.
func buildIndex(symbols []string, offset uint64) map[string]uint64 {
	index := make(map[string]uint64)

	for i, sym := range symbols {
		if sym != "" {
			if _, ok := index[sym]; !ok {
				index[sym] = offset + uint64(i)
			}
		}
	}

	return index
}
