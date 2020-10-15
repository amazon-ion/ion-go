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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedSymbolTable(t *testing.T) {
	st := NewSharedSymbolTable("test", 2, []string{
		"abc",
		"def",
		"foo'bar",
		"null",
		"def",
		"ghi",
	})

	assert.Equal(t, "test", st.Name())
	assert.Equal(t, 2, st.Version())
	assert.Equal(t, 6, int(st.MaxID()))

	testFindByName(t, st, "def", 2)
	testFindByName(t, st, "null", 4)
	testFindByName(t, st, "bogus", 0)

	testFindByID(t, st, 0, "")
	testFindByID(t, st, 2, "def")
	testFindByID(t, st, 4, "null")
	testFindByID(t, st, 7, "")

	testFindSymbolToken(t, st, "def", NewSymbolTokenFromString("def"))
	testFindSymbolToken(t, st, "foo'bar", NewSymbolTokenFromString("foo'bar"))

	testString(t, st, `$ion_shared_symbol_table::{name:"test",version:2,symbols:["abc","def","foo'bar","null","def","ghi"]}`)
}

func TestLocalSymbolTable(t *testing.T) {
	st := NewLocalSymbolTable(nil, []string{"foo", "bar"})

	assert.Equal(t, 11, int(st.MaxID()))

	testFindByName(t, st, "$ion", 1)
	testFindByName(t, st, "foo", 10)
	testFindByName(t, st, "bar", 11)
	testFindByName(t, st, "bogus", 0)

	testFindByID(t, st, 0, "")
	testFindByID(t, st, 1, "$ion")
	testFindByID(t, st, 10, "foo")
	testFindByID(t, st, 11, "bar")
	testFindByID(t, st, 12, "")

	testFindSymbolToken(t, st, "foo", NewSymbolTokenFromString("foo"))
	testFindSymbolToken(t, st, "bar", NewSymbolTokenFromString("bar"))
	testFindSymbolToken(t, st, "$ion", NewSymbolTokenFromString("$ion"))

	testString(t, st, `$ion_symbol_table::{symbols:["foo","bar"]}`)
}

func TestLocalSymbolTableWithImports(t *testing.T) {
	shared := NewSharedSymbolTable("shared", 1, []string{
		"foo",
		"bar",
	})
	imports := []SharedSymbolTable{shared}

	st := NewLocalSymbolTable(imports, []string{
		"foo2",
		"bar2",
	})

	assert.Equal(t, 13, int(st.MaxID()))

	testFindByName(t, st, "$ion", 1)
	testFindByName(t, st, "$ion_shared_symbol_table", 9)
	testFindByName(t, st, "foo", 10)
	testFindByName(t, st, "bar", 11)
	testFindByName(t, st, "foo2", 12)
	testFindByName(t, st, "bar2", 13)
	testFindByName(t, st, "bogus", 0)

	testFindByID(t, st, 0, "")
	testFindByID(t, st, 1, "$ion")
	testFindByID(t, st, 9, "$ion_shared_symbol_table")
	testFindByID(t, st, 10, "foo")
	testFindByID(t, st, 11, "bar")
	testFindByID(t, st, 12, "foo2")
	testFindByID(t, st, 13, "bar2")
	testFindByID(t, st, 14, "")

	testFindSymbolToken(t, st, "foo", NewSymbolTokenFromString("foo"))
	testFindSymbolToken(t, st, "bar", NewSymbolTokenFromString("bar"))
	testFindSymbolToken(t, st, "foo2", NewSymbolTokenFromString("foo2"))
	testFindSymbolToken(t, st, "bar2", NewSymbolTokenFromString("bar2"))

	testString(t, st, `$ion_symbol_table::{imports:[{name:"shared",version:1,max_id:2}],symbols:["foo2","bar2"]}`)
}

func TestSymbolTableBuilder(t *testing.T) {
	b := NewSymbolTableBuilder()

	id, ok := b.Add("name")
	assert.False(t, ok, "Add(name) returned true")
	assert.Equal(t, 4, int(id), "Add(name) returned %v", id)

	id, ok = b.Add("foo")
	assert.True(t, ok, "Add(foo) returned false")
	assert.Equal(t, 10, int(id), "Add(foo) returned %v", id)

	id, ok = b.Add("foo")
	assert.False(t, ok, "Second Add(foo) returned true")
	assert.Equal(t, 10, int(id), "Second Add(foo) returned %v", id)

	st := b.Build()
	assert.Equal(t, 10, int(st.MaxID()), "maxid returned %v", st.MaxID())

	testFindByName(t, st, "$ion", 1)
	testFindByName(t, st, "foo", 10)
	testFindByName(t, st, "bogus", 0)

	testFindByID(t, st, 1, "$ion")
	testFindByID(t, st, 10, "foo")
	testFindByID(t, st, 11, "")
}

func testFindByName(t *testing.T, st SymbolTable, sym string, expected uint64) {
	t.Run("FindByName("+sym+")", func(t *testing.T) {
		actual, ok := st.FindByName(sym)
		if expected == 0 {
			require.False(t, ok)
		} else {
			require.True(t, ok)
			assert.Equal(t, expected, actual)
		}
	})
}

func testFindByID(t *testing.T, st SymbolTable, id uint64, expected string) {
	t.Run(fmt.Sprintf("FindByID(%v)", id), func(t *testing.T) {
		actual, ok := st.FindByID(id)
		if expected == "" {
			require.False(t, ok)
		} else {
			require.True(t, ok)
			assert.Equal(t, expected, actual)
		}
	})
}

func testFindSymbolToken(t *testing.T, st SymbolTable, sym string, expected SymbolToken) {
	t.Run("Find("+sym+")", func(t *testing.T) {
		actual := st.Find(sym)
		require.NotNil(t, actual)

		assert.True(t, actual.Equal(&expected), "expected %v, got %v", expected, actual)
	})
}

func testString(t *testing.T, st SymbolTable, expected string) {
	t.Run("String()", func(t *testing.T) {
		actual := st.String()
		assert.Equal(t, expected, actual)
	})
}

func newString(value string) *string {
	return &value
}
