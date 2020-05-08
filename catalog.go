package ion

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// A Catalog provides access to shared symbol tables.
type Catalog interface {
	FindExact(name string, version int) SharedSymbolTable
	FindLatest(name string) SharedSymbolTable
}

// A basicCatalog wraps an in-memory collection of shared symbol tables.
type basicCatalog struct {
	ssts   map[string]SharedSymbolTable
	latest map[string]SharedSymbolTable
}

// NewCatalog creates a new basic catalog containing the given symbol tables.
func NewCatalog(ssts ...SharedSymbolTable) Catalog {
	cat := &basicCatalog{
		ssts:   make(map[string]SharedSymbolTable),
		latest: make(map[string]SharedSymbolTable),
	}
	for _, sst := range ssts {
		cat.add(sst)
	}
	return cat
}

// Add adds a shared symbol table to the catalog.
func (c *basicCatalog) add(sst SharedSymbolTable) {
	key := fmt.Sprintf("%v/%v", sst.Name(), sst.Version())
	c.ssts[key] = sst

	cur, ok := c.latest[sst.Name()]
	if !ok || sst.Version() > cur.Version() {
		c.latest[sst.Name()] = sst
	}
}

// FindExact attempts to find a shared symbol table with the given name and version.
func (c *basicCatalog) FindExact(name string, version int) SharedSymbolTable {
	key := fmt.Sprintf("%v/%v", name, version)
	return c.ssts[key]
}

// FindLatest finds the shared symbol table with the given name and largest version.
func (c *basicCatalog) FindLatest(name string) SharedSymbolTable {
	return c.latest[name]
}

// A System is a reader factory wrapping a catalog.
type System struct {
	Catalog Catalog
}

// NewReader creates a new reader using this system's catalog.
func (s System) NewReader(in io.Reader) Reader {
	return NewReaderCat(in, s.Catalog)
}

// NewReaderStr creates a new reader using this system's catalog.
func (s System) NewReaderStr(in string) Reader {
	return NewReaderCat(strings.NewReader(in), s.Catalog)
}

// NewReaderBytes creates a new reader using this system's catalog.
func (s System) NewReaderBytes(in []byte) Reader {
	return NewReaderCat(bytes.NewReader(in), s.Catalog)
}

// Unmarshal unmarshals Ion data using this system's catalog.
func (s System) Unmarshal(data []byte, v interface{}) error {
	r := s.NewReaderBytes(data)
	d := NewDecoder(r)
	return d.DecodeTo(v)
}

// UnmarshalStr unmarshals Ion data using this system's catalog.
func (s System) UnmarshalStr(data string, v interface{}) error {
	r := s.NewReaderStr(data)
	d := NewDecoder(r)
	return d.DecodeTo(v)
}
