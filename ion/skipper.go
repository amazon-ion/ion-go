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
	"fmt"
	"io"
)

// SkipContainerContents skips over the contents of a container of the given type.
func (t *tokenizer) SkipContainerContents(typ Type) error {
	switch typ {
	case StructType:
		return t.skipStructHelper()
	case ListType:
		return t.skipListHelper()
	case SexpType:
		return t.skipSexpHelper()
	default:
		panic(fmt.Sprintf("invalid container type: %v", typ))
	}
}

// Skips whitespace and a double-colon token, if there is one.
func (t *tokenizer) SkipDoubleColon() (bool, bool, error) {
	ws, err := t.skipWhitespaceHelper()
	if err != nil {
		return false, false, err
	}

	ok, err := t.skipDoubleColon()
	if err != nil {
		return false, false, err
	}

	return ok, ws, nil
}

// Peeks ahead to see if the next token is a dot, and
// if so skips it. If not, leaves the next token unconsumed.
func (t *tokenizer) SkipDot() (bool, error) {
	c, err := t.peek()
	if err != nil {
		return false, err
	}
	if c != '.' {
		return false, nil
	}

	_, err = t.read()
	if err != nil {
		return false, err
	}
	return true, nil
}

// SkipLobWhitespace skips whitespace when we're inside a blob
// or clob where comments are not allowed.
func (t *tokenizer) SkipLobWhitespace() (int, error) {
	c, _, err := t.skipLobWhitespace()
	return c, err
}

// SkipValue skips to the end of the current value, if the caller
// didn't bother to consume it before calling Next again.
func (t *tokenizer) skipValue() (int, error) {
	var c int
	var err error

	switch t.token {
	case tokenNumber:
		c, err = t.skipNumber()
	case tokenBinary:
		c, err = t.skipBinary()
	case tokenHex:
		c, err = t.skipHex()
	case tokenTimestamp:
		c, err = t.skipTimestamp()
	case tokenSymbol:
		c, err = t.skipSymbol()
	case tokenSymbolQuoted:
		c, err = t.skipSymbolQuoted()
	case tokenSymbolOperator:
		c, err = t.skipSymbolOperator()
	case tokenString:
		c, err = t.skipString()
	case tokenLongString:
		c, err = t.skipLongString()
	case tokenOpenDoubleBrace:
		c, err = t.skipBlob()
	case tokenOpenBrace:
		c, err = t.skipStruct()
	case tokenOpenParen:
		c, err = t.skipSexp()
	case tokenOpenBracket:
		c, err = t.skipList()
	default:
		panic(fmt.Sprintf("skipValue called with token=%v", t.token))
	}

	if err != nil {
		return 0, err
	}

	if isWhitespace(c) {
		c, _, err = t.skipWhitespace()
		if err != nil {
			return 0, err
		}
	}

	t.unfinished = false
	return c, nil
}

// SkipNumber skips a (non-binary, non-hex) number.
func (t *tokenizer) skipNumber() (int, error) {
	c, err := t.read()
	if err != nil {
		return 0, err
	}

	if c == '-' {
		c, err = t.read()
		if err != nil {
			return 0, err
		}
	}

	c, err = t.skipDigits(c)
	if err != nil {
		return 0, err
	}

	if c == '.' {
		c, err = t.read()
		if err != nil {
			return 0, err
		}
		c, err = t.skipDigits(c)
		if err != nil {
			return 0, err
		}
	}

	if c == 'd' || c == 'D' || c == 'e' || c == 'E' {
		c, err = t.read()
		if err != nil {
			return 0, err
		}
		if c == '+' || c == '-' {
			c, err = t.read()
			if err != nil {
				return 0, err
			}
		}
		c, err = t.skipDigits(c)
		if err != nil {
			return 0, err
		}
	}

	ok, err := t.isStopChar(c)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, t.invalidChar(c)
	}
	return c, nil
}

// SkipBinary skips a binary literal value.
func (t *tokenizer) skipBinary() (int, error) {
	isB := func(c int) bool {
		return c == 'b' || c == 'B'
	}
	isBinaryDigit := func(c int) bool {
		return c == '0' || c == '1'
	}
	return t.skipRadix(isB, isBinaryDigit)
}

// SkipHex skips a hex value.
func (t *tokenizer) skipHex() (int, error) {
	isX := func(c int) bool {
		return c == 'x' || c == 'X'
	}
	return t.skipRadix(isX, isHexDigit)
}

func (t *tokenizer) skipRadix(isRadixMarker, isValidForRadix matcher) (int, error) {
	c, err := t.read()
	if err != nil {
		return 0, err
	}

	if c == '-' {
		c, err = t.read()
		if err != nil {
			return 0, err
		}
	}

	if c != '0' {
		return 0, t.invalidChar(c)
	}
	if err = t.expect(isRadixMarker); err != nil {
		return 0, err
	}

	for {
		c, err = t.read()
		if err != nil {
			return 0, err
		}
		if !isValidForRadix(c) {
			break
		}
	}

	ok, err := t.isStopChar(c)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, t.invalidChar(c)
	}

	return c, nil
}

// SkipTimestamp skips a timestamp value, returning the next character.
func (t *tokenizer) skipTimestamp() (int, error) {
	// Read the first four digits, yyyy.
	c, err := t.skipTimestampDigits(4)
	if err != nil {
		return 0, err
	}
	if c == 'T' {
		// yyyyT
		return t.read()
	}
	if c != '-' {
		return 0, t.invalidChar(c)
	}

	// Read the next two, yyyy-mm.
	c, err = t.skipTimestampDigits(2)
	if err != nil {
		return 0, err
	}
	if c == 'T' {
		// yyyy-mmT
		return t.read()
	}
	if c != '-' {
		return 0, t.invalidChar(c)
	}

	// Read the day.
	c, err = t.skipTimestampDigits(2)
	if err != nil {
		return 0, err
	}
	if c != 'T' {
		// yyyy-mm-dd.
		return t.skipTimestampFinish(c)
	}

	c, err = t.read()
	if err != nil {
		return 0, err
	}
	if !isDigit(c) {
		// yyyy-mm-ddT(+hh:mm)?
		c, err = t.skipTimestampOffset(c)
		if err != nil {
			return 0, err
		}
		return t.skipTimestampFinish(c)
	}

	// Already read the first hour digit above.
	c, err = t.skipTimestampDigits(1)
	if err != nil {
		return 0, err
	}
	if c != ':' {
		return 0, t.invalidChar(c)
	}

	c, err = t.skipTimestampDigits(2)
	if err != nil {
		return 0, err
	}
	if c != ':' {
		// yyyy-mm-ddThh:mmZ
		c, err = t.skipTimestampOffsetOrZ(c)
		if err != nil {
			return 0, err
		}
		return t.skipTimestampFinish(c)
	}

	c, err = t.skipTimestampDigits(2)
	if err != nil {
		return 0, err
	}
	if c != '.' {
		// yyyy-mm-ddThh:mm:ssZ
		c, err = t.skipTimestampOffsetOrZ(c)
		if err != nil {
			return 0, err
		}
		return t.skipTimestampFinish(c)
	}

	// yyyy-mm-ddThh:mm:ss.ssssZ
	c, err = t.read()
	if err != nil {
		return 0, err
	}
	if isDigit(c) {
		c, err = t.skipDigits(c)
		if err != nil {
			return 0, err
		}
	}

	c, err = t.skipTimestampOffsetOrZ(c)
	if err != nil {
		return 0, err
	}
	return t.skipTimestampFinish(c)
}

// SkipTimestampOffsetOrZ skips a (required) timestamp offset value or
// letter 'Z' (indicating UTC).
func (t *tokenizer) skipTimestampOffsetOrZ(c int) (int, error) {
	if c == '-' || c == '+' {
		return t.skipTimestampOffset(c)
	}
	if c == 'z' || c == 'Z' {
		return t.read()
	}
	return 0, t.invalidChar(c)
}

// SkipTimestampOffset skips an (optional) +-hh:mm timestamp zone offset
// value.
func (t *tokenizer) skipTimestampOffset(c int) (int, error) {
	if c != '-' && c != '+' {
		return c, nil
	}

	c, err := t.skipTimestampDigits(2)
	if err != nil {
		return 0, err
	}
	if c != ':' {
		return 0, t.invalidChar(c)
	}
	return t.skipTimestampDigits(2)
}

// SkipTimestampDigits skips a bounded sequence of digits inside a
// timestamp.
func (t *tokenizer) skipTimestampDigits(n int) (int, error) {
	for n > 0 {
		if err := t.expect(func(c int) bool {
			return isDigit(c)
		}); err != nil {
			return 0, err
		}
		n--
	}

	return t.read()
}

// SkipTimestampFinish makes sure the character after a timestamp
// value is a valid ending point. If so, it returns it.
func (t *tokenizer) skipTimestampFinish(c int) (int, error) {
	ok, err := t.isStopChar(c)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, t.invalidChar(c)
	}
	return c, nil
}

// SkipSymbol skips a normal symbol and returns the next character.
func (t *tokenizer) skipSymbol() (int, error) {
	c, err := t.read()
	if err != nil {
		return 0, err
	}

	for isIdentifierPart(c) {
		c, err = t.read()
		if err != nil {
			return 0, err
		}
	}

	return c, nil
}

// SkipSymbolQuoted skips a quoted symbol and returns the next char.
func (t *tokenizer) skipSymbolQuoted() (int, error) {
	if err := t.skipSymbolQuotedHelper(); err != nil {
		return 0, err
	}
	return t.read()
}

// SkipSymbolQuotedHelper skips a quoted symbol.
func (t *tokenizer) skipSymbolQuotedHelper() error {
	for {
		c, err := t.read()
		if err != nil {
			return err
		}

		switch c {
		case -1, '\n':
			return t.invalidChar(c)

		case '\'':
			return nil

		case '\\':
			if _, err := t.read(); err != nil {
				return err
			}
		}
	}
}

// SkipSymbolOperator skips an operator-style symbol inside an sexp.
func (t *tokenizer) skipSymbolOperator() (int, error) {
	c, err := t.read()
	if err != nil {
		return 0, err
	}

	for isOperatorChar(c) {
		c, err = t.read()
		if err != nil {
			return 0, err
		}
	}

	return c, nil
}

// SkipString skips over a "-enclosed string, returning the next char.
func (t *tokenizer) skipString() (int, error) {
	if err := t.skipStringHelper(); err != nil {
		return 0, err
	}
	return t.read()
}

// SkipStringHelper skips over a "-enclosed string.
func (t *tokenizer) skipStringHelper() error {
	for {
		c, err := t.read()
		if err != nil {
			return err
		}

		switch c {
		case -1, '\n':
			return t.invalidChar(c)

		case '"':
			return nil

		case '\\':
			if _, err := t.read(); err != nil {
				return err
			}
		}
	}
}

// SkipLongString skips over a triple-quote-enclosed string, returning the next
// character after the closing triple-quote.
func (t *tokenizer) skipLongString() (int, error) {
	if err := t.skipLongStringHelper(t.skipCommentsHandler); err != nil {
		return 0, err
	}
	return t.read()
}

// SkipLongStringHelper skips over a triple-quote-enclosed string.
func (t *tokenizer) skipLongStringHelper(handler commentHandler) error {
	for {
		c, err := t.read()
		if err != nil {
			return err
		}

		switch c {
		case -1:
			return t.invalidChar(c)

		case '\'':
			ok, _, err := t.skipEndOfLongString(handler)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}

		case '\\':
			if _, err = t.read(); err != nil {
				return err
			}
		}
	}
}

// SkipEndOfLongString is called after reading a ' to determine if we've hit the end
// of the long string, and if we have consumed any ' characters. Also, it can detect
// if another long string starts after the current one; in that case, it returns
// false indicating this is not the end of the long string, and true for consumed '
// as we have read the closing triple-quote of the first long string.
func (t *tokenizer) skipEndOfLongString(handler commentHandler) (bool, bool, error) {
	isConsumed := false
	// We just read a ', check for two more ''s.
	cs, err := t.peekN(2)
	if err != nil && err != io.EOF {
		return false, isConsumed, err
	}

	// If it's not a triple-quote, keep going.
	if len(cs) < 2 || cs[0] != '\'' || cs[1] != '\'' {
		return false, isConsumed, nil
	}

	// Consume the triple-quote.
	err = t.skipN(2)
	isConsumed = true
	if err != nil {
		return false, isConsumed, err
	}

	// Consume any additional whitespace/comments.
	c, _, err := t.skipWhitespaceWith(handler)
	if err != nil {
		return false, isConsumed, err
	}

	// Check if it's another triple-quote; if so, keep going.
	if c == '\'' {
		ok, err := t.IsTripleQuote()
		if err != nil {
			return false, isConsumed, err
		}
		if ok {
			return false, isConsumed, nil
		}
	}

	t.unread(c)
	return true, isConsumed, nil
}

// SkipBlob skips over a blob value, returning the next character.
func (t *tokenizer) skipBlob() (int, error) {
	if err := t.skipBlobHelper(); err != nil {
		return 0, err
	}
	return t.read()
}

// SkipBlobHelper skips over a blob value, stopping after reading the
// final '}'.
func (t *tokenizer) skipBlobHelper() error {
	c, _, err := t.skipLobWhitespace()
	if err != nil {
		return err
	}

	// https://github.com/amazon-ion/ion-go/issues/115
	for c != '}' {
		c, _, err = t.skipLobWhitespace()
		if err != nil {
			return err
		}
		if c == -1 {
			return t.invalidChar(c)
		}
	}

	return t.expect(func(c int) bool {
		return c == '}'
	})
}

func (t *tokenizer) skipStruct() (int, error) {
	return t.skipContainer('}')
}

func (t *tokenizer) skipStructHelper() error {
	return t.skipContainerHelper('}')
}

func (t *tokenizer) skipSexp() (int, error) {
	return t.skipContainer(')')
}

func (t *tokenizer) skipSexpHelper() error {
	return t.skipContainerHelper(')')
}

// SkipList skips forward past a list that the caller doesn't care to
// step in to.
func (t *tokenizer) skipList() (int, error) {
	return t.skipContainer(']')
}

func (t *tokenizer) skipListHelper() error {
	return t.skipContainerHelper(']')
}

// SkipContainer skips a container terminated by the given char and
// returns the next character.
func (t *tokenizer) skipContainer(term int) (int, error) {
	if err := t.skipContainerHelper(term); err != nil {
		return 0, err
	}
	return t.read()
}

// SkipContainerHelper skips over a container terminated by the given
// char.
func (t *tokenizer) skipContainerHelper(term int) error {
	if term != ']' && term != ')' && term != '}' {
		panic(fmt.Sprintf("unexpected character: %q. Expected one of the closing container characters: ] } )", term))
	}

	for {
		c, _, err := t.skipWhitespace()
		if err != nil {
			return err
		}

		switch c {
		case -1:
			return t.invalidChar(c)

		case term:
			return nil

		case '"':
			if err := t.skipStringHelper(); err != nil {
				return err
			}

		case '\'':
			ok, err := t.IsTripleQuote()
			if err != nil {
				return err
			}
			if ok {
				if err = t.skipLongStringHelper(t.skipCommentsHandler); err != nil {
					return err
				}
			} else {
				if err = t.skipSymbolQuotedHelper(); err != nil {
					return err
				}
			}

		case '(':
			if err := t.skipContainerHelper(')'); err != nil {
				return err
			}

		case '[':
			if err := t.skipContainerHelper(']'); err != nil {
				return err
			}

		case '{':
			c, err := t.peek()
			if err != nil {
				return err
			}

			if c == '{' {
				if _, err := t.read(); err != nil {
					return err
				}
				if err := t.skipBlobHelper(); err != nil {
					return err
				}
			} else if c == '}' {
				if _, err := t.read(); err != nil {
					return err
				}
			} else {
				if err := t.skipContainerHelper('}'); err != nil {
					return err
				}
			}
		}
	}
}

// SkipDigits skips a sequence of digits starting with the
// given character.
func (t *tokenizer) skipDigits(c int) (int, error) {
	var err error
	for err == nil && isDigit(c) {
		c, err = t.read()
	}
	return c, err
}

// SkipWhitespace skips whitespace (and comments) when we're out
// in normal parsing territory.
func (t *tokenizer) skipWhitespace() (int, bool, error) {
	return t.skipWhitespaceWith(t.skipCommentsHandler)
}

// SkipWhitespaceHelper is a 'helper' form of SkipWhitespace that
// unreads the first non-whitespace char instead of returning it.
func (t *tokenizer) skipWhitespaceHelper() (bool, error) {
	c, ok, err := t.skipWhitespace()
	if err != nil {
		return false, err
	}
	t.unread(c)
	return ok, err
}

// SkipLobWhitespace skips whitespace when we're inside a blob
// or clob, where comments are not allowed.
func (t *tokenizer) skipLobWhitespace() (int, bool, error) {
	// Comments are not allowed inside a lob value; if we see a '/',
	// it's the start of a base64-encoded value.
	return t.skipWhitespaceWith(stopForCommentsHandler)
}

// CommentHandler is a strategy for handling comments. Returns true
// if it found and handled a comment, false if it didn't find a
// comment, and returns an error if it choked on the comment.
type commentHandler func() (bool, error)

// SkipWhitespaceWith skips whitespace using the given strategy for
// handling comments--generally speaking, either skipping over them
// using skipCommentsHandler, or stopping with a stopForCommentsHandler.
// Returns the first non-whitespace character it reads, and whether it
// actually skipped anything to find it.
func (t *tokenizer) skipWhitespaceWith(handler commentHandler) (int, bool, error) {
	skipped := false
	for {
		c, err := t.read()
		if err != nil {
			return 0, skipped, err
		}

		switch c {
		case ' ', '\t', '\n', '\r':
			// Skipped.

		case '/':
			comment, err := handler()
			if err != nil {
				return 0, skipped, err
			}
			if !comment {
				return '/', skipped, nil
			}

		default:
			return c, skipped, nil
		}
		skipped = true
	}
}

// StopForCommentsHandler is a commentHandler that stops skipping
// whitespace when it finds a (potential) comment. Use it when you
// expect a '/' to be an actual '/', not a comment.
func stopForCommentsHandler() (bool, error) {
	return false, nil
}

// ensureNoCommentsHandler is a commentHandler that returns an
// error if any comments are found, else no error is returned.
func (t *tokenizer) ensureNoCommentsHandler() (bool, error) {
	return false, &UnexpectedTokenError{"comments are not allowed within a clob", t.Pos() - 1}
}

// SkipCommentsHandler is a commentHandler that skips over any
// comments it finds.
func (t *tokenizer) skipCommentsHandler() (bool, error) {
	// We've just read a '/', which might be the start of a comment.
	// Peek ahead to see if it is, and if so skip over it.
	c, err := t.peek()
	if err != nil {
		return false, err
	}

	switch c {
	case '/':
		return true, t.skipSingleLineComment()
	case '*':
		return true, t.skipBlockComment()
	default:
		return false, nil
	}
}

// SkipSingleLineComment skips over the body of a single-line comment,
// terminated by the end of the line (or file).
func (t *tokenizer) skipSingleLineComment() error {
	for {
		c, err := t.read()
		if err != nil {
			return err
		}

		if c == -1 || c == '\n' {
			return nil
		}
	}
}

// SkipBlockComment skips over the body of a block comment, terminated
// by a '*/' sequence.
func (t *tokenizer) skipBlockComment() error {
	star := false
	for {
		c, err := t.read()
		if err != nil {
			return err
		}
		if c == -1 {
			return t.invalidChar(c)
		}

		if star && c == '/' {
			return nil
		}

		star = c == '*'
	}
}

// Peeks ahead to see if the next token is a double colon, and
// if so skips it. If not, leaves the next token unconsumed.
func (t *tokenizer) skipDoubleColon() (bool, error) {
	cs, err := t.peekN(2)
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if cs[0] == ':' && cs[1] == ':' {
		err = t.skipN(2)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}
