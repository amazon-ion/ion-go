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
		value = Clob{annotations: annotations, text: doClobReplacements(item.Val)}
	case lex.IonClobLong:
		text := item.Val
		for peek := p.peekNonComment(); peek.Type == lex.IonClobLong; peek = p.peekNonComment() {
			text = append(text, p.next().Val...)
		}
		value = Clob{annotations: annotations, text: doClobReplacements(text)}
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
		text = append(text, doStringReplacements(p.next().Val)...)
	}

	return String{annotations: annotations, text: text}
}

// doStringReplacements converts escaped characters into their equivalent
// character while handling cases involving \r.
func doStringReplacements(str []byte) []byte {
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

// doClobReplacements is like doStringReplacements but is restricted to escapes that CLOBs have
func doClobReplacements(str []byte) []byte {
	strLen := len(str)
	ret := make([]byte, 0, strLen)
	for index := 0; index < strLen; index++ {
		switch ch := str[index]; ch {
		case '\r':
			// We need to treat "\r\n" as "\n", so skip extra if what comes next is "\n".
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
				// Decode the hex sequence
				index += 2
				octet, _ := strconv.ParseUint(string(str[index:index+2]), 16, 8)
				index += 2
				ret = append(ret, byte(octet))
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
	return Symbol{annotations: annotations, quoted: quoted, text: append([]byte{}, doStringReplacements(item.Val)...)}
}
