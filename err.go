package ion

import "fmt"

// A UsageError is returned when you use a Reader or Writer in an inappropriate way.
type UsageError struct {
	API string
	Msg string
}

func (e *UsageError) Error() string {
	return fmt.Sprintf("ion: usage error in %v: %v", e.API, e.Msg)
}

// An IOError is returned when there is an error reading from or writing to an
// underlying io.Reader or io.Writer.
type IOError struct {
	Err error
}

func (e *IOError) Error() string {
	return fmt.Sprintf("ion: i/o error: %v", e.Err)
}

// A SyntaxError is returned when a Reader encounters invalid input for which no more
// specific error type is defined.
type SyntaxError struct {
	Msg    string
	Offset uint64
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("ion: syntax error: %v (offset %v)", e.Msg, e.Offset)
}

// An UnexpectedEOFError is returned when a Reader unexpectedly encounters an
// io.EOF error.
type UnexpectedEOFError struct {
	Offset uint64
}

func (e *UnexpectedEOFError) Error() string {
	return fmt.Sprintf("ion: unexpected end of input (offset %v)", e.Offset)
}

// An UnsupportedVersionError is returned when a Reader encounters a binary version
// marker with a version that this library does not understand.
type UnsupportedVersionError struct {
	Major  int
	Minor  int
	Offset uint64
}

func (e *UnsupportedVersionError) Error() string {
	return fmt.Sprintf("ion: unsupported version %v.%v (offset %v)", e.Major, e.Minor, e.Offset)
}

// An InvalidTagByteError is returned when a binary Reader encounters an invalid
// tag byte.
type InvalidTagByteError struct {
	Byte   byte
	Offset uint64
}

func (e *InvalidTagByteError) Error() string {
	return fmt.Sprintf("ion: invalid tag byte 0x%02X (offset %v)", e.Byte, e.Offset)
}

// An UnexpectedRuneError is returned when a text Reader encounters an unexpected rune.
type UnexpectedRuneError struct {
	Rune   rune
	Offset uint64
}

func (e *UnexpectedRuneError) Error() string {
	return fmt.Sprintf("ion: unexpected rune %q (offset %v)", e.Rune, e.Offset)
}

// An UnexpectedTokenError is returned when a text Reader encounters an unexpected
// token.
type UnexpectedTokenError struct {
	Token  string
	Offset uint64
}

func (e *UnexpectedTokenError) Error() string {
	return fmt.Sprintf("ion: unexpected token '%v' (offset %v)", e.Token, e.Offset)
}
