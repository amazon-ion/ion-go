package ion

import (
	"strings"
)

// A SymbolTable maps binary-representation symbol IDs to
// text-representation strings and vice versa.
type SymbolTable interface {
	// MaxID returns the maximum ID this symbol table defines.
	MaxID() int
	// FindByName finds the ID of a symbol by its name.
	FindByName(symbol string) (int, bool)
	// FindByID finds the name of a symbol given its ID.
	FindByID(id int) (string, bool)
	// WriteTo serializes the symbol table to an ion.Writer.
	WriteTo(w Writer) error
	// String returns an ion text representation of the symbol table.
	String() string
}

// A SharedSymbolTable is distributed out-of-band and referenced from
// a LocalSymbolTable to save space.
type SharedSymbolTable interface {
	SymbolTable

	Name() string
	Version() int
}

type sharedSymbolTable struct {
	name    string
	version int
	symbols []string
	index   map[string]int
}

// NewSharedSymbolTable creates a new shared symbol table.
func NewSharedSymbolTable(name string, version int, symbols []string) SharedSymbolTable {
	if name == "" {
		panic("name must be non-empty")
	}
	if version < 1 {
		panic("version must be at least one")
	}

	index, copy := buildIndex(symbols, 0)

	return &sharedSymbolTable{
		name:    name,
		version: version,
		symbols: copy,
		index:   index,
	}
}

func buildIndex(symbols []string, offset int) (map[string]int, []string) {
	index := map[string]int{}
	copy := []string{}

	for _, sym := range symbols {
		if _, ok := index[sym]; !ok {
			copy = append(copy, sym)
			index[sym] = offset + len(copy)
		}
	}

	return index, copy
}

func (s *sharedSymbolTable) Name() string {
	return s.name
}

func (s *sharedSymbolTable) Version() int {
	return s.version
}

func (s *sharedSymbolTable) MaxID() int {
	return len(s.symbols)
}

func (s *sharedSymbolTable) FindByName(sym string) (int, bool) {
	id, ok := s.index[sym]
	return id, ok
}

func (s *sharedSymbolTable) FindByID(id int) (string, bool) {
	if id <= 0 || id > len(s.symbols) {
		return "", false
	}
	return s.symbols[id-1], true
}

func (s *sharedSymbolTable) WriteTo(w Writer) error {
	w.TypeAnnotation("$ion_shared_symbol_table")
	w.BeginStruct()

	w.FieldName("name")
	w.WriteString(s.name)

	w.FieldName("version")
	w.WriteInt(int64(s.version))

	w.FieldName("symbols")
	w.BeginList()

	for _, sym := range s.symbols {
		w.WriteString(sym)
	}

	w.EndList() // symbols

	w.EndStruct()
	return w.Err()
}

func (s *sharedSymbolTable) String() string {
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

// A LocalSymbolTable is transmitted in-band along with the binary data
// it describes. It may include SharedSymbolTables by reference.
type localSymbolTable struct {
	imports     []SharedSymbolTable
	offsets     []int
	maxImportID int

	symbols []string
	index   map[string]int
}

// NewLocalSymbolTable creates a new local symbol table.
func NewLocalSymbolTable(imports []SharedSymbolTable, symbols []string) SymbolTable {
	imps, offsets, maxID := processImports(imports)
	index, copy := buildIndex(symbols, maxID)

	return &localSymbolTable{
		imports:     imps,
		offsets:     offsets,
		maxImportID: maxID,
		symbols:     copy,
		index:       index,
	}
}

func processImports(imports []SharedSymbolTable) ([]SharedSymbolTable, []int, int) {
	var imps []SharedSymbolTable
	if len(imports) > 0 && imports[0].Name() == "$ion" {
		imps = make([]SharedSymbolTable, len(imports))
		copy(imps, imports)
	} else {
		imps = make([]SharedSymbolTable, len(imports)+1)
		imps[0] = V1SystemSymbolTable
		copy(imps[1:], imports)
	}

	maxID := 0
	offsets := make([]int, len(imps))
	for i, imp := range imps {
		offsets[i] = maxID
		maxID += imp.MaxID()
	}

	return imps, offsets, maxID
}

func (t *localSymbolTable) MaxID() int {
	return t.maxImportID + len(t.symbols)
}

func (t *localSymbolTable) FindByName(s string) (int, bool) {
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

func (t *localSymbolTable) FindByID(id int) (string, bool) {
	if id <= 0 {
		return "", false
	}
	if id <= t.maxImportID {
		return t.findByIDInImports(id)
	}

	// Local to this symbol table.
	idx := id - t.maxImportID - 1
	if idx < len(t.symbols) {
		return t.symbols[idx], true
	}

	return "", false
}

func (t *localSymbolTable) findByIDInImports(id int) (string, bool) {
	i := 1
	off := 0

	for ; i < len(t.imports); i++ {
		if id <= t.offsets[i] {
			break
		}
		off = t.offsets[i]
	}

	return t.imports[i-1].FindByID(id - off)
}

func (t *localSymbolTable) WriteTo(w Writer) error {
	w.TypeAnnotation("$ion_symbol_table")
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
			w.WriteInt(int64(imp.MaxID()))

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

	w.EndStruct()
	return w.Err()
}

func (t *localSymbolTable) String() string {
	buf := strings.Builder{}

	w := NewTextWriter(&buf)
	t.WriteTo(w)

	return buf.String()
}

// A SymbolTableBuilder helps you iteratively build a local symbol table.
type SymbolTableBuilder interface {
	SymbolTable

	// Add adds a symbol to this symbol table.
	Add(symbol string) (int, bool)
	// Build creates an immutable local symbol table.
	Build() SymbolTable
}

type symbolTableBuilder struct {
	localSymbolTable
}

// NewSymbolTableBuilder creates a new symbol table builder with the given imports.
func NewSymbolTableBuilder(imports ...SharedSymbolTable) SymbolTableBuilder {
	imps, offsets, maxID := processImports(imports)
	return &symbolTableBuilder{
		localSymbolTable{
			imports:     imps,
			offsets:     offsets,
			maxImportID: maxID,
			index:       make(map[string]int),
		},
	}
}

func (b *symbolTableBuilder) Add(symbol string) (int, bool) {
	if id, ok := b.FindByName(symbol); ok {
		return id, false
	}

	b.symbols = append(b.symbols, symbol)
	id := b.maxImportID + len(b.symbols)
	b.index[symbol] = id

	return id, true
}

func (b *symbolTableBuilder) Build() SymbolTable {
	symbols := append([]string{}, b.symbols...)
	index := make(map[string]int)
	for s, i := range b.index {
		index[s] = i
	}

	return &localSymbolTable{
		imports:     b.imports,
		offsets:     b.offsets,
		maxImportID: b.maxImportID,
		symbols:     symbols,
		index:       index,
	}
}
