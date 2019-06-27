package ion

import (
	"math"
	"testing"
)

func TestSymbols(t *testing.T) {
	r := NewTextReaderString("'null'::foo bar a::b::'baz'")

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
	if r.Err() != nil {
		t.Error(r.Err())
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

func TestSpecialSymbols(t *testing.T) {
	r := NewTextReaderString("null\nnull.struct\ntrue\nfalse\nnan")

	// null
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != NullType {
			t.Errorf("expected type=NullType, got %v", r.Type())
		}
		if !r.IsNull() {
			t.Error("expected isNull=true, got false")
		}
	}

	// null.struct
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != StructType {
			t.Errorf("expected type=StructType, got %v", r.Type())
		}
		if !r.IsNull() {
			t.Error("expected isNull=true, got false")
		}
	}

	// true
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != BoolType {
			t.Errorf("expected type=BoolType, got %v", r.Type())
		}
		val, err := r.BoolValue()
		if err != nil {
			t.Fatal(err)
		}
		if !val {
			t.Error("expected value=true, got false")
		}
	}

	// false
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != BoolType {
			t.Errorf("expected type=BoolType, got %v", r.Type())
		}
		val, err := r.BoolValue()
		if err != nil {
			t.Fatal(err)
		}
		if val {
			t.Error("expected value=false, got true")
		}
	}

	// nan
	{
		if !r.Next() {
			t.Fatal("next returned false")
		}
		if r.Type() != FloatType {
			t.Errorf("expected type=FloatType, got %v", r.Type())
		}
		val, err := r.FloatValue()
		if err != nil {
			t.Fatal(err)
		}
		if !math.IsNaN(val) {
			t.Errorf("expected value=NaN, got %v", val)
		}
	}

	if r.Next() {
		t.Error("next returned true")
	}
}
