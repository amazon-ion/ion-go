package ion

import (
	"fmt"
	"math/big"
	"time"
)

// A Type represents the type of an Ion Value.
type Type uint8

const (
	// NoType is returned by a Reader that is not currently pointing at a value.
	NoType Type = iota

	// NullType is the type of the (unqualified) Ion null value.
	NullType

	// BoolType is the type of an Ion boolean, true or false.
	BoolType

	// IntType is the type of a signed Ion integer of arbitrary size.
	IntType

	// FloatType is the type of a fixed-precision Ion floating-point value.
	FloatType

	// DecimalType is the type of an arbitrary-precision Ion decimal value.
	DecimalType

	// TimestampType is the type of an arbitrary-precision Ion timestamp.
	TimestampType

	// SymbolType is the type of an Ion symbol, mapped to an integer ID by a SymbolTable
	// to (potentially) save space.
	SymbolType

	// StringType is the type of a non-symbol Unicode string, represented directly.
	StringType

	// ClobType is the type of a character large object. Like a BlobType, it stores an
	// arbitrary sequence of bytes, but it represents them in text form as an escaped-ASCII
	// string rather than a base64-encoded string.
	ClobType

	// BlobType is the type of a binary large object; a sequence of arbitrary bytes.
	BlobType

	// ListType is the type of a list, recursively containing zero or more Ion values.
	ListType

	// SexpType is the type of an s-expression. Like a ListType, it contains a sequence
	// of zero or more Ion values, but with a lisp-like syntax when encoded as text.
	SexpType

	// StructType is the type of a structure, recursively containing a sequence of named
	// (by an Ion symbol) Ion values.
	StructType
)

// String implements fmt.Stringer for Type.
func (t Type) String() string {
	switch t {
	case NoType:
		return "<no type>"
	case NullType:
		return "null"
	case BoolType:
		return "bool"
	case IntType:
		return "int"
	case FloatType:
		return "float"
	case DecimalType:
		return "decimal"
	case TimestampType:
		return "timestamp"
	case StringType:
		return "string"
	case SymbolType:
		return "symbol"
	case BlobType:
		return "blob"
	case ClobType:
		return "clob"
	case StructType:
		return "struct"
	case ListType:
		return "list"
	case SexpType:
		return "sexp"
	default:
		return fmt.Sprintf("<unknown type %v>", uint8(t))
	}
}

// IntSize returns the size of an integer, allowing you to pick the
// appropriate Reader method to call to retrieve the value without loss.
type IntSize uint8

const (
	// NullInt is the size of null.int and other things that aren't actually ints.
	NullInt IntSize = iota
	// Int32 is an integer that can be losslessly stored in an int32.
	Int32
	// Int64 is an integer that can be losslessly stored in an int64.
	Int64
	// BigInt is an integer that can only be losslessly stored in a big.Int.
	BigInt
)

// String implements fmt.Stringer for IntSize.
func (i IntSize) String() string {
	switch i {
	case NullInt:
		return "null.int"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case BigInt:
		return "big.Int"
	default:
		return fmt.Sprintf("<unknown size %v>", uint8(i))
	}
}

// A Reader reads a stream of Ion values.
//
// The Reader has a logical position within the stream of values, influencing the
// values returnedd from its methods. Initially, the Reader is positioned before the
// first value in the stream. A call to Next advances the Reader to the first value
// in the stream, with subsequent calls advancing to subsequent values. When a call to
// Next moves the Reader to the position after the final value in the stream, it returns
// false, making it easy to loop through the values in a stream.
//
// 	var r Reader
// 	for r.Next() {
// 		// ...
// 	}
//
// Next also returns false in case of error. This can be distinguished from a legitimate
// end-of-stream by calling Err after exiting the loop.
//
// When positioned on an Ion value, the type of the value can be retrieved by calling
// Type. If it has an associated field name (inside a struct) or annotations, they can
// be read by calling FieldName and Annotations respectively.
//
// For atomic values, an appropriate XxxValue method can be called to read the value.
// For lists, sexps, and structs, you should instead call StepIn to move the Reader in
// to the contained sequence of values. The Reader will initially be positioned before
// the first value in the container. Calling Next without calling StepIn will skip over
// the composite value and return the next value in the outer value stream.
//
// At any point while reading through a composite value, including when Next returns false
// to indicate the end of the contained values, you may call StepOut to move back to the
// outer sequence of values. The Reader will be positioned at the end of the composite value,
// such that a call to Next will move to the immediately-following value (if any).
//
// 	r := NewTextReaderStr("[foo, bar] [")
// 	for r.Next() {
// 		if err := r.StepIn(); err != nil {
// 			return err
// 		}
// 		for r.Next() {
// 			fmt.Println(r.StringValue())
// 		}
// 		if err := r.StepOut(); err != nil {
// 			return err
// 		}
// 	}
// 	if err := r.Err(); err != nil {
// 		return err
// 	}
//
type Reader interface {

	// SymbolTable returns the current symbol table, or nil if there isn't one.
	// Text Readers do not, generally speaking, have an associated symbol table.
	// Binary Readers do.
	SymbolTable() SymbolTable

	// Next advances the Reader to the next position in the current value stream.
	// It returns true if this is the position of an Ion value, and false if it
	// is not. On error, it returns false and sets Err.
	Next() bool

	// Err returns an error if a previous call call to Next has failed.
	Err() error

	// Type returns the type of the Ion value the Reader is currently positioned on.
	// It returns NoType if the Reader is positioned before or after a value.
	Type() Type

	// IsNull returns true if the current value is an explicit null. This may be true
	// even if the Type is not NullType (for example, null.struct has type Struct). Yes,
	// that's a bit confusing.
	IsNull() bool

	// FieldName returns the field name associated with the current value. It returns
	// the empty string if there is no current value or the current value has no field
	// name.
	FieldName() string

	// Annotations returns the set of annotations associated with the current value.
	// It returns nil if there is no current value or the current value has no annotations.
	Annotations() []string

	// StepIn steps in to the current value if it is a container. It returns an error if there
	// is no current value or if the value is not a container. On success, the Reader is
	// positioned before the first value in the container.
	StepIn() error

	// StepOut steps out of the current container value being read. It returns an error if
	// this Reader is not currently stepped in to a container. On success, the Reader is
	// positioned after the end of the container, but before any subsequent values in the
	// stream.
	StepOut() error

	// BoolValue returns the current value as a boolean (if that makes sense). It returns
	// an error if the current value is not an Ion bool.
	BoolValue() (bool, error)

	// IntSize returns the size of integer needed to losslessly represent the current value
	// (if that makes sense). It returns an error if the current value is not an Ion int.
	IntSize() (IntSize, error)

	// IntValue returns the current value as a 32-bit integer (if that makes sense). It
	// returns an error if the current value is not an Ion integer or requires more than
	// 32 bits to represent losslessly.
	IntValue() (int, error)

	// Int64Value returns the current value as a 64-bit integer (if that makes sense). It
	// returns an error if the current value is not an Ion integer or requires more than
	// 64 bits to represent losslessly.
	Int64Value() (int64, error)

	// BigIntValue returns the current value as a big.Integer (if that makes sense). It
	// returns an error if the current value is not an Ion integer.
	BigIntValue() (*big.Int, error)

	// FloatValue returns the current value as a 64-bit floating point number (if that
	// makes sense). It returns an error if the current value is not an Ion float.
	FloatValue() (float64, error)

	// DecimalValue returns the current value as an arbitrary-precision Decimal (if that
	// makes sense). It returns an error if the current value is not an Ion decimal.
	DecimalValue() (*Decimal, error)

	// TimeValue returns the current value as a timestamp (if that makes sense). It returns
	// an error if the current value is not an Ion timestamp.
	TimeValue() (time.Time, error)

	// StringValue returns the current value as a string (if that makes sense). It returns
	// an error if the current value is not an Ion symbol or an Ion string.
	StringValue() (string, error)

	// ByteValue returns the current value as a byte slice (if that makes sense). It returns
	// an error if the current value is not an Ion clob or an Ion blob.
	ByteValue() ([]byte, error)
}

// A Writer writes Ion values to an output stream.
type Writer interface {
	InStruct() bool
	InList() bool
	InSexp() bool
	Err() error

	FieldName(val string)
	Annotation(val string)
	Annotations(vals ...string)

	BeginStruct()
	EndStruct()

	BeginList()
	EndList()

	BeginSexp()
	EndSexp()

	WriteNull()
	WriteNullWithType(t Type)

	WriteBool(val bool)

	WriteInt(val int64)
	WriteBigInt(val *big.Int)
	WriteFloat(val float64)
	WriteDecimal(val *Decimal)

	WriteTimestamp(val time.Time)

	WriteSymbol(val string)
	WriteString(val string)

	WriteBlob(val []byte)
	WriteClob(val []byte)

	Finish() error
}
