package ion

import (
	"encoding/base64"
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

// WriteNull writes an untyped null.
func (w *textWriter) WriteNull() error {
	return w.writeValue("Writer.WriteNull", textNulls[NoType])
}

// WriteNullType writes a typed null.
func (w *textWriter) WriteNullType(t Type) error {
	return w.writeValue("Writer.WriteNullType", textNulls[t])
}

// WriteBool writes a boolean value.
func (w *textWriter) WriteBool(val bool) error {
	str := "false"
	if val {
		str = "true"
	}
	return w.writeValue("Writer.WriteBool", str)
}

// WriteInt writes an integer value.
func (w *textWriter) WriteInt(val int64) error {
	return w.writeValue("Writer.WriteInt", fmt.Sprintf("%d", val))
}

// WriteUint writes an unsigned integer value.
func (w *textWriter) WriteUint(val uint64) error {
	return w.writeValue("Writer.WriteUint", fmt.Sprintf("%d", val))
}

// WriteBigInt writes a (big) integer value.
func (w *textWriter) WriteBigInt(val *big.Int) error {
	return w.writeValue("Writer.WriteBigInt", val.String())
}

// WriteFloat writes a floating-point value.
func (w *textWriter) WriteFloat(val float64) error {
	return w.writeValue("Writer.WriteFloat", formatFloat(val))
}

// WriteDecimal writes an arbitrary-precision decimal value.
func (w *textWriter) WriteDecimal(val *Decimal) error {
	return w.writeValue("Writer.WriteDecimal", val.String())
}

// WriteTimestamp writes a timestamp.
func (w *textWriter) WriteTimestamp(val time.Time) error {
	return w.writeValue("Writer.WriteTimestamp", val.Format(time.RFC3339Nano))
}

// WriteSymbol writes a symbol.
func (w *textWriter) WriteSymbol(val string) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue("Writer.WriteSymbol"); w.err != nil {
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
	if w.err = w.beginValue("Writer.WriteString"); w.err != nil {
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
	if w.err = w.beginValue("Writer.WriteBlob"); w.err != nil {
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
	if w.err = w.beginValue("Writer.WriteBlob"); w.err != nil {
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
		w.err = w.begin("Writer.BeginList", ctxInList, '[')
	}
	return w.err
}

// EndList finishes writing a list.
func (w *textWriter) EndList() error {
	if w.err == nil {
		w.err = w.end("Writer.EndList", ctxInList, ']')
	}
	return w.err
}

// BeginSexp begins writing an s-expression.
func (w *textWriter) BeginSexp() error {
	if w.err == nil {
		w.err = w.begin("Writer.BeginSexp", ctxInSexp, '(')
	}
	return w.err
}

// EndSexp finishes writing an s-expression.
func (w *textWriter) EndSexp() error {
	if w.err == nil {
		w.err = w.end("Writer.EndSexp", ctxInSexp, ')')
	}
	return w.err
}

// BeginStruct begins writing a struct.
func (w *textWriter) BeginStruct() error {
	if w.err == nil {
		w.err = w.begin("Writer.BeginStruct", ctxInStruct, '{')
	}
	return w.err
}

// EndStruct finishes writing a struct.
func (w *textWriter) EndStruct() error {
	if w.err == nil {
		w.err = w.end("Writer.EndStruct", ctxInStruct, '}')
	}
	return w.err
}

// Finish finishes writing the current datagram.
func (w *textWriter) Finish() error {
	if w.err != nil {
		return w.err
	}
	if w.ctx.peek() != ctxAtTopLevel {
		return &UsageError{"Writer.Finish", "not at top level"}
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
func (w *textWriter) writeValue(api string, val string) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue(api); w.err != nil {
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
func (w *textWriter) beginValue(api string) error {
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
			return &UsageError{api, "field name not set"}
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
func (w *textWriter) begin(api string, t ctx, c byte) error {
	if err := w.beginValue(api); err != nil {
		return err
	}

	w.ctx.push(t)
	w.needsSeparator = false

	return writeRawChar(c, w.out)
}

// end finishes writing a container of the given type
func (w *textWriter) end(api string, t ctx, c byte) error {
	if w.ctx.peek() != t {
		return &UsageError{api, "not in that kind of container"}
	}

	if err := writeRawChar(c, w.out); err != nil {
		return err
	}

	w.clear()
	w.ctx.pop()
	w.endValue()

	return nil
}
