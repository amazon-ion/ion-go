package ion

import (
	"strings"
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	test := func(str string, eval string, expectedPrecision TimestampPrecision) {
		t.Run(str, func(t *testing.T) {
			val, err := parseTimestamp(str)
			if err != nil {
				t.Fatal(err)
			}

			et, err := time.Parse(time.RFC3339Nano, eval)
			if err != nil {
				t.Fatal(err)
			}

			if !val.dateTime.Equal(et) {
				t.Errorf("expected %v, got %v", eval, val)
			}

			if val.precision != expectedPrecision {
				t.Errorf("expected %v, got %v", expectedPrecision.String(), val.precision.String())
			}
		})
	}

	test("1234T", "1234-01-01T00:00:00Z", Year)
	test("1234-05T", "1234-05-01T00:00:00Z", Month)
	test("1234-05-06", "1234-05-06T00:00:00Z", Day)
	test("1234-05-06T", "1234-05-06T00:00:00Z", Day)
	test("1234-05-06T07:08Z", "1234-05-06T07:08:00Z", Minute)
	test("1234-05-06T07:08:09Z", "1234-05-06T07:08:09Z", Second)
	test("1234-05-06T07:08:09.100Z", "1234-05-06T07:08:09.100Z", Second)
	test("1234-05-06T07:08:09.100100Z", "1234-05-06T07:08:09.100100Z", Second)

	// Test rounding of >=9 fractional seconds.
	test("1234-05-06T07:08:09.000100100Z", "1234-05-06T07:08:09.000100100Z", Second)
	test("1234-05-06T07:08:09.100100100Z", "1234-05-06T07:08:09.100100100Z", Second)
	test("1234-05-06T07:08:09.00010010044Z", "1234-05-06T07:08:09.000100100Z", Second)
	test("1234-05-06T07:08:09.00010010044Z", "1234-05-06T07:08:09.000100100Z", Second)
	test("1234-05-06T07:08:09.00010010055Z", "1234-05-06T07:08:09.000100101Z", Second)
	test("1234-05-06T07:08:09.00010010099Z", "1234-05-06T07:08:09.000100101Z", Second)
	test("1234-05-06T07:08:09.99999999999Z", "1234-05-06T07:08:10.000000000Z", Second)
	test("1234-12-31T23:59:59.99999999999Z", "1235-01-01T00:00:00.000000000Z", Second)
	test("1234-05-06T07:08:09.000100100+09:10", "1234-05-06T07:08:09.000100100+09:10", Second)
	test("1234-05-06T07:08:09.100100100-10:11", "1234-05-06T07:08:09.100100100-10:11", Second)
	test("1234-05-06T07:08:09.00010010044+09:10", "1234-05-06T07:08:09.000100100+09:10", Second)
	test("1234-05-06T07:08:09.00010010055-10:11", "1234-05-06T07:08:09.000100101-10:11", Second)
	test("1234-05-06T07:08:09.00010010099+09:10", "1234-05-06T07:08:09.000100101+09:10", Second)
	test("1234-05-06T07:08:09.99999999999-10:11", "1234-05-06T07:08:10.000000000-10:11", Second)
	test("1234-12-31T23:59:59.99999999999+09:10", "1235-01-01T00:00:00.000000000+09:10", Second)

	test("1234-05-06T07:08+09:10", "1234-05-06T07:08:00+09:10", Minute)
	test("1234-05-06T07:08:09-10:11", "1234-05-06T07:08:09-10:11", Second)
}

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
	test("$123", "$123")

	test("123", "'123'")
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
	test("$123", false)

	test("123", true)
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
