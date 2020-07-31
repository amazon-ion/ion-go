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
	"testing"
)

type Item struct {
	ID          int    `ion:"id"`
	Name        string `ion:"name"`
	Description string `ion:"description"`
}

func TestCatalog(t *testing.T) {
	sst := NewSharedSymbolTable("item", 1, []string{
		"item",
		"id",
		"name",
		"description",
	})

	buf := bytes.Buffer{}
	out := NewBinaryWriter(&buf, sst)

	for i := 0; i < 10; i++ {
		out.Annotation("item")
		MarshalTo(out, &Item{
			ID:          i,
			Name:        fmt.Sprintf("Item %v", i),
			Description: fmt.Sprintf("The %vth test item", i),
		})
	}
	if err := out.Finish(); err != nil {
		t.Fatal(err)
	}

	bs := buf.Bytes()

	sys := System{Catalog: NewCatalog(sst)}
	in := sys.NewReaderBytes(bs)

	i := 0
	for ; ; i++ {
		item := Item{}
		err := UnmarshalFrom(in, &item)
		if err == ErrNoInput {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if item.ID != i {
			t.Errorf("expected id=%v, got %v", i, item.ID)
		}
	}

	if i != 10 {
		t.Errorf("expected i=10, got %v", i)
	}
}
