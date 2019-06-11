package ion

import (
	"io"
)

// Does this symbol need to be quoted in text form?
func symbolNeedsQuoting(sym string) bool {
	if sym == "" || sym == "null" || sym == "true" || sym == "false" || sym == "nan" {
		return true
	}

	if isSymbolRef(sym) {
		return true
	}

	if !isIdentifierStart(sym[0]) {
		return true
	}

	for i := 1; i < len(sym); i++ {
		if !isIdentifierPart(sym[i]) {
			return true
		}
	}

	return false
}

// Is this the text form of a symbol reference ($<integer>)?
func isSymbolRef(sym string) bool {
	if len(sym) == 0 || sym[0] != '$' {
		return false
	}

	if len(sym) == 1 {
		return false
	}

	for i := 1; i < len(sym); i++ {
		if !isDigit(sym[i]) {
			return false
		}
	}

	return true
}

// Is this a valid first character for an identifier?
func isIdentifierStart(c byte) bool {
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
func isIdentifierPart(c byte) bool {
	return isIdentifierStart(c) || isDigit(c)
}

// Is this a digit?
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// Write the given symbol out, quoting and encoding if necessary.
func writeSymbol(sym string, out io.Writer) error {
	if symbolNeedsQuoting(sym) {
		if err := writeRawChar('\'', out); err != nil {
			return err
		}
		if err := writeEscapedSymbol(sym, out); err != nil {
			return err
		}
		return writeRawChar('\'', out)
	} else {
		return writeRawString(sym, out)
	}
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
	for i := 0; i<len(str); i++ {
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

var hexChars = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

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
func writeRawString(s string, out io.Writer) error {
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
