package ion

import (
	"bytes"
	"fmt"
	"io"
)

// A Catalog stores shared symbol tables and serves as a reader factory.
type Catalog struct {
	ssts map[string]SharedSymbolTable
}

// Add adds a shared symbol table to the catalog.
func (c *Catalog) Add(sst SharedSymbolTable) {
	key := fmt.Sprintf("%v/%v", sst.Name(), sst.Version())
	c.ssts[key] = sst
}

// Find attempts to find a shared symbol table with the given name and version.
func (c *Catalog) Find(name string, version int) SharedSymbolTable {
	key := fmt.Sprintf("%v/%v", name, version)
	return c.ssts[key]
}

// NewReader creates a new reader using this catalog.
func (c *Catalog) NewReader(in io.Reader) Reader {
	return newReader(in, c)
}

// NewReaderBytes creates a new reader using this catalog.
func (c *Catalog) NewReaderBytes(in []byte) Reader {
	return newReader(bytes.NewReader(in), c)
}
