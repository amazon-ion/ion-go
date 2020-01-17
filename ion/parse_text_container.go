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
	"github.com/amzn/ion-go/internal/lex"
)

// This file contains text parsers for List, SExp, and Struct.

func (p *parser) parseList(annotations []Symbol) List {
	if item := p.next(); item.Type != lex.IonListStart {
		p.panicf("expected list start but found %s", item)
	}

	var values []Value
	var prev lex.Item
	for item := p.peekNonComment(); item.Type != lex.IonListEnd && item.Type != lex.IonError; prev, item = item, p.peekNonComment() {
		if item.Type == lex.IonComma {
			if prev.Type == 0 {
				p.panicf("list may not start with a comma")
			}
			if prev.Type == lex.IonComma {
				p.panicf("list must have a value defined between commas")
			}
			p.next()
			continue
		} else if prev.Type != lex.IonComma && prev.Type != 0 {
			p.panicf("list items must be separated by commas")
		}

		values = append(values, p.parseValue(false))
	}

	// Eat the end of the list.  An improperly terminated list creates an error
	// before we hit this spot, but check it just to be safe.
	if item := p.next(); item.Type != lex.IonListEnd {
		p.panicf("expected list end but found %s", item)
	}

	return List{annotations: annotations, values: values}
}

func (p *parser) parseSExpression(annotations []Symbol) SExp {
	if item := p.next(); item.Type != lex.IonSExpStart {
		p.panicf("expected s-expression start but found %s", item)
	}

	var values []Value
	for item := p.peekNonComment(); item.Type != lex.IonSExpEnd && item.Type != lex.IonError; item = p.peekNonComment() {
		values = append(values, p.parseValue(true))
	}

	// Eat the end of the s-expression.    An improperly terminated s-expression creates an error
	// before we hit this spot, but check it just to be safe.
	if item := p.next(); item.Type != lex.IonSExpEnd {
		p.panicf("expected s-expression end but found %s", item)
	}

	return SExp{annotations: annotations, values: values}
}

func (p *parser) parseStruct(annotations []Symbol) Struct {
	if item := p.next(); item.Type != lex.IonStructStart {
		p.panicf("expected struct start but found %s", item)
	}

	var values []StructField
	var prev lex.Item
	for item := p.peekNonComment(); item.Type != lex.IonStructEnd && item.Type != lex.IonError; prev, item = item, p.peekNonComment() {
		if item.Type == lex.IonComma {
			if prev.Type == 0 {
				p.panicf("struct may not start with a comma")
			}
			if prev.Type == lex.IonComma {
				p.panicf("struct must have a field defined between commas")
			}
			p.next()
			continue
		} else if prev.Type != lex.IonComma && prev.Type != 0 {
			p.panicf("struct fields must be separated by commas")
		}

		// Struct field names are not allowed to have annotations.
		// It's possible for the symbol that gets parsed to be a special reserved
		// Symbol, e.g. true, that resolves to a non-Symbol type.  We need to put
		// that back into a Symbol for the struct.
		parsed := p.parseSymbol(nil, false)
		if pt := parsed.Type(); pt == TypeBool || pt == TypeNull || pt == TypeFloat || pt == TypeDecimal {
			p.panicf("invalid type for field: %s", pt)
		}

		symbol, ok := parsed.(Symbol)
		if !ok {
			symbol = Symbol{text: parsed.Text()}
		}

		if item = p.nextNonComment(); item.Type != lex.IonColon {
			p.panicf("expected colon after symbol in struct but found %s", item)
		}
		value := p.parseValue(false)
		values = append(values, StructField{Symbol: symbol, Value: value})
	}

	// Eat the end of the structure.  An improperly terminated struct creates an error
	// before we hit this spot, but check it just to be safe.
	if item := p.next(); item.Type != lex.IonStructEnd {
		p.panicf("expected struct end but found %s", item)
	}

	return Struct{annotations: annotations, fields: values}
}
