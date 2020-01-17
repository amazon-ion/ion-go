/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

package lex

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	eof = -1

	// Dot is not included and must be checked individually.
	operatorRunes = "!#%&*+-/;<=>?@^`|~"

	// \v is a vertical tab
	whitespaceRunes = " \t\n\r\f\v"
)

// The state of the scanner as a function that returns the next state.
type stateFn func(*Lexer) stateFn

// Lexer represents the state of scanning the input text.
type Lexer struct {
	input      []byte    // the data being scanned
	state      stateFn   // the next lexing function to enter
	pos        int       // current position in the input
	itemStart  int       // start position of the current item
	width      int       // width of last rune read from input
	lastPos    int       // position of most recent item returned by NextItem
	items      chan Item // channel of scanned items
	containers []byte    // keep track of container starts and ends
}

// New creates a new scanner for the input data.  This is the lexing half of the
// Lexer / parser.  Basic validation is done in the Lexer for a loose sense of
// correctness, but the rigid correctness is enforced in the parser.
func New(input []byte) *Lexer {
	x := &Lexer{
		input: input,
		items: make(chan Item),
	}
	go x.run()
	return x
}

// NextItem returns the next item from the input.
func (x *Lexer) NextItem() Item {
	item := <-x.items
	x.lastPos = item.Pos
	return item
}

// LineNumber returns the line number that the Lexer last stopped at.
func (x *Lexer) LineNumber() int {
	// Count the number of newlines, then add 1 for the line we're currently on.
	return bytes.Count(x.input[:x.lastPos], []byte("\n")) + 1
}

// run the state machine for the Lexer.
func (x *Lexer) run() {
	for x.state = lexValue; x.state != nil; {
		x.state = x.state(x)
	}
}

// next returns the next rune in the input.  If there is a problem decoding
// the rune, then utf8.RuneError is returned.
func (x *Lexer) next() rune {
	if x.pos >= len(x.input) {
		x.width = 0
		return eof
	}
	r, w := utf8.DecodeRune(x.input[x.pos:])
	x.width = w
	x.pos += x.width
	return r
}

// peek returns, but does not consume, the next rune from the input.
func (x *Lexer) peek() rune {
	if x.pos >= len(x.input) {
		return eof
	}
	r, _ := utf8.DecodeRune(x.input[x.pos:])
	return r
}

// backup steps back one rune. Can only be called once per call of next().
func (x *Lexer) backup() {
	x.pos -= x.width
}

// emit sends an item representing the current Lexer state and the given type
// onto the items channel.
func (x *Lexer) emit(it itemType) {
	x.items <- Item{Type: it, Pos: x.itemStart, Val: x.input[x.itemStart:x.pos]}
	x.itemStart = x.pos
}

// ignore sets the itemStart point to the current position, thereby "ignoring" any
// input between the two points.
func (x *Lexer) ignore() {
	x.itemStart = x.pos
}

// emitAndIgnoreTripleQuoteEnd backs up three spots (a triple quote), emits the given
// itemType, then goes forward three spots to ignore the triple quote
func (x *Lexer) emitAndIgnoreTripleQuoteEnd(itemType itemType) {
	x.width = 1
	x.backup()
	x.backup()
	x.backup()
	x.emit(itemType)

	x.next()
	x.next()
	x.next()
	x.ignore()
}

// errorf emits an error token and returns nil to stop lexing.
func (x *Lexer) errorf(format string, args ...interface{}) stateFn {
	x.items <- Item{Type: IonError, Pos: x.itemStart, Val: []byte(fmt.Sprintf(format, args...))}
	return nil
}

// error emits an error token and returns nil to stop lexing.
func (x *Lexer) error(message string) stateFn {
	x.items <- Item{Type: IonError, Pos: x.itemStart, Val: []byte(message)}
	return nil
}

// lexValue scans for a value, which can be an annotation, number, symbol, list,
// struct, or s-expression.
func lexValue(x *Lexer) stateFn {
	switch ch := x.next(); {
	case ch == eof:
		x.emit(IonEOF)
		return nil
	case isWhitespace(ch):
		x.ignore()
		return lexValue
	case ch == ':':
		return lexColons
	case ch == '\'':
		return lexSingleQuote
	case ch == '"':
		return lexString
	case ch == ',':
		x.emit(IonComma)
		return lexValue
	case ch == '[':
		return lexList
	case ch == ']':
		return lexListEnd
	case ch == '(':
		return lexSExp
	case ch == ')':
		return lexSExpEnd
	case ch == '{':
		if x.peek() == '{' {
			x.next()
			return lexBinary
		}
		return lexStruct
	case ch == '}':
		return lexStructEnd
	case ch == '/':
		// Comment handling needs to come before operator handling because the
		// start of a comment is also an operator.  Treat it as an operator if
		// the following character doesn't adhere to one of the comment standards.
		switch x.peek() {
		case '/':
			x.next()
			return lexLineComment
		case '*':
			x.next()
			return lexBlockComment
		}
		x.emit(IonOperator)
		return lexValue
	case isOperator(ch) || ch == '.':
		// - is both an operator and a signal that a number is starting.  Since
		// infinity is represented as +inf or -inf, we need to take that into
		// account as well.
		if (ch == '+' && x.peek() == 'i') || (ch == '-' && (isNumber(x.peek()) || x.peek() == 'i' || x.peek() == '_')) {
			x.backup()
			return lexNumber
		}
		// An operator can consist of multiple characters.
		for next := x.peek(); isOperator(next); next = x.peek() {
			x.next()
		}
		x.emit(IonOperator)
		return lexValue
	case isIdentifierSymbolStart(ch):
		x.backup()
		return lexSymbol
	case isNumericStart(ch):
		x.backup()
		return lexNumber
	default:
		return x.errorf("invalid start of a value: %#U", ch)
	}
}

// lexColons expects one colon to be scanned and checks to see if there is
// a second before emitting.  Returns lexValue.
func lexColons(x *Lexer) stateFn {
	if x.peek() == ':' {
		x.next()
		x.emit(IonDoubleColon)
	} else {
		x.emit(IonColon)
	}

	return lexValue
}

// lexLineComment scans a comment while parsing values. The comment is
// terminated by a newline.  lexValue is returned.
func lexLineComment(x *Lexer) stateFn {
	// Ignore the preceding "//" characters.
	x.ignore()
	for {
		ch := x.next()
		if ch == utf8.RuneError {
			return x.error("error parsing rune")
		}
		if isEndOfLine(ch) || ch == eof {
			x.backup()
			break
		}
	}
	x.emit(IonCommentLine)
	return lexValue
}

// lexBlockComment scans a block comment. The comment is terminated by */
// lexTopLevel is returned since we don't know what is going to come next.
func lexBlockComment(x *Lexer) stateFn {
	// Ignore the preceding "/*" characters.
	x.ignore()
	for {
		ch := x.next()
		if ch == eof {
			return x.error("unexpected end of file while lexing block comment")
		}
		if ch == utf8.RuneError {
			return x.error("error parsing rune")
		}
		if ch == '*' && x.peek() == '/' {
			x.backup()
			break
		}
	}

	x.emit(IonCommentBlock)
	// Ignore the trailing "*/" characters.
	x.next()
	x.next()
	x.ignore()

	return lexValue
}

// eatWhitespace eats up all of the text until a non-whitespace character is encountered.
func eatWhitespace(x *Lexer) {
	for isWhitespace(x.peek()) {
		x.next()
	}
	x.ignore()
}

// isWhitespace returns if the given rune is considered to be a form of whitespace.
func isWhitespace(ch rune) bool {
	return bytes.ContainsRune([]byte(whitespaceRunes), ch)
}

// isEndOfLine returns true if the given rune is an end-of-line character.
func isEndOfLine(ch rune) bool {
	return ch == '\r' || ch == '\n'
}

// isOperator returns true if the given rune is one of the operator chars (not including dot).
func isOperator(ch rune) bool {
	return bytes.ContainsRune([]byte(operatorRunes), ch)
}

// accept consumes the next rune if it's from the given set of valid runes.
func (x *Lexer) accept(valid string) bool {
	if strings.IndexRune(valid, x.peek()) >= 0 {
		x.next()
		return true
	}
	return false
}

// acceptString consumes the as many runes from the given string as possible.
// If it hits a rune it can't accept, then it backs up and returns false.
func (x *Lexer) acceptString(valid string) bool {
	for _, ch := range valid {
		if x.peek() != ch {
			return false
		}
		x.next()
	}
	return true
}

// acceptRun consumes as many runes as possible from the given set set of valid runes.
// Stops at either an unacceptable rune, EOF, or if any of the noRepeat runes are encountered
// twice consecutively.
func (x *Lexer) acceptRun(valid string, noRepeat string) int {
	inRepeat := false
	count := 0
	// Use peek so that we can still back up if the rune we fail on is EOF.
	for ch := x.peek(); strings.IndexRune(valid, ch) >= 0; ch = x.peek() {
		x.next()
		count++
		isRepeatRune := strings.IndexRune(noRepeat, ch) >= 0
		if isRepeatRune && inRepeat {
			break
		}
		inRepeat = isRepeatRune
	}
	return count
}
