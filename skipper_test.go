package ion

import (
	"testing"
)

func TestSkipNumber(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipNumber)

	test("", -1)
	test("0", -1)
	test("-1234567890,", ',')
	test("1.2 ", ' ')
	test("1d45\n", '\n')
	test("1.4e-12//", '/')

	testErr("1.2d3d", "unexpected char 'd'")
}

func TestSkipBinary(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipBinary)

	test("0b0", -1)
	test("-0b10 ", ' ')
	test("0b010101,", ',')

	testErr("0b2", "unexpected char '2'")
}

func TestSkipHex(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipHex)

	test("0x0", -1)
	test("-0x0F ", ' ')
	test("0x1234567890abcdefABCDEF,", ',')

	testErr("0x0G", "unexpected char 'G'")
}

func TestSkipTimestamp(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipTimestamp)

	test("2001T", -1)
	test("2001-01T,", ',')
	test("2001-01-02}", '}')
	test("2001-01-02T ", ' ')
	test("2001-01-02T+00:00\t", '\t')
	test("2001-01-02T-00:00\n", '\n')
	test("2001-01-02T03:04+00:00 ", ' ')
	test("2001-01-02T03:04-00:00 ", ' ')
	test("2001-01-02T03:04Z ", ' ')
	test("2001-01-02T03:04z ", ' ')
	test("2001-01-02T03:04:05Z ", ' ')
	test("2001-01-02T03:04:05+00:00 ", ' ')
	test("2001-01-02T03:04:05.666Z ", ' ')
	test("2001-01-02T03:04:05.666666z ", ' ')

	testErr("", "unexpected EOF")
	testErr("2001", "unexpected EOF")
	testErr("2001z", "unexpected char 'z'")
	testErr("20011", "unexpected char '1'")
	testErr("2001-0", "unexpected EOF")
	testErr("2001-01", "unexpected EOF")
	testErr("2001-01-02Tz", "unexpected char 'z'")
	testErr("2001-01-02T03", "unexpected EOF")
	testErr("2001-01-02T03z", "unexpected char 'z'")
	testErr("2001-01-02T03:04x ", "unexpected char 'x'")
	testErr("2001-01-02T03:04:05x ", "unexpected char 'x'")
}

func TestSkipSymbol(t *testing.T) {
	test, _ := testSkip(t, (*tokenizer).skipSymbol)

	test("f", -1)
	test("foo:", ':')
	test("foo,", ',')
	test("foo ", ' ')
	test("foo\n", '\n')
	test("foo]", ']')
	test("foo}", '}')
	test("foo)", ')')
	test("foo\\n", '\\')
}

func TestSkipSymbolQuoted(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipSymbolQuoted)

	test("'", -1)
	test("foo',", ',')
	test("foo\\'bar':", ':')
	test("foo\\\nbar',", ',')

	testErr("foo", "unexpected EOF")
	testErr("foo\n", "unexpected char '\\n'")
}

func TestSkipSymbolOperator(t *testing.T) {
	test, _ := testSkip(t, (*tokenizer).skipSymbolOperator)

	test("+", -1)
	test("++", -1)
	test("+= ", ' ')
	test("%b", 'b')
}

func TestSkipString(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipString)

	test("\"", -1)
	test("\",", ',')
	test("foo\\\"bar\"], \"\"", ']')
	test("foo\\\nbar\" \t\t\t", ' ')

	testErr("foobar", "unexpected EOF")
	testErr("foobar\n", "unexpected char '\\n'")
}

func TestSkipLongString(t *testing.T) {
	test, _ := testSkip(t, (*tokenizer).skipLongString)

	test("'''", -1)
	test("''',", ',')
	test("abc''',", ',')
	test("abc'''   }", '}')
	test("abc''' /*more*/ '''def'''\t//more\r\n]", ']')
}

func TestSkipBlob(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipBlob)

	test("}}", -1)
	test("oogboog}},{{}}", ',')
	test("'''not encoded'''}}\n", '\n')

	testErr("", "unexpected EOF")
	testErr("oogboog", "unexpected EOF")
	testErr("oogboog}", "unexpected EOF")
	testErr("oog}{boog", "unexpected char '{'")
}

func TestSkipList(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipList)

	test("]", -1)
	test("[]],", ',')
	test("[123, \"]\", ']']] ", ' ')

	testErr("abc, def, ", "unexpected EOF")
}

type skipFunc func(*tokenizer) (int, error)
type skipTestFunc func(string, int)
type skipTestErrFunc func(string, string)

func testSkip(t *testing.T, f skipFunc) (skipTestFunc, skipTestErrFunc) {
	test := func(str string, ec int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, err := f(tok)
			if err != nil {
				t.Fatal(err)
			}
			if c != ec {
				t.Errorf("expected '%c', got '%c'", ec, c)
			}
		})
	}
	testErr := func(str string, e string) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			_, err := f(tok)
			if err == nil || err.Error() != e {
				t.Errorf("expected err=%v, got err=%v", e, err)
			}
		})
	}
	return test, testErr
}
