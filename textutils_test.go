package ion

import (
	"strings"
	"testing"
)

func TestWriteSymbol(t *testing.T) {
	test := func(sym, expected string) {
		t.Run(expected, func(t *testing.T) {
			buf := strings.Builder{}
			if err := writeSymbol(sym, &buf); err != nil {
				t.Fatal(err)
			}
			actual := buf.String()
			if actual != expected {
				t.Errorf("expected \"%v\", got \"%v\"", expected, actual)
			}
		})
	}

	test("", "''")
	test("null", "'null'")
	test("null.null", "'null.null'")

	test("basic", "basic")
	test("_basic_", "_basic_")
	test("$basic$", "$basic$")

	test("123", "'123'")
	test("$123", "'$123'")
	test("abc'def", "'abc\\'def'")
	test("abc\"def", "'abc\"def'")
}

func TestSymbolNeedsQuoting(t *testing.T) {
	test := func(sym string, expected bool) {
		t.Run(sym, func(t *testing.T) {
			actual := symbolNeedsQuoting(sym)
			if actual != expected {
				t.Errorf("expected %v, got %v", expected, actual)
			}
		})
	}

	test("", true)
	test("null", true)
	test("true", true)
	test("false", true)
	test("nan", true)

	test("basic", false)
	test("_basic_", false)
	test("basic$123", false)
	test("$", false)
	test("$basic", false)

	test("123", true)
	test("$123", true)
	test("abc.def", true)
	test("abc,def", true)
	test("abc:def", true)
	test("abc{def", true)
	test("abc}def", true)
	test("abc[def", true)
	test("abc]def", true)
	test("abc'def", true)
	test("abc\"def", true)
}

func TestIsSymbolRef(t *testing.T) {
	test := func(sym string, expected bool) {
		t.Run(sym, func(t *testing.T) {
			actual := isSymbolRef(sym)
			if actual != expected {
				t.Errorf("expected %v, got %v", expected, actual)
			}
		})
	}

	test("", false)
	test("1", false)
	test("a", false)
	test("$", false)
	test("$1", true)
	test("$1234567890", true)
	test("$a", false)
	test("$1234a567890", false)
}

func TestWriteEscapedSymbol(t *testing.T) {
	test := func(sym, expected string) {
		t.Run(expected, func(t *testing.T) {
			buf := strings.Builder{}
			if err := writeEscapedSymbol(sym, &buf); err != nil {
				t.Fatal(err)
			}
			actual := buf.String()
			if actual != expected {
				t.Errorf("bad encoding of \"%v\": \"%v\"",
					expected, actual)
			}
		})
	}

	test("basic", "basic")
	test("\"basic\"", "\"basic\"")
	test("o'clock", "o\\'clock")
	test("c:\\", "c:\\\\")
}

func TestWriteEscapedChar(t *testing.T) {
	test := func(c byte, expected string) {
		t.Run(expected, func(t *testing.T) {
			buf := strings.Builder{}
			if err := writeEscapedChar(c, &buf); err != nil {
				t.Fatal(err)
			}
			actual := buf.String()
			if actual != expected {
				t.Errorf("bad encoding of '%v': \"%v\"",
					expected, actual)
			}
		})
	}

	test(0, "\\0")
	test('\n', "\\n")
	test(1, "\\x01")
	test('\xFF', "\\xFF")
}
