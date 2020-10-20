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
	"bufio"
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
)

// trs is the state of the text reader.
type trs uint8

const (
	trsDone trs = iota
	trsBeforeFieldName
	trsBeforeTypeAnnotations
	trsBeforeContainer
	trsAfterValue
)

func (s trs) String() string {
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

// A textReader is a Reader that reads text Ion.
type textReader struct {
	reader

	tok   tokenizer
	state trs
	cat   Catalog
}

func newTextReaderBuf(in *bufio.Reader, cat Catalog) Reader {
	tr := textReader{
		cat: cat,
		tok: tokenizer{
			in: in,
		},
		state: trsBeforeTypeAnnotations,
	}
	tr.lst = V1SystemSymbolTable

	return &tr
}

// Next moves the reader to the next value.
func (t *textReader) Next() bool {
	if t.state == trsDone || t.eof {
		return false
	}

	// If we haven't fully read the current value, skip over it.
	err := t.finishValue()
	if err != nil {
		t.explode(err)
		return false
	}

	t.clear()

	// Loop until we've consumed enough tokens to know what the next value is.
	for {
		if err := t.tok.Next(); err != nil {
			t.explode(err)
			return false
		}

		var done bool
		var err error

		switch t.state {
		case trsAfterValue:
			done, err = t.nextAfterValue()
		case trsBeforeFieldName:
			done, err = t.nextBeforeFieldName()
		case trsBeforeTypeAnnotations:
			done, err = t.nextBeforeTypeAnnotations()
		default:
			panic(fmt.Sprintf("unexpected state: %v", t.state))
		}
		if err != nil {
			t.explode(err)
			return false
		}

		if done {
			// We're done reading tokens. If we hit the end of the current sequence,
			// return false. Otherwise, we've got a value for the caller.
			return !t.eof
		}
	}
}

// NextAfterValue moves to the next value when we're in the
// AfterValue state.
func (t *textReader) nextAfterValue() (bool, error) {
	tok := t.tok.Token()
	switch tok {
	case tokenComma:
		// There's another value coming; eat the comma and move to the
		// appropriate next state.
		switch t.ctx.peek() {
		case ctxInStruct:
			t.state = trsBeforeFieldName
		case ctxInList:
			t.state = trsBeforeTypeAnnotations
		default:
			panic(fmt.Sprintf("unexpected context: %v", t.ctx.peek()))
		}
		return false, nil

	case tokenCloseBrace:
		// No more values in this struct.
		if t.ctx.peek() == ctxInStruct {
			t.eof = true
			return true, nil
		}
		return false, &UnexpectedTokenError{"}", t.tok.Pos() - 1}

	case tokenCloseBracket:
		// No more values in this list.
		if t.ctx.peek() == ctxInList {
			t.eof = true
			return true, nil
		}
		return false, &UnexpectedTokenError{"]", t.tok.Pos() - 1}

	default:
		return false, &UnexpectedTokenError{tok.String(), t.tok.Pos() - 1}
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

	case tokenSymbol, tokenSymbolQuoted, tokenString, tokenLongString:
		// Read the field name.
		val, err := t.tok.ReadValue(tok)
		if err != nil {
			return false, err
		}
		if tok == tokenSymbol {
			if err := t.verifyUnquotedSymbol(val, "field name"); err != nil {
				return false, err
			}
		}

		if tok == tokenSymbolQuoted {
			t.fieldName = &SymbolToken{Text: &val, LocalSID: SymbolIDUnknown}
		} else {
			st, err := newSymbolToken(t.SymbolTable(), val)
			if err != nil {
				return false, err
			}
			t.fieldName = &st
		}

		// Skip over the following colon.
		if err = t.tok.Next(); err != nil {
			return false, err
		}
		if tok = t.tok.Token(); tok != tokenColon {
			return false, &UnexpectedTokenError{tok.String(), t.tok.Pos() - 1}
		}

		t.state = trsBeforeTypeAnnotations

		return false, nil

	default:
		return false, &UnexpectedTokenError{tok.String(), t.tok.Pos() - 1}
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
		return false, &UnexpectedEOFError{t.tok.Pos() - 1}

	case tokenSymbolOperator, tokenDot:
		if t.ctx.peek() != ctxInSexp {
			// Operators can only appear inside an sexp.
			return false, &UnexpectedTokenError{tok.String(), t.tok.Pos() - 1}
		}
		fallthrough

	case tokenSymbolQuoted, tokenSymbol:
		val, err := t.tok.ReadValue(tok)
		if err != nil {
			return false, err
		}

		ok, ws, err := t.tok.SkipDoubleColon()
		if err != nil {
			return false, err
		}

		if ok {
			// val was an annotation; remember it and keep going.
			if tok == tokenSymbol {
				if err := t.verifyUnquotedSymbol(val, "annotation"); err != nil {
					return false, err
				}
			} else if tok == tokenSymbolOperator {
				return false, &SyntaxError{
					"annotations that include a '" + val + "' must be enclosed in quotes", t.tok.Pos() - 1}
			}

			var token SymbolToken
			if tok == tokenSymbolQuoted {
				token = SymbolToken{Text: &val, LocalSID: SymbolIDUnknown}
			} else {
				token, err = newSymbolToken(t.SymbolTable(), val)
				if err != nil {
					return false, err
				}
			}

			t.annotations = append(t.annotations, token)
			return false, nil
		}

		if tok == tokenSymbolQuoted {
			t.value = &SymbolToken{Text: &val, LocalSID: SymbolIDUnknown}
			t.valueType = SymbolType
			t.state = t.stateAfterValue()
		} else {
			if err := t.onSymbol(val, tok, ws); err != nil {
				return false, err
			}
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
		t.value = StructType

		ctx := t.ctx.peek()
		if ctx == ctxAtTopLevel && isIonSymbolTable(t.annotations) {
			if t.IsNull() {
				t.clear()
				t.lst = V1SystemSymbolTable
				return false, nil
			}

			st, err := readLocalSymbolTable(t, t.cat)
			if err == nil {
				t.lst = st
				return false, nil
			}
			return false, err
		}

		return true, nil

	case tokenOpenBracket:
		t.state = trsBeforeContainer
		t.valueType = ListType
		t.value = ListType
		return true, nil

	case tokenOpenParen:
		t.state = trsBeforeContainer
		t.valueType = SexpType
		t.value = SexpType
		return true, nil

	case tokenCloseBracket:
		// No more values in this list.
		if t.ctx.peek() == ctxInList {
			t.eof = true
			return true, nil
		}
		return false, &UnexpectedTokenError{"]", t.tok.Pos() - 1}

	case tokenCloseParen:
		// No more values in this sexp.
		if t.ctx.peek() == ctxInSexp {
			t.eof = true
			return true, nil
		}
		return false, &UnexpectedTokenError{")", t.tok.Pos() - 1}

	default:
		return false, &UnexpectedTokenError{tok.String(), t.tok.Pos() - 1}
	}
}

// StepIn steps in to a container.
func (t *textReader) StepIn() error {
	if t.err != nil {
		return t.err
	}
	if t.state != trsBeforeContainer {
		return &UsageError{"Reader.StepIn", fmt.Sprintf("cannot step in to a %v", t.valueType)}
	}

	ctx := containerTypeToCtx(t.valueType)
	t.ctx.push(ctx)

	if ctx == ctxInStruct {
		t.state = trsBeforeFieldName
	} else {
		t.state = trsBeforeTypeAnnotations
	}
	t.clear()

	t.tok.SetFinished()
	return nil
}

// StepOut steps out of a container.
func (t *textReader) StepOut() error {
	if t.err != nil {
		return t.err
	}

	ctx := t.ctx.peek()
	if ctx == ctxAtTopLevel {
		return &UsageError{"Reader.StepOut", "cannot step out of top-level datagram"}
	}
	ctype := ctxToContainerType(ctx)

	// Finish off whatever value *inside* the container that we're currently reading.
	_, err := t.tok.FinishValue()
	if err != nil {
		t.explode(err)
		return err
	}

	// If we haven't seen the end of the container yet, skip values until we find it.
	if !t.eof {
		if err := t.tok.SkipContainerContents(ctype); err != nil {
			t.explode(err)
			return err
		}
	}

	t.ctx.pop()
	t.state = t.stateAfterValue()
	t.clear()
	t.eof = false

	return nil
}

// VerifyUnquotedSymbol checks for certain 'special' values that are returned from
// the tokenizer as symbols but cannot be used as field names or annotations.
func (t *textReader) verifyUnquotedSymbol(val string, ctx string) error {
	switch val {
	case "null", "true", "false", "nan":
		return &SyntaxError{fmt.Sprintf("unquoted keyword '%v' as %v", val, ctx), t.tok.Pos() - 1}
	}
	return nil
}

// OnSymbol handles finding a symbol-token value.
func (t *textReader) onSymbol(val string, tok token, ws bool) error {
	valueType := SymbolType
	var value interface{} = val

	if tok == tokenSymbol || tok == tokenSymbolOperator || tok == tokenDot {
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
		default:
			st, err := newSymbolToken(t.SymbolTable(), val)
			if err != nil {
				return err
			}
			value = &st
		}
	}

	t.state = t.stateAfterValue()
	t.valueType = valueType
	t.value = value

	return nil
}

// OnNull handles finding a null token.
func (t *textReader) onNull(ws bool) (Type, error) {
	if !ws {
		ok, err := t.tok.SkipDot()
		if err != nil {
			return NoType, err
		}
		if ok {
			return t.readNullType()
		}
	}
	return NullType, nil
}

// readNullType reads the null.{this} type symbol.
func (t *textReader) readNullType() (Type, error) {
	if err := t.tok.Next(); err != nil {
		return NoType, err
	}
	if t.tok.Token() != tokenSymbol {
		msg := fmt.Sprintf("invalid symbol null.%v", t.tok.Token())
		return NoType, &SyntaxError{msg, t.tok.Pos() - 1}
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
		msg := fmt.Sprintf("invalid symbol null.%v", t.tok.Token())
		return NoType, &SyntaxError{msg, t.tok.Pos() - 1}
	}
}

// OnNumber handles finding a number token.
func (t *textReader) onNumber(tok token) error {
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
			panic(fmt.Sprintf("unexpected type %v", tt))
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
		panic(fmt.Sprintf("unexpected token type %v", tok))
	}

	t.state = t.stateAfterValue()
	t.valueType = valueType
	t.value = value

	return nil
}

// OnTimestamp handles finding a timestamp token.
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

// OnLob handles finding a [bc]lob token.
func (t *textReader) onLob() error {
	c, err := t.tok.SkipLobWhitespace()
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

		val, err = t.tok.ReadShortClob()
		if err != nil {
			return err
		}
	} else if c == '\'' {
		// Long clob.
		ok, err := t.tok.IsTripleQuote()
		if err != nil {
			return err
		}
		if !ok {
			return t.tok.invalidChar(c)
		}

		valType = ClobType

		val, err = t.tok.ReadLongClob()
		if err != nil {
			return err
		}
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

// FinishValue finishes reading the current value, if there is one.
func (t *textReader) finishValue() error {
	ok, err := t.tok.FinishValue()
	if err != nil {
		return err
	}

	if ok {
		t.state = t.stateAfterValue()
	}

	return nil
}

func (t *textReader) stateAfterValue() trs {
	ctx := t.ctx.peek()
	switch ctx {
	case ctxInList, ctxInStruct:
		return trsAfterValue
	case ctxInSexp, ctxAtTopLevel:
		return trsBeforeTypeAnnotations
	default:
		panic(fmt.Sprintf("invalid ctx %v", ctx))
	}
}

// Explode explodes the reader state when something unexpected
// happens and further calls to Next are a bad idea.
func (t *textReader) explode(err error) {
	t.state = trsDone
	t.err = err
}
