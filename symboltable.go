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
	return uint64(s.maxID)
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

func (s *sst) FindByName(sym string) (uint64, bool) {
	id, ok := s.index[sym]
	return uint64(id), ok
}

func (s *sst) FindByID(id uint64) (string, bool) {
	if id <= 0 || id > uint64(len(s.symbols)) {
		return "", false
	}
	return s.symbols[id-1], true
}

func (s *sst) WriteTo(w Writer) error {
	w.Annotation("$ion_shared_symbol_table")
	w.BeginStruct()
	{
		w.FieldName("name")
		w.WriteString(s.name)

		w.FieldName("version")
		w.WriteInt(int64(s.version))

		w.FieldName("symbols")
		w.BeginList()
		{
			for _, sym := range s.symbols {
				w.WriteString(sym)
			}
		}
		w.EndList()
	}
	return w.EndStruct()
}

func (s *sst) String() string {
	buf := strings.Builder{}

	w := NewTextWriter(&buf)
	s.WriteTo(w)

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
	buf := strings.Builder{}
	w := NewTextWriter(&buf)
	w.Annotations("$ion_shared_symbol_table", "bogus")
	w.BeginStruct()

	w.FieldName("name")
	w.WriteString(s.name)

	w.FieldName("version")
	w.WriteInt(int64(s.version))

	w.FieldName("max_id")
	w.WriteUint(s.maxID)

	w.EndStruct()
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

	w.Annotation("$ion_symbol_table")
	w.BeginStruct()

	if len(t.imports) > 1 {
		w.FieldName("imports")
		w.BeginList()
		for i := 1; i < len(t.imports); i++ {
			imp := t.imports[i]
			w.BeginStruct()

			w.FieldName("name")
			w.WriteString(imp.Name())

			w.FieldName("version")
			w.WriteInt(int64(imp.Version()))

			w.FieldName("max_id")
			w.WriteUint(imp.MaxID())

			w.EndStruct()
		}
		w.EndList()
	}

	if len(t.symbols) > 0 {
		w.FieldName("symbols")

		w.BeginList()
		for _, sym := range t.symbols {
			w.WriteString(sym)
		}
		w.EndList()
	}

	return w.EndStruct()
}

func (t *lst) String() string {
	buf := strings.Builder{}

	w := NewTextWriter(&buf)
	t.WriteTo(w)

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
		index[s] = uint64(i)
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
