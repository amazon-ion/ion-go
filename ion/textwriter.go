/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package ion

import (
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
)

// TextWriterOpts defines a set of bit flag options for text writers.
type TextWriterOpts uint8

const (
	// TextWriterQuietFinish disables emiting a newline in Finish(). Convenient if you
	// know you're only emiting one datagram; dangerous if there's a chance you're going
	// to emit another datagram using the same Writer.
	TextWriterQuietFinish TextWriterOpts = 1

	// TextWriterPretty enables pretty-printing mode.
	TextWriterPretty TextWriterOpts = 2
)

// textWriter is a writer that writes human-readable text
type textWriter struct {
	writer
	opts           TextWriterOpts
	needsSeparator bool
	emptyContainer bool
	emptyStream    bool
	indent         int

	lstb     SymbolTableBuilder
	wroteLST bool
}

// NewTextWriter returns a new text writer that will construct a
// local symbol table as it is written to.
func NewTextWriter(out io.Writer, sts ...SharedSymbolTable) Writer {
	return NewTextWriterOpts(out, 0, sts...)
}

// NewTextWriterOpts returns a new text writer with the given options.
func NewTextWriterOpts(out io.Writer, opts TextWriterOpts, sts ...SharedSymbolTable) Writer {
	return &textWriter{
		writer:      writer{out: out},
		opts:        opts,
		emptyStream: true,
		lstb:        NewSymbolTableBuilder(sts...),
	}
}

// WriteNull writes an untyped null.
func (w *textWriter) WriteNull() error {
	return w.writeValue("Writer.WriteNull", textNulls[NoType], writeRawString)
}

// WriteNullType writes a typed null.
func (w *textWriter) WriteNullType(t Type) error {
	return w.writeValue("Writer.WriteNullType", textNulls[t], writeRawString)
}

// WriteBool writes a boolean value.
func (w *textWriter) WriteBool(val bool) error {
	str := "false"
	if val {
		str = "true"
	}
	return w.writeValue("Writer.WriteBool", str, writeRawString)
}

// WriteInt writes an integer value.
func (w *textWriter) WriteInt(val int64) error {
	return w.writeValue("Writer.WriteInt", fmt.Sprintf("%d", val), writeRawString)
}

// WriteUint writes an unsigned integer value.
func (w *textWriter) WriteUint(val uint64) error {
	return w.writeValue("Writer.WriteUint", fmt.Sprintf("%d", val), writeRawString)
}

// WriteBigInt writes a (big) integer value.
func (w *textWriter) WriteBigInt(val *big.Int) error {
	return w.writeValue("Writer.WriteBigInt", val.String(), writeRawString)
}

// WriteFloat writes a floating-point value.
func (w *textWriter) WriteFloat(val float64) error {
	return w.writeValue("Writer.WriteFloat", formatFloat(val), writeRawString)
}

// WriteDecimal writes an arbitrary-precision decimal value.
func (w *textWriter) WriteDecimal(val *Decimal) error {
	return w.writeValue("Writer.WriteDecimal", val.String(), writeRawString)
}

// WriteTimestamp writes a timestamp.
func (w *textWriter) WriteTimestamp(val Timestamp) error {
	return w.writeValue("Writer.WriteTimestamp", val.String(), writeRawString)
}

// WriteSymbol writes a symbol given a SymbolToken.
func (w *textWriter) WriteSymbol(val SymbolToken) error {
	return w.writeValue("Writer.WriteSymbol", val, writeSymbol)
}

// WriteSymbolFromString writes a symbol given a string.
func (w *textWriter) WriteSymbolFromString(val string) error {
	return w.writeValue("Writer.WriteSymbolFromString", val, writeSymbolFromString)
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
	_, err := enc.Write(val)
	if err != nil {
		return err
	}
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

	if !w.emptyStream && w.opts&TextWriterQuietFinish == 0 {
		if w.err = writeRawChar('\n', w.out); w.err != nil {
			return w.err
		}
		w.needsSeparator = false
		w.emptyStream = true
	}

	w.clear()
	return nil
}

// pretty returns true if we're pretty-printing.
func (w *textWriter) pretty() bool {
	return w.opts&TextWriterPretty == TextWriterPretty
}

// writeValue writes a stringified value to the output stream.
func (w *textWriter) writeValue(api string, val interface{}, fn func(interface{}, io.Writer) error) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue(api); w.err != nil {
		return w.err
	}

	if w.err = fn(val, w.out); w.err != nil {
		return w.err
	}

	w.endValue()
	return nil
}

// beginValue begins the process of writing a value, by writing out
// a separator (if needed), field name (if in a struct), and type
// annotations (if any).
func (w *textWriter) beginValue(api string) error {
	// We have to record/empty these before calling w.lst.WriteTo(), which
	// will end up using/modifying them.
	name := w.fieldName
	as := w.annotations
	w.clear()

	// If we have a local symbol table and haven't written it out yet, do that now.
	if !w.wroteLST {
		w.wroteLST = true
		lst := w.lstb.Build()
		if err := lst.WriteTo(w); err != nil {
			return err
		}
	}

	if w.needsSeparator {
		if err := w.writeSeparator(); err != nil {
			return err
		}
	}

	if w.emptyContainer {
		if w.pretty() {
			if err := writeRawChar('\n', w.out); err != nil {
				return err
			}
		}
	}

	if w.pretty() {
		if err := w.writeIndent(); err != nil {
			return err
		}
	}

	if w.IsInStruct() {
		w.fieldName = name
		if err := w.writeFieldName(api); err != nil {
			return err
		}
	}

	w.annotations = append(w.annotations, as...)
	if len(w.annotations) > 0 {
		if err := w.writeAnnotations(); err != nil {
			return err
		}
	}

	return nil
}

// writeSeparator writes out the character or characters that separate values.
func (w *textWriter) writeSeparator() error {
	var sep string

	switch w.ctx.peek() {
	case ctxInStruct, ctxInList:
		// In a struct or a list, values are separated by commas.
		if w.pretty() {
			sep = ",\n"
		} else {
			sep = ","
		}

	case ctxInSexp:
		// In an sexp, values are separated by whitespace.
		if w.pretty() {
			sep = "\n"
		} else {
			sep = " "
		}

	default:
		// At the top level, values are separated by newlines.
		sep = "\n"
	}

	return writeRawString(sep, w.out)
}

// writeFieldName writes a field name inside a struct.
func (w *textWriter) writeFieldName(api string) error {
	if w.fieldName == nil {
		return &UsageError{api, "field name not set"}
	}
	name := w.fieldName
	w.fieldName = nil

	if err := writeSymbol(*name, w.out); err != nil {
		return err
	}

	sep := ":"
	if w.pretty() {
		sep = ": "
	}

	return writeRawString(sep, w.out)
}

// writeAnnotations writes out the annotations for a value.
func (w *textWriter) writeAnnotations() error {
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

	return nil
}

// endValue finishes the process of writing a value.
func (w *textWriter) endValue() {
	w.needsSeparator = true
	w.emptyContainer = false
	w.emptyStream = false
}

// begin starts writing a container of the given type.
func (w *textWriter) begin(api string, t ctx, c byte) error {
	if err := w.beginValue(api); err != nil {
		return err
	}

	w.ctx.push(t)
	w.indent++
	w.needsSeparator = false
	w.emptyContainer = true

	return writeRawChar(c, w.out)
}

// end finishes writing a container of the given type
func (w *textWriter) end(api string, t ctx, c byte) error {
	if w.ctx.peek() != t {
		return &UsageError{api, "not in that kind of container"}
	}

	w.indent--

	if !w.emptyContainer && w.pretty() {
		if err := writeRawChar('\n', w.out); err != nil {
			return err
		}
		if err := w.writeIndent(); err != nil {
			return err
		}
	}

	if err := writeRawChar(c, w.out); err != nil {
		return err
	}

	w.clear()
	w.ctx.pop()
	w.endValue()

	return nil
}

// writeIndent writes out tabs to indent a pretty-printed value.
func (w *textWriter) writeIndent() error {
	for i := 0; i < w.indent; i++ {
		if err := writeRawChar('\t', w.out); err != nil {
			return err
		}
	}
	return nil
}
