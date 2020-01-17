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

package lex

import (
	"fmt"
	"strconv"
)

// A token returned from the Lexer.
type Item struct {
	Type itemType // The type of this Item.
	Pos  int      // The starting position, in bytes, of this Item in the input.
	Val  []byte   // The value of this Item.
}

// String satisfies Stringer.
func (i Item) String() string {
	_, typeKnown := itemTypeMap[i.Type]
	switch {
	case i.Type == IonEOF:
		return "EOF"
	case i.Type == IonIllegal:
		return string(i.Val)
	case len(i.Val) > 75:
		return fmt.Sprintf("<%.75s>...", i.Val)
	case !typeKnown:
		return fmt.Sprintf("%s <%s>", i.Type, i.Val)
	}
	// We use the '<' and '>' characters because both single and double quotes are
	// used extensively as are brackets, braces, and parens.
	return fmt.Sprintf("<%s>", i.Val)
}

const (
	IonIllegal itemType = iota
	IonError
	IonEOF

	IonBlob
	IonClobLong
	IonClobShort
	IonCommentBlock
	IonCommentLine
	IonDecimal
	IonFloat
	IonInfinity
	IonInt
	IonIntBinary
	IonIntHex
	IonList
	IonNull
	IonSExp
	IonString
	IonStringLong
	IonStruct
	IonSymbol
	IonSymbolQuoted
	IonTimestamp

	IonBinaryStart // {{
	IonBinaryEnd   // }}
	IonColon       // :
	IonDoubleColon // ::
	IonComma       // ,
	IonDot         // .
	IonOperator    // One of !#%&*+\\-/;<=>?@^`|~
	IonStructStart // {
	IonStructEnd   // }
	IonListStart   // [
	IonListEnd     // ]
	IonSExpStart   // (
	IonSExpEnd     // )
)

// Type of the lex Item.
type itemType int

var itemTypeMap = map[itemType]string{
	IonIllegal: "Illegal",
	IonError:   "Error",
	IonEOF:     "EOF",
	IonNull:    "Null",

	IonBlob:         "Blob",
	IonClobLong:     "ClobLong",
	IonClobShort:    "ClobShort",
	IonCommentBlock: "BlockComment",
	IonCommentLine:  "LineComment",
	IonDecimal:      "Decimal",
	IonInfinity:     "Infinity",
	IonInt:          "Int",
	IonIntBinary:    "BinaryInt",
	IonIntHex:       "HexInt",
	IonFloat:        "Float",
	IonList:         "List",
	IonSExp:         "SExp",
	IonString:       "String",
	IonStringLong:   "LongString",
	IonStruct:       "Struct",
	IonSymbol:       "Symbol",
	IonSymbolQuoted: "QuotedSymbol",
	IonTimestamp:    "Timestamp",

	IonBinaryStart: "{{",
	IonBinaryEnd:   "}}",
	IonColon:       ":",
	IonDoubleColon: "::",
	IonComma:       ",",
	IonDot:         ".",
	IonOperator:    "Operator",
	IonStructStart: "{",
	IonStructEnd:   "}",
	IonListStart:   "[",
	IonListEnd:     "]",
	IonSExpStart:   "(",
	IonSExpEnd:     ")",
}

func (i itemType) String() string {
	if s, ok := itemTypeMap[i]; ok {
		return s
	}
	return "Unknown itemType " + strconv.Itoa(int(i))
}
