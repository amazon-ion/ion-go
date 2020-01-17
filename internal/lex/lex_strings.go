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
)

const (
	// Runes that can follow a slash as part of an escape sequence.
	escapeAbleRunes = "abtnfrv?0xuU'\"/\r\n\\"
)

// This file contains the state functions for lexing string and symbol types.  It
// does not contain state functions for the binary text types Blob and Clob.

// lexSymbol scans an annotation or symbol and returns lexValue.
func lexSymbol(x *Lexer) stateFn {
	isNull := false
	for {
		switch ch := x.next(); {
		case isIdentifierSymbolPart(ch):
		case ch == '.' && string(x.input[x.itemStart:x.pos]) == "null.":
			// There is a special case where a dot is okay within an identifier symbol,
			// and that is when it is part of one of the null types.
			isNull = true
		case isIdentifierSymbolEnd(ch):
			x.backup()
			if isNull {
				x.emit(IonNull)
			} else {
				x.emit(IonSymbol)
			}
			return lexValue
		default:
			return x.errorf("bad character as part of symbol: %#U", ch)
		}
	}
}

// lexString scans a quoted string.
func lexString(x *Lexer) stateFn {
	// Ignore the opening quote.
	x.ignore()

Loop:
	for {
		switch ch := x.next(); {
		case ch == '\\':
			if fn := handleEscapedRune(x); fn != nil {
				return fn
			}
		case isEndOfLine(ch) || ch == eof:
			return x.error("unterminated quoted string")
		case ch == '"':
			x.backup()
			break Loop
		case isStringPart(ch):
		// Yay, happy string character.
		default:
			return x.errorf("bad character as part of string: %#U", ch)
		}
	}

	x.emit(IonString)
	// Ignore the closing quote.
	x.next()
	x.ignore()
	return lexValue
}

// lexSingleQuote determines whether the single quote is for the start of a quoted
// symbol, an empty quotedSymbol, or the first of three single quotes that denotes
// a long string
func lexSingleQuote(x *Lexer) stateFn {
	// Need to distinguish between an empty symbol, e.g. '', and the
	// start of a "long string", e.g. '''
	if x.peek() == '\'' {
		x.next()
		if x.peek() == '\'' {
			// Triple quote!  Dive into lexing a quoted long string.
			x.next()
			return lexLongString
		} else {
			// Ignore the opening and closing quotes.
			x.ignore()
			// Empty quoted symbol.  Emit it and move on.
			x.emit(IonSymbolQuoted)
			return lexValue
		}
	}
	return lexQuotedSymbol
}

// lexLongString scans a long string.  It is up to the parser to join multiple
// long strings together.
func lexLongString(x *Lexer) stateFn {
	// Ignore the initial triple quote.
	x.ignore()

	count := 0

Loop:
	for {
		// Keep consuming single quotes until they stop.  If there was a run
		// of three or more then the string was ended and we can break the loop.
		switch ch := x.next(); {
		case ch == '\\':
			if fn := handleEscapedRune(x); fn != nil {
				return fn
			}
		case ch == eof:
			if count >= 3 {
				break Loop
			}
			return x.error("unterminated long string")
		case ch == '\'':
			count++
		case isLongStringPart(ch):
			if count >= 3 {
				x.backup()
				break Loop
			}
			count = 0
		default:
			if count >= 3 {
				x.backup()
				break Loop
			}
			return x.errorf("bad character as part of long string: %#U", ch)
		}
	}

	x.emitAndIgnoreTripleQuoteEnd(IonStringLong)
	return lexValue
}

// lexQuotedSymbol scans an annotation or symbol that is surrounded
// in quotes and returns lexValue.
func lexQuotedSymbol(x *Lexer) stateFn {
	// Ignore the opening quote.
	x.ignore()

Loop:
	for {
		switch ch := x.next(); {
		case isQuotedSymbolPart(ch):
		case ch == '\\':
			if fn := handleEscapedRune(x); fn != nil {
				return fn
			}
		case ch == eof:
			return x.error("unterminated quoted symbol")
		case ch == '\'':
			x.backup()
			break Loop
		default:
			return x.errorf("bad character as part of quoted symbol: %#U", ch)
		}
	}

	x.emit(IonSymbolQuoted)
	// Ignore the closing quote.
	x.next()
	x.ignore()
	return lexValue
}

// handleEscapedRune checks the character after an escape character '\' within
// a string.  If the escaped character is not valid, e.g. EOF, then an error
// stateFn is returned.
func handleEscapedRune(x *Lexer) stateFn {
	// Escaping the EOF isn't cool.
	ch := x.next()
	if ch == eof {
		return x.error("unterminated sequence")
	}
	if !isEscapeAble(ch) {
		return x.errorf("invalid character after escape: %#U", ch)
	}

	// Both the \x and \u escapes should be followed by a specific number
	// of unicode characters (2 and 4 respectively).
	switch ch {
	case 'x':
		// '\x' HEX_DIGIT HEX_DIGIT
		if !bytes.ContainsRune([]byte(hexDigits), x.next()) || !bytes.ContainsRune([]byte(hexDigits), x.next()) {
			x.backup()
			return x.errorf("invalid character as part of hex escape: %#U", x.peek())
		}
	case 'u':
		// '\u' HEX_DIGIT_QUARTET
		if !bytes.ContainsRune([]byte(hexDigits), x.next()) || !bytes.ContainsRune([]byte(hexDigits), x.next()) ||
			!bytes.ContainsRune([]byte(hexDigits), x.next()) || !bytes.ContainsRune([]byte(hexDigits), x.next()) {
			return unicodeEscapeError(x)
		}
	case 'U':
		// '\U000'  HEX_DIGIT_QUARTET HEX_DIGIT or
		// '\U0010' HEX_DIGIT_QUARTET
		if x.next() != '0' || x.next() != '0' {
			return unicodeEscapeError(x)
		}
		switch next := x.next(); next {
		case '0':
			// Eat the hex digit that is expected to be a 0 in the other case.
			if !bytes.ContainsRune([]byte(hexDigits), x.next()) {
				return unicodeEscapeError(x)
			}
		case '1':
			if x.next() != '0' {
				return unicodeEscapeError(x)
			}
		default:
			return unicodeEscapeError(x)
		}
		if !bytes.ContainsRune([]byte(hexDigits), x.next()) || !bytes.ContainsRune([]byte(hexDigits), x.next()) ||
			!bytes.ContainsRune([]byte(hexDigits), x.next()) || !bytes.ContainsRune([]byte(hexDigits), x.next()) {
			return unicodeEscapeError(x)
		}
	}

	// If the file is from Windows then the escape character may be
	// trying to escape both /r and /n.
	if pk := x.peek(); ch == '\r' && pk == '\n' {
		x.next()
	}
	return nil
}

// unicodeEscapeError is a convenience function that backs up the lexer and emits an invalid
// character error with that character.
func unicodeEscapeError(x *Lexer) stateFn {
	x.backup()
	return x.errorf("invalid character as part of unicode escape: %#U", x.peek())
}

// isIdentifierSymbolEnd returns if the given rune is one of the container end characters.
func isContainerEnd(ch rune) bool {
	return ch == ')' || ch == ']' || ch == '}'
}

// isIdentifierSymbolStart returns if the given rune is a valid start of a symbol.
// IDENTIFIER_SYMBOL: [$_a-zA-Z] ([$_a-zA-Z] | DEC_DIGIT)*
func isIdentifierSymbolStart(ch rune) bool {
	return ch == '$' || ch == '_' || ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

// isIdentifierSymbolPart returns if the given rune is a valid part of an
// identifier symbol.
// IDENTIFIER_SYMBOL: [$_a-zA-Z] ([$_a-zA-Z] | DEC_DIGIT)*
func isIdentifierSymbolPart(ch rune) bool {
	return ch == '$' || ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || isNumber(ch)
}

// isIdentifierSymbolEnd returns if the given rune is a valid end character
// for an identifier symbol.
func isIdentifierSymbolEnd(ch rune) bool {
	return ch == ':' || ch == ',' || ch == '.' || isContainerEnd(ch) || isWhitespace(ch) || isOperator(ch) || ch == eof
}

// isQuotedSymbolPart returns if the given rune is a valid part of a quoted symbol.
// SYMBOL_TEXT: (TEXT_ESCAPE | SYMBOL_TEXT_ALLOWED)*
// SYMBOL_TEXT_ALLOWED
//    : '\u0020'..'\u0026' // no C1 control characters and no U+0027 single quote
//    | '\u0028'..'\u005B' // no U+005C backslash
//    | '\u005D'..'\u10FFFF'
//    | WS_NOT_NL
// Note: The backslash character is a valid escape-able character, so it is valid within
// a quoted symbol even though it isn't allowed here.
func isQuotedSymbolPart(ch rune) bool {
	return (ch >= 0x0020 && ch <= 0x0026) || (ch >= 0x0028 && ch <= 0x005B) || (ch >= 0x005D && ch <= 0x10FFFF) || isSpace(ch)
}

// isStringPart returns if the given rune is a valid part of a double-quoted string.
// STRING_SHORT_TEXT_ALLOWED
//    : '\u0020'..'\u0021' // no C1 control characters and no U+0022 double quote
//    | '\u0023'..'\u005B' // no U+005C backslash
//    | '\u005D'..'\u10FFFF'
//    | WS_NOT_NL
// Note: The backslash character is a valid escape-able character, so it is valid within
// a double-quoted string even though it isn't allowed here.
func isStringPart(ch rune) bool {
	return (ch >= 0x0020 && ch <= 0x0021) || (ch >= 0x0023 && ch <= 0x005B) || (ch >= 0x005D && ch <= 0x10FFFF) || isSpace(ch)
}

// isLongStringPart returns if the given rune is a valid part of a long string.
// STRING_LONG_TEXT_ALLOWED
//    : '\u0020'..'\u005B' // no C1 control characters and no U+005C backslash
//    | '\u005D'..'\u10FFFF'
//    | WS
// Note: The backslash character is a valid escape-able character, so it is valid within
// a long string even though it isn't allowed here.
func isLongStringPart(ch rune) bool {
	return (ch >= 0x0020 && ch <= 0x005B) || (ch >= 0x005D && ch <= 0x10FFFF) || isWhitespace(ch)
}

// isEscapeAble returns if the given rune is a character that is allowed to follow the
// escape character U+005C backslash.
func isEscapeAble(ch rune) bool {
	return bytes.ContainsRune([]byte(escapeAbleRunes), ch)
}
