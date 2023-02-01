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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type token int

const (
	tokenError token = iota

	tokenEOF // End of input

	tokenNumber        // Haven't seen enough to know which, yet
	tokenBinary        // 0b[01]+
	tokenHex           // 0x[0-9a-fA-F]+
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

const clobText = true
const nonClobText = false

func (t token) String() string {
	switch t {
	case tokenError:
		return "<error>"
	case tokenEOF:
		return "<EOF>"
	case tokenNumber:
		return "<number>"
	case tokenBinary:
		return "<binary>"
	case tokenHex:
		return "<hex>"
	case tokenFloatInf:
		return "+inf"
	case tokenFloatMinusInf:
		return "-inf"
	case tokenTimestamp:
		return "<timestamp>"
	case tokenSymbol:
		return "<symbol>"
	case tokenSymbolQuoted:
		return "<quoted-symbol>"
	case tokenSymbolOperator:
		return "<operator>"

	case tokenString:
		return "<string>"
	case tokenLongString:
		return "<long-string>"

	case tokenDot:
		return "."
	case tokenComma:
		return ","
	case tokenColon:
		return ":"
	case tokenDoubleColon:
		return "::"

	case tokenOpenParen:
		return "("
	case tokenCloseParen:
		return ")"

	case tokenOpenBrace:
		return "{"
	case tokenCloseBrace:
		return "}"

	case tokenOpenBracket:
		return "["
	case tokenCloseBracket:
		return "]"

	case tokenOpenDoubleBrace:
		return "{{"
	case tokenCloseDoubleBrace:
		return "}}"

	default:
		return "<???>"
	}
}

type tokenizer struct {
	in     *bufio.Reader
	buffer []int

	token      token
	unfinished bool
	pos        uint64
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
func (t *tokenizer) Token() token {
	return t.token
}

func (t *tokenizer) Pos() uint64 {
	return t.pos
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
		return t.ok(tokenEOF, true)

	case c == ':':
		c2, err := t.peek()
		if err != nil {
			return err
		}
		if c2 == ':' {
			_, err = t.read()
			if err != nil {
				return err
			}
			return t.ok(tokenDoubleColon, false)
		}
		return t.ok(tokenColon, false)

	case c == '{':
		c2, err := t.peek()
		if err != nil {
			return err
		}
		if c2 == '{' {
			_, err = t.read()
			if err != nil {
				return err
			}
			return t.ok(tokenOpenDoubleBrace, true)
		}
		return t.ok(tokenOpenBrace, true)

	case c == '}':
		return t.ok(tokenCloseBrace, false)

	case c == '[':
		return t.ok(tokenOpenBracket, true)

	case c == ']':
		return t.ok(tokenCloseBracket, false)

	case c == '(':
		return t.ok(tokenOpenParen, true)

	case c == ')':
		return t.ok(tokenCloseParen, false)

	case c == ',':
		return t.ok(tokenComma, false)

	case c == '.':
		c2, err := t.peek()
		if err != nil {
			return err
		}
		if isOperatorChar(c2) {
			t.unread(c)
			return t.ok(tokenSymbolOperator, true)
		}
		if c2 == ' ' || isIdentifierPart(c2) {
			t.unread(c)
		}

		return t.ok(tokenDot, false)

	case c == '\'':
		ok, err := t.IsTripleQuote()
		if err != nil {
			return err
		}
		if ok {
			return t.ok(tokenLongString, true)
		}
		return t.ok(tokenSymbolQuoted, true)

	case c == '+':
		ok, err := t.isInf(c)
		if err != nil {
			return err
		}
		if ok {
			return t.ok(tokenFloatInf, false)
		}
		t.unread(c)
		return t.ok(tokenSymbolOperator, true)

	case c == '-':
		c2, err := t.peek()
		if err != nil {
			return err
		}

		if isDigit(c2) {
			_, err = t.read()
			if err != nil {
				return err
			}

			tt, err := t.scanForNumericType(c2)
			if err != nil {
				return err
			}
			if tt == tokenTimestamp {
				// can't have negative timestamps.
				return t.invalidChar(c2)
			}
			t.unread(c2)
			t.unread(c)
			return t.ok(tt, true)
		}

		ok, err := t.isInf(c)
		if err != nil {
			return err
		}
		if ok {
			return t.ok(tokenFloatMinusInf, false)
		}

		t.unread(c)
		return t.ok(tokenSymbolOperator, true)

	case isOperatorChar(c):
		t.unread(c)
		return t.ok(tokenSymbolOperator, true)

	case c == '"':
		return t.ok(tokenString, true)

	case isIdentifierStart(c):
		t.unread(c)
		return t.ok(tokenSymbol, true)

	case isDigit(c):
		tt, err := t.scanForNumericType(c)
		if err != nil {
			return err
		}

		t.unread(c)
		return t.ok(tt, true)

	default:
		return t.invalidChar(c)
	}
}

func (t *tokenizer) ok(tok token, more bool) error {
	t.token = tok
	t.unfinished = more
	return nil
}

// SetFinished marks the current token finished (indicating that the caller has
// chosen to step in to a list, sexp, or struct and Next should not skip over its
// contents in search of the next token).
func (t *tokenizer) SetFinished() {
	t.unfinished = false
}

// FinishValue skips to the end of the current value if (and only if)
// we're currently in the middle of reading it.
func (t *tokenizer) FinishValue() (bool, error) {
	if !t.unfinished {
		return false, nil
	}

	c, err := t.skipValue()
	if err != nil {
		return true, err
	}

	t.unread(c)
	t.unfinished = false
	return true, nil
}

// ReadValue reads the value of a token of the given type.
func (t *tokenizer) ReadValue(tok token) (string, error) {
	var str string
	var err error

	switch tok {
	case tokenSymbol:
		str, err = t.readSymbol()
	case tokenSymbolQuoted:
		str, err = t.readQuotedSymbol()
	case tokenSymbolOperator, tokenDot:
		str, err = t.readOperator()
	case tokenString:
		str, err = t.readString()
	case tokenLongString:
		str, err = t.readLongString()
	case tokenBinary:
		str, err = t.readBinary()
	case tokenHex:
		str, err = t.readHex()
	case tokenTimestamp:
		str, err = t.readTimestamp()
	default:
		panic(fmt.Sprintf("unsupported token type %v", tok))
	}

	if err != nil {
		return "", err
	}

	t.unfinished = false
	return str, nil
}

// ReadNumber reads a number and determines the type.
func (t *tokenizer) ReadNumber() (string, Type, error) {
	w := strings.Builder{}

	c, err := t.read()
	if err != nil {
		return "", NoType, err
	}

	if c == '-' {
		w.WriteByte('-')
		c, err = t.read()
		if err != nil {
			return "", NoType, err
		}
	}

	first := c
	oldlen := w.Len()

	c, err = t.readDigits(c, &w)
	if err != nil {
		return "", NoType, err
	}

	if first == '0' {
		if w.Len()-oldlen > 1 {
			return "", NoType, &SyntaxError{"invalid leading zeroes", t.pos - 1}
		}
	}

	tt := IntType

	if c == '.' {
		w.WriteByte('.')
		tt = DecimalType

		if c, err = t.read(); err != nil {
			return "", NoType, err
		}
		if c, err = t.readDigits(c, &w); err != nil {
			return "", NoType, err
		}
	}

	switch c {
	case 'e', 'E':
		tt = FloatType

		w.WriteByte(byte(c))
		if c, err = t.readExponent(&w); err != nil {
			return "", NoType, err
		}

	case 'd', 'D':
		tt = DecimalType

		w.WriteByte(byte(c))
		if c, err = t.readExponent(&w); err != nil {
			return "", NoType, err
		}
	}

	ok, err := t.isStopChar(c)
	if err != nil {
		return "", NoType, err
	}
	if !ok {
		return "", NoType, t.invalidChar(c)
	}
	t.unread(c)

	return w.String(), tt, nil
}

func (t *tokenizer) readExponent(w io.ByteWriter) (int, error) {
	c, err := t.read()
	if err != nil {
		return 0, err
	}

	if c == '+' || c == '-' {
		err = w.WriteByte(byte(c))
		if err != nil {
			return 0, err
		}
		if c, err = t.read(); err != nil {
			return 0, err
		}
	}

	return t.readDigits(c, w)
}

func (t *tokenizer) readDigits(c int, w io.ByteWriter) (int, error) {
	if !isDigit(c) {
		return c, nil
	}
	err := w.WriteByte(byte(c))
	if err != nil {
		return 0, err
	}

	return t.readRadixDigits(isDigit, w)
}

// ReadSymbol reads an unquoted symbol value.
func (t *tokenizer) readSymbol() (string, error) {
	ret := strings.Builder{}

	c, err := t.peek()
	if err != nil {
		return "", err
	}

	for isIdentifierPart(c) {
		ret.WriteByte(byte(c))
		_, err = t.read()
		if err != nil {
			return "", err
		}
		c, err = t.peek()
		if err != nil {
			return "", err
		}
	}

	return ret.String(), nil
}

// ReadQuotedSymbol reads a quoted symbol.
func (t *tokenizer) readQuotedSymbol() (string, error) {
	ret := strings.Builder{}

	for {
		c, err := t.read()
		if err != nil {
			return "", err
		}

		switch c {
		case -1, '\n':
			return "", t.invalidChar(c)

		case '\'':
			return ret.String(), nil

		case '\\':
			c, err = t.peek()
			if err != nil {
				return "", err
			}

			if c == '\n' {
				_, err = t.read()
				if err != nil {
					return "", err
				}
				continue
			}

			r, err := t.readEscapedChar(nonClobText)
			if err != nil {
				return "", err
			}
			ret.WriteRune(r)

		default:
			ret.WriteByte(byte(c))
		}
	}
}

func (t *tokenizer) readOperator() (string, error) {
	ret := strings.Builder{}

	c, err := t.peek()
	if err != nil {
		return "", err
	}

	for isOperatorChar(c) {
		ret.WriteByte(byte(c))
		_, err = t.read()
		if err != nil {
			return "", err
		}
		c, err = t.peek()
		if err != nil {
			return "", err
		}
	}

	return ret.String(), nil
}

// ReadString reads a quoted string.
func (t *tokenizer) readString() (string, error) {
	ret := strings.Builder{}

	for {
		c, err := t.read()
		if err != nil {
			return "", err
		}
		// -1 denotes EOF, and new lines are not allowed in short string
		if c == -1 || c == '\n' || isProhibitedControlChar(c) {
			return "", t.invalidChar(c)
		}

		switch c {
		case '"':
			return ret.String(), nil

		case '\\':
			err = processBackslashInString(t, &ret)
			if err != nil {
				return "", err
			}

		default:
			ret.WriteByte(byte(c))
		}
	}
}

// ReadClob reads a quoted clob.
func (t *tokenizer) readClob() ([]byte, error) {
	var ret []byte

	for {
		c, err := t.read()
		if err != nil {
			return nil, err
		}
		// -1 denotes EOF, and new lines are not allowed in short string
		if c == -1 || c == '\n' || isProhibitedControlChar(c) || !isASCII(c) {
			return nil, t.invalidChar(c)
		}

		switch c {
		case '"':
			if ret == nil {
				// The first character is the closing " , which means an empty clob.
				return []byte{}, nil
			}
			return ret, nil

		case '\\':
			err = processBackslashInClob(t, &ret)
			if err != nil {
				return nil, err
			}

		default:
			ret = append(ret, byte(c))
		}
	}
}

// ReadLongString reads a triple-quoted string.
func (t *tokenizer) readLongString() (string, error) {
	ret := strings.Builder{}

	for {
		c, err := t.read()
		if err != nil {
			return "", err
		}
		// -1 denotes EOF
		if c == -1 || isProhibitedControlChar(c) {
			return "", t.invalidChar(c)
		}

		switch c {
		case '\'':
			isEndOfString, isConsumed, err := t.skipEndOfLongString(t.skipCommentsHandler)
			if err != nil {
				return "", err
			}
			if isEndOfString {
				return ret.String(), nil
			}
			if !isConsumed {
				// No character has been consumed. It is a single '.
				ret.WriteByte(byte(c))
			}
		case '\\':
			err = processBackslashInString(t, &ret)
			if err != nil {
				return "", err
			}

		default:
			ret.WriteByte(byte(c))
		}
	}
}

// ReadLongClob reads a triple-quoted clob.
func (t *tokenizer) readLongClob() ([]byte, error) {
	var ret []byte

	for {
		c, err := t.read()
		if err != nil {
			return nil, err
		}
		// -1 denotes EOF
		if c == -1 || isProhibitedControlChar(c) || !isASCII(c) {
			return nil, t.invalidChar(c)
		}

		switch c {
		case '\'':
			isEndOfString, isConsumed, err := t.skipEndOfLongString(t.ensureNoCommentsHandler)
			if err != nil {
				return nil, err
			}
			if isEndOfString {
				if ret == nil {
					// The first character is the closing ''' , which means an empty clob.
					return []byte{}, nil
				}
				return ret, nil
			}
			if !isConsumed {
				// No character has been consumed. It is a single '.
				ret = append(ret, byte(c))
			}
		case '\\':
			err = processBackslashInClob(t, &ret)
			if err != nil {
				return nil, err
			}

		default:
			ret = append(ret, byte(c))
		}
	}
}

// ReadEscapedChar reads an escaped character.
func (t *tokenizer) readEscapedChar(isClob bool) (rune, error) {
	// We just read the '\', grab the next char.
	c, err := t.read()
	if err != nil {
		return 0, err
	}

	switch c {
	case '0':
		return '\x00', nil
	case 'a':
		return '\a', nil
	case 'b':
		return '\b', nil
	case 't':
		return '\t', nil
	case 'n':
		return '\n', nil
	case 'f':
		return '\f', nil
	case 'r':
		return '\r', nil
	case 'v':
		return '\v', nil
	case '?':
		return '?', nil
	case '/':
		return '/', nil
	case '\'':
		return '\'', nil
	case '"':
		return '"', nil
	case '\\':
		return '\\', nil
	case 'U':
		if isClob {
			return 0, t.invalidChar('U')
		}
		return t.readHexEscapeSeq(8)
	case 'u':
		if isClob {
			return 0, t.invalidChar('u')
		}
		return t.readHexEscapeSeq(4)
	case 'x':
		return t.readHexEscapeSeq(2)
	}

	return 0, &SyntaxError{fmt.Sprintf("bad escape sequence '\\%c'", c), t.pos - 2}
}

func (t *tokenizer) readHexEscapeSeq(length int) (rune, error) {
	val := rune(0)

	for length > 0 {
		c, err := t.read()
		if err != nil {
			return 0, err
		}

		d, err := t.fromHex(c)
		if err != nil {
			return 0, err
		}

		val = (val << 4) | rune(d)
		length--
	}

	return val, nil
}

func (t *tokenizer) fromHex(c int) (int, error) {
	if c >= '0' && c <= '9' {
		return c - '0', nil
	}
	if c >= 'a' && c <= 'f' {
		return 10 + (c - 'a'), nil
	}
	if c >= 'A' && c <= 'F' {
		return 10 + (c - 'A'), nil
	}
	return 0, t.invalidChar(c)
}

func (t *tokenizer) readBinary() (string, error) {
	isB := func(c int) bool {
		return c == 'b' || c == 'B'
	}
	isDigit := func(c int) bool {
		return c == '0' || c == '1'
	}
	return t.readRadix(isB, isDigit)
}

func (t *tokenizer) readHex() (string, error) {
	isX := func(c int) bool {
		return c == 'x' || c == 'X'
	}
	return t.readRadix(isX, isHexDigit)
}

func (t *tokenizer) readRadix(isRadixMarker, isValidForRadix matcher) (string, error) {
	w := strings.Builder{}

	c, err := t.read()
	if err != nil {
		return "", err
	}

	if c == '-' {
		w.WriteByte('-')
		c, err = t.read()
		if err != nil {
			return "", err
		}
	}

	if c != '0' {
		return "", t.invalidChar(c)
	}
	w.WriteByte('0')

	c, err = t.read()
	if err != nil {
		return "", err
	}
	if !isRadixMarker(c) {
		return "", t.invalidChar(c)
	}
	w.WriteByte(byte(c))

	// At this point we have either 0x or 0b, and it cannot be followed by _
	nextChar, err2 := t.peek()
	if err2 != nil {
		return "", err
	}
	if nextChar == '_' {
		return "", t.invalidChar(c)
	}
	c, err = t.readRadixDigits(isValidForRadix, &w)
	if err != nil {
		return "", err
	}

	ok, err := t.isStopChar(c)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", t.invalidChar(c)
	}
	t.unread(c)

	return w.String(), nil
}

func (t *tokenizer) readRadixDigits(isValidForRadix matcher, w io.ByteWriter) (int, error) {
	var c int
	var err error

	for {
		c, err = t.read()
		if err != nil {
			return 0, err
		}
		if c == '_' {
			nextChar, err := t.peek()
			if err != nil {
				return 0, err
			}
			if !isValidForRadix(nextChar) {
				return 0, t.invalidChar(c)
			}
			continue
		}
		if !isValidForRadix(c) {
			return c, nil
		}
		err := w.WriteByte(byte(c))
		if err != nil {
			return 0, err
		}
	}
}

func (t *tokenizer) readTimestamp() (string, error) {
	w := strings.Builder{}

	c, err := t.readTimestampDigits(4, &w)
	if err != nil {
		return "", err
	}
	if c == 'T' {
		// yyyyT
		w.WriteByte('T')
		return w.String(), nil
	}
	if c != '-' {
		return "", t.invalidChar(c)
	}
	w.WriteByte('-')

	if c, err = t.readTimestampDigits(2, &w); err != nil {
		return "", err
	}
	if c == 'T' {
		// yyyy-mmT
		w.WriteByte('T')
		return w.String(), nil
	}
	if c != '-' {
		return "", t.invalidChar(c)
	}
	w.WriteByte('-')

	if c, err = t.readTimestampDigits(2, &w); err != nil {
		return "", err
	}
	if c != 'T' {
		// yyyy-mm-dd
		return t.readTimestampFinish(c, &w)
	}
	w.WriteByte('T')

	if c, err = t.read(); err != nil {
		return "", err
	}
	if !isDigit(c) {
		// yyyy-mm-ddT(+hh:mm)?
		if c, err = t.readTimestampOffset(c, &w); err != nil {
			return "", err
		}
		return t.readTimestampFinish(c, &w)
	}
	w.WriteByte(byte(c))

	if c, err = t.readTimestampDigits(1, &w); err != nil {
		return "", err
	}
	if c != ':' {
		return "", t.invalidChar(c)
	}
	w.WriteByte(':')

	if c, err = t.readTimestampDigits(2, &w); err != nil {
		return "", err
	}
	if c != ':' {
		// yyyy-mm-ddThh:mmZ
		if c, err = t.readTimestampOffsetOrZ(c, &w); err != nil {
			return "", err
		}
		return t.readTimestampFinish(c, &w)
	}
	w.WriteByte(':')

	if c, err = t.readTimestampDigits(2, &w); err != nil {
		return "", err
	}
	if c != '.' {
		// yyyy-mm-ddThh:mm:ssZ
		if c, err = t.readTimestampOffsetOrZ(c, &w); err != nil {
			return "", err
		}
		return t.readTimestampFinish(c, &w)
	}
	w.WriteByte('.')

	// yyyy-mm-ddThh:mm:ss.ssssZ
	if c, err = t.read(); err != nil {
		return "", err
	}
	if isDigit(c) {
		if c, err = t.readDigits(c, &w); err != nil {
			return "", err
		}
	}

	if c, err = t.readTimestampOffsetOrZ(c, &w); err != nil {
		return "", err
	}
	return t.readTimestampFinish(c, &w)
}

func (t *tokenizer) readTimestampOffsetOrZ(c int, w io.ByteWriter) (int, error) {
	if c == '-' || c == '+' {
		return t.readTimestampOffset(c, w)
	}
	if c == 'z' || c == 'Z' {
		err := w.WriteByte(byte(c))
		if err != nil {
			return 0, err
		}
		return t.read()
	}
	return 0, t.invalidChar(c)
}

func (t *tokenizer) readTimestampOffset(c int, w io.ByteWriter) (int, error) {
	if c != '-' && c != '+' {
		return c, nil
	}
	err := w.WriteByte(byte(c))
	if err != nil {
		return 0, err
	}

	c, err = t.readTimestampDigits(2, w)
	if err != nil {
		return 0, err
	}
	if c != ':' {
		return 0, t.invalidChar(c)
	}
	err = w.WriteByte(':')
	if err != nil {
		return 0, err
	}
	return t.readTimestampDigits(2, w)
}

func (t *tokenizer) readTimestampDigits(n int, w io.ByteWriter) (int, error) {
	for n > 0 {
		c, err := t.read()
		if err != nil {
			return 0, err
		}
		if !isDigit(c) {
			return 0, t.invalidChar(c)
		}
		err = w.WriteByte(byte(c))
		if err != nil {
			return 0, err
		}
		n--
	}
	return t.read()
}

func (t *tokenizer) readTimestampFinish(c int, w fmt.Stringer) (string, error) {
	ok, err := t.isStopChar(c)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", t.invalidChar(c)
	}
	t.unread(c)
	return w.String(), nil
}

func (t *tokenizer) ReadBlob() (string, error) {
	w := strings.Builder{}

	var (
		c   int
		err error
	)

	for {
		if c, _, err = t.skipLobWhitespace(); err != nil {
			return "", err
		}
		if c == -1 {
			return "", t.invalidChar(c)
		}
		if c == '}' {
			break
		}
		w.WriteByte(byte(c))
	}

	if c, err = t.read(); err != nil {
		return "", err
	}
	if c != '}' {
		return "", t.invalidChar(c)
	}

	t.unfinished = false
	return w.String(), nil
}

func (t *tokenizer) ReadShortClob() ([]byte, error) {
	val, err := t.readClob()
	if err != nil {
		return nil, err
	}

	c, _, err := t.skipLobWhitespace()
	if err != nil {
		return nil, err
	}
	if c != '}' {
		return nil, t.invalidChar(c)
	}

	if c, err = t.read(); err != nil {
		return nil, err
	}
	if c != '}' {
		return nil, t.invalidChar(c)
	}

	t.unfinished = false
	return val, nil
}

func (t *tokenizer) ReadLongClob() ([]byte, error) {
	val, err := t.readLongClob()
	if err != nil {
		return nil, err
	}

	c, _, err := t.skipLobWhitespace()
	if err != nil {
		return nil, err
	}
	if c != '}' {
		return nil, t.invalidChar(c)
	}

	if c, err = t.read(); err != nil {
		return nil, err
	}
	if c != '}' {
		return nil, t.invalidChar(c)
	}

	t.unfinished = false
	return val, nil
}

// IsTripleQuote returns true if this is a triple-quote sequence; i.e.:
//
//	'''
func (t *tokenizer) IsTripleQuote() (bool, error) {
	// We've just read a '\'', check if the next two are too.
	cs, err := t.peekN(2)
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if cs[0] == '\'' && cs[1] == '\'' {
		err = t.skipN(2)
		if err != nil {
			return false, err
		}
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
		err = t.skipN(3)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	if cs[3] == '/' && len(cs) > 4 && (cs[4] == '/' || cs[4] == '*') {
		err = t.skipN(3)
		if err != nil {
			return false, err
		}
		// +-inf followed immediately by a comment works too.
		return true, nil
	}

	return false, nil
}

// ScanForNumericType attempts to determine what type of number we
// have by peeking at a finite number of characters. We can rule
// out binary (0b...), hex (0x...), and timestamps (....-) via this
// method. There are a couple other cases where we *could* distinguish,
// but it's unclear that it's worth it.
func (t *tokenizer) scanForNumericType(c int) (token, error) {
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
	return tokenNumber, nil
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
		return t.invalidChar(c)
	}
	return nil
}

// InvalidChar returns an error complaining that the given character was
// unexpected.
func (t *tokenizer) invalidChar(c int) error {
	if c == -1 {
		return &UnexpectedEOFError{t.pos - 1}
	}
	return &UnexpectedRuneError{rune(c), t.pos - 1}
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
	t.pos++
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
		return 0, &IOError{err}
	}

	// Normalize \r and \r\n to just \n.
	if c == '\r' {
		cs, err := t.in.Peek(1)
		if err != nil && err != io.EOF {
			// Not EOF, because we haven't dealt with the '\r' yet.
			return 0, &IOError{err}
		}
		if len(cs) > 0 && cs[0] == '\n' {
			// Skip over the '\n' as well.
			_, err = t.in.ReadByte()
			if err != nil {
				return 0, err
			}
		}
		return '\n', nil
	}

	return int(c), nil
}

// Unread pushes a character (or -1) back into the input stream to
// be read again later.
func (t *tokenizer) unread(c int) {
	t.pos--
	t.buffer = append(t.buffer, c)
}

func isProhibitedControlChar(c int) bool {
	// Values between 0 to 31 are non-displayable ASCII characters; except for new line and white space characters.
	if c < 0x00 || c > 0x1F {
		return false
	}
	if isStringWhitespace(c) || isNewLineChar(c) {
		return false
	}
	return true
}

func isStringWhitespace(c int) bool {
	return c == 0x09 || // horizontal tab
		c == 0x0B || // vertical tab
		c == 0x0C // form feed
}

func isNewLineChar(c int) bool {
	return c == 0x0A || // new line
		c == 0x0D // carriage return
}

// isASCII returns true if c is a 7-bit ASCII character.
func isASCII(c int) bool {
	return c < 0x80
}

func processBackslashInString(t *tokenizer, sb *strings.Builder) error {
	c, err := t.peek()
	if err != nil {
		return err
	}

	if c == '\n' {
		_, err = t.read()
		if err != nil {
			return err
		}
		return nil
	}

	r, err := t.readEscapedChar(nonClobText)
	if err != nil {
		return err
	}
	sb.WriteRune(r)
	return nil
}

func processBackslashInClob(t *tokenizer, ret *[]byte) error {
	c, err := t.peek()
	if err != nil {
		return err
	}

	if c == '\n' {
		_, err = t.read()
		if err != nil {
			return err
		}
		return nil
	}

	r, err := t.readEscapedChar(clobText)
	if err != nil {
		return err
	}
	*ret = append(*ret, byte(r))
	return nil
}
