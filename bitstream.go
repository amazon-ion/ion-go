package ion

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

type bss uint8

const (
	bssBeforeValue bss = iota
	bssOnValue
	bssBeforeFieldID
	bssOnFieldID
)

type bitcode uint8

const (
	bitcodeNone bitcode = iota
	bitcodeEOF
	bitcodeBVM
	bitcodeNull
	bitcodeBool
	bitcodeInt
	bitcodeNegInt
	bitcodeFloat
	bitcodeDecimal
	bitcodeTimestamp
	bitcodeSymbol
	bitcodeString
	bitcodeClob
	bitcodeBlob
	bitcodeList
	bitcodeSexp
	bitcodeStruct
	bitcodeFieldID
	bitcodeAnnotation
)

func (b bitcode) String() string {
	switch b {
	case bitcodeNone:
		return "none"
	case bitcodeEOF:
		return "eof"
	case bitcodeBVM:
		return "bvm"
	case bitcodeBool:
		return "bool"
	case bitcodeInt:
		return "int"
	case bitcodeNegInt:
		return "negint"
	case bitcodeFloat:
		return "float"
	case bitcodeDecimal:
		return "decimal"
	case bitcodeTimestamp:
		return "timestamp"
	case bitcodeSymbol:
		return "symbol"
	case bitcodeString:
		return "string"
	case bitcodeClob:
		return "clob"
	case bitcodeBlob:
		return "blob"
	case bitcodeList:
		return "list"
	case bitcodeSexp:
		return "sexp"
	case bitcodeStruct:
		return "struct"
	case bitcodeFieldID:
		return "fieldid"
	case bitcodeAnnotation:
		return "annotation"
	default:
		return fmt.Sprintf("<invalid bitcode 0x%2X>", uint8(b))
	}
}

type bitnode struct {
	code bitcode
	end  uint64
}

type bitstack struct {
	arr []bitnode
}

func (b *bitstack) empty() bool {
	return len(b.arr) == 0
}

func (b *bitstack) peek() bitnode {
	if len(b.arr) == 0 {
		return bitnode{}
	}
	return b.arr[len(b.arr)-1]
}

func (b *bitstack) push(code bitcode, end uint64) {
	b.arr = append(b.arr, bitnode{code, end})
}

func (b *bitstack) pop() {
	if len(b.arr) == 0 {
		panic("pop called on empty bitstack")
	}
	b.arr = b.arr[:len(b.arr)-1]
}

type bitstream struct {
	in    *bufio.Reader
	pos   uint64
	state bss
	stack bitstack

	code bitcode
	null bool
	len  uint64
}

func (b *bitstream) Init(in io.Reader) {
	b.in = bufio.NewReader(in)
}

func (b *bitstream) InitBytes(in []byte) {
	b.Init(bytes.NewReader(in))
}

func (b *bitstream) Code() bitcode {
	return b.code
}

func (b *bitstream) Null() bool {
	return b.null
}

func (b *bitstream) Len() uint64 {
	return b.len
}

func (b *bitstream) Next() error {
	// If we have an unread value, skip over it to the next one.
	switch b.state {
	case bssOnValue, bssOnFieldID:
		if err := b.SkipValue(); err != nil {
			return err
		}
	}

	// If we're at the end of the current container, stop and make the user step out.
	if !b.stack.empty() {
		cur := b.stack.peek()
		if b.pos == cur.end {
			b.code = bitcodeEOF
			return nil
		}
	}

	// If it's time to read a field id, do that.
	if b.state == bssBeforeFieldID {
		b.code = bitcodeFieldID
		b.state = bssOnFieldID
		return nil
	}

	// Otherwise it's time to read a value. Read the tag byte.
	c, err := b.read()
	if err != nil {
		return err
	}

	// Found the actual end of the file.
	if c == -1 {
		b.code = bitcodeEOF
		return nil
	}

	code, len := parseTag(c)
	if code == bitcodeNone {
		return fmt.Errorf("ion: invalid tag byte: 0x%X", c)
	}

	b.state = bssOnValue

	// This value is actually a BVM. It's invalid if we're not at the top level.
	if code == bitcodeAnnotation && len == 0 {
		if !b.stack.empty() {
			return errors.New("ion: BVM in a container")
		}
		b.code = bitcodeBVM
		b.len = 3
		return nil
	}

	// This value is actually a null.
	if len == 0x0F {
		b.code = code
		b.null = true
		return nil
	}

	// This value's actual len is encoded as a separate varUint.
	if len == 0x0E {
		len, err = b.readVarUint()
		if err != nil {
			return err
		}
	}

	b.code = code
	b.len = len
	return nil
}

func (b *bitstream) SkipValue() error {
	switch b.state {
	case bssBeforeFieldID, bssBeforeValue:
		return nil

	case bssOnFieldID:
		if err := b.skipVarUint(); err != nil {
			return err
		}
		b.state = bssBeforeValue

	case bssOnValue:
		if b.len > 0 {
			if err := b.skip(b.len); err != nil {
				return err
			}

			if b.stack.peek().code == bitcodeStruct {
				b.state = bssBeforeFieldID
			} else {
				b.state = bssBeforeValue
			}
		}
	}

	b.code = bitcodeNone
	b.null = false
	b.len = 0

	return nil
}

func (b *bitstream) StepIn() {
	switch b.code {
	case bitcodeStruct:
		b.state = bssBeforeFieldID

	case bitcodeList, bitcodeSexp:
		b.state = bssBeforeValue

	default:
		panic(fmt.Sprintf("called StepIn with code=%v", b.code))
	}

	b.stack.push(b.code, b.pos+b.len)
	b.code = bitcodeNone
	b.len = 0
}

func (b *bitstream) StepOut() error {
	if b.stack.empty() {
		panic("called StepOut at top level")
	}

	cur := b.stack.peek()
	b.stack.pop()

	if cur.end < b.pos {
		panic("end greater than b.pos")
	}

	diff := cur.end - b.pos
	if err := b.skip(diff); err != nil {
		return err
	}

	if b.stack.peek().code == bitcodeStruct {
		b.state = bssBeforeFieldID
	} else {
		b.state = bssBeforeValue
	}

	b.code = bitcodeNone
	b.null = false
	b.len = 0

	return nil
}

func (b *bitstream) ReadBVM() (byte, byte, error) {
	if b.code != bitcodeBVM {
		return 0, 0, errors.New("ion: not a bvm")
	}

	major, err := b.read()
	if err != nil {
		return 0, 0, err
	}
	if major == -1 {
		return 0, 0, errors.New("ion: unexpected end of input")
	}

	minor, err := b.read()
	if err != nil {
		return 0, 0, err
	}
	if minor == -1 {
		return 0, 0, errors.New("ion: unexpected end of input")
	}

	end, err := b.read()
	if err != nil {
		return 0, 0, err
	}
	if end == -1 {
		return 0, 0, errors.New("ion: unexpected end of input")
	}

	if end != 0xEA {
		return 0, 0, fmt.Errorf("ion: invalid BVM (0xE0 0x%X 0x%X 0x%X)", major, minor, end)
	}

	b.state = bssBeforeValue
	b.code = bitcodeNone
	b.len = 0

	return byte(major), byte(minor), nil
}

func (b *bitstream) ReadAnnotations() ([]uint64, error) {
	if b.code != bitcodeAnnotation {
		return nil, errors.New("ion: not an annotation")
	}

	alen, lenlen, err := b.readVarUintLen(b.len)
	if err != nil {
		return nil, err
	}

	if b.len-lenlen <= alen {
		// The size of the annotation is larger than the remaining free space inside the
		// annotation container.
		return nil, errors.New("ion: malformed annotation")
	}

	as := []uint64{}
	for alen > 0 {
		id, idlen, err := b.readVarUintLen(alen)
		if err != nil {
			return nil, err
		}

		as = append(as, id)
		alen -= idlen
	}

	b.state = bssBeforeValue
	b.code = bitcodeNone
	b.len = 0

	return as, nil
}

func (b *bitstream) ReadFieldID() (uint64, error) {
	if b.code != bitcodeFieldID {
		return 0, errors.New("ion: not a field id")
	}

	id, err := b.readVarUint()
	if err != nil {
		return 0, err
	}

	b.state = bssBeforeValue
	b.code = bitcodeNone

	return id, nil
}

func (b *bitstream) readVarUint() (uint64, error) {
	r, _, err := b.readVarUintLen(10)
	return r, err
}

func (b *bitstream) readVarUintLen(max uint64) (uint64, uint64, error) {
	r := uint64(0)
	l := uint64(0)

	for {
		c, err := b.read()
		if err != nil {
			return 0, 0, err
		}
		if c == -1 {
			return 0, 0, errors.New("ion: unexpected end of input")
		}

		l++

		r = (r << 7) ^ uint64(c&0x7F)
		if c&0x80 != 0 {
			return r, l, nil
		}

		if l == max {
			return 0, 0, errors.New("ion: varuint too large")
		}
	}
}

func (b *bitstream) skipVarUint() error {
	for {
		c, err := b.read()
		if err != nil {
			return err
		}
		if c == -1 {
			return errors.New("ion: unexpected end of input")
		}
		if c&0x80 != 0 {
			return nil
		}
	}
}

var bitcodes = []bitcode{
	bitcodeNull,       // 0x00
	bitcodeBool,       // 0x10
	bitcodeInt,        // 0x20
	bitcodeNegInt,     // 0x30
	bitcodeFloat,      // 0x40
	bitcodeDecimal,    // 0x50
	bitcodeTimestamp,  // 0x60
	bitcodeSymbol,     // 0x70
	bitcodeString,     // 0x80
	bitcodeClob,       // 0x90
	bitcodeBlob,       // 0xA0
	bitcodeList,       // 0xB0
	bitcodeSexp,       // 0xC0
	bitcodeStruct,     // 0xD0
	bitcodeAnnotation, // 0xE0
}

func parseTag(c int) (bitcode, uint64) {
	high := (c >> 4) & 0x0F
	low := c & 0x0F

	code := bitcodeNone
	if high < len(bitcodes) {
		code = bitcodes[high]
	}

	return code, uint64(low)
}

func (b *bitstream) read() (int, error) {
	c, err := b.in.ReadByte()
	if err == io.EOF {
		return -1, nil
	}
	if err != nil {
		return 0, err
	}

	b.pos++
	return int(c), nil
}

func (b *bitstream) skip(n uint64) error {
	_, err := b.in.Discard(int(n))
	if err == io.EOF {
		return nil
	}
	b.pos += n
	return err
}
