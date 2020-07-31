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

// NewReaderString creates a new reader using this system's catalog.
func (s System) NewReaderString(in string) Reader {
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

// UnmarshalString unmarshals Ion data using this system's catalog.
func (s System) UnmarshalString(data string, v interface{}) error {
	r := s.NewReaderString(data)
	d := NewDecoder(r)
	return d.DecodeTo(v)
}
