package ion

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"time"
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
	bitcodeFalse
	bitcodeTrue
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
	case bitcodeFalse:
		return "false"
	case bitcodeTrue:
		return "true"
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

	if code == bitcodeAnnotation {
		switch len {
		case 0:
			// This value is actually a BVM. It's invalid if we're not at the top level.
			if !b.stack.empty() {
				return errors.New("ion: BVM in a container")
			}
			b.code = bitcodeBVM
			b.len = 3
			return nil

		case 0x0F:
			// No such thing as a null annotation.
			return fmt.Errorf("ion: invalid tag byte: 0x%X", c)
		}
	}

	// Booleans are a bit special.
	if code == bitcodeFalse {
		switch len {
		case 0, 0x0F:
			break
		case 1:
			code = bitcodeTrue
			len = 0
		default:
			// Other forms of bool are invalid.
			return fmt.Errorf("ion: invalid tag byte: 0x%X", c)
		}
	}

	if len == 0x0F {
		// This value is actually a null.
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
		}
		b.state = b.stateAfterValue()
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
	if diff > 0 {
		if err := b.skip(diff); err != nil {
			return err
		}
	}

	b.state = b.stateAfterValue()
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

func (b *bitstream) ReadInt() (interface{}, error) {
	switch b.code {
	case bitcodeInt, bitcodeNegInt:
	default:
		return "", errors.New("ion: not an integer")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return "", err
	}

	var ret interface{}
	switch {
	case len(bs) == 0:
		// Special case for zero.
		ret = int64(0)

	case len(bs) < 8, (len(bs) == 8 && bs[0]&0x80 == 0):
		// It'll fit in an int64.
		i := int64(0)
		for _, b := range bs {
			i <<= 8
			i |= int64(b)
		}
		if b.code == bitcodeNegInt {
			i = -i
		}
		ret = i

	default:
		// Need to go big.Int.
		i := new(big.Int).SetBytes(bs)
		if b.code == bitcodeNegInt {
			i = i.Neg(i)
		}
		ret = i
	}

	b.state = b.stateAfterValue()
	b.code = bitcodeNone
	b.len = 0

	return ret, nil
}

func (b *bitstream) ReadFloat() (float64, error) {
	if b.code != bitcodeFloat {
		return 0, errors.New("ion: not a float")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return 0, err
	}

	var ret float64
	switch len(bs) {
	case 0:
		ret = 0

	case 4:
		ui := binary.BigEndian.Uint32(bs)
		ret = float64(math.Float32frombits(ui))

	case 8:
		ui := binary.BigEndian.Uint64(bs)
		ret = math.Float64frombits(ui)

	default:
		return 0, errors.New("ion: invalid float size")
	}

	b.state = b.stateAfterValue()
	b.code = bitcodeNone
	b.len = 0

	return ret, nil
}

func (b *bitstream) ReadDecimal() (*Decimal, error) {
	if b.code != bitcodeDecimal {
		return nil, errors.New("ion: not a decimal")
	}
	if b.len == 0 {
		return NewDecimalInt(0), nil
	}

	d, err := b.readDecimal(b.len)
	if err != nil {
		return nil, err
	}

	b.state = b.stateAfterValue()
	b.code = bitcodeNone
	b.len = 0

	return d, nil
}

func (b *bitstream) ReadTimestamp() (time.Time, error) {
	if b.code != bitcodeTimestamp {
		return time.Time{}, errors.New("ion: not a timestamp")
	}

	offset, olen, err := b.readVarIntLen(b.len)
	if err != nil {
		return time.Time{}, err
	}
	b.len -= olen

	ts := []int{1, 1, 1, 0, 0, 0}
	for i := 0; b.len > 0 && i < 6; i++ {
		val, vlen, err := b.readVarUintLen(b.len)
		if err != nil {
			return time.Time{}, err
		}
		b.len -= vlen
		ts[i] = int(val)
	}

	nsecs, err := b.readNsecs()
	if err != nil {
		return time.Time{}, err
	}

	utc := time.Date(ts[0], time.Month(ts[1]), ts[2], ts[3], ts[4], ts[5], int(nsecs), time.UTC)

	return utc.In(time.FixedZone("fixed", int(offset)*60)), nil
}

func (b *bitstream) readNsecs() (int64, error) {
	d, err := b.readDecimal(b.len)
	if err != nil {
		return 0, err
	}
	return d.ShiftL(9).Trunc()
}

func (b *bitstream) readDecimal(len uint64) (*Decimal, error) {
	exp := int64(0)
	coef := new(big.Int)

	if len > 0 {
		val, vlen, err := b.readVarIntLen(len)
		if err != nil {
			return nil, err
		}
		exp = val
		len -= vlen
	}

	if len > 0 {
		if err := b.readIntTo(len, coef); err != nil {
			return nil, err
		}
	}

	return NewDecimal(coef, int(exp)), nil
}

func (b *bitstream) ReadSymbol() (uint64, error) {
	if b.code != bitcodeSymbol {
		return 0, errors.New("ion: not a symbol")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return 0, err
	}

	b.state = b.stateAfterValue()
	b.code = bitcodeNone
	b.len = 0

	if len(bs) == 0 {
		return 0, nil
	}
	if len(bs) > 8 {
		return 0, errors.New("ion: symbol id out of range")
	}

	ret := uint64(0)
	for _, b := range bs {
		ret <<= 8
		ret |= uint64(b)
	}
	return ret, nil
}

func (b *bitstream) ReadString() (string, error) {
	if b.code != bitcodeString {
		return "", errors.New("ion: not a string")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return "", err
	}

	b.state = b.stateAfterValue()
	b.code = bitcodeNone
	b.len = 0

	return string(bs), nil
}

func (b *bitstream) ReadBytes() ([]byte, error) {
	if b.code != bitcodeClob && b.code != bitcodeBlob {
		return nil, errors.New("ion: not a lob")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return nil, err
	}

	b.state = b.stateAfterValue()
	b.code = bitcodeNone
	b.len = 0

	return bs, nil
}

func (b *bitstream) readIntTo(len uint64, ret *big.Int) error {
	bs, err := b.readN(len)
	if err != nil {
		return err
	}

	neg := (bs[0]&0x80 != 0)
	bs[0] &= 0x7F
	if bs[0] == 0 {
		bs = bs[1:]
	}

	ret.SetBytes(bs)
	if neg {
		ret.Neg(ret)
	}

	return nil
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

		r <<= 7
		r ^= uint64(c & 0x7F)
		l++

		if c&0x80 != 0 {
			return r, l, nil
		}

		if l == max {
			return 0, 0, errors.New("ion: varuint too large")
		}
	}
}

func (b *bitstream) readVarIntLen(max uint64) (int64, uint64, error) {
	c, err := b.read()
	if err != nil {
		return 0, 0, err
	}
	if c == -1 {
		return 0, 0, errors.New("ion: unexpected end of input")
	}

	sign := int64(1)
	if c&0x40 != 0 {
		sign = -1
	}

	r := int64(c & 0x3F)
	l := uint64(1)

	if c&0x80 != 0 {
		return r * sign, l, nil
	}

	for {
		c, err := b.read()
		if err != nil {
			return 0, 0, err
		}
		if c == -1 {
			return 0, 0, errors.New("ion: unexpected end of input")
		}

		r <<= 7
		r ^= int64(c & 0x7F)
		l++

		if c&0x80 != 0 {
			return r * sign, l, nil
		}

		if l == max {
			return 0, 0, errors.New("ion: varint too large")
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

func (b *bitstream) stateAfterValue() bss {
	if b.stack.peek().code == bitcodeStruct {
		return bssBeforeFieldID
	}
	return bssBeforeValue
}

var bitcodes = []bitcode{
	bitcodeNull,       // 0x00
	bitcodeFalse,      // 0x10
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

func (b *bitstream) readN(n uint64) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}

	bs := make([]byte, n)
	_, err := b.in.Read(bs)
	if err == io.EOF {
		return nil, errors.New("ion: unexpected end of input")
	}
	if err != nil {
		return nil, err
	}

	b.pos += n
	return bs, nil
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
