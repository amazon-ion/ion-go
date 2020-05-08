package ion

import (
	"bytes"
	"fmt"
	"testing"
)

type Item struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
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
