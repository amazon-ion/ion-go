/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

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
