package text

import (
	"strings"
	"testing"
)

func TestWriteSymbol(t *testing.T) {
	test := func(t *testing.T, sym, expected string) {
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

	test(t, "", "''")
	test(t, "null", "'null'")
	test(t, "null.null", "'null.null'")

	test(t, "basic", "basic")
	test(t, "_basic_", "_basic_")
	test(t, "$basic$", "$basic$")

	test(t, "123", "'123'")
	test(t, "$123", "'$123'")
	test(t, "abc'def", "'abc\\'def'")
	test(t, "abc\"def", "'abc\"def'")
}

func TestNeedsQuoting(t *testing.T) {
	test := func(t *testing.T, sym string, expected bool) {
		t.Run(sym, func(t *testing.T) {
			actual := needsQuoting(sym)
			if actual != expected {
				t.Errorf("expected %v, got %v", expected, actual)
			}
		})
	}

	test(t, "", true)
	test(t, "null", true)
	test(t, "true", true)
	test(t, "false", true)
	test(t, "nan", true)

	test(t, "basic", false)
	test(t, "_basic_", false)
	test(t, "basic$123", false)
	test(t, "$", false)
	test(t, "$basic", false)

	test(t, "123", true)
	test(t, "$123", true)
	test(t, "abc.def", true)
	test(t, "abc,def", true)
	test(t, "abc:def", true)
	test(t, "abc{def", true)
	test(t, "abc}def", true)
	test(t, "abc[def", true)
	test(t, "abc]def", true)
	test(t, "abc'def", true)
	test(t, "abc\"def", true)
}

func TestIsSymbolRef(t *testing.T) {
	testIsSymbolRef(t, "", false)
	testIsSymbolRef(t, "1", false)
	testIsSymbolRef(t, "a", false)
	testIsSymbolRef(t, "$", false)
	testIsSymbolRef(t, "$1", true)
	testIsSymbolRef(t, "$1234567890", true)
	testIsSymbolRef(t, "$a", false)
	testIsSymbolRef(t, "$1234a567890", false)
}

func testIsSymbolRef(t *testing.T, sym string, expected bool) {
	t.Run(sym, func(t *testing.T) {
		actual := isSymbolRef(sym)
		if actual != expected {
			t.Errorf("expected %v, got %v", expected, actual)
		}
	})
}

func TestWriteEscapedSymbol(t *testing.T) {
	testWriteEscapedSymbol(t, "basic", "basic")
	testWriteEscapedSymbol(t, "\"basic\"", "\"basic\"")
	testWriteEscapedSymbol(t, "o'clock", "o\\'clock")
	testWriteEscapedSymbol(t, "c:\\", "c:\\\\")
}

func testWriteEscapedSymbol(t *testing.T, sym, expected string) {
	t.Run(expected, func(t *testing.T) {
		buf := strings.Builder{}
		if err := writeEscapedSymbol(sym, &buf); err != nil {
			t.Fatal(err)
		}
		actual := buf.String()
		if actual != expected {
			t.Errorf("bad encoding of \"%v\": \"%v\"", expected, actual)
		}
	})
}

func TestWriteEscapedChar(t *testing.T) {
	testWriteEscapedChar(t, 0, "\\0")
	testWriteEscapedChar(t, '\n', "\\n")
	testWriteEscapedChar(t, 1, "\\x01")
	testWriteEscapedChar(t, '\xFF', "\\xFF")
}

func testWriteEscapedChar(t *testing.T, c byte, expected string) {
	t.Run(expected, func(t *testing.T) {
		buf := strings.Builder{}
		if err := writeEscapedChar(c, &buf); err != nil {
			t.Fatal(err)
		}
		actual := buf.String()
		if actual != expected {
			t.Errorf("bad encoding of '%v': \"%v\"", expected, actual)
		}
	})
}
