package ion

import (
	"errors"
	"io"
	"math/big"
	"time"
)

// A Writer writes a stream of Ion values.
//
// The various Write methods write atomic values to the current output stream. The
// Begin methods begin writing a list, sexp, or struct respectively. Subsequent
// calls to Write will write values inside of the container until a matching
// End method is called.
//
// 	var w Writer
// 	w.BeginSexp()
// 	{
// 		w.WriteInt(1)
// 		w.WriteSymbol("+")
// 		w.WriteInt(1)
// 	}
// 	w.EndSexp()
//
// When writing values inside a struct, the FieldName method must be called before
// each value to set the value's field name. The Annotation method may likewise
// be called before writing any value to add an annotation to the value.
//
// 	var w Writer
// 	w.Annotation("user")
// 	w.BeginStruct()
// 	{
// 		w.FieldName("id")
// 		w.WriteString("qu33nb33")
// 		w.FieldName("name")
// 		w.WriteString("Beyoncé")
// 	}
// 	w.EndStruct()
//
// When you're done writing values, you should call Finish to ensure everything has
// been flushed from in-memory buffers. While individual methods all return an error
// on failure, implementations will remember any errors, no-op subsequent calls, and
// return the previous error. This lets you keep code a bit cleaner by only checking
// the return value of the final method call (generally Finish).
//
// 	var w Writer
// 	writeSomeStuff(w)
// 	if err := w.Finish(); err != nil {
// 		return err
// 	}
//
type Writer interface {

	// FieldName sets the field name for the next value written.
	FieldName(val string) error

	// Annotation adds a single annotation to the next value written.
	Annotation(val string) error

	// Annotations adds multiple annotations to the next value written.
	Annotations(vals ...string) error

	// WriteNull writes an untyped null value.
	WriteNull() error
	// WriteNullType writes a null value with a type qualifier, e.g. null.bool.
	WriteNullType(t Type) error

	// WriteBool writes a boolean value.
	WriteBool(val bool) error

	// WriteInt writes an integer value.
	WriteInt(val int64) error
	// WriteUint writes an unsigned integer value.
	WriteUint(val uint64) error
	// WriteBigInt writes a big integer value.
	WriteBigInt(val *big.Int) error
	// WriteFloat writes a floating-point value.
	WriteFloat(val float64) error
	// WriteDecimal writes an arbitrary-precision decimal value.
	WriteDecimal(val *Decimal) error

	// WriteTimestamp writes a timestamp value.
	WriteTimestamp(val time.Time) error

	// WriteSymbol writes a symbol value.
	WriteSymbol(val string) error
	// WriteString writes a string value.
	WriteString(val string) error

	// WriteClob writes a clob value.
	WriteClob(val []byte) error
	// WriteBlob writes a blob value.
	WriteBlob(val []byte) error

	// BeginList begins writing a list value.
	BeginList() error
	// EndList finishes writing a list value.
	EndList() error

	// BeginSexp begins writing an s-expression value.
	BeginSexp() error
	// EndSexp finishes writing an s-expression value.
	EndSexp() error

	// BeginStruct begins writing a struct value.
	BeginStruct() error
	// EndStruct finishes writing a struct value.
	EndStruct() error

	// Finish finishes writing values and flushes any buffered data.
	Finish() error
}

// A writer holds shared stuff for all writers.
type writer struct {
	out io.Writer
	ctx ctxstack
	err error

	fieldName   string
	annotations []string
}

// FieldName sets the field name for the next value written.
// It may only be called while writing a struct.
func (w *writer) FieldName(val string) error {
	if w.err != nil {
		return w.err
	}
	if !w.InStruct() {
		w.err = errors.New("ion: Writer.FieldName called when not writing a struct")
		return w.err
	}

	w.fieldName = val
	return nil
}

// Annotation adds an annotation to the next value written.
func (w *writer) Annotation(val string) error {
	if w.err == nil {
		w.annotations = append(w.annotations, val)
	}
	return w.err
}

// Annotations adds one or more annotations to the next value written.
func (w *writer) Annotations(val ...string) error {
	if w.err == nil {
		w.annotations = append(w.annotations, val...)
	}
	return w.err
}

// InStruct returns true if we're currently writing a struct.
func (w *writer) InStruct() bool {
	return w.ctx.peek() == ctxInStruct
}

// Clear clears field name and annotations after writing a value.
func (w *writer) clear() {
	w.fieldName = ""
	w.annotations = nil
}
