package ion

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"time"
)

// A cstack is a stack of containers.
type cstack struct {
	arr []*container
}

func (c *cstack) peek() *container {
	if len(c.arr) == 0 {
		return nil
	}
	return c.arr[len(c.arr)-1]
}

func (c *cstack) push(code byte) {
	c.arr = append(c.arr, &container{code: code})
}

func (c *cstack) pop() {
	if len(c.arr) == 0 {
		panic("pop called at top level")
	}
	c.arr = c.arr[:len(c.arr)-1]
}

type binaryWriterLST struct {
	writer
	cs  cstack
	lst SymbolTable

	wroteLST bool
}

// NewBinaryWriter creates a new binary writer.
func NewBinaryWriter(out io.Writer, lst SymbolTable) Writer {
	return &binaryWriterLST{
		writer: writer{
			out: out,
		},
		lst: lst,
	}
}

func (w *binaryWriterLST) write(c bufnode) error {
	p := w.cs.peek()
	if p == nil {
		return c.WriteTo(w.out)
	}
	p.Add(c)
	return nil
}

func (w *binaryWriterLST) writeTag(code byte, len int) error {
	buf := bytes.Buffer{}
	writeTag(&buf, code, uint64(len))
	return w.write(atom(buf.Bytes()))
}

func (w *binaryWriterLST) writeLST() error {
	if _, err := w.out.Write([]byte{0xE0, 0x01, 0x00, 0xEA}); err != nil {
		return err
	}

	// Prevent recursion...
	w.wroteLST = true

	return w.lst.WriteTo(w)
}

func (w *binaryWriterLST) beginValue() error {
	// Have to record/empty these before calling writeLST, which
	// will end up modifying them. Ugh.
	name := w.fieldName
	w.fieldName = ""
	tas := w.typeAnnotations
	w.typeAnnotations = nil

	if !w.wroteLST {
		if err := w.writeLST(); err != nil {
			return err
		}
	}

	if w.InStruct() {
		if name == "" {
			return errors.New("ion: field name not set")
		}

		id, ok := w.lst.FindByName(name)
		if !ok {
			return fmt.Errorf("ion: symbol '%v' not defined", name)
		}
		if id < 0 {
			panic("negative id")
		}

		if err := w.write(fieldname(id)); err != nil {
			return err
		}
	}

	if len(tas) > 0 {
		w.cs.push(0xE0)

		ids := make([]uint64, len(tas))
		idlen := uint64(0)

		for i, a := range tas {
			id, ok := w.lst.FindByName(a)
			if !ok {
				return fmt.Errorf("ion: symbol '%v' not defined", a)
			}
			if id < 0 {
				panic("negative id")
			}
			ids[i] = uint64(id)
			idlen += varUintLen(uint64(id))
		}

		buf := bytes.Buffer{}
		buf.Write(packVarUint(idlen))

		for _, id := range ids {
			buf.Write(packVarUint(id))
		}

		if err := w.write(atom(buf.Bytes())); err != nil {
			return err
		}
	}

	return nil
}

func (w *binaryWriterLST) endValue() error {
	cur := w.cs.peek()
	if cur != nil && cur.code == 0xE0 {
		// If we're in an annotation container, write it up a level now that we
		// know the length of the value.
		w.cs.pop()
		return w.write(cur)
	}
	return nil
}

func (w *binaryWriterLST) writeValue(f func() []byte) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	val := f()

	if err := w.write(atom(val)); err != nil {
		return err
	}

	return w.endValue()
}

func (w *binaryWriterLST) writeValueStreaming(f func() error) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	if err := f(); err != nil {
		return err
	}

	return w.endValue()
}

func (w *binaryWriterLST) begin(t ctxType, code byte) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	w.ctx.push(t)
	w.cs.push(code)

	return nil
}

func (w *binaryWriterLST) end(t ctxType) error {
	if w.ctx.peek() != t {
		return errors.New("ion: not in that kind of container")
	}

	cur := w.cs.peek()
	if cur != nil {
		w.cs.pop()
		if err := w.write(cur); err != nil {
			return err
		}
	}

	w.fieldName = ""
	w.typeAnnotations = nil
	w.ctx.pop()

	return w.endValue()
}

func (w *binaryWriterLST) BeginStruct() {
	if w.err != nil {
		return
	}
	w.err = w.begin(ctxInStruct, 0xD0)
}

func (w *binaryWriterLST) EndStruct() {
	if w.err != nil {
		return
	}
	w.err = w.end(ctxInStruct)
}

func (w *binaryWriterLST) BeginList() {
	if w.err != nil {
		return
	}
	w.err = w.begin(ctxInList, 0xB0)
}

func (w *binaryWriterLST) EndList() {
	if w.err != nil {
		return
	}
	w.err = w.end(ctxInList)
}

func (w *binaryWriterLST) BeginSexp() {
	if w.err != nil {
		return
	}
	w.err = w.begin(ctxInSexp, 0xC0)
}

func (w *binaryWriterLST) EndSexp() {
	if w.err != nil {
		return
	}
	w.err = w.end(ctxInSexp)
}

func (w *binaryWriterLST) WriteNull() {
	w.WriteNullWithType(NullType)
}

func (w *binaryWriterLST) WriteNullWithType(t Type) {
	if w.err != nil {
		return
	}

	w.err = w.writeValue(func() []byte {
		var b byte
		switch t {
		case NoType, NullType:
			b = 0x0F
		case BoolType:
			b = 0x1F
		case IntType:
			b = 0x2F
		case FloatType:
			b = 0x4F
		case DecimalType:
			b = 0x5F
		case TimestampType:
			b = 0x6F
		case SymbolType:
			b = 0x7F
		case StringType:
			b = 0x8F
		case ClobType:
			b = 0x9F
		case BlobType:
			b = 0xAF
		case ListType:
			b = 0xBF
		case SexpType:
			b = 0xCF
		case StructType:
			b = 0xDF
		default:
			panic(fmt.Sprintf("invalid type: %v", t))
		}

		return []byte{b}
	})
}

func (w *binaryWriterLST) WriteBool(val bool) {
	if w.err != nil {
		return
	}

	w.err = w.writeValue(func() []byte {
		if val {
			return []byte{0x11}
		}
		return []byte{0x10}
	})
}

func (w *binaryWriterLST) WriteInt(val int64) {
	if w.err != nil {
		return
	}

	w.err = w.writeValueStreaming(func() error {
		if val == 0 {
			return w.write(atom([]byte{0x20}))
		}

		code := byte(0x20)
		mag := uint64(val)

		if val < 0 {
			code = 0x30
			mag = uint64(-val)
		}

		bs := packUint(mag)

		if err := w.writeTag(code, len(bs)); err != nil {
			return err
		}
		return w.write(atom(bs))
	})
}

func (w *binaryWriterLST) WriteBigInt(val *big.Int) {
	if w.err != nil {
		return
	}

	w.err = w.writeValueStreaming(func() error {
		sign := val.Sign()
		if sign == 0 {
			return w.write(atom([]byte{0x20}))
		}

		code := byte(0x20)
		if sign < 0 {
			code = 0x30
		}

		bs := val.Bytes()

		if err := w.writeTag(code, len(bs)); err != nil {
			return err
		}
		return w.write(atom(bs))
	})
}

func (w *binaryWriterLST) WriteFloat(val float64) {
	if w.err != nil {
		return
	}

	w.err = w.writeValue(func() []byte {
		if val == 0 {
			return []byte{0x40}
		}

		bs := make([]byte, 9)
		bs[0] = 0x48

		bits := math.Float64bits(val)
		binary.BigEndian.PutUint64(bs[1:], bits)

		return bs
	})
}

func (w *binaryWriterLST) WriteDecimal(val *Decimal) {
	if w.err != nil {
		return
	}

	w.writeValueStreaming(func() error {
		coef, exp := val.CoEx()

		ebs := []byte{}
		if exp != 0 {
			ebs = packVarInt(int64(exp))
		}

		cbs := packBigInt(coef)

		if err := w.writeTag(0x50, len(cbs)+len(ebs)); err != nil {
			return err
		}

		if len(ebs) > 0 {
			if err := w.write(atom(ebs)); err != nil {
				return err
			}
		}

		if len(cbs) > 0 {
			if err := w.write(atom(cbs)); err != nil {
				return err
			}
		}

		return nil
	})
}

func (w *binaryWriterLST) WriteTimestamp(val time.Time) {
	if w.err != nil {
		return
	}

	w.err = w.writeValueStreaming(func() error {
		bs := packTime(val)
		if err := w.writeTag(0x60, len(bs)); err != nil {
			return err
		}
		return w.write(atom(bs))
	})
}

func (w *binaryWriterLST) WriteSymbol(val string) {
	if w.err != nil {
		return
	}

	id, ok := w.lst.FindByName(val)
	if !ok {
		w.err = fmt.Errorf("ion: symbol '%v' not defined in local symbol table", val)
		return
	}

	w.err = w.writeValueStreaming(func() error {
		bs := packUint(uint64(id))
		if err := w.writeTag(0x70, len(bs)); err != nil {
			return err
		}
		return w.write(atom(bs))
	})
}

func (w *binaryWriterLST) WriteString(val string) {
	if w.err != nil {
		return
	}

	w.err = w.writeValueStreaming(func() error {
		if len(val) == 0 {
			return w.write(atom([]byte{0x80}))
		}

		bs := []byte(val)

		if err := w.writeTag(0x80, len(bs)); err != nil {
			return err
		}
		return w.write(atom(bs))
	})
}

func (w *binaryWriterLST) WriteClob(val []byte) {
	if w.err != nil {
		return
	}

	w.err = w.writeValueStreaming(func() error {
		if err := w.writeTag(0x90, len(val)); err != nil {
			return err
		}
		return w.write(atom(val))
	})
}

func (w *binaryWriterLST) WriteBlob(val []byte) {
	if w.err != nil {
		return
	}

	w.err = w.writeValueStreaming(func() error {
		if err := w.writeTag(0xA0, len(val)); err != nil {
			return err
		}
		return w.write(atom(val))
	})
}

func (w *binaryWriterLST) WriteValue(val interface{}) {
	w.err = errors.New("not yet implemented")
}

func (w *binaryWriterLST) Finish() error {
	if w.err != nil {
		return w.err
	}
	if w.ctx.peek() != ctxAtTopLevel {
		w.err = errors.New("ion: not at top level")
		return w.err
	}

	// TODO: Flush all them buffers mate!

	return nil
}
