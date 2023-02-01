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
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/big"
	"time"
)

// A binaryWriter writes binary ion.
type binaryWriter struct {
	writer
	bufs bufstack

	lst  SymbolTable
	lstb SymbolTableBuilder

	wroteLST bool
}

// NewBinaryWriter creates a new binary writer that will construct a
// local symbol table as it is written to.
func NewBinaryWriter(out io.Writer, sts ...SharedSymbolTable) Writer {
	w := &binaryWriter{
		writer: writer{
			out: out,
		},
		lstb: NewSymbolTableBuilder(sts...),
	}
	w.bufs.push(&datagram{})
	return w
}

// NewBinaryWriterLST creates a new binary writer with a pre-built local
// symbol table.
func NewBinaryWriterLST(out io.Writer, lst SymbolTable) Writer {
	return &binaryWriter{
		writer: writer{
			out: out,
		},
		lst: lst,
	}
}

// WriteNull writes an untyped null.
func (w *binaryWriter) WriteNull() error {
	return w.writeValue("Writer.WriteNull", []byte{0x0F})
}

// WriteNullType writes a typed null.
func (w *binaryWriter) WriteNullType(t Type) error {
	return w.writeValue("Writer.WriteNullType", []byte{binaryNulls[t]})
}

// WriteBool writes a bool.
func (w *binaryWriter) WriteBool(val bool) error {
	b := byte(0x10)
	if val {
		b = 0x11
	}
	return w.writeValue("Writer.WriteBool", []byte{b})
}

// WriteInt writes an integer.
func (w *binaryWriter) WriteInt(val int64) error {
	if val == 0 {
		return w.writeValue("Writer.WriteInt", []byte{0x20})
	}

	code := byte(0x20)
	mag := uint64(val)

	if val < 0 {
		code = 0x30
		mag = uint64(-val)
	}

	length := uintLen(mag)
	bufLength := length + tagLen(length)

	buf := make([]byte, 0, bufLength)
	buf = appendTag(buf, code, length)
	buf = appendUint(buf, mag)

	return w.writeValue("Writer.WriteInt", buf)
}

// WriteUint writes an unsigned integer.
func (w *binaryWriter) WriteUint(val uint64) error {
	if val == 0 {
		return w.writeValue("Writer.WriteUint", []byte{0x20})
	}

	length := uintLen(val)
	bufLength := length + tagLen(length)

	buf := make([]byte, 0, bufLength)
	buf = appendTag(buf, 0x20, length)
	buf = appendUint(buf, val)

	return w.writeValue("Writer.WriteUint", buf)
}

// WriteBigInt writes a big integer.
func (w *binaryWriter) WriteBigInt(val *big.Int) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue("Writer.WriteBigInt"); w.err != nil {
		return w.err
	}

	if w.err = w.writeBigInt(val); w.err != nil {
		return w.err
	}

	w.err = w.endValue()
	return w.err
}

// WriteBigInt writes the actual big integer value.
func (w *binaryWriter) writeBigInt(val *big.Int) error {
	sign := val.Sign()
	if sign == 0 {
		return w.write([]byte{0x20})
	}

	code := byte(0x20)
	if sign < 0 {
		code = 0x30
	}

	bs := val.Bytes()

	bl := uint64(len(bs))
	if bl < 64 {
		bufLength := bl + tagLen(bl)
		buf := make([]byte, 0, bufLength)

		buf = appendTag(buf, code, bl)
		buf = append(buf, bs...)
		return w.write(buf)
	}

	// no sense in copying, emit tag separately.
	if err := w.writeTag(code, bl); err != nil {
		return err
	}
	return w.write(bs)
}

// WriteFloat writes a floating-point value.
func (w *binaryWriter) WriteFloat(val float64) error {
	if val == 0 && !math.Signbit(val) {
		// Positive zero is represented as just one byte.
		return w.writeValue("Writer.WriteFloat", []byte{0x40})
	} else if math.IsNaN(val) {
		return w.writeValue("Writer.WriteFloat", []byte{0x44, 0x7F, 0xC0, 0x00, 0x00})
	}

	var bs []byte

	// Can this be losslessly represented as a float32?
	if val == float64(float32(val)) {
		bs = make([]byte, 5)
		bs[0] = 0x44

		bits := math.Float32bits(float32(val))
		binary.BigEndian.PutUint32(bs[1:], bits)
	} else {
		bs = make([]byte, 9)
		bs[0] = 0x48

		bits := math.Float64bits(val)
		binary.BigEndian.PutUint64(bs[1:], bits)
	}

	return w.writeValue("Writer.WriteFloat", bs)
}

// WriteDecimal writes a decimal value.
func (w *binaryWriter) WriteDecimal(val *Decimal) error {
	coef, exp := val.CoEx()

	// If the value is positive 0. (aka 0d0) then L is zero, there are no length or
	// representation fields, and the entire value is encoded as the single byte 0x50.
	if coef.Sign() == 0 && int64(exp) == 0 && !val.isNegZero {
		return w.writeValue("Writer.WriteDecimal", []byte{0x50})
	}

	// Otherwise, length or representation fields are present and must be considered.
	vlength := varIntLen(int64(exp))

	if val.isNegZero {
		vlength++
	} else {
		vlength += bigIntLen(coef)
	}

	bufLength := vlength + tagLen(vlength)
	buf := make([]byte, 0, bufLength)

	buf = appendTag(buf, 0x50, vlength)
	buf = appendVarInt(buf, int64(exp))

	if val.isNegZero {
		buf = append(buf, 0x80)
	} else {
		buf = appendBigInt(buf, coef)
	}

	return w.writeValue("Writer.WriteDecimal", buf)
}

// WriteTimestamp writes a timestamp value.
func (w *binaryWriter) WriteTimestamp(val Timestamp) error {
	_, offset := val.dateTime.Zone()
	offset /= 60
	val.dateTime = val.dateTime.In(time.UTC)

	vlength := timestampLen(offset, val)
	bufLength := vlength + tagLen(vlength)

	buf := make([]byte, 0, bufLength)

	buf = appendTag(buf, 0x60, vlength)
	buf = appendTimestamp(buf, offset, val)

	return w.writeValue("Writer.WriteTimestamp", buf)
}

// WriteSymbol writes a symbol value given a SymbolToken.
func (w *binaryWriter) WriteSymbol(val SymbolToken) error {
	var id uint64
	if val.LocalSID != SymbolIDUnknown {
		id = uint64(val.LocalSID)
	} else if val.Text != nil {
		id, w.err = w.resolveFromSymbolTable("Writer.WriteSymbol", *val.Text)
		if w.err != nil {
			return w.err
		}
	} else {
		return &UsageError{"Writer.WriteSymbol", "symbol token without defined text or symbol id is invalid"}
	}

	return w.writeSymbolFromID("Writer.WriteSymbol", id)
}

// WriteSymbolFromString writes a symbol value given a string that is expected to be in the symbol table.
// Returns an error if string is not in symbol table.
func (w *binaryWriter) WriteSymbolFromString(val string) error {
	var id uint64
	id, w.err = w.resolve("Writer.WriteSymbolFromString", val)
	if w.err != nil {
		return w.err
	}

	return w.writeSymbolFromID("Writer.WriteSymbolFromString", id)
}

func (w *binaryWriter) writeSymbolFromID(api string, id uint64) error {
	vlength := uintLen(id)
	bufLength := vlength + tagLen(vlength)
	buf := make([]byte, 0, bufLength)

	buf = appendTag(buf, 0x70, vlength)
	buf = appendUint(buf, id)

	return w.writeValue(api, buf)
}

// WriteString writes a string.
func (w *binaryWriter) WriteString(val string) error {
	if len(val) == 0 {
		return w.writeValue("Writer.WriteString", []byte{0x80})
	}

	vlength := uint64(len(val))
	bufLength := vlength + tagLen(vlength)
	buf := make([]byte, 0, bufLength)

	buf = appendTag(buf, 0x80, vlength)
	buf = append(buf, val...)

	return w.writeValue("Writer.WriteString", buf)
}

// WriteClob writes a clob.
func (w *binaryWriter) WriteClob(val []byte) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue("Writer.WriteClob"); w.err != nil {
		return w.err
	}

	if w.err = w.writeLob(0x90, val); w.err != nil {
		return w.err
	}

	w.err = w.endValue()
	return w.err
}

// WriteBlob writes a blob.
func (w *binaryWriter) WriteBlob(val []byte) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue("Writer.WriteBlob"); w.err != nil {
		return w.err
	}

	if w.err = w.writeLob(0xA0, val); w.err != nil {
		return w.err
	}

	w.err = w.endValue()
	return w.err
}

func (w *binaryWriter) writeLob(code byte, val []byte) error {
	vlength := uint64(len(val))

	if vlength < 64 {
		bufLength := vlength + tagLen(vlength)
		buf := make([]byte, 0, bufLength)

		buf = appendTag(buf, code, vlength)
		buf = append(buf, val...)

		return w.write(buf)
	}

	if err := w.writeTag(code, vlength); err != nil {
		return err
	}
	return w.write(val)
}

// BeginList begins writing a list.
func (w *binaryWriter) BeginList() error {
	if w.err == nil {
		w.err = w.begin("Writer.BeginList", ctxInList, 0xB0)
	}
	return w.err
}

// EndList finishes writing a list.
func (w *binaryWriter) EndList() error {
	if w.err == nil {
		w.err = w.end("Writer.EndList", ctxInList)
	}
	return w.err
}

// BeginSexp begins writing an s-expression.
func (w *binaryWriter) BeginSexp() error {
	if w.err == nil {
		w.err = w.begin("Writer.BeginSexp", ctxInSexp, 0xC0)
	}
	return w.err
}

// EndSexp finishes writing an s-expression.
func (w *binaryWriter) EndSexp() error {
	if w.err == nil {
		w.err = w.end("Writer.EndSexp", ctxInSexp)
	}
	return w.err
}

// BeginStruct begins writing a struct.
func (w *binaryWriter) BeginStruct() error {
	if w.err == nil {
		w.err = w.begin("Writer.BeginStruct", ctxInStruct, 0xD0)
	}
	return w.err
}

// EndStruct finishes writing a struct.
func (w *binaryWriter) EndStruct() error {
	if w.err == nil {
		w.err = w.end("Writer.EndStruct", ctxInStruct)
	}
	return w.err
}

// Finish finishes writing a datagram.
func (w *binaryWriter) Finish() error {
	if w.err != nil {
		return w.err
	}
	if w.ctx.peek() != ctxAtTopLevel {
		return &UsageError{"Writer.Finish", "not at top level"}
	}

	w.clear()
	w.wroteLST = false

	seq := w.bufs.peek()
	if seq != nil {
		w.bufs.pop()
		if w.bufs.peek() != nil {
			panic("at top level but too many bufseqs")
		}

		lst := w.lstb.Build()
		if err := w.writeLST(lst); err != nil {
			return err
		}
		if w.err = w.emit(seq); w.err != nil {
			return w.err
		}
	}

	return nil
}

// Emit emits the given node. If we're currently at the top level, that
// means actually emitting to the output stream. If not, we emit append
// to the current bufseq.
func (w *binaryWriter) emit(node bufnode) error {
	s := w.bufs.peek()
	if s == nil {
		return node.EmitTo(w.out)
	}
	s.Append(node)
	return nil
}

// Write emits the given bytes as an atom.
func (w *binaryWriter) write(bs []byte) error {
	return w.emit(atom(bs))
}

// WriteValue writes a serialized value to the output stream.
func (w *binaryWriter) writeValue(api string, val []byte) error {
	if w.err != nil {
		return w.err
	}
	if w.err = w.beginValue(api); w.err != nil {
		return w.err
	}

	if w.err = w.write(val); w.err != nil {
		return w.err
	}

	w.err = w.endValue()
	return w.err
}

// WriteTag writes out a type+length tag. Use me when you've already got the value to
// be written as a []byte and don't want to copy it.
func (w *binaryWriter) writeTag(code byte, length uint64) error {
	tl := tagLen(length)

	tag := make([]byte, 0, tl)
	tag = appendTag(tag, code, length)

	return w.write(tag)
}

// WriteLST writes out a local symbol table.
func (w *binaryWriter) writeLST(lst SymbolTable) error {
	if err := w.write([]byte{0xE0, 0x01, 0x00, 0xEA}); err != nil {
		return err
	}
	return lst.WriteTo(w)
}

// BeginValue begins the process of writing a value by writing out
// its field name and annotations.
func (w *binaryWriter) beginValue(api string) error {
	// We have to record/empty these before calling writeLST, which
	// will end up using/modifying them. Ugh.
	name := w.fieldName
	as := w.annotations
	w.clear()

	// If we have a local symbol table and haven't written it out yet, do that now.
	if w.lst != nil && !w.wroteLST {
		w.wroteLST = true
		if err := w.writeLST(w.lst); err != nil {
			return err
		}
	}

	if w.IsInStruct() {
		if name == nil {
			return &UsageError{api, "field name not set"}
		}

		var id uint64
		if name.LocalSID != SymbolIDUnknown {
			id = uint64(name.LocalSID)
		} else if name.Text != nil {
			var err error
			id, err = w.resolve(api, *name.Text)
			if err != nil {
				return err
			}
		} else {
			return &UsageError{api, "field name symbol token does not have defined text or symbol id."}
		}

		buf := make([]byte, 0, 10)
		buf = appendVarUint(buf, id)
		if err := w.write(buf); err != nil {
			return err
		}
	}

	if len(as) > 0 {
		ids := make([]uint64, len(as))
		idlen := uint64(0)

		var id uint64
		var err error
		for i, a := range as {
			if a.Text != nil {
				id, err = w.resolve(api, *a.Text)
				if err != nil {
					return err
				}
			} else if a.LocalSID != SymbolIDUnknown {
				id = uint64(a.LocalSID)
			} else {
				return &UsageError{api, "invalid annotation symbol token"}
			}

			ids[i] = id
			idlen += varUintLen(id)
		}

		bufLength := idlen + varUintLen(idlen)
		buf := make([]byte, 0, bufLength)

		buf = appendVarUint(buf, idlen)
		for _, id := range ids {
			buf = appendVarUint(buf, id)
		}

		// https://github.com/amazon-ion/ion-go/issues/120
		w.bufs.push(&container{code: 0xE0})
		if err := w.write(buf); err != nil {
			return err
		}
	}

	return nil
}

// EndValue ends the process of writing a value by flushing it and its annotations
// up a level, if needed.
func (w *binaryWriter) endValue() error {
	seq := w.bufs.peek()
	if seq != nil {
		if c, ok := seq.(*container); ok && c.code == 0xE0 {
			w.bufs.pop()
			return w.emit(seq)
		}
	}
	return nil
}

// Begin begins writing a new container.
func (w *binaryWriter) begin(api string, t ctx, code byte) error {
	if err := w.beginValue(api); err != nil {
		return err
	}

	w.ctx.push(t)
	w.bufs.push(&container{code: code})

	return nil
}

// End ends writing a container, emitting its buffered contents up a level in the stack.
func (w *binaryWriter) end(api string, t ctx) error {
	if w.ctx.peek() != t {
		return &UsageError{api, "not in that kind of container"}
	}

	seq := w.bufs.peek()
	if seq != nil {
		w.bufs.pop()
		if err := w.emit(seq); err != nil {
			return err
		}
	}

	w.clear()
	w.ctx.pop()

	return w.endValue()
}

// Resolve resolves a symbol to its ID.
func (w *binaryWriter) resolve(api, sym string) (uint64, error) {
	if id, ok := symbolIdentifier(sym); ok {
		return uint64(id), nil
	}

	return w.resolveFromSymbolTable(api, sym)
}

func (w *binaryWriter) resolveFromSymbolTable(api, sym string) (uint64, error) {
	if w.lst != nil {
		id, ok := w.lst.FindByName(sym)
		if !ok {
			return 0, &UsageError{api, fmt.Sprintf("symbol '%v' not defined", sym)}
		}
		return id, nil
	}

	id, _ := w.lstb.Add(sym)
	return id, nil
}
