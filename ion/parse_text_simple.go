/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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
	"bytes"
	"github.com/amzn/ion-go/internal/lex"
	"math"
	"strconv"
	"unicode/utf8"
)

// This file contains text parsers for Null, Padding, Bool, Symbol, String, Blob, and Clob.

func (p *parser) parseBinary(annotations []Symbol) Value {
	// Eat the IonBinaryStart.
	if item := p.next(); item.Type != lex.IonBinaryStart {
		p.panicf("expected binary start but found %q", item)
	}

	var value Value
	switch item := p.next(); item.Type {
	case lex.IonBlob:
		value = Blob{annotations: annotations, text: removeAny(item.Val, []byte(" \t\n\r\f\v"))}
	case lex.IonClobShort:
		value = Clob{annotations: annotations, text: p.doClobReplacements(item.Val)}
	case lex.IonClobLong:
		text := item.Val
		for peek := p.peekNonComment(); peek.Type == lex.IonClobLong; peek = p.peekNonComment() {
			text = append(text, p.next().Val...)
		}
		value = Clob{annotations: annotations, text: p.doClobReplacements(text)}
	default:
		p.panicf("expected a blob or clob but found %q", item)
	}

	if item := p.next(); item.Type != lex.IonBinaryEnd {
		p.panicf("expected binary end but found %q", item)
	}

	return value
}

func (p *parser) parseLongString(annotations []Symbol) String {
	text := []byte{}
	for item := p.peekNonComment(); item.Type == lex.IonStringLong; item = p.peekNonComment() {
		text = append(text, p.doStringReplacements(p.next().Val)...)
	}

	return String{annotations: annotations, text: text}
}

// decodeHex takes a slice that is the input buffer and decodes it into a slice containing UTF-8 code units.
// The start parameter is the index in the slice start of a hex-encoded rune to decode.
// The escapeLen parameter is the length of the escape to decode and must be a length
// that is a power of two, no longer than size of uint32, and within the length of the input slice.
func (p *parser) decodeHex(input []byte, start int, hexLen int) []byte {
	// Length must be a power of two and no larger that size of uint32.
	if hexLen <= 0 || hexLen > 8 || hexLen%2 != 0 {
		// calling code must give us a proper slice...
		p.panicf("hex escape is invalid length (%d)", hexLen)
	}
	// Construct a working slice to decode with.
	inLen := len(input)
	if start < 0 || start >= inLen {
		p.panicf("start of hex escape (%d) is negative or greater than or equal to input length (%d)", start, inLen)
	}
	end := start + hexLen
	if end > inLen {
		p.panicf("end of hex escape (%d) is greater than input length (%d)", end, inLen)
	}
	hex := input[start:end]

	// Decode the hex string into a UTF-32 scalar.
	buf := make([]byte, utf8.UTFMax)
	var cp uint64
	for i := 0; i < hexLen; i += 2 {
		octet, errParse := strconv.ParseUint(string(hex[i:i+2]), 16, 8)
		if errParse != nil {
			p.panicf("invalid hex escape (%q) was not caught by lexer: %v", hex, errParse)
		}
		cp = (cp << 8) | octet
	}

	// Now serialize back as UTF-8 code units.
	encodeLen := utf8.EncodeRune(buf, rune(cp))

	return buf[0:encodeLen]
}

// doStringReplacements converts escaped characters into their equivalent
// character while handling cases involving \r.
func (p *parser) doStringReplacements(str []byte) []byte {
	strLen := len(str)
	ret := make([]byte, 0, strLen)
	for index := 0; index < strLen; index++ {
		switch ch := str[index]; ch {
		case '\r':
			// Turn \r into \n.
			ret = append(ret, '\n')
			// We need to treat both "\r\n" and "\r" as "\n", so
			// skip extra if what comes next is "\n".
			if index < strLen-1 && str[index+1] == '\n' {
				index++
			}
		case '\\':
			if index >= strLen-1 {
				continue
			}
			// We have an escape character.  Do different things depending on
			// what we are escaping.
			switch next := str[index+1]; next {
			case '\r':
				// Newline being escaped.
				index++
				// Newline being escaped may be \r\n or just \r.
				if index < strLen-1 && str[index+1] == '\n' {
					index++
				}
			case '\n':
				// Newline being escaped.
				index++
			case 'r', 'n':
				// Treat both "\\r" and "\\n" as "\n"
				ret = append(ret, '\n')
				index++
			case '\'', '"', '\\':
				ret = append(ret, next)
				index++
			case 'x':
				index += 2
				data := p.decodeHex(str, index, 2)
				index += 1
				ret = append(ret, data...)
			case 'u':
				index += 2
				data := p.decodeHex(str, index, 4)
				index += 3
				ret = append(ret, data...)
			case 'U':
				index += 2
				data := p.decodeHex(str, index, 8)
				index += 7
				ret = append(ret, data...)
			default:
				// Don't have anything special to do with the next character, so
				// just add the current character and let the next one get added
				// as normal.
				ret = append(ret, ch)
			}
		default:
			ret = append(ret, ch)
		}
	}

	return ret
}

// doClobReplacements is like doStringReplacements but is restricted to escapes that CLOBs have.
func (p *parser) doClobReplacements(str []byte) []byte {
	strLen := len(str)
	ret := make([]byte, 0, strLen)
	for index := 0; index < strLen; index++ {
		switch ch := str[index]; ch {
		case '\r':
			// We normalize "\r" and "\r\n" as "\n".
			if index < strLen-1 && str[index+1] == '\n' {
				index++
			}
			ret = append(ret, '\n')
		case '\\':
			if index >= strLen-1 {
				continue
			}
			// We have an escape character.  Do different things depending on
			// what we are escaping.
			switch next := str[index+1]; next {
			case '\r':
				// Newline being escaped.
				index++
				// Newline being escaped may be \r\n or just \r.
				if index < strLen-1 && str[index+1] == '\n' {
					index++
				}
			case '\n':
				// Newline being escaped.
				index++
			case 'n':
				ret = append(ret, '\n')
				index++
			case 'r':
				ret = append(ret, '\r')
				index++
			case '\'', '"', '\\':
				ret = append(ret, next)
				index++
			case 'x':
				index += 2
				data := p.decodeHex(str, index, 2)
				index += 1
				ret = append(ret, data...)
			default:
				// Don't have anything special to do with the next character, so
				// just add the current character and let the next one get added
				// as normal.
				ret = append(ret, ch)
			}
		default:
			ret = append(ret, ch)
		}
	}

	return ret
}

// removeAny removes any occurrence of any of the given bytes from the given
// data and returns the result.
func removeAny(data []byte, any []byte) []byte {
	ret := make([]byte, 0, len(data))
	for _, ch := range data {
		if fnd := bytes.IndexRune(any, rune(ch)); fnd >= 0 {
			continue
		}
		ret = append(ret, ch)
	}

	return ret
}

func (p *parser) parseNull(annotations []Symbol) Null {
	item := p.next()

	switch string(item.Val) {
	case "null.blob":
		return Null{annotations: annotations, typ: TypeBlob}
	case "null.bool":
		return Null{annotations: annotations, typ: TypeBool}
	case "null.clob":
		return Null{annotations: annotations, typ: TypeClob}
	case "null.decimal":
		return Null{annotations: annotations, typ: TypeDecimal}
	case "null.float":
		return Null{annotations: annotations, typ: TypeFloat}
	case "null.int":
		return Null{annotations: annotations, typ: TypeInt}
	case "null.list":
		return Null{annotations: annotations, typ: TypeList}
	case "null.null":
		return Null{annotations: annotations, typ: TypeNull}
	case "null.sexp":
		return Null{annotations: annotations, typ: TypeSExp}
	case "null.string":
		return Null{annotations: annotations, typ: TypeString}
	case "null.struct":
		return Null{annotations: annotations, typ: TypeStruct}
	case "null.symbol":
		return Null{annotations: annotations, typ: TypeSymbol}
	case "null.timestamp":
		return Null{annotations: annotations, typ: TypeTimestamp}
	default:
		p.panicf("invalid null type: %v", item)
	}

	// Not reach-able, but Go doesn't know that p.panicf always panics.
	return Null{}
}

// parseSymbol parses a quoted or unquoted symbol.  There are several reserved symbols
// that hold special meaning, e.g. null.bool, that the lexer does not differentiate
// from other symbols.  This method treats the reserved symbols differently and returns
// the correct type.
func (p *parser) parseSymbol(annotations []Symbol, allowOperator bool) Value {
	item := p.next()
	// Include IonString here since struct fields are ostensibly symbols, but
	// quoted strings can be used to express them.  IonOperator is basically a
	// specialized version of Symbol.  Long strings have more extensive parsing
	// so back up and kick off that process.
	switch item.Type {
	case lex.IonOperator:
		if !allowOperator {
			p.panicf("operator not allowed here: %v", item)
		}
	case lex.IonNull:
		// Null has its own special parsing, so backup and give that a go.
		p.backup()
		return p.parseNull(annotations)
	case lex.IonSymbol, lex.IonSymbolQuoted, lex.IonString:
	case lex.IonStringLong:
		p.backup()
		return p.parseLongString(annotations)
	default:
		p.panicf("expected operation, symbol, quoted symbol, or string but found %v", item)
	}

	quoted := item.Type == lex.IonSymbolQuoted || item.Type == lex.IonString || item.Type == lex.IonStringLong
	if !quoted {
		switch string(item.Val) {
		case "true":
			return Bool{annotations: annotations, isSet: true, value: true}
		case "false":
			return Bool{annotations: annotations, isSet: true, value: false}
		case "nan":
			nan := math.NaN()
			return Float{isSet: true, value: &nan}
		case "null":
			return Null{annotations: annotations}
		}
	}

	// TODO: Figure out why the bytes in item.Val get overwritten when we don't
	//       make an explicit copy of the data.
	//return Symbol{annotations: annotations, quoted: quoted, text: doStringReplacements(item.Val)}
	return Symbol{annotations: annotations, quoted: quoted, text: append([]byte{}, p.doStringReplacements(item.Val)...)}
}
