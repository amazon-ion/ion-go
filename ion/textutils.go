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
	"math/big"
	"strconv"
	"strings"
)

// Does this symbol need to be quoted in text form?
func symbolNeedsQuoting(sym string) bool {
	switch sym {
	case "", "null", "true", "false", "nan":
		return true
	}

	if !isIdentifierStart(int(sym[0])) {
		return true
	}

	for i := 1; i < len(sym); i++ {
		if !isIdentifierPart(int(sym[i])) {
			return true
		}
	}

	return false
}

// Is this a valid first character for an identifier?
func isIdentifierStart(c int) bool {
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c == '_' || c == '$' {
		return true
	}
	return false
}

// Is this a valid character for later in an identifier?
func isIdentifierPart(c int) bool {
	return isIdentifierStart(c) || isDigit(c)
}

// Is this a valid hex digit?
func isHexDigit(c int) bool {
	if isDigit(c) {
		return true
	}
	if c >= 'a' && c <= 'f' {
		return true
	}
	if c >= 'A' && c <= 'F' {
		return true
	}
	return false
}

// Is this a digit?
func isDigit(c int) bool {
	return c >= '0' && c <= '9'
}

// Is this a valid part of an operator symbol?
func isOperatorChar(c int) bool {
	switch c {
	case '!', '#', '%', '&', '*', '+', '-', '.', '/', ';', '<', '=',
		'>', '?', '@', '^', '`', '|', '~':
		return true
	default:
		return false
	}
}

// Does this character mark the end of a normal (unquoted) value? Does
// *not* check for the start of a comment, because that requires two
// characters. Use tokenizer.isStopChar(c) or check for it yourself.
func isStopChar(c int) bool {
	switch c {
	case -1, '{', '}', '[', ']', '(', ')', ',', '"', '\'',
		' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

// Is this character whitespace?
func isWhitespace(c int) bool {
	switch c {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// Formats a float64 in Ion text style.
func formatFloat(val float64) string {
	str := strconv.FormatFloat(val, 'e', -1, 64)

	// Ion uses lower case for special values.
	switch str {
	case "NaN":
		return "nan"
	case "+Inf":
		return "+inf"
	case "-Inf":
		return "-inf"
	}

	idx := strings.Index(str, "e")
	if idx < 0 {
		// We need to add an 'e' or it will get interpreted as an Ion decimal.
		str += "e0"
	} else if idx+2 < len(str) && str[idx+2] == '0' {
		// FormatFloat returns exponents with a leading ±0 in some cases; strip it.
		str = str[:idx+2] + str[idx+3:]
	}

	return str
}

// Write the given symbol out.
func writeSymbol(val interface{}, out io.Writer) error {
	token := val.(SymbolToken)

	var text string
	if token.Text != nil {
		text = *token.Text

		if _, ok := symbolIdentifier(text); ok {
			// Wrap text value in single quotes if the symbol's text is a symbol identifier
			// (ie. of form $n for some integer n)
			// This is done to distinguish from actual symbol table mappings.
			text = fmt.Sprintf("'%v'", text)
			return writeRawString(text, out)
		}
	} else if token.LocalSID != SymbolIDUnknown {
		text = fmt.Sprintf("$%v", token.LocalSID)
	} else {
		return fmt.Errorf("ion: invalid symbol token")
	}

	return writeSymbolFromString(text, out)
}

// Write the given symbol out, quoting and encoding if necessary.
func writeSymbolFromString(val interface{}, out io.Writer) error {
	sym := val.(string)

	if symbolNeedsQuoting(sym) {
		if err := writeRawChar('\'', out); err != nil {
			return err
		}
		if err := writeEscapedSymbol(sym, out); err != nil {
			return err
		}
		return writeRawChar('\'', out)
	}
	return writeRawString(sym, out)
}

// Write the given symbol out, escaping any characters that need escaping.
func writeEscapedSymbol(sym string, out io.Writer) error {
	for i := 0; i < len(sym); i++ {
		c := sym[i]
		if c < 32 || c == '\\' || c == '\'' {
			if err := writeEscapedChar(c, out); err != nil {
				return err
			}
		} else {
			if err := writeRawChar(c, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// Write the given string out, escaping any characters that need escaping.
func writeEscapedString(str string, out io.Writer) error {
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c < 32 || c == '\\' || c == '"' {
			if err := writeEscapedChar(c, out); err != nil {
				return err
			}
		} else {
			if err := writeRawChar(c, out); err != nil {
				return err
			}
		}
	}
	return nil
}

// Write out the given character in escaped form.
func writeEscapedChar(c byte, out io.Writer) error {
	switch c {
	case 0:
		return writeRawString("\\0", out)
	case '\a':
		return writeRawString("\\a", out)
	case '\b':
		return writeRawString("\\b", out)
	case '\t':
		return writeRawString("\\t", out)
	case '\n':
		return writeRawString("\\n", out)
	case '\f':
		return writeRawString("\\f", out)
	case '\r':
		return writeRawString("\\r", out)
	case '\v':
		return writeRawString("\\v", out)
	case '\'':
		return writeRawString("\\'", out)
	case '"':
		return writeRawString("\\\"", out)
	case '\\':
		return writeRawString("\\\\", out)
	default:
		buf := []byte{'\\', 'x', hexChars[(c>>4)&0xF], hexChars[c&0xF]}
		return writeRawChars(buf, out)
	}
}

// Write out the given raw string.
func writeRawString(val interface{}, out io.Writer) error {
	s := val.(string)

	_, err := out.Write([]byte(s))
	return err
}

// Write out the given raw character sequence.
func writeRawChars(cs []byte, out io.Writer) error {
	_, err := out.Write(cs)
	return err
}

// Write out the given raw character.
func writeRawChar(c byte, out io.Writer) error {
	_, err := out.Write([]byte{c})
	return err
}

func parseFloat(str string) (float64, error) {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		if ne, ok := err.(*strconv.NumError); ok {
			if ne.Err == strconv.ErrRange {
				// Ignore me, val will be +-inf which is fine.
				return val, nil
			}
		}
	}
	return val, err
}

func parseDecimal(str string) (*Decimal, error) {
	return ParseDecimal(str)
}

func parseInt(str string, radix int) (interface{}, error) {
	digits := str

	switch radix {
	case 10:
		// All set.

	case 2, 16:
		neg := false
		if digits[0] == '-' {
			neg = true
			digits = digits[1:]
		}

		// Skip over the '0x' prefix.
		digits = digits[2:]
		if neg {
			digits = "-" + digits
		}

	default:
		panic("unsupported radix")
	}

	i, err := strconv.ParseInt(digits, radix, 64)
	if err == nil {
		return i, nil
	}
	if err.(*strconv.NumError).Err != strconv.ErrRange {
		return nil, err
	}

	bi, ok := (&big.Int{}).SetString(digits, radix)
	if !ok {
		return nil, &strconv.NumError{
			Func: "ParseInt",
			Num:  str,
			Err:  strconv.ErrSyntax,
		}
	}

	return bi, nil
}

func parseTimestamp(val string) (Timestamp, error) {
	return ParseTimestamp(val)
}
