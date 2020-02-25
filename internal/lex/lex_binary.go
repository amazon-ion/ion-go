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

// This file contains the state functions for lexing Blobs and Clobs.

// lexBinary emits IonBinaryStart, determines if the contained text is a Blob or Clob
// and emits the corresponding function.
func lexBinary(x *Lexer) stateFn {
	x.emit(IonBinaryStart)
	eatWhitespace(x)
	// We can tell the difference between a Blob and a Clob by the presence of an
	// opening quote character.
	if ch := x.peek(); ch == '\'' || ch == '"' {
		return lexClob
	}
	return lexBlob
}

// lexBlob reads the base64-encoded blob, keeping all whitespace.  If the blob does
// not have the correct number of padding characters, then an error is returned.
func lexBlob(x *Lexer) stateFn {
	eqCount := 0
	charCount := 0

Loop:
	for {
		switch ch := x.next(); {
		case ch == eof:
			return x.error("unterminated blob")
		case ch == '}':
			if ch = x.peek(); ch != '}' {
				return x.errorf("invalid end to blob, expected } but found: %#U", ch)
			}
			// We can back up again since we know for a fact that the previous
			// character is the same width as the character we just peeked.
			x.backup()
			break Loop
		case isBlobText(ch):
			if eqCount > 0 {
				return x.error("base64 character found after padding character")
			}
			charCount++
		case ch == '=':
			eqCount++
		case isWhitespace(ch):
		default:
			return x.errorf("invalid rune as part of blob string: %#U", ch)
		}
	}

	if eqCount > 2 {
		return x.error("too much padding for base64 encoding")
	}

	if (charCount+eqCount)%4 != 0 {
		return x.error("invalid base64 encoding")
	}

	x.emit(IonBlob)
	x.next()
	x.next()
	x.emit(IonBinaryEnd)

	return lexValue
}

// lexClob determines whether the clob uses the long or short string format then
// returns the corresponding stateFn.
func lexClob(x *Lexer) stateFn {
	// Determine if we are a short or long string.  If we are a long string then
	// we need to keep looking for long strings before we end.
	if ch := x.next(); ch == '"' {
		return lexClobShort
	}
	x.backup()
	return lexClobLong
}

// lexClobShort consumes Clob text between double-quotes, similar to a String but is
// limited in what characters are legal.
func lexClobShort(x *Lexer) stateFn {
	// Ignore the opening quote.
	x.ignore()

Loop:
	for {
		switch ch := x.next(); {
		case isClobShortText(ch):
		case ch == '\\':
			switch r := x.next(); {
			case r == eof:
				return x.error("unterminated short clob")
			case !isEscapeAble(r):
				return x.errorf("invalid character after escape: %#U", r)
			case r == '\r':
				// check for CR LF
				if x.peek() == '\n' {
					x.next()
				}
			case r == 'x' || r == 'X':
				// If what is being escaped is a hex character, then we still
				// need to make sure that escaped character is allowed.
				if !bytes.ContainsRune([]byte(hexDigits), x.next()) || !bytes.ContainsRune([]byte(hexDigits), x.next()) {
					x.backup()
					return x.errorf("invalid character as part of hex escape: %#U", x.peek())
				}
			case r == 'u' || r == 'U':
				return x.error("unicode escape is not valid in clob")
			}
		case isEndOfLine(ch) || ch == eof:
			return x.error("unterminated short clob")
		case ch == '"':
			x.backup()
			break Loop
		default:
			return x.errorf("invalid rune as part of short clob string: %#U", ch)
		}
	}

	x.emit(IonClobShort)
	// Ignore the closing quote.
	x.next()
	x.ignore()

	eatWhitespace(x)
	if ch := x.next(); ch != '}' {
		return x.errorf("invalid end to short clob, expected } but found: %#U", ch)
	}
	if ch := x.next(); ch != '}' {
		return x.errorf("invalid end to short clob, expected second } but found: %#U", ch)
	}

	x.emit(IonBinaryEnd)
	return lexValue
}

// lexClobLong consumes Clob text between one or more sets of triple single-quotes, similar
// to a Long String but is limited in what characters are legal.
func lexClobLong(x *Lexer) stateFn {
	// emitSingleLongClob returns true as long as it is able to process a long
	// string.  It returns false if it cannot, e.g. it encounters the end of the
	// Clob.  If it encounters an error, then a state function is returned.
	for lexed, errFn := emitSingleLongClob(x); lexed || errFn != nil; lexed, errFn = emitSingleLongClob(x) {
		if errFn != nil {
			return errFn
		}
	}

	// emitSingleLongClob returns an error if we have an invalid ending of the
	// Clob, so we can safely eat the ending "}}".
	x.next()
	x.next()
	x.emit(IonBinaryEnd)

	return lexValue
}

// emitSingleLongClob eats optional whitespace then expects either the end of a Clob
// or the opening of a long string.  If we encountered the end of a Clob, then false and a nil
// state function are returned.  We then read the Clob version of a long string and return
// true and a nil state function if through the end of the long string is read without issue.
// If there is an issue, then false and an error state function are returned.
func emitSingleLongClob(x *Lexer) (bool, stateFn) {
	// eat any whitespace.
	eatWhitespace(x)

	// If the next character is the closing of a binary blob, then check it out
	// and consume it if it is.
	if x.peek() == '}' {
		x.next()
		if ch := x.peek(); ch != '}' {
			return false, x.errorf("expected a second } but found: %#U", ch)
		}
		// We can back up again since we know for a fact that the previous
		// character is the same width as the character we just peeked.
		x.backup()
		return false, nil
	}

	// Ensure that we have three single quotes to start the clob, then ignore them.
	if x.next() != '\'' || x.next() != '\'' || x.next() != '\'' {
		x.backup()
		return false, x.errorf("expected end of a Clob or start of a long string but found: %#U", x.next())
	}
	x.ignore()

	for {
		switch ch := x.next(); {
		case isClobLongText(ch):
		case ch == '\\':
			// Eat whatever is after the escape character unless it's an EOF.
			if r := x.next(); r == eof {
				return false, x.error("unterminated long clob")
			}
		case ch == eof:
			return false, x.error("unterminated long clob")
		case ch == '\'':
			count := 1
			for next := x.next(); next == '\''; next = x.next() {
				count++
			}
			// We have reached a character after the end of our long string, which is
			// three single quotes. Need to back up over both that character and the
			// three single quotes, emit the long string, then eat the single quotes.
			if count >= 3 {
				x.backup()
				x.emitAndIgnoreTripleQuoteEnd(IonClobLong)

				return true, nil
			}
		default:
			return false, x.errorf("invalid rune as part of long clob string: %#U", ch)
		}
	}

}

// isBlobText returns if the given rune is a valid Blob rune.  Note that whitespace is not included
// since there are rules for when certain non-blob-text characters can occur.
func isBlobText(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '+' || ch == '/'
}

// isClobLongText returns if the given rune is a valid part of a long-quoted Clob.
// CLOB_LONG_TEXT_ALLOWED
//	    : '\u0020'..'\u0026' // no U+0027 single quote
//	    | '\u0028'..'\u005B' // no U+005C blackslash
//	    | '\u005D'..'\u007F'
//	    | WS
//	    ;
func isClobLongText(ch rune) bool {
	return (ch >= 0x0020 && ch <= 0x0026) || (ch >= 0x0028 && ch <= 0x005B) || (ch >= 0x005D && ch <= 0x07F) || isWhitespace(ch)
}

// isClobShortText returns if the given rune is a valid part of a short-quoted Clob.
// 	CLOB_SHORT_TEXT_ALLOWED
//	    : '\u0020'..'\u0021' // no U+0022 double quote
//	    | '\u0023'..'\u005B' // no U+005C backslash
//	    | '\u005D'..'\u007F'
//	    | WS_NOT_NL
//	    ;
func isClobShortText(ch rune) bool {
	return (ch >= 0x0020 && ch <= 0x0021) || (ch >= 0x0023 && ch <= 0x005B) || (ch >= 0x005D && ch <= 0x007F) || isSpace(ch)
}

// isSpace returns if the given rune is a valid space character (not newline).
// WS_NOT_NL
//    : '\u0009' // tab
//    | '\u000B' // vertical tab
//    | '\u000C' // form feed
//    | '\u0020' // space
func isSpace(ch rune) bool {
	return ch == 0x09 || ch == 0x0B || ch == 0x0C || ch == 0x20
}
