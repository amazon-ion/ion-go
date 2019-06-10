package text

import (
	"io"
)

func needsQuoting(sym string) bool {
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

func isIdentifierPart(c byte) bool {
	return isIdentifierStart(c) || isDigit(c)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func writeSymbol(sym string, out io.Writer) error {
	if needsQuoting(sym) {
		if err := writeChar('\'', out); err != nil {
			return err
		}
		if err := writeEscapedSymbol(sym, out); err != nil {
			return err
		}
		return writeChar('\'', out)
	} else {
		return writeString(sym, out)
	}
}

func writeEscapedSymbol(sym string, out io.Writer) error {
	for i := 0; i < len(sym); i++ {
		c := sym[i]
		if c < 32 || c == '\\' || c == '\'' {
			if err := writeEscapedChar(c, out); err != nil {
				return err
			}
		} else {
			if err := writeChar(c, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeEscapedString(str string, out io.Writer) error {
	for i := 0; i<len(str); i++ {
		c := str[i]
		if c < 32 || c == '\\' || c == '"' {
			if err := writeEscapedChar(c, out); err != nil {
				return err
			}
		} else {
			if err := writeChar(c, out); err != nil {
				return err
			}
		}
	}
	return nil
}

var hexChars = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

func writeEscapedChar(c byte, out io.Writer) error {
	switch c {
	case 0:
		return writeString("\\0", out)
	case '\a':
		return writeString("\\a", out)
	case '\b':
		return writeString("\\b", out)
	case '\t':
		return writeString("\\t", out)
	case '\n':
		return writeString("\\n", out)
	case '\f':
		return writeString("\\f", out)
	case '\r':
		return writeString("\\r", out)
	case '\v':
		return writeString("\\v", out)
	case '\'':
		return writeString("\\'", out)
	case '"':
		return writeString("\\\"", out)
	case '\\':
		return writeString("\\\\", out)
	default:
		buf := []byte{'\\', 'x', hexChars[(c>>4)&0xF], hexChars[c&0xF]}
		return writeChars(buf, out)
	}
}

func writeString(s string, out io.Writer) error {
	_, err := out.Write([]byte(s))
	return err
}

func writeChars(cs []byte, out io.Writer) error {
	_, err := out.Write(cs)
	return err
}

func writeChar(c byte, out io.Writer) error {
	_, err := out.Write([]byte{c})
	return err
}
