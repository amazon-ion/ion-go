package ion

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// TextWriterOpts defines a set of bit flag options for text writers.
type TextWriterOpts uint8

const (
	// OptQuietFinish disables emiting a newline in Finish(). Convenient if you know
	// you're only emiting one datagram; dangerous if there's a chance you're going to
	// emit another datagram using the same Writer.
	OptQuietFinish TextWriterOpts = 1
)

// textWriter is a writer that writes human-readable text
type textWriter struct {
	writer
	needsSeparator bool
	opts           TextWriterOpts
}

// NewTextWriter returns a new text writer.
func NewTextWriter(out io.Writer) Writer {
	return NewTextWriterOpts(out, 0)
}

// NewTextWriterOpts returns a new text writer with the given options.
func NewTextWriterOpts(out io.Writer, opts TextWriterOpts) Writer {
	return &textWriter{
		writer: writer{
			out: out,
		},
		opts: opts,
	}
}

// beginValue begins the process of writing a value, by writing out
// a separator (if needed), field name (if in a struct), and type
// annotations (if any).
func (w *textWriter) beginValue() error {
	if w.needsSeparator {
		var sep byte
		switch w.ctx.peek() {
		case ctxInStruct, ctxInList:
			sep = ','
		case ctxInSexp:
			sep = ' '
		default:
			sep = '\n'
		}

		if err := writeRawChar(sep, w.out); err != nil {
			return err
		}
	}

	if w.InStruct() {
		if w.fieldName == "" {
			return errors.New("field name not set")
		}
		name := w.fieldName
		w.fieldName = ""

		if err := writeSymbol(name, w.out); err != nil {
			return err
		}
		if err := writeRawChar(':', w.out); err != nil {
			return err
		}
	}

	if len(w.typeAnnotations) > 0 {
		as := w.typeAnnotations
		w.typeAnnotations = nil

		for _, a := range as {
			if err := writeSymbol(a, w.out); err != nil {
				return err
			}
			if err := writeRawString("::", w.out); err != nil {
				return err
			}
		}
	}

	return nil
}

// endValue finishes the process of writing a value.
func (w *textWriter) endValue() {
	w.needsSeparator = true
}

// begin starts writing a container of the given type.
func (w *textWriter) begin(t ctxType, c byte) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	w.ctx.push(t)
	w.needsSeparator = false

	return writeRawChar(c, w.out)
}

// end finishes writing a container of the given type
func (w *textWriter) end(t ctxType, c byte) error {
	if w.ctx.peek() != t {
		return errors.New("not in that kind of container")
	}

	if err := writeRawChar(c, w.out); err != nil {
		return err
	}

	w.fieldName = ""
	w.typeAnnotations = nil
	w.ctx.pop()
	w.endValue()

	return nil
}

// BeginStruct begins writing a struct.
func (w *textWriter) BeginStruct() {
	if w.err != nil {
		return
	}
	w.err = w.begin(ctxInStruct, '{')
}

// EndStruct finishes writing a struct.
func (w *textWriter) EndStruct() {
	if w.err != nil {
		return
	}
	w.err = w.end(ctxInStruct, '}')
}

// BeginList begins writing a list.
func (w *textWriter) BeginList() {
	if w.err != nil {
		return
	}
	w.err = w.begin(ctxInList, '[')
}

// EndList finishes writing a list.
func (w *textWriter) EndList() {
	if w.err != nil {
		return
	}
	w.err = w.end(ctxInList, ']')
}

// BeginSexp begins writing an s-expression.
func (w *textWriter) BeginSexp() {
	if w.err != nil {
		return
	}
	w.err = w.begin(ctxInSexp, '(')
}

// EndSexp finishes writing an s-expression.
func (w *textWriter) EndSexp() {
	if w.err != nil {
		return
	}
	w.err = w.end(ctxInSexp, ')')
}

// writeValue writes a value whose raw encoding is produced by the
// given function.
func (w *textWriter) writeValue(f func() string) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	sym := f()
	if err := writeRawString(sym, w.out); err != nil {
		return err
	}

	w.endValue()
	return nil
}

// writeValue writes a value by calling the given function, which is
// expected to write the raw value to w.out.
func (w *textWriter) writeValueStreaming(f func() error) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	if err := f(); err != nil {
		return err
	}

	w.endValue()
	return nil
}

// WriteNull writes an untyped null.
func (w *textWriter) WriteNull() {
	w.WriteNullWithType(NullType)
}

// WriteNullWithType writes a typed null.
func (w *textWriter) WriteNullWithType(t Type) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		switch t {
		case NoType, NullType:
			return "null"
		case BoolType:
			return "null.bool"
		case IntType:
			return "null.int"
		case FloatType:
			return "null.float"
		case DecimalType:
			return "null.decimal"
		case TimestampType:
			return "null.timestamp"
		case StringType:
			return "null.string"
		case SymbolType:
			return "null.symbol"
		case BlobType:
			return "null.blob"
		case ClobType:
			return "null.clob"
		case StructType:
			return "null.struct"
		case ListType:
			return "null.list"
		case SexpType:
			return "null.sexp"
		default:
			panic(fmt.Sprintf("invalid type: %v", t))
		}
	})
}

// WriteBool writes a boolean value.
func (w *textWriter) WriteBool(val bool) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		if val {
			return "true"
		}
		return "false"
	})
}

// WriteInt writes an integer value.
func (w *textWriter) WriteInt(val int64) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		return fmt.Sprintf("%d", val)
	})
}

// WriteBigInt writes a (big) integer value.
func (w *textWriter) WriteBigInt(val *big.Int) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		return val.String()
	})
}

// WriteFloat writes a floating-point value.
func (w *textWriter) WriteFloat(val float64) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		// Built-in go formatting isn't quite up to the task. :(
		str := strconv.FormatFloat(val, 'e', -1, 64)

		switch str {
		case "NaN":
			return "nan"
		case "+Inf":
			return "+inf"
		case "-Inf":
			return "-inf"
		default:
			break
		}

		idx := strings.Index(str, "e")
		if idx < 0 {
			str += "e0"
		} else if idx+2 < len(str) && str[idx+2] == '0' {
			str = str[:idx+2] + str[idx+3:]
		}

		return str
	})
}

// WriteDecimal writes an arbitrary-precision decimal value.
func (w *textWriter) WriteDecimal(val *Decimal) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		return val.String()
	})
}

// WriteTimestamp writes a timestamp.
func (w *textWriter) WriteTimestamp(val time.Time) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		return val.Format(time.RFC3339Nano)
	})
}

// WriteSymbol writes a symbol.
func (w *textWriter) WriteSymbol(val string) {
	if w.err != nil {
		return
	}
	w.err = w.writeValueStreaming(func() error {
		return writeSymbol(val, w.out)
	})
}

// WriteString writes a string.
func (w *textWriter) WriteString(val string) {
	if w.err != nil {
		return
	}
	w.err = w.writeValueStreaming(func() error {
		if err := writeRawChar('"', w.out); err != nil {
			return err
		}
		if err := writeEscapedString(val, w.out); err != nil {
			return err
		}
		return writeRawChar('"', w.out)
	})
}

// WriteBlob writes a blob.
func (w *textWriter) WriteBlob(val []byte) {
	if w.err != nil {
		return
	}
	w.err = w.writeValueStreaming(func() error {
		if err := writeRawString("{{", w.out); err != nil {
			return err
		}

		enc := base64.NewEncoder(base64.StdEncoding, w.out)
		enc.Write(val)
		if err := enc.Close(); err != nil {
			return err
		}

		return writeRawString("}}", w.out)
	})
}

// WriteClob writes a clob.
func (w *textWriter) WriteClob(val []byte) {
	if w.err != nil {
		return
	}
	w.err = w.writeValueStreaming(func() error {
		if err := writeRawString("{{\"", w.out); err != nil {
			return err
		}

		for _, c := range val {
			if c < 32 || c == '\\' || c == '"' || c > 0x7F {
				if err := writeEscapedChar(c, w.out); err != nil {
					return err
				}
			} else {
				if err := writeRawChar(c, w.out); err != nil {
					return err
				}
			}
		}

		return writeRawString("\"}}", w.out)
	})
}

func (w *textWriter) WriteValue(val interface{}) {
	m := Encoder{
		w:        w,
		sortMaps: true,
	}
	m.Encode(val)
}

// Finish finishes the current datagram.
func (w *textWriter) Finish() error {
	if w.err != nil {
		return w.err
	}
	if w.ctx.peek() != ctxAtTopLevel {
		w.err = errors.New("not at top level")
		return w.err
	}

	if w.opts&OptQuietFinish == 0 {
		if w.err = writeRawChar('\n', w.out); w.err != nil {
			return w.err
		}
	}

	w.fieldName = ""
	w.typeAnnotations = nil
	w.needsSeparator = false
	return nil
}
