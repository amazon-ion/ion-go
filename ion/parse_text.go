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
	"fmt"
	"io"
	"io/ioutil"
	"runtime"

	"github.com/amzn/ion-go/internal/lex"
	"github.com/pkg/errors"
)

// ParseText parses all of the bytes from the given Reader into an
// instance of Digest.  Assume that the entire contents of reading the
// given reader will be kept in memory.  This allows for lazy evaluation
// of values, e.g. don't turn "1234" into an int unless its value is
// accessed.
func ParseText(reader io.Reader) (*Digest, error) {
	text, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read ion text to parse")
	}

	t := &parser{}
	if err := t.Parse(text); err != nil {
		return nil, err
	}

	return t.digest, nil
}

type parser struct {
	digest    *Digest
	lex       *lex.Lexer
	token     [3]lex.Item
	peekCount int
}

// panicf formats the given error and panics.  Panicking provides a quick exit from
// whatever depth of parsing we are at and is recovered from during the call to Parse().
func (p *parser) panicf(format string, args ...interface{}) {
	format = fmt.Sprintf("parsing line %d - %s", p.lex.LineNumber(), format)
	panic(fmt.Errorf(format, args...))
}

func (p *parser) next() lex.Item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lex.NextItem()
		if p.token[0].Type == lex.IonError {
			p.panicf("Encountered error lexing the next value: %v", p.token[0])
		}
	}
	return p.token[p.peekCount]
}

// Backs the input stream up one item.
func (p *parser) backup() {
	p.peekCount++
}

// Returns the next non-comment item, consuming all comments.
func (p *parser) nextNonComment() (item lex.Item) {
	for {
		item = p.next()
		if item.Type != lex.IonCommentBlock && item.Type != lex.IonCommentLine {
			break
		}
	}
	return item
}

// Returns but does not consume the next non-comment token,
// while consuming all comments.
func (p *parser) peekNonComment() lex.Item {
	var item lex.Item
	for {
		item = p.next()
		if item.Type != lex.IonCommentBlock && item.Type != lex.IonCommentLine {
			break
		}
	}
	p.backup()
	return item
}

// recover is a handler that turns panics into error returns from Parse.  The
// panic is retained if it is of the runtime.Error variety.
func (p *parser) recover(err *error) {
	e := recover()
	if e != nil {
		// We only want to capture errors that we panic on.
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		*err = e.(error)
	}
	return
}

// Parses the given input string and makes the resulting parse tree available
// in the Root object of the tree.  The nodes that are parsed are assigned the
// given priority.
func (p *parser) Parse(text []byte) (err error) {
	defer p.recover(&err)
	p.lex = lex.New(text)
	p.parse()
	return nil
}

func (p *parser) parse() {
	var values []Value
	for item := p.peekNonComment(); item.Type != lex.IonEOF; item = p.peekNonComment() {
		values = append(values, p.parseValue(false))
	}

	// If there is a version marker then it will be the first symbol.
	if len(values) > 0 && values[0].Type() == TypeSymbol {
		sym := values[0].Text()
		if bytes.HasPrefix(sym, []byte("$ion_")) && !bytes.Equal(sym, []byte("$ion_1_0")) {
			p.panicf("unsupported ION version %s", sym)
		}
	}

	p.digest = &Digest{values: values}
}

func (p *parser) parseValue(allowOperator bool) Value {
	var annotations []Symbol
	for {
		item := p.peekNonComment()
		switch item.Type {
		case lex.IonError:
			p.panicf("unable to parse input: " + item.String())
		case lex.IonBinaryStart:
			return p.parseBinary(annotations)
		case lex.IonDecimal:
			return p.parseDecimal(annotations)
		case lex.IonFloat, lex.IonInfinity:
			return p.parseFloat(annotations)
		case lex.IonInt:
			return p.parseInt(annotations, intBase10)
		case lex.IonIntBinary:
			return p.parseInt(annotations, intBase2)
		case lex.IonIntHex:
			return p.parseInt(annotations, intBase16)
		case lex.IonListStart:
			return p.parseList(annotations)
		case lex.IonNull:
			return p.parseNull(annotations)
		case lex.IonOperator:
			if !allowOperator {
				p.panicf("operator not allowed outside s-expression %v", item)
			}
			return p.parseSymbol(annotations, true)
		case lex.IonSExpStart:
			return p.parseSExpression(annotations)
		case lex.IonString:
			return String{annotations: annotations, text: p.doStringReplacements(p.next().Val)}
		case lex.IonStringLong:
			return p.parseLongString(annotations)
		case lex.IonStructStart:
			return p.parseStruct(annotations)
		case lex.IonSymbol, lex.IonSymbolQuoted:
			symbol := p.parseSymbol(annotations, false)
			if item = p.peekNonComment(); item.Type == lex.IonDoubleColon {
				annotation, ok := symbol.(Symbol)
				if !ok {
					p.panicf("invalid annotation type %q", symbol.Type())
				}
				// Annotations themselves don't have annotations.
				annotation.annotations = nil
				annotations = append(annotations, annotation)
				fmt.Printf("Annotations: %#v\n", annotations)
				p.nextNonComment()
				continue
			}
			return symbol
		case lex.IonTimestamp:
			return p.parseTimestamp(annotations)
		default:
			p.panicf("unexpected item type %q", item.Type)
		}
	}
}
