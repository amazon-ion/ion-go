package ion

import (
	"fmt"
	"math/big"
	"time"
)

// Type is the type of an Ion Value.
type Type uint8

const (
	// NoType is returned by a Reader that's not currently pointing at a value.
	NoType Type = iota
	// NullType is the type of the (unqualified) null value.
	NullType
	// BoolType is the type of a boolean, true or false.
	BoolType
	// IntType is the type of a signed integer of arbitrary size.
	IntType
	// FloatType is the type of a 64-bit floating-point value.
	FloatType
	// DecimalType is the type of an arbitrary-precision decimal value.
	DecimalType
	// TimestampType is the type of a timestamp.
	TimestampType
	// StringType is the type of a Unicode string.
	StringType
	// SymbolType is the type of an interned string.
	SymbolType
	// BlobType is the type of a binary large object.
	BlobType
	// ClobType is the type of a character large object.
	ClobType
	// StructType is the type of a structure.
	StructType
	// ListType is the type of a list.
	ListType
	// SexpType is the type of an s-expression.
	SexpType
)

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

// A Reader reads Ion values from an input stream.
type Reader interface {
	SymbolTable() SymbolTable

	Next() bool
	Type() Type
	Err() error

	FieldName() string
	TypeAnnotations() []string
	IsNull() bool

	StepIn() error
	StepOut() error

	BoolValue() (bool, error)
	IntValue() (int, error)
	Int64Value() (int64, error)
	BigIntValue() (*big.Int, error)
	FloatValue() (float64, error)
	DecimalValue() (*Decimal, error)

	TimeValue() (time.Time, error)
	StringValue() (string, error)

	ByteValue() ([]byte, error)
}

// A Writer writes Ion values to an output stream.
type Writer interface {
	InStruct() bool
	InList() bool
	InSexp() bool
	Err() error

	FieldName(val string)
	TypeAnnotation(val string)
	TypeAnnotations(vals ...string)

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
