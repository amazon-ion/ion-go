package ion

import (
	"strings"
	"testing"
)

func TestSymbols(t *testing.T) {
	r := NewTextReader(strings.NewReader("'null'::foo bar a::b::'baz'"))

	test := func(etas []string, eval string) {
		if !r.Next() {
			t.Fatal("next returned false")
		}

		if r.Type() != SymbolType {
			t.Fatalf("expected type=symbol, got type=%v", r.Type())
		}

		if !strequals(r.TypeAnnotations(), etas) {
			t.Errorf("expected tas=%v, got tas=%v", etas, r.TypeAnnotations())
		}

		val, err := r.StringValue()
		if err != nil {
			t.Fatal(err)
		}

		if val != eval {
			t.Errorf("expected val=%v, got val=%v", eval, val)
		}
	}

	test([]string{"null"}, "foo")
	test([]string{}, "bar")
	test([]string{"a", "b"}, "baz")

	if r.Next() {
		t.Errorf("next unexpectedly returned true")
	}
}

func strequals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
