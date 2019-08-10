package ion

import (
	"io"
	"testing"
)

func TestNext(t *testing.T) {
	tok := tokenizeString("foo::'foo':[] 123, {})")

	next := func(tt token) {
		if err := tok.Next(); err != nil {
			t.Fatal(err)
		}
		if tok.Token() != tt {
			t.Fatalf("expected %v, got %v", tt, tok.Token())
		}
	}

	next(tokenSymbol)
	next(tokenDoubleColon)
	next(tokenSymbolQuoted)
	next(tokenColon)
	next(tokenOpenBracket)
	next(tokenNumber)
	next(tokenComma)
	next(tokenOpenBrace)
}

func TestReadSymbol(t *testing.T) {
	test := func(str string, expected string, next token) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			if err := tok.Next(); err != nil {
				t.Fatal(err)
			}

			if tok.Token() != tokenSymbol {
				t.Fatal("not a symbol")
			}

			actual, err := tok.readSymbol()
			if err != nil {
				t.Fatal(err)
			}

			if actual != expected {
				t.Errorf("expected '%v', got '%v'", expected, actual)
			}

			if err := tok.Next(); err != nil {
				t.Fatal(err)
			}
			if tok.Token() != next {
				t.Errorf("expected next=%v, got next=%v", next, tok.Token())
			}
		})
	}

	test("a", "a", tokenEOF)
	test("abc", "abc", tokenEOF)
	test("null +inf", "null", tokenFloatInf)
	test("false,", "false", tokenComma)
	test("nan]", "nan", tokenCloseBracket)
}

func TestReadSymbols(t *testing.T) {
	tok := tokenizeString("foo bar baz beep boop null")
	expected := []string{"foo", "bar", "baz", "beep", "boop", "null"}

	for i := 0; i < len(expected); i++ {
		if err := tok.Next(); err != nil {
			t.Fatal(err)
		}
		if tok.Token() != tokenSymbol {
			t.Fatalf("expected %v, got %v", tokenSymbol, tok.Token())
		}

		val, err := tok.readSymbol()
		if err != nil {
			t.Fatal(err)
		}

		if val != expected[i] {
			t.Errorf("expected %v, got %v", expected[i], val)
		}
	}
}

func TestReadQuotedSymbol(t *testing.T) {
	test := func(str string, expected string, next int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			if err := tok.Next(); err != nil {
				t.Fatal(err)
			}

			if tok.Token() != tokenSymbolQuoted {
				t.Fatal("not a quoted symbol")
			}

			actual, err := tok.readQuotedSymbol()
			if err != nil {
				t.Fatal(err)
			}

			if actual != expected {
				t.Errorf("expected '%v', got '%v'", expected, actual)
			}

			c, err := tok.read()
			if err != nil {
				t.Fatal(err)
			}
			if c != next {
				t.Errorf("expected next=%q, got next=%q", next, c)
			}
		})
	}

	test("'a'", "a", -1)
	test("'a b c'", "a b c", -1)
	test("'null' ", "null", ' ')
	test("'false',", "false", ',')
	test("'nan']", "nan", ']')

	test("'a\\'b'", "a'b", -1)
	test("'a\\\nb'", "ab", -1)
	test("'a\\\\b'", "a\\b", -1)
	test("'a\x20b'", "a b", -1)
	test("'a\\u2248b'", "a≈b", -1)
	test("'a\\U0001F44Db'", "a👍b", -1)
}

func TestReadTimestamp(t *testing.T) {
	test := func(str string, eval string, next int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			if err := tok.Next(); err != nil {
				t.Fatal(err)
			}
			if tok.Token() != tokenTimestamp {
				t.Fatalf("unexpected token %v", tok.Token())
			}

			val, err := tok.ReadValue(tokenTimestamp)
			if err != nil {
				t.Fatal(err)
			}
			if val != eval {
				t.Errorf("expected %v, got %v", eval, val)
			}

			c, err := tok.read()
			if err != nil {
				t.Fatal(err)
			}
			if c != next {
				t.Errorf("expected %q, got %q", next, c)
			}
		})
	}

	test("2001T", "2001T", -1)
	test("2001-01T,", "2001-01T", ',')
	test("2001-01-02}", "2001-01-02", '}')
	test("2001-01-02T ", "2001-01-02T", ' ')
	test("2001-01-02T+00:00\t", "2001-01-02T+00:00", '\t')
	test("2001-01-02T-00:00\n", "2001-01-02T-00:00", '\n')
	test("2001-01-02T03:04+00:00 ", "2001-01-02T03:04+00:00", ' ')
	test("2001-01-02T03:04-00:00 ", "2001-01-02T03:04-00:00", ' ')
	test("2001-01-02T03:04Z ", "2001-01-02T03:04Z", ' ')
	test("2001-01-02T03:04z ", "2001-01-02T03:04z", ' ')
	test("2001-01-02T03:04:05Z ", "2001-01-02T03:04:05Z", ' ')
	test("2001-01-02T03:04:05+00:00 ", "2001-01-02T03:04:05+00:00", ' ')
	test("2001-01-02T03:04:05.666Z ", "2001-01-02T03:04:05.666Z", ' ')
	test("2001-01-02T03:04:05.666666z ", "2001-01-02T03:04:05.666666z", ' ')
}

func TestIsTripleQuote(t *testing.T) {
	test := func(str string, eok bool, next int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)

			ok, err := tok.IsTripleQuote()
			if err != nil {
				t.Fatal(err)
			}
			if ok != eok {
				t.Errorf("expected ok=%v, got ok=%v", eok, ok)
			}

			read(t, tok, next)
		})
	}

	test("''string'''", true, 's')
	test("'string'''", false, '\'')
	test("'", false, '\'')
	test("", false, -1)
}

func TestIsInf(t *testing.T) {
	test := func(str string, eok bool, next int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, err := tok.read()
			if err != nil {
				t.Fatal(err)
			}

			ok, err := tok.isInf(c)
			if err != nil {
				t.Fatal(err)
			}

			if ok != eok {
				t.Errorf("expected %v, got %v", eok, ok)
			}

			c, err = tok.read()
			if err != nil {
				t.Fatal(err)
			}
			if c != next {
				t.Errorf("expected '%c', got '%c'", next, c)
			}
		})
	}

	test("+inf", true, -1)
	test("-inf", true, -1)
	test("+inf ", true, ' ')
	test("-inf\t", true, '\t')
	test("-inf\n", true, '\n')
	test("+inf,", true, ',')
	test("-inf}", true, '}')
	test("+inf)", true, ')')
	test("-inf]", true, ']')
	test("+inf//", true, '/')
	test("+inf/*", true, '/')

	test("+inf/", false, 'i')
	test("-inf/0", false, 'i')
	test("+int", false, 'i')
	test("-iot", false, 'i')
	test("+unf", false, 'u')
	test("_inf", false, 'i')

	test("-in", false, 'i')
	test("+i", false, 'i')
	test("+", false, -1)
	test("-", false, -1)
}

func TestScanForNumericType(t *testing.T) {
	test := func(str string, ett token) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, err := tok.read()
			if err != nil {
				t.Fatal(err)
			}

			tt, err := tok.scanForNumericType(c)
			if err != nil {
				t.Fatal(err)
			}
			if tt != ett {
				t.Errorf("expected %v, got %v", ett, tt)
			}
		})
	}

	test("0b0101", tokenBinary)
	test("0B", tokenBinary)
	test("0xABCD", tokenHex)
	test("0X", tokenHex)
	test("0000-00-00", tokenTimestamp)
	test("0000T", tokenTimestamp)

	test("0", tokenNumber)
	test("1b0101", tokenNumber)
	test("1B", tokenNumber)
	test("1x0101", tokenNumber)
	test("1X", tokenNumber)
	test("1234", tokenNumber)
	test("12345", tokenNumber)
	test("1,23T", tokenNumber)
	test("12,3T", tokenNumber)
	test("123,T", tokenNumber)
}

func TestSkipWhitespace(t *testing.T) {
	test := func(str string, eok bool, ec int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, ok, err := tok.skipWhitespace()
			if err != nil {
				t.Fatal(err)
			}

			if ok != eok {
				t.Errorf("expected ok=%v, got ok=%v", eok, ok)
			}
			if c != ec {
				t.Errorf("expected c='%c', got c='%c'", ec, c)
			}
		})
	}

	test("/ 0)", false, '/')
	test("xyz_", false, 'x')
	test(" / 0)", true, '/')
	test(" xyz_", true, 'x')
	test(" \t\r\n / 0)", true, '/')
	test("\t\t  // comment\t\r\n\t\t  x", true, 'x')
	test(" \r\n /* comment *//* \r\n comment */x", true, 'x')
}

func TestSkipLobWhitespace(t *testing.T) {
	test := func(str string, eok bool, ec int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, ok, err := tok.skipLobWhitespace()
			if err != nil {
				t.Fatal(err)
			}

			if ok != eok {
				t.Errorf("expected ok=%v, got ok=%v", eok, ok)
			}
			if c != ec {
				t.Errorf("expected c='%c', got c='%c'", ec, c)
			}
		})
	}

	test("///=", false, '/')
	test("xyz_", false, 'x')
	test(" ///=", true, '/')
	test(" xyz_", true, 'x')
	test("\r\n\t///=", true, '/')
	test("\r\n\txyz_", true, 'x')
}

func TestSkipCommentsHandler(t *testing.T) {
	t.Run("SingleLine", func(t *testing.T) {
		tok := tokenizeString("/comment\nok")
		ok, err := tok.skipCommentsHandler()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected ok=true, got ok=false")
		}

		read(t, tok, 'o')
		read(t, tok, 'k')
		read(t, tok, -1)
	})

	t.Run("Block", func(t *testing.T) {
		tok := tokenizeString("*comm\nent*/ok")
		ok, err := tok.skipCommentsHandler()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Error("expected ok=true, got ok=false")
		}

		read(t, tok, 'o')
		read(t, tok, 'k')
		read(t, tok, -1)
	})

	t.Run("FalseAlarm", func(t *testing.T) {
		tok := tokenizeString(" 0)")
		ok, err := tok.skipCommentsHandler()
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Error("expected ok=false, got ok=true")
		}

		read(t, tok, ' ')
		read(t, tok, '0')
		read(t, tok, ')')
		read(t, tok, -1)
	})
}

func TestSkipSingleLineComment(t *testing.T) {
	tok := tokenizeString("single-line comment\r\nok")
	err := tok.skipSingleLineComment()
	if err != nil {
		t.Fatal(err)
	}

	read(t, tok, 'o')
	read(t, tok, 'k')
	read(t, tok, -1)
}

func TestSkipSingleLineCommentOnLastLine(t *testing.T) {
	tok := tokenizeString("single-line comment")
	err := tok.skipSingleLineComment()
	if err != nil {
		t.Fatal(err)
	}

	read(t, tok, -1)
}

func TestSkipBlockComment(t *testing.T) {
	tok := tokenizeString("this is/ a\nmulti-line /** comment.**/ok")
	err := tok.skipBlockComment()
	if err != nil {
		t.Fatal(err)
	}

	read(t, tok, 'o')
	read(t, tok, 'k')
	read(t, tok, -1)
}

func TestSkipInvalidBlockComment(t *testing.T) {
	tok := tokenizeString("this is a comment that never ends")
	err := tok.skipBlockComment()
	if err == nil {
		t.Error("did not fail on bad block comment")
	}
}

func TestPeekN(t *testing.T) {
	tok := tokenizeString("abc\r\ndef")

	peekN(t, tok, 1, nil, 'a')
	peekN(t, tok, 2, nil, 'a', 'b')
	peekN(t, tok, 3, nil, 'a', 'b', 'c')

	read(t, tok, 'a')
	read(t, tok, 'b')

	peekN(t, tok, 3, nil, 'c', '\n', 'd')
	peekN(t, tok, 2, nil, 'c', '\n')
	peekN(t, tok, 3, nil, 'c', '\n', 'd')

	read(t, tok, 'c')
	read(t, tok, '\n')
	read(t, tok, 'd')

	peekN(t, tok, 3, io.EOF, 'e', 'f')
	peekN(t, tok, 3, io.EOF, 'e', 'f')
	peekN(t, tok, 2, nil, 'e', 'f')

	read(t, tok, 'e')
	read(t, tok, 'f')
	read(t, tok, -1)

	peekN(t, tok, 10, io.EOF)
}

func peekN(t *testing.T, tok *tokenizer, n int, ee error, ecs ...int) {
	cs, err := tok.peekN(n)
	if err != ee {
		t.Fatalf("expected err=%v, got err=%v", ee, err)
	}
	if !equal(ecs, cs) {
		t.Errorf("expected %v, got %v", ecs, cs)
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestPeek(t *testing.T) {
	tok := tokenizeString("abc")

	peek(t, tok, 'a')
	peek(t, tok, 'a')
	read(t, tok, 'a')

	peek(t, tok, 'b')
	tok.unread('a')

	peek(t, tok, 'a')
	read(t, tok, 'a')
	read(t, tok, 'b')
	peek(t, tok, 'c')
	peek(t, tok, 'c')

	read(t, tok, 'c')
	peek(t, tok, -1)
	peek(t, tok, -1)
	read(t, tok, -1)
}

func peek(t *testing.T, tok *tokenizer, expected int) {
	c, err := tok.peek()
	if err != nil {
		t.Fatal(err)
	}
	if c != expected {
		t.Errorf("expected %v, got %v", expected, c)
	}
}

func TestReadUnread(t *testing.T) {
	tok := tokenizeString("abc\rd\ne\r\n")

	read(t, tok, 'a')
	tok.unread('a')

	read(t, tok, 'a')
	read(t, tok, 'b')
	read(t, tok, 'c')
	tok.unread('c')
	tok.unread('b')

	read(t, tok, 'b')
	read(t, tok, 'c')
	read(t, tok, '\n')
	tok.unread('\n')

	read(t, tok, '\n')
	read(t, tok, 'd')
	read(t, tok, '\n')
	read(t, tok, 'e')
	read(t, tok, '\n')
	read(t, tok, -1)

	tok.unread(-1)
	tok.unread('\n')

	read(t, tok, '\n')
	read(t, tok, -1)
	read(t, tok, -1)
}

func TestTokenToString(t *testing.T) {
	for i := tokenError; i <= tokenCloseDoubleBrace+1; i++ {
		str := i.String()
		if str == "" {
			t.Errorf("expected non-empty string for token %v", int(i))
		}
	}
}

func read(t *testing.T, tok *tokenizer, expected int) {
	c, err := tok.read()
	if err != nil {
		t.Fatal(err)
	}
	if c != expected {
		t.Errorf("expected %v, got %v", expected, c)
	}
}
