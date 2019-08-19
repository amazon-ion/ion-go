package ion

import "fmt"

// A Catalog stores shared symbol tables.
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
