package ion

import (
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const (
	// layoutMinutesAndOffset layout for time.date with yyyy-mm-ddThh:mm±hh:mm format. time.Parse()
	// uses "2006-01-02T15:04Z07:00" explicitly as a string value to identify this format.
	layoutMinutesAndOffset = "2006-01-02T15:04Z07:00"

	// layoutMinutesZulu layout for time.date with yyyy-mm-ddThh:mmZ format. time.Parse()
	// uses "2006-01-02T15:04Z07:00" explicitly as a string value to identify this format.
	layoutMinutesZulu = "2006-01-02T15:04Z"

	// layoutNanosecondsAndOffset layout for time.date of RFC3339Nano format.
	// Such as 2006-01-02T15:04:05.999999999Z07:00.
	layoutNanosecondsAndOffset = time.RFC3339Nano

	// layoutSecondsAndOffset layout for time.date of RFC3339 format.
	// Such as: 2006-01-02T15:04:05Z07:00.
	layoutSecondsAndOffset = time.RFC3339
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

// Is this the text form of a symbol reference ($<integer>)?
func isSymbolRef(sym string) bool {
	if len(sym) == 0 || sym[0] != '$' {
		return false
	}

	if len(sym) == 1 {
		return false
	}

	for i := 1; i < len(sym); i++ {
		if !isDigit(int(sym[i])) {
			return false
		}
	}

	return true
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

func parseTimestamp(val string) (time.Time, error) {
	if len(val) < 5 {
		return invalidTimestamp(val)
	}

	year, err := strconv.ParseInt(val[:4], 10, 32)
	if err != nil || year < 1 {
		return invalidTimestamp(val)
	}
	if len(val) == 5 && (val[4] == 't' || val[4] == 'T') {
		// yyyyT
		return tryCreateTimeDate(val, year, 1, 1)
	}
	if val[4] != '-' {
		return invalidTimestamp(val)
	}

	if len(val) < 8 {
		return invalidTimestamp(val)
	}

	month, err := strconv.ParseInt(val[5:7], 10, 32)
	if err != nil {
		return invalidTimestamp(val)
	}

	if len(val) == 8 && (val[7] == 't' || val[7] == 'T') {
		// yyyy-mmT
		return tryCreateTimeDate(val, year, month, 1)
	}
	if val[7] != '-' {
		return invalidTimestamp(val)
	}

	if len(val) < 10 {
		return invalidTimestamp(val)
	}

	day, err := strconv.ParseInt(val[8:10], 10, 32)
	if err != nil {
		return invalidTimestamp(val)
	}

	if len(val) == 10 || (len(val) == 11 && (val[10] == 't' || val[10] == 'T')) {
		// yyyy-mm-dd or yyyy-mm-ddT
		return tryCreateTimeDate(val, year, month, day)
	}
	if val[10] != 't' && val[10] != 'T' {
		return invalidTimestamp(val)
	}

	// At this point timestamp must have hour:minute
	if len(val) < 17 {
		return invalidTimestamp(val)
	}
	if val[16] == 'z' || val[16] == 'Z' {
		return time.Parse(layoutMinutesZulu, val)
	}
	if val[16] == '+' || val[16] == '-' {
		if isValidOffset(val, 16) {
			return time.Parse(layoutMinutesAndOffset, val)
		}
		return invalidTimestamp(val)
	}
	if val[16] == ':' {
		//yyyy-mm-ddThh:mm:ss
		if len(val) < 20 {
			return invalidTimestamp(val)
		}

		idx := 19
		if val[idx] == '.' {
			idx++
			for idx < len(val) && isDigit(int(val[idx])) {
				idx++
			}
		}

		if val[idx] == 'z' || val[idx] == 'Z' {
			if idx >= 29 {
				// Too much precision for a go Time.
				// TODO: We should probably round instead of truncating? Ah well.
				return time.Parse(layoutNanosecondsAndOffset, val[:29]+val[idx:])
			}
			return time.Parse(layoutSecondsAndOffset, val)
		}

		if val[idx] == '+' || val[idx] == '-' {
			if isValidOffset(val, idx) {
				if idx >= 29 {
					// Too much precision for a go Time.
					// TODO: We should probably round instead of truncating? Ah well.
					return time.Parse(layoutNanosecondsAndOffset, val[:29]+val[idx:])
				}
				return time.Parse(layoutSecondsAndOffset, val)
			}
			return invalidTimestamp(val)
		}
	}
	return invalidTimestamp(val)
}

func tryCreateTimeDate(val string, year int64, month int64, day int64) (time.Time, error) {
	date := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)
	// time.Date converts 2000-01-32 input to 2000-02-01
	if int(year) != date.Year() || time.Month(month) != date.Month() || int(day) != date.Day() {
		return invalidTimestamp(val)
	}
	return date, nil
}

func isValidOffset(val string, idx int) bool {
	// +hh:mm
	if idx+5 > len(val) || val[idx+3] != ':' {
		return false
	}

	hourOffset, errH := strconv.ParseInt(val[idx+1:idx+3], 10, 32)
	minuteOffset, errM := strconv.ParseInt(val[idx+4:], 10, 32)
	if errH != nil || errM != nil {
		return false
	}

	return hourOffset < 24 && minuteOffset < 60
}

func invalidTimestamp(val string) (time.Time, error) {
	return time.Time{}, fmt.Errorf("ion: invalid timestamp: %v", val)
}
