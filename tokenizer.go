package ion

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

type tokenType int

const (
	tokenError tokenType = iota

	tokenEOF // End of input

	tokenNumeric       // Haven't seen enough to know which, yet
	tokenInt           // [0-9]+
	tokenBinary        // 0b[01]+
	tokenHex           // 0x[0-9a-fA-F]+
	tokenDecimal       // [0-9]+.[0-9]+d[0-9]+
	tokenFloat         // [0-9]+.[0-9]+e[0-9]+
	tokenFloatInf      // +inf
	tokenFloatMinusInf // -inf
	tokenTimestamp     // 2001-01-01T00:00:00.000Z

	tokenSymbol         // [a-zA-Z_]+
	tokenSymbolQuoted   // '[^']+'
	tokenSymbolOperator // +-/*

	tokenString     // "[^"]+"
	tokenLongString // '''[^']+'''

	tokenDot         // .
	tokenComma       // ,
	tokenColon       // :
	tokenDoubleColon // ::

	tokenOpenParen        // (
	tokenCloseParen       // )
	tokenOpenBrace        // {
	tokenCloseBrace       // }
	tokenOpenBracket      // [
	tokenCloseBracket     // ]
	tokenOpenDoubleBrace  // {{
	tokenCloseDoubleBrace // }}
)

func (t tokenType) String() string {
	switch t {
	case tokenError:
		return "error"
	case tokenEOF:
		return "EOF"
	case tokenNumeric:
		return "numeric"
	case tokenInt:
		return "int"
	case tokenBinary:
		return "binary"
	case tokenHex:
		return "hex"
	case tokenDecimal:
		return "decimal"
	case tokenFloat:
		return "float"
	case tokenFloatInf:
		return "+inf"
	case tokenFloatMinusInf:
		return "-inf"
	case tokenTimestamp:
		return "timestamp"
	case tokenSymbol:
		return "symbol"
	case tokenSymbolQuoted:
		return "symbolQuoted"
	case tokenSymbolOperator:
		return "symbolOperator"

	case tokenString:
		return "string"
	case tokenLongString:
		return "longstring"

	case tokenDot:
		return "dot"
	case tokenComma:
		return "comma"
	case tokenColon:
		return "colon"
	case tokenDoubleColon:
		return "doublecolon"

	case tokenOpenParen:
		return "openparen"
	case tokenCloseParen:
		return "closeparen"

	case tokenOpenBrace:
		return "openbrace"
	case tokenCloseBrace:
		return "closebrace"

	case tokenOpenBracket:
		return "openbracket"
	case tokenCloseBracket:
		return "closebracket"
	case tokenOpenDoubleBrace:
		return "opendoublebrace"
	case tokenCloseDoubleBrace:
		return "closedoublebrace"

	default:
		return "<???>"
	}
}

type tokenizer struct {
	in     *bufio.Reader
	buffer []int

	token      tokenType
	unfinished bool
}

func tokenizeString(in string) *tokenizer {
	return tokenizeBytes([]byte(in))
}

func tokenizeBytes(in []byte) *tokenizer {
	return tokenize(bytes.NewReader(in))
}

func tokenize(in io.Reader) *tokenizer {
	return &tokenizer{
		in: bufio.NewReader(in),
	}
}

// Token returns the type of the current token.
func (t *tokenizer) Token() tokenType {
	return t.token
}

// Next advances to the next token in the input stream.
func (t *tokenizer) Next() error {
	var c int
	var err error

	if t.unfinished {
		c, err = t.skipValue()
	} else {
		c, _, err = t.skipWhitespace()
	}

	if err != nil {
		return err
	}

	switch {
	case c == -1:
		return t.finish(tokenEOF, true)

	case c == '/':
		t.unread(c)
		return t.finish(tokenSymbolOperator, true)

	case c == ':':
		c2, err := t.peek()
		if err != nil {
			return err
		}
		if c2 == ':' {
			t.read()
			return t.finish(tokenDoubleColon, false)
		} else {
			return t.finish(tokenColon, false)
		}

	case c == '{':
		c2, err := t.peek()
		if err != nil {
			return err
		}
		if c2 == '{' {
			t.read()
			return t.finish(tokenOpenDoubleBrace, true)
		} else {
			return t.finish(tokenOpenBrace, true)
		}

	case c == '}':
		return t.finish(tokenCloseBrace, false)

	case c == '[':
		return t.finish(tokenOpenBracket, true)

	case c == ']':
		return t.finish(tokenCloseBracket, false)

	case c == '(':
		return t.finish(tokenOpenParen, true)

	case c == ')':
		return t.finish(tokenCloseParen, false)

	case c == ',':
		return t.finish(tokenComma, false)

	case c == '.':
		c2, err := t.peek()
		if err != nil {
			return err
		}
		if isOperatorChar(c2) {
			t.unread(c)
			return t.finish(tokenSymbolOperator, true)
		} else {
			return t.finish(tokenDot, false)
		}

	case c == '\'':
		ok, err := t.isTripleQuote()
		if err != nil {
			return err
		}
		if ok {
			return t.finish(tokenLongString, true)
		} else {
			return t.finish(tokenSymbolQuoted, true)
		}

	case c == '+':
		ok, err := t.isInf(c)
		if err != nil {
			return err
		}
		if ok {
			return t.finish(tokenFloatInf, false)
		} else {
			t.unread(c)
			return t.finish(tokenSymbolOperator, true)
		}

	case isOperatorChar(c):
		t.unread(c)
		return t.finish(tokenSymbolOperator, true)

	case c == '"':
		return t.finish(tokenString, true)

	case isIdentifierStart(c):
		t.unread(c)
		return t.finish(tokenSymbol, true)

	case isDigit(c):
		tt, err := t.scanForNumericType(c)
		if err != nil {
			return err
		}

		t.unread(c)
		return t.finish(tt, true)

	case c == '-':
		c2, err := t.peek()
		if err != nil {
			return err
		}

		if isDigit(c2) {
			t.read()
			tt, err := t.scanForNumericType(c2)
			if err != nil {
				return err
			}
			if tt == tokenTimestamp {
				// can't have negative timestamps.
				return invalidChar(c2)
			}
			t.unread(c2)
			return t.finish(tt, true)
		}

		ok, err := t.isInf(c)
		if err != nil {
			return err
		}
		if ok {
			return t.finish(tokenFloatMinusInf, false)
		}

		t.unread(c)
		return t.finish(tokenSymbolOperator, true)

	default:
		return invalidChar(c)
	}
}

func (t *tokenizer) finish(token tokenType, more bool) error {
	t.token = token
	t.unfinished = more
	return nil
}

// IsTripleQuote returns true if this is a triple-quote sequence (''').
func (t *tokenizer) isTripleQuote() (bool, error) {
	// We've just read a '\'', check if the next two are too.
	cs, err := t.peekN(2)
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if cs[0] == '\'' && cs[1] == '\'' {
		t.skipN(2)
		return true, nil
	}

	return false, nil
}

// IsInf returns true if the given character begins a '+inf' or
// '-inf' keyword.
func (t *tokenizer) isInf(c int) (bool, error) {
	if c != '+' && c != '-' {
		return false, nil
	}

	cs, err := t.peekN(5)
	if err != nil && err != io.EOF {
		return false, err
	}

	if len(cs) < 3 || cs[0] != 'i' || cs[1] != 'n' || cs[2] != 'f' {
		// Definitely not +-inf.
		return false, nil
	}

	if len(cs) == 3 || isStopChar(cs[3]) {
		// Cleanly-terminated +-inf.
		t.skipN(3)
		return true, nil
	}

	if cs[3] == '/' && len(cs) > 4 && (cs[4] == '/' || cs[4] == '*') {
		t.skipN(3)
		// +-inf followed immediately by a comment works too.
		return true, nil
	}

	return false, nil
}

// ScanForNumericType attempts to determine what type of number we
// have by peeking at a fininte number of characters. We can rule
// out binary (0b...), hex (0x...), and timestamps (....-) via this
// method. There are a couple other cases where we *could* distinguish,
// but it's unclear that it's worth it.
func (t *tokenizer) scanForNumericType(c int) (tokenType, error) {
	if !isDigit(c) {
		panic("scanForNumericType with non-digit")
	}

	cs, err := t.peekN(4)
	if err != nil && err != io.EOF {
		return tokenError, err
	}

	if c == '0' && len(cs) > 0 {
		switch {
		case cs[0] == 'b' || cs[0] == 'B':
			return tokenBinary, nil

		case cs[0] == 'x' || cs[0] == 'X':
			return tokenHex, nil
		}
	}

	if len(cs) >= 4 {
		if isDigit(cs[0]) && isDigit(cs[1]) && isDigit(cs[2]) {
			if cs[3] == '-' || cs[3] == 'T' {
				return tokenTimestamp, nil
			}
		}
	}

	// Can't tell yet; wait until actually reading it to find out.
	return tokenNumeric, nil
}

// Is this character a valid way to end a 'normal' (unquoted) value?
// Peeks in case of '/', so don't call it with a character you've
// peeked.
func (t *tokenizer) isStopChar(c int) (bool, error) {
	if isStopChar(c) {
		return true, nil
	}

	if c == '/' {
		c2, err := t.peek()
		if err != nil {
			return false, err
		}
		if c2 == '/' || c2 == '*' {
			// Comment, also all done.
			return true, nil
		}
	}

	return false, nil
}

type matcher func(int) bool

// Expect reads a byte of input and asserts that it matches some
// condition, returning an error if it does not.
func (t *tokenizer) expect(f matcher) error {
	c, err := t.read()
	if err != nil {
		return err
	}
	if !f(c) {
		return invalidChar(c)
	}
	return nil
}

// InvalidChar returns an error complaining that the given character was
// unexpected.
func invalidChar(c int) error {
	if c == -1 {
		return errors.New("unexpected EOF")
	}
	return fmt.Errorf("unexpected char %q", c)
}

// SkipN skips over the next n bytes of input. Presumably you've
// already peeked at them, and decided they're not worth keeping.
func (t *tokenizer) skipN(n int) error {
	for i := 0; i < n; i++ {
		c, err := t.read()
		if err != nil {
			return err
		}
		if c == -1 {
			break
		}
	}
	return nil
}

// PeekN peeks at the next n bytes of input. Unlike read/peek, does
// NOT return -1 to indicate EOF. If it cannot peek N bytes ahead
// because of an EOF (or other error), it returns the bytes it was
// able to peek at along with the error.
func (t *tokenizer) peekN(n int) ([]int, error) {
	var ret []int
	var err error

	// Read ahead.
	for i := 0; i < n; i++ {
		var c int
		c, err = t.read()
		if err != nil {
			break
		}
		if c == -1 {
			err = io.EOF
			break
		}
		ret = append(ret, c)
	}

	// Put back the ones we got.
	if err == io.EOF {
		t.unread(-1)
	}
	for i := len(ret) - 1; i >= 0; i-- {
		t.unread(ret[i])
	}

	return ret, err
}

// Peek at the next byte of input without removing it. Other conditions
// from Read all apply.
func (t *tokenizer) peek() (int, error) {
	if len(t.buffer) > 0 {
		// Short-circuit and peek from the buffer.
		return t.buffer[len(t.buffer)-1], nil
	}

	c, err := t.read()
	if err != nil {
		return 0, err
	}

	t.unread(c)
	return c, nil
}

// Read reads a byte of input from the underlying reader. EOF is
// returned as (-1, nil) rather than (0, io.EOF), because I find it
// easier to reason about that way. Newlines are normalized to '\n'.
func (t *tokenizer) read() (int, error) {
	if len(t.buffer) > 0 {
		// We've already peeked ahead; read from our buffer.
		c := t.buffer[len(t.buffer)-1]
		t.buffer = t.buffer[:len(t.buffer)-1]
		return c, nil
	}

	c, err := t.in.ReadByte()
	if err == io.EOF {
		return -1, nil
	}
	if err != nil {
		return 0, err
	}

	// Normalize \r and \r\n to just \n.
	if c == '\r' {
		cs, err := t.in.Peek(1)
		if err != nil && err != io.EOF {
			// Not EOF, because we haven't dealt with the '\r' yet.
			return 0, err
		}
		if len(cs) > 0 && cs[0] == '\n' {
			// Skip over the '\n' as well.
			t.in.ReadByte()
		}
		return '\n', nil
	}

	return int(c), nil
}

// Unread pushes a character (or -1) back into the input stream to
// be read again later.
func (t *tokenizer) unread(c int) {
	t.buffer = append(t.buffer, c)
}
