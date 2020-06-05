package ion

import (
	"bufio"
	"bytes"
	"encoding/binary"
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

// A bitstream is a low-level parser for binary Ion values.
type bitstream struct {
	in    *bufio.Reader
	pos   uint64
	state bss
	stack bitstack

	code bitcode
	null bool
	len  uint64
}

// Init initializes this stream with the given bufio.Reader.
func (b *bitstream) Init(in *bufio.Reader) {
	b.in = in
}

// InitBytes initializes this stream with the given bytes.
func (b *bitstream) InitBytes(in []byte) {
	b.in = bufio.NewReader(bytes.NewReader(in))
}

// Code returns the typecode of the current value.
func (b *bitstream) Code() bitcode {
	return b.code
}

// IsNull returns true if the current value is null.
func (b *bitstream) IsNull() bool {
	return b.null
}

// Pos returns the current position.
func (b *bitstream) Pos() uint64 {
	return b.pos
}

// Len returns the length of the current value.
func (b *bitstream) Len() uint64 {
	return b.len
}

// Next advances the stream to the next value.
func (b *bitstream) Next() error {
	// If we have an unread value, skip over it to get to the next one.
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

	// Found the end of the file.
	if c == -1 {
		b.code = bitcodeEOF
		return nil
	}

	// Parse the tag.
	code, len := parseTag(c)
	if code == bitcodeNone {
		return &InvalidTagByteError{byte(c), b.pos - 1}
	}

	b.state = bssOnValue

	if code == bitcodeAnnotation {
		switch len {
		case 0:
			// This value is actually a BVM. It's invalid if we're not at the top level.
			if !b.stack.empty() {
				return &SyntaxError{"invalid BVM in a container", b.pos - 1}
			}
			b.code = bitcodeBVM
			b.len = 3
			return nil

		case 0x0F:
			// No such thing as a null annotation.
			return &InvalidTagByteError{byte(c), b.pos - 1}
		}
	}

	// Booleans are a bit special; the 'length' stores the value.
	if code == bitcodeFalse {
		switch len {
		case 0, 0x0F:
			break
		case 1:
			code = bitcodeTrue
			len = 0
		default:
			// Other forms are invalid.
			return &InvalidTagByteError{byte(c), b.pos - 1}
		}
	}

	if len == 0x0F {
		// This value is actually a null.
		b.code = code
		b.null = true
		return nil
	}

	pos := b.pos
	rem := b.remaining()

	// This value's actual len is encoded as a separate varUint.
	if len == 0x0E {
		var lenlen uint64
		len, lenlen, err = b.readVarUintLen(rem)
		if err != nil {
			return err
		}
		rem -= lenlen
	}

	if len > rem {
		msg := fmt.Sprintf("value overruns its container: %v vs %v", len, rem)
		return &SyntaxError{msg, pos - 1}
	}

	b.code = code
	b.len = len
	return nil
}

// SkipValue skips over the current value.
func (b *bitstream) SkipValue() error {
	switch b.state {
	case bssBeforeFieldID, bssBeforeValue:
		// No current value to skip yet.
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

	default:
		panic(fmt.Sprintf("invalid state %v", b.state))
	}

	b.clear()
	return nil
}

// StepIn steps in to a container.
func (b *bitstream) StepIn() {
	switch b.code {
	case bitcodeStruct:
		b.state = bssBeforeFieldID

	case bitcodeList, bitcodeSexp:
		b.state = bssBeforeValue

	default:
		panic(fmt.Sprintf("StepIn called with b.code=%v", b.code))
	}

	b.stack.push(b.code, b.pos+b.len)
	b.clear()
}

// StepOut steps out of a container.
func (b *bitstream) StepOut() error {
	if b.stack.empty() {
		panic("StepOut called at top level")
	}

	cur := b.stack.peek()
	b.stack.pop()

	if cur.end < b.pos {
		panic(fmt.Sprintf("end (%v) greater than b.pos (%v)", cur.end, b.pos))
	}
	diff := cur.end - b.pos

	// Skip over anything left in the container we're stepping out of.
	if diff > 0 {
		if err := b.skip(diff); err != nil {
			return err
		}
	}

	b.state = b.stateAfterValue()
	b.clear()

	return nil
}

// ReadBVM reads a binary version marker, returning its major and minor version.
func (b *bitstream) ReadBVM() (byte, byte, error) {
	if b.code != bitcodeBVM {
		panic("not a BVM")
	}

	major, err := b.read1()
	if err != nil {
		return 0, 0, err
	}

	minor, err := b.read1()
	if err != nil {
		return 0, 0, err
	}

	end, err := b.read1()
	if err != nil {
		return 0, 0, err
	}

	if end != 0xEA {
		msg := fmt.Sprintf("invalid BVM: 0xE0 0x%02X 0x%02X 0x%02X", major, minor, end)
		return 0, 0, &SyntaxError{msg, b.pos - 4}
	}

	b.state = bssBeforeValue
	b.clear()

	return byte(major), byte(minor), nil
}

// ReadFieldID reads a field ID.
func (b *bitstream) ReadFieldID() (uint64, error) {
	if b.code != bitcodeFieldID {
		panic("not a field ID")
	}

	id, err := b.readVarUint()
	if err != nil {
		return 0, err
	}

	b.state = bssBeforeValue
	b.code = bitcodeNone

	return id, nil
}

// ReadAnnotationIDs reads a set of annotation IDs.
func (b *bitstream) ReadAnnotationIDs() ([]uint64, error) {
	if b.code != bitcodeAnnotation {
		panic("not an annotation")
	}

	alen, lenlen, err := b.readVarUintLen(b.len)
	if err != nil {
		return nil, err
	}

	if b.len-lenlen <= alen {
		// The size of the annotations is larger than the remaining free space inside the
		// annotation container.
		return nil, &SyntaxError{"malformed annotation", b.pos - lenlen}
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
	b.clear()

	return as, nil
}

// ReadInt reads an integer value.
func (b *bitstream) ReadInt() (interface{}, error) {
	if b.code != bitcodeInt && b.code != bitcodeNegInt {
		panic("not an integer")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return "", err
	}

	var ret interface{}
	switch {
	case b.len == 0:
		// Special case for zero.
		ret = int64(0)

	case b.len < 8, (b.len == 8 && bs[0]&0x80 == 0):
		// It'll fit in an int64.
		i := int64(0)
		for _, b := range bs {
			i <<= 8
			i ^= int64(b)
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
	b.clear()

	return ret, nil
}

// ReadFloat reads a float value.
func (b *bitstream) ReadFloat() (float64, error) {
	if b.code != bitcodeFloat {
		panic("not a float")
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
		return 0, &SyntaxError{"invalid float size", b.pos - b.len}
	}

	b.state = b.stateAfterValue()
	b.clear()

	return ret, nil
}

// ReadDecimal reads a decimal value.
func (b *bitstream) ReadDecimal() (*Decimal, error) {
	if b.code != bitcodeDecimal {
		panic("not a decimal")
	}

	d, err := b.readDecimal(b.len)
	if err != nil {
		return nil, err
	}

	b.state = b.stateAfterValue()
	b.clear()

	return d, nil
}

// ReadTimestamp reads a timestamp value.
func (b *bitstream) ReadTimestamp() (time.Time, error) {
	if b.code != bitcodeTimestamp {
		panic("not a timestamp")
	}

	len := b.len

	offset, olen, err := b.readVarIntLen(len)
	if err != nil {
		return time.Time{}, err
	}
	len -= olen

	ts := []int{1, 1, 1, 0, 0, 0}
	for i := 0; len > 0 && i < 6; i++ {
		val, vlen, err := b.readVarUintLen(len)
		if err != nil {
			return time.Time{}, err
		}
		len -= vlen
		ts[i] = int(val)

		// When i is 3, it means we are setting hour component. A timestamp with
		// hour, must follow by minute. Hence, len cannot be zero at this point.
		if i == 3 && len == 0 {
			return time.Time{},
				&SyntaxError{"Invalid timestamp - Hour cannot be present without minute", b.pos}
		}
	}

	nsecs, err := b.readNsecs(len)
	if err != nil {
		return time.Time{}, err
	}

	b.state = b.stateAfterValue()
	b.clear()

	return tryCreateTimeWithNSecAndOffset(ts, nsecs, offset)
}

func tryCreateTimeWithNSecAndOffset(ts []int, nsecs int, offset int64) (time.Time, error) {
	date := time.Date(ts[0], time.Month(ts[1]), ts[2], ts[3], ts[4], ts[5], nsecs, time.UTC)
	// time.Date converts 2000-01-32 input to 2000-02-01
	if ts[0] != date.Year() || time.Month(ts[1]) != date.Month() || ts[2] != date.Day() {
		return time.Time{}, fmt.Errorf("ion: invalid timestamp")
	}

	return date.In(time.FixedZone("fixed", int(offset)*60)), nil
}

// ReadNsecs reads the fraction part of a timestamp and truncates it to nanoseconds.
func (b *bitstream) readNsecs(len uint64) (int, error) {
	d, err := b.readDecimal(len)
	if err != nil {
		return 0, err
	}

	nsec, err := d.ShiftL(9).Trunc()
	if err != nil || nsec < 0 || nsec > 999999999 {
		msg := fmt.Sprintf("invalid timestamp fraction: %v", d)
		return 0, &SyntaxError{msg, b.pos}
	}

	return int(nsec), nil
}

// ReadDecimal reads a decimal value of the given length: an exponent encoded as a
// varInt, followed by an integer coefficient taking up the remaining bytes.
func (b *bitstream) readDecimal(len uint64) (*Decimal, error) {
	exp := int64(0)
	coef := new(big.Int)

	if len > 0 {
		val, vlen, err := b.readVarIntLen(len)
		if err != nil {
			return nil, err
		}

		if val > math.MaxInt32 || val < math.MinInt32 {
			msg := fmt.Sprintf("decimal exponent out of range: %v", val)
			return nil, &SyntaxError{msg, b.pos - vlen}
		}

		exp = val
		len -= vlen
	}

	if len > 0 {
		if err := b.readBigInt(len, coef); err != nil {
			return nil, err
		}
	}

	return NewDecimal(coef, int32(exp)), nil
}

// ReadSymbolID reads a symbol value.
func (b *bitstream) ReadSymbolID() (uint64, error) {
	if b.code != bitcodeSymbol {
		panic("not a symbol")
	}

	if b.len > 8 {
		return 0, &SyntaxError{"symbol id too large", b.pos}
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return 0, err
	}

	b.state = b.stateAfterValue()
	b.clear()

	ret := uint64(0)
	for _, b := range bs {
		ret <<= 8
		ret ^= uint64(b)
	}
	return ret, nil
}

// ReadString reads a string value.
func (b *bitstream) ReadString() (string, error) {
	if b.code != bitcodeString {
		panic("not a string")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return "", err
	}

	b.state = b.stateAfterValue()
	b.clear()

	return string(bs), nil
}

// ReadBytes reads a blob or clob value.
func (b *bitstream) ReadBytes() ([]byte, error) {
	if b.code != bitcodeClob && b.code != bitcodeBlob {
		panic("not a lob")
	}

	bs, err := b.readN(b.len)
	if err != nil {
		return nil, err
	}

	b.state = b.stateAfterValue()
	b.clear()

	return bs, nil
}

// Clear clears the current code and len.
func (b *bitstream) clear() {
	b.code = bitcodeNone
	b.null = false
	b.len = 0
}

// ReadBigInt reads a fixed-length integer of the given length and stores
// the value in the given big.Int.
func (b *bitstream) readBigInt(len uint64, ret *big.Int) error {
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

// ReadVarUint reads a variable-length-encoded uint.
func (b *bitstream) readVarUint() (uint64, error) {
	val, _, err := b.readVarUintLen(b.remaining())
	return val, err
}

// ReadVarUintLen reads a variable-length-encoded uint of at most max bytes,
// returning the value and its actual length in bytes.
func (b *bitstream) readVarUintLen(max uint64) (uint64, uint64, error) {
	if max > 10 {
		max = 10
	}

	val := uint64(0)
	len := uint64(0)

	for {
		if len >= max {
			return 0, 0, &SyntaxError{"varuint too large", b.pos}
		}

		c, err := b.read1()
		if err != nil {
			return 0, 0, err
		}

		val <<= 7
		val ^= uint64(c & 0x7F)
		len++

		if c&0x80 != 0 {
			return val, len, nil
		}
	}
}

// SkipVarUint skips over a variable-length-encoded uint.
func (b *bitstream) skipVarUint() error {
	_, err := b.skipVarUintLen(b.remaining())
	return err
}

// SkipVarUintLen skips over a variable-length-encoded uint of at most max bytes.
func (b *bitstream) skipVarUintLen(max uint64) (uint64, error) {
	if max > 10 {
		max = 10
	}

	len := uint64(0)
	for {
		if len >= max {
			return 0, &SyntaxError{"varuint too large", b.pos - len}
		}

		c, err := b.read1()
		if err != nil {
			return 0, err
		}

		len++

		if c&0x80 != 0 {
			return len, nil
		}
	}
}

// Remaining returns the number of bytes remaining in the current container.
func (b *bitstream) remaining() uint64 {
	if b.stack.empty() {
		return math.MaxUint64
	}

	end := b.stack.peek().end
	if b.pos > end {
		panic(fmt.Sprintf("pos (%v) > end (%v)", b.pos, end))
	}

	return end - b.pos
}

// ReadVarIntLen reads a variable-length-encoded int of at most max bytes,
// returning the value and its actual length in bytes
func (b *bitstream) readVarIntLen(max uint64) (int64, uint64, error) {
	if max == 0 {
		return 0, 0, &SyntaxError{"varint too large", b.pos}
	}
	if max > 10 {
		max = 10
	}

	// Read the first byte, which contains the sign bit.
	c, err := b.read1()
	if err != nil {
		return 0, 0, err
	}

	sign := int64(1)
	if c&0x40 != 0 {
		sign = -1
	}

	val := int64(c & 0x3F)
	len := uint64(1)

	// Check if that was the last (only) byte.
	if c&0x80 != 0 {
		return val * sign, len, nil
	}

	for {
		if len >= max {
			return 0, 0, &SyntaxError{"varint too large", b.pos - len}
		}

		c, err := b.read1()
		if err != nil {
			return 0, 0, err
		}

		val <<= 7
		val ^= int64(c & 0x7F)
		len++

		if c&0x80 != 0 {
			return val * sign, len, nil
		}
	}
}

// StateAfterValue returns the state this stream is in after reading a value.
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

// ParseTag parses a tag byte into a typecode and a length.
func parseTag(c int) (bitcode, uint64) {
	high := (c >> 4) & 0x0F
	low := c & 0x0F

	code := bitcodeNone
	if high < len(bitcodes) {
		code = bitcodes[high]
	}

	return code, uint64(low)
}

// ReadN reads the next n bytes of input from the underlying stream.
func (b *bitstream) readN(n uint64) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}

	bs := make([]byte, n)
	actual, err := io.ReadFull(b.in, bs)
	b.pos += uint64(actual)

	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil, &UnexpectedEOFError{b.pos}
	}
	if err != nil {
		return nil, &IOError{err}
	}

	return bs, nil
}

// Read1 reads the next byte of input from the underlying stream, returning
// an UnexpectedEOFError if it's an EOF.
func (b *bitstream) read1() (int, error) {
	c, err := b.read()
	if err != nil {
		return 0, err
	}
	if c == -1 {
		return 0, &UnexpectedEOFError{b.pos}
	}
	return c, nil
}

// Read reads the next byte of input from the underlying stream. It returns
// -1 instead of io.EOF if we've hit the end of the stream, because I find
// that easier to reason about.
func (b *bitstream) read() (int, error) {
	c, err := b.in.ReadByte()
	b.pos++

	if err == io.EOF {
		return -1, nil
	}
	if err != nil {
		return 0, &IOError{err}
	}

	return int(c), nil
}

// Skip skips n bytes of input from the underlying stream.
func (b *bitstream) skip(n uint64) error {
	actual, err := b.in.Discard(int(n))
	b.pos += uint64(actual)

	if err == io.EOF {
		return nil
	}
	if err != nil {
		return &IOError{err}
	}

	return nil
}

// A bitnode represents a container value, including its typecode and
// the offset at which it (supposedly) ends.
type bitnode struct {
	code bitcode
	end  uint64
}

// A stack of bitnodes representing container values that we're currently
// stepped in to.
type bitstack struct {
	arr []bitnode
}

// Empty returns true if this bitstack is empty.
func (b *bitstack) empty() bool {
	return len(b.arr) == 0
}

// Peek peeks at the top bitnode on the stack.
func (b *bitstack) peek() bitnode {
	if len(b.arr) == 0 {
		return bitnode{}
	}
	return b.arr[len(b.arr)-1]
}

// Push pushes a bitnode onto the stack.
func (b *bitstack) push(code bitcode, end uint64) {
	b.arr = append(b.arr, bitnode{code, end})
}

// Pop pops a bitnode from the stack.
func (b *bitstack) pop() {
	if len(b.arr) == 0 {
		panic("pop called on empty bitstack")
	}
	b.arr = b.arr[:len(b.arr)-1]
}
