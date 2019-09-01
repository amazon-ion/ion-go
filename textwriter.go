package ion

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"
)

// TextWriterOpts defines a set of bit flag options for text writers.
type TextWriterOpts uint8

const (
	// TextWriterQuietFinish disables emiting a newline in Finish(). Convenient if you
	// know you're only emiting one datagram; dangerous if there's a chance you're going
	// to emit another datagram using the same Writer.
	TextWriterQuietFinish TextWriterOpts = 1
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

// // writeValue writes a value whose raw encoding is produced by the
// // given function.
// func (w *textWriter) writeValue(f func() string) error {
// 	if err := w.beginValue(); err != nil {
// 		return err
// 	}

// 	sym := f()
// 	if err := writeRawString(sym, w.out); err != nil {
// 		return err
// 	}

// 	w.endValue()
// 	return nil
// }

// // writeValue writes a value by calling the given function, which is
// // expected to write the raw value to w.out.
// func (w *textWriter) writeValueStreaming(f func() error) error {
// 	if err := w.beginValue(); err != nil {
// 		return err
// 	}

// 	if err := f(); err != nil {
// 		return err
// 	}

// 	w.endValue()
// 	return nil
// }

// WriteNull writes an untyped null.
func (w *textWriter) WriteNull() error {
	return w.WriteNullType(NoType)
}

// WriteNullType writes a typed null.
func (w *textWriter) WriteNullType(t Type) error {
	return w.writeValue(textNulls[t])
}

// WriteBool writes a boolean value.
func (w *textWriter) WriteBool(val bool) error {
	str := "false"
	if val {
		str = "true"
	}
	return w.writeValue(str)
}

// WriteInt writes an integer value.
func (w *textWriter) WriteInt(val int64) error {
	return w.writeValue(fmt.Sprintf("%d", val))
}

// WriteBigInt writes a (big) integer value.
func (w *textWriter) WriteBigInt(val *big.Int) error {
	return w.writeValue(val.String())
}

// WriteFloat writes a floating-point value.
func (w *textWriter) WriteFloat(val float64) error {
	return w.writeValue(formatFloat(val))
}

// WriteDecimal writes an arbitrary-precision decimal value.
func (w *textWriter) WriteDecimal(val *Decimal) error {
	return w.writeValue(val.String())
}

// WriteTimestamp writes a timestamp.
func (w *textWriter) WriteTimestamp(val time.Time) error {
	return w.writeValue(val.Format(time.RFC3339Nano))
}

// WriteSymbol writes a symbol.
func (w *textWriter) WriteSymbol(val string) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue(); w.err != nil {
		return w.err
	}

	if w.err = writeSymbol(val, w.out); w.err != nil {
		return w.err
	}

	w.endValue()
	return nil
}

// WriteString writes a string.
func (w *textWriter) WriteString(val string) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue(); w.err != nil {
		return w.err
	}

	if w.err = writeRawChar('"', w.out); w.err != nil {
		return w.err
	}
	if w.err = writeEscapedString(val, w.out); w.err != nil {
		return w.err
	}
	if w.err = writeRawChar('"', w.out); w.err != nil {
		return w.err
	}

	w.endValue()
	return nil
}

// WriteClob writes a clob.
func (w *textWriter) WriteClob(val []byte) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue(); w.err != nil {
		return w.err
	}

	if w.err = writeRawString("{{\"", w.out); w.err != nil {
		return w.err
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
	if w.err = writeRawString("\"}}", w.out); w.err != nil {
		return w.err
	}

	w.endValue()
	return nil
}

// WriteBlob writes a blob.
func (w *textWriter) WriteBlob(val []byte) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue(); w.err != nil {
		return w.err
	}

	if w.err = writeRawString("{{", w.out); w.err != nil {
		return w.err
	}

	enc := base64.NewEncoder(base64.StdEncoding, w.out)
	enc.Write(val)
	if w.err = enc.Close(); w.err != nil {
		return w.err
	}

	if w.err = writeRawString("}}", w.out); w.err != nil {
		return w.err
	}

	w.endValue()
	return nil
}

// BeginList begins writing a list.
func (w *textWriter) BeginList() error {
	if w.err == nil {
		w.err = w.begin(ctxInList, '[')
	}
	return w.err
}

// EndList finishes writing a list.
func (w *textWriter) EndList() error {
	if w.err == nil {
		w.err = w.end(ctxInList, ']')
	}
	return w.err
}

// BeginSexp begins writing an s-expression.
func (w *textWriter) BeginSexp() error {
	if w.err == nil {
		w.err = w.begin(ctxInSexp, '(')
	}
	return w.err
}

// EndSexp finishes writing an s-expression.
func (w *textWriter) EndSexp() error {
	if w.err == nil {
		w.err = w.end(ctxInSexp, ')')
	}
	return w.err
}

// BeginStruct begins writing a struct.
func (w *textWriter) BeginStruct() error {
	if w.err == nil {
		w.err = w.begin(ctxInStruct, '{')
	}
	return w.err
}

// EndStruct finishes writing a struct.
func (w *textWriter) EndStruct() error {
	if w.err == nil {
		w.err = w.end(ctxInStruct, '}')
	}
	return w.err
}

// Finish finishes writing the current datagram.
func (w *textWriter) Finish() error {
	if w.err != nil {
		return w.err
	}
	if w.ctx.peek() != ctxAtTopLevel {
		return errors.New("ion: Finish not at top level")
	}

	if w.opts&TextWriterQuietFinish == 0 {
		if w.err = writeRawChar('\n', w.out); w.err != nil {
			return w.err
		}
		w.needsSeparator = false
	}

	w.clear()
	return nil
}

// writeValue writes a stringified value to the output stream.
func (w *textWriter) writeValue(val string) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue(); w.err != nil {
		return w.err
	}

	if w.err = writeRawString(val, w.out); w.err != nil {
		return w.err
	}

	w.endValue()
	return nil
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

	if w.inStruct() {
		if w.fieldName == "" {
			return errors.New("ion: field name not set")
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

	if len(w.annotations) > 0 {
		as := w.annotations
		w.annotations = nil

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
func (w *textWriter) begin(t ctx, c byte) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	w.ctx.push(t)
	w.needsSeparator = false

	return writeRawChar(c, w.out)
}

// end finishes writing a container of the given type
func (w *textWriter) end(t ctx, c byte) error {
	if w.ctx.peek() != t {
		return errors.New("ion: End called with wrong container type")
	}

	if err := writeRawChar(c, w.out); err != nil {
		return err
	}

	w.clear()
	w.ctx.pop()
	w.endValue()

	return nil
}
