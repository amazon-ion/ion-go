package ion

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type textReaderState uint8

const (
	trsDone textReaderState = iota
	trsBeforeFieldName
	trsBeforeTypeAnnotations
	trsBeforeScalar
	trsBeforeContainer
	trsInValue
	trsAfterValue
)

func (s textReaderState) String() string {
	switch s {
	case trsDone:
		return "<done>"
	case trsBeforeFieldName:
		return "<beforeFieldName>"
	case trsBeforeTypeAnnotations:
		return "<beforeTypeAnnotations>"
	case trsBeforeContainer:
		return "<beforeContainer>"
	case trsAfterValue:
		return "<afterValue>"
	default:
		return strconv.Itoa(int(s))
	}
}

type textReader struct {
	tok   tokenizer
	state textReaderState
	ctx   ctx
	eof   bool
	err   error

	fieldName       string
	typeAnnotations []string
	valueType       Type
	value           interface{}
}

// NewTextReader creates a new text reader.
func NewTextReader(in io.Reader) Reader {
	return &textReader{
		tok: tokenizer{
			in: bufio.NewReader(in),
		},
		state: trsBeforeTypeAnnotations,
	}
}

// NewTextReaderString creates a new text reader from a string.
func NewTextReaderString(str string) Reader {
	return NewTextReader(strings.NewReader(str))
}

func (t *textReader) SymbolTable() SymbolTable {
	// Text content doesn't have a symbol table.
	return nil
}

func (t *textReader) Next() bool {
	if t.state == trsDone || t.eof {
		return false
	}

	err := t.finishValue()
	if err != nil {
		t.explode(err)
		return false
	}

	t.fieldName = ""
	t.typeAnnotations = nil
	t.valueType = NoType
	t.value = nil

	if err := t.tok.Next(); err != nil {
		t.explode(err)
		return false
	}

	for {
		var f func() (bool, error)

		switch t.state {
		case trsAfterValue:
			f = t.nextAfterValue
		case trsBeforeFieldName:
			f = t.nextBeforeFieldName
		case trsBeforeTypeAnnotations:
			f = t.nextBeforeTypeAnnotations
		default:
			panic("invalid state")
		}

		done, err := f()
		if err != nil {
			t.explode(err)
			return false
		}
		if done {
			return !t.eof
		}

		if err := t.tok.Next(); err != nil {
			t.explode(err)
			return false
		}
	}
}

// NextAfterValue moves to the next value when we're in the
// AfterValue state.
func (t *textReader) nextAfterValue() (bool, error) {
	tok := t.tok.Token()
	switch tok {
	case tokenComma:
		// Another value coming; eat the comma and move to the
		// appropriate next state.
		switch t.ctx.peek() {
		case ctxInStruct:
			t.state = trsBeforeFieldName
		case ctxInList:
			t.state = trsBeforeTypeAnnotations
		default:
			panic("invalid state")
		}
		return false, nil

	case tokenCloseBrace:
		// No more values in this struct.
		if t.ctx.peek() == ctxInStruct {
			t.eof = true
			return true, nil
		}
		return false, errors.New("unexpected token '}'")

	case tokenCloseBracket:
		// No more values in this list.
		if t.ctx.peek() == ctxInList {
			t.eof = true
			return true, nil
		}
		return false, errors.New("unexpected token ']'")

	default:
		return false, fmt.Errorf("unexpected token '%v'", tok)
	}
}

// NextBeforeFieldName moves to the next value when we're in the
// BeforeFieldName state.
func (t *textReader) nextBeforeFieldName() (bool, error) {
	tok := t.tok.Token()
	switch tok {
	case tokenCloseBrace:
		// No more values in this struct.
		t.eof = true
		return true, nil

	case tokenSymbol, tokenSymbolQuoted:
		// Read the field name.
		val, err := t.tok.ReadValue(tok)
		if err != nil {
			return false, err
		}
		if tok == tokenSymbol {
			if err := verifyUnquotedSymbol(val, "field name"); err != nil {
				return false, err
			}
		}

		// Skip over the following colon.
		if err = t.tok.Next(); err != nil {
			return false, err
		}
		if tok = t.tok.Token(); tok != tokenColon {
			return false, fmt.Errorf("unexpected token '%v'", tok)
		}

		t.fieldName = val
		t.state = trsBeforeTypeAnnotations

		return false, nil

	default:
		return false, fmt.Errorf("unexpected token '%v'", tok)
	}
}

// NextBeforeTypeAnnotations moves to the next value when we're in the
// BeforeTypeAnnotations state.
func (t *textReader) nextBeforeTypeAnnotations() (bool, error) {
	tok := t.tok.Token()
	switch tok {
	case tokenEOF:
		if t.ctx.peek() == ctxAtTopLevel {
			t.eof = true
			return true, nil
		}
		return false, errors.New("unexpected EOF")

	case tokenSymbol, tokenSymbolQuoted:
		val, err := t.tok.ReadValue(tok)
		if err != nil {
			return false, err
		}

		ws, err := t.tok.skipWhitespaceHelper()
		if err != nil {
			return false, err
		}

		ok, err := t.tok.skipDoubleColon()
		if err != nil {
			return false, err
		}
		if ok {
			// val was a type annotation; remember it and keep going.
			if tok == tokenSymbol {
				if err := verifyUnquotedSymbol(val, "type annotation"); err != nil {
					return false, err
				}
			}
			t.typeAnnotations = append(t.typeAnnotations, val)
			return false, nil
		}

		// val was a legit symbol value.
		if err := t.onSymbol(val, tok, ws); err != nil {
			return false, err
		}
		return true, nil

	case tokenString, tokenLongString:
		val, err := t.tok.ReadValue(tok)
		if err != nil {
			return false, err
		}

		t.state = t.stateAfterValue()
		t.valueType = StringType
		t.value = val
		return true, nil

	case tokenBinary, tokenHex, tokenNumber, tokenFloatInf, tokenFloatMinusInf:
		if err := t.onNumber(tok); err != nil {
			return false, err
		}
		return true, nil

	case tokenTimestamp:
		if err := t.onTimestamp(); err != nil {
			return false, err
		}
		return true, nil

	case tokenOpenDoubleBrace:
		if err := t.onLob(); err != nil {
			return false, err
		}
		return true, nil

	case tokenOpenBrace:
		t.state = trsBeforeContainer
		t.valueType = StructType
		return true, nil

	case tokenOpenBracket:
		t.state = trsBeforeContainer
		t.valueType = ListType
		return true, nil

	case tokenOpenParen:
		t.state = trsBeforeContainer
		t.valueType = SexpType
		return true, nil

	case tokenCloseBrace:
		// No more values in this struct.
		if t.ctx.peek() == ctxInStruct {
			t.eof = true
			return true, nil
		}
		return false, errors.New("unexpected token '}'")

	case tokenCloseBracket:
		// No more values in this list.
		if t.ctx.peek() == ctxInList {
			t.eof = true
			return true, nil
		}
		return false, errors.New("unexpected token ']'")

	case tokenCloseParen:
		// No more values in this sexp.
		if t.ctx.peek() == ctxInSexp {
			t.eof = true
			return true, nil
		}
		return false, errors.New("unexpected token ')'")

	default:
		return false, fmt.Errorf("unexpected token '%v'", tok)
	}
}

func verifyUnquotedSymbol(val string, ctx string) error {
	switch val {
	case "null", "true", "false", "nan":
		return fmt.Errorf("cannot use unquoted keyword %v as %v", val, ctx)
	}
	return nil
}

func (t *textReader) onSymbol(val string, tok tokenType, ws bool) error {
	valueType := SymbolType
	var value interface{} = val

	if tok == tokenSymbol {
		switch val {
		case "null":
			vt, err := t.onNull(ws)
			if err != nil {
				return err
			}
			valueType = vt
			value = nil

		case "true":
			valueType = BoolType
			value = true

		case "false":
			valueType = BoolType
			value = false

		case "nan":
			valueType = FloatType
			value = math.NaN()
		}
	}

	t.state = t.stateAfterValue()
	t.valueType = valueType
	t.value = value

	return nil
}

func (t *textReader) onNull(ws bool) (Type, error) {
	if !ws {
		ok, err := t.tok.skipDot()
		if err != nil {
			return NoType, err
		}
		if ok {
			return t.readNullType()
		}
	}

	return NullType, nil
}

func (t *textReader) readNullType() (Type, error) {
	if err := t.tok.Next(); err != nil {
		return NoType, err
	}
	if t.tok.Token() != tokenSymbol {
		return NoType, fmt.Errorf("unexpected token %v after null", t.tok.Token())
	}

	val, err := t.tok.ReadValue(tokenSymbol)
	if err != nil {
		return NoType, err
	}

	switch val {
	case "null":
		return NullType, nil
	case "bool":
		return BoolType, nil
	case "int":
		return IntType, nil
	case "float":
		return FloatType, nil
	case "decimal":
		return DecimalType, nil
	case "timestamp":
		return TimestampType, nil
	case "symbol":
		return SymbolType, nil
	case "string":
		return StringType, nil
	case "blob":
		return BlobType, nil
	case "clob":
		return ClobType, nil
	case "list":
		return ListType, nil
	case "struct":
		return StructType, nil
	case "sexp":
		return SexpType, nil
	default:
		return NoType, fmt.Errorf("invalid symbol null.%v", val)
	}
}

func (t *textReader) onNumber(tok tokenType) error {
	var valueType Type
	var value interface{}

	switch tok {
	case tokenBinary:
		val, err := t.tok.ReadValue(tok)
		if err != nil {
			return err
		}

		valueType = IntType
		value, err = parseInt(val, 2)
		if err != nil {
			return err
		}

	case tokenHex:
		val, err := t.tok.ReadValue(tok)
		if err != nil {
			return err
		}

		valueType = IntType
		value, err = parseInt(val, 16)
		if err != nil {
			return err
		}

	case tokenNumber:
		val, tt, err := t.tok.ReadNumber()
		if err != nil {
			return err
		}

		valueType = tt

		switch tt {
		case IntType:
			value, err = parseInt(val, 10)
		case FloatType:
			value, err = parseFloat(val)
		case DecimalType:
			value, err = parseDecimal(val)
		default:
			panic("unexpected type")
		}

		if err != nil {
			return err
		}

	case tokenFloatInf:
		valueType = FloatType
		value = math.Inf(1)

	case tokenFloatMinusInf:
		valueType = FloatType
		value = math.Inf(-1)

	default:
		panic("unexpected token type")
	}

	t.state = t.stateAfterValue()
	t.valueType = valueType
	t.value = value

	return nil
}

func (t *textReader) onTimestamp() error {
	val, err := t.tok.ReadValue(tokenTimestamp)
	if err != nil {
		return err
	}

	value, err := parseTimestamp(val)
	if err != nil {
		return err
	}

	t.state = t.stateAfterValue()
	t.valueType = TimestampType
	t.value = value

	return nil
}

func (t *textReader) onLob() error {
	c, _, err := t.tok.skipLobWhitespace()
	if err != nil {
		return err
	}

	var (
		valType Type
		val     []byte
	)

	if c == '"' {
		// Short clob.
		valType = ClobType

		str, err := t.tok.ReadShortClob()
		if err != nil {
			return err
		}

		val = []byte(str)

	} else if c == '\'' {
		// Long clob.
		ok, err := t.tok.isTripleQuote()
		if err != nil {
			return err
		}
		if !ok {
			return invalidChar(c)
		}

		valType = ClobType

		str, err := t.tok.ReadLongClob()
		if err != nil {
			return err
		}

		val = []byte(str)

	} else {
		// Normal blob.
		valType = BlobType
		t.tok.unread(c)

		b64, err := t.tok.ReadBlob()
		if err != nil {
			return err
		}

		val, err = base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return err
		}
	}

	t.state = t.stateAfterValue()
	t.valueType = valType
	t.value = val

	return nil
}

func (t *textReader) Type() Type {
	return t.valueType
}

func (t *textReader) Err() error {
	return t.err
}

func (t *textReader) FieldName() string {
	return t.fieldName
}

func (t *textReader) TypeAnnotations() []string {
	return t.typeAnnotations
}

func (t *textReader) IsNull() bool {
	return t.value == nil
}

func (t *textReader) StepIn() error {
	if t.err != nil {
		return t.err
	}
	if t.state != trsBeforeContainer {
		return errors.New("invalid state")
	}

	var ctx ctxType
	switch t.valueType {
	case StructType:
		ctx = ctxInStruct
	case ListType:
		ctx = ctxInList
	case SexpType:
		ctx = ctxInSexp
	default:
		panic("trsBeforeContainer with unexpected valueType")
	}
	t.ctx.push(ctx)

	if ctx == ctxInStruct {
		t.state = trsBeforeFieldName
	} else {
		t.state = trsBeforeTypeAnnotations
	}

	// TODO: Make this less hacky.
	t.tok.unfinished = false
	return nil
}

func (t *textReader) StepOut() error {
	if t.err != nil {
		return t.err
	}

	ctx := t.ctx.peek()
	if ctx == ctxAtTopLevel {
		return errors.New("invalid state")
	}

	_, err := t.tok.finishValue()
	if err != nil {
		t.explode(err)
		return err
	}

	if !t.eof {
		// Haven't seen the end of the container yet; skip until we
		// find it.
		switch t.ctx.peek() {
		case ctxInStruct:
			err = t.tok.skipStructHelper()
		case ctxInList:
			err = t.tok.skipListHelper()
		case ctxInSexp:
			err = t.tok.skipSexpHelper()
		default:
			panic("invalid ctx")
		}

		if err != nil {
			t.explode(err)
			return err
		}
	}

	t.ctx.pop()
	t.state = t.stateAfterValue()
	t.valueType = NoType
	t.value = nil

	return nil
}

func (t *textReader) BoolValue() (bool, error) {
	if t.valueType == BoolType {
		if t.value == nil {
			return false, nil
		}
		return t.value.(bool), nil
	}
	return false, errors.New("value is not a bool")
}

func (t *textReader) IntValue() (int, error) {
	i, err := t.Int64Value()
	if err != nil {
		return 0, err
	}
	if i > math.MaxInt32 || i < math.MinInt32 {
		return 0, errors.New("value out of bounds")
	}
	return int(i), nil
}

func (t *textReader) Int64Value() (int64, error) {
	if t.valueType == IntType {
		if t.value == nil {
			return 0, nil
		}

		if i, ok := t.value.(int64); ok {
			return i, nil
		}

		bi := t.value.(*big.Int)
		if bi.IsInt64() {
			return bi.Int64(), nil
		}

		return 0, errors.New("value out of bounds")
	}
	return 0, errors.New("value is not an int")
}

func (t *textReader) BigIntValue() (*big.Int, error) {
	if t.valueType == IntType {
		if t.value == nil {
			return nil, nil
		}
		if i, ok := t.value.(int64); ok {
			return big.NewInt(i), nil
		}
		return t.value.(*big.Int), nil
	}
	return nil, errors.New("value is not an int")
}

func (t *textReader) FloatValue() (float64, error) {
	if t.valueType == FloatType {
		if t.value == nil {
			return 0.0, nil
		}
		return t.value.(float64), nil
	}
	// TODO: Cast ints/decimals?
	return 0.0, errors.New("value is not a float")
}

func (t *textReader) DecimalValue() (*Decimal, error) {
	switch t.valueType {
	case DecimalType:
		if t.value == nil {
			return nil, nil
		}
		return t.value.(*Decimal), nil
	}
	// TODO: Cast floats/ints?
	return nil, errors.New("value is not a decimal")
}

func (t *textReader) TimeValue() (time.Time, error) {
	switch t.valueType {
	case TimestampType:
		if t.value == nil {
			return time.Time{}, nil
		}
		return t.value.(time.Time), nil
	}
	return time.Time{}, errors.New("value is not a timestamp")
}

func (t *textReader) StringValue() (string, error) {
	switch t.valueType {
	case StringType, SymbolType:
		if t.value == nil {
			return "", nil
		}
		return t.value.(string), nil

	default:
		return "", errors.New("value is not a string")
	}
}

func (t *textReader) ByteValue() ([]byte, error) {
	switch t.valueType {
	case BlobType, ClobType:
		if t.value == nil {
			return nil, nil
		}
		return t.value.([]byte), nil
	}
	return nil, errors.New("value is not a byte array")
}

// FinishValue finishes reading the current value, if there is one.
func (t *textReader) finishValue() error {
	ok, err := t.tok.finishValue()
	if err != nil {
		return err
	}

	if ok {
		t.state = t.stateAfterValue()
	}

	return nil
}

func (t *textReader) stateAfterValue() textReaderState {
	switch t.ctx.peek() {
	case ctxInList, ctxInStruct:
		return trsAfterValue
	case ctxInSexp, ctxAtTopLevel:
		return trsBeforeTypeAnnotations
	default:
		panic("invalid ctx")
	}
}

// Explode explodes the reader state when something unexpected
// happens and further calls to Next are a bad idea.
func (t *textReader) explode(err error) {
	t.state = trsDone
	t.err = err
}
