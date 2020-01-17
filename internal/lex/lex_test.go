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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	tEOF         = Item{Type: IonEOF}
	tBinaryStart = Item{Type: IonBinaryStart, Val: []byte("{{")}
	tBinaryEnd   = Item{Type: IonBinaryEnd, Val: []byte("}}")}
	tColon       = Item{Type: IonColon, Val: []byte(":")}
	tComma       = Item{Type: IonComma, Val: []byte(",")}
	tDoubleColon = Item{Type: IonDoubleColon, Val: []byte("::")}
	tListStart   = Item{Type: IonListStart, Val: []byte("[")}
	tListEnd     = Item{Type: IonListEnd, Val: []byte("]")}
	tSExpStart   = Item{Type: IonSExpStart, Val: []byte("(")}
	tSExpEnd     = Item{Type: IonSExpEnd, Val: []byte(")")}
	tStructStart = Item{Type: IonStructStart, Val: []byte("{")}
	tStructEnd   = Item{Type: IonStructEnd, Val: []byte("}")}
)

func TestLex(t *testing.T) {
	binInt := func(val string) Item { return Item{Type: IonIntBinary, Val: []byte(val)} }
	blob := func(val string) Item { return Item{Type: IonBlob, Val: []byte(val)} }
	blockComment := func(val string) Item { return Item{Type: IonCommentBlock, Val: []byte(val)} }
	clobLong := func(val string) Item { return Item{Type: IonClobLong, Val: []byte(val)} }
	clobShort := func(val string) Item { return Item{Type: IonClobShort, Val: []byte(val)} }
	decimal := func(val string) Item { return Item{Type: IonDecimal, Val: []byte(val)} }
	doubleQuote := func(val string) Item { return Item{Type: IonString, Val: []byte(val)} }
	err := func(val string) Item { return Item{Type: IonError, Val: []byte(val)} }
	float := func(val string) Item { return Item{Type: IonFloat, Val: []byte(val)} }
	hexInt := func(val string) Item { return Item{Type: IonIntHex, Val: []byte(val)} }
	integer := func(val string) Item { return Item{Type: IonInt, Val: []byte(val)} }
	lineComment := func(val string) Item { return Item{Type: IonCommentLine, Val: []byte(val)} }
	longString := func(val string) Item { return Item{Type: IonStringLong, Val: []byte(val)} }
	nullItem := func(val string) Item { return Item{Type: IonNull, Val: []byte(val)} }
	operator := func(val string) Item { return Item{Type: IonOperator, Val: []byte(val)} }
	quotedSym := func(val string) Item { return Item{Type: IonSymbolQuoted, Val: []byte(val)} }
	symbol := func(val string) Item { return Item{Type: IonSymbol, Val: []byte(val)} }
	timestamp := func(val string) Item { return Item{Type: IonTimestamp, Val: []byte(val)} }

	tests := []struct {
		name     string
		input    []byte
		expected []Item
	}{
		// Empty things.

		{
			name:     "nil input",
			expected: []Item{tEOF},
		},
		{
			name:     "only whitespace",
			input:    []byte(" \t\r\n\f\v"),
			expected: []Item{tEOF},
		},
		{
			name:     "empty list",
			input:    []byte("[]"),
			expected: []Item{tListStart, tListEnd, tEOF},
		},
		{
			name:     "empty struct",
			input:    []byte("{}"),
			expected: []Item{tStructStart, tStructEnd, tEOF},
		},
		{
			name:     "empty s-expression",
			input:    []byte("()"),
			expected: []Item{tSExpStart, tSExpEnd, tEOF},
		},
		{
			name:     "empty single quote",
			input:    []byte("''"),
			expected: []Item{quotedSym(""), tEOF},
		},
		{
			name:     "empty double quote",
			input:    []byte(`""`),
			expected: []Item{doubleQuote(""), tEOF},
		},
		{
			name:     "empty triple single quote",
			input:    []byte("''''''"),
			expected: []Item{longString(""), tEOF},
		},
		{
			name:     "empty symbol to symbol",
			input:    []byte("'':    abc"),
			expected: []Item{quotedSym(""), tColon, symbol("abc"), tEOF},
		},
		{
			name: "empty annotation to symbol",
			input: []byte("''::	abc"),
			expected: []Item{quotedSym(""), tDoubleColon, symbol("abc"), tEOF},
		},

		// Simple symbols, strings, nulls, and comments.

		{
			name:     "symbol to symbol",
			input:    []byte("'':\nabc"),
			expected: []Item{quotedSym(""), tColon, symbol("abc"), tEOF},
		},
		{
			name:     "symbol to quoted symbol",
			input:    []byte("'':\n'abc'"),
			expected: []Item{quotedSym(""), tColon, quotedSym("abc"), tEOF},
		},
		{
			name:     "symbol to double quoted symbol",
			input:    []byte(`'':"abc"`),
			expected: []Item{quotedSym(""), tColon, doubleQuote("abc"), tEOF},
		},
		{
			name:     "long string",
			input:    []byte("'''\"'''"),
			expected: []Item{longString(`"`), tEOF},
		},
		{
			name:     "long string with single quote",
			input:    []byte("'''''''"),
			expected: []Item{longString("'"), tEOF},
		},
		{
			name:     "quoted string",
			input:    []byte(`"\\"`),
			expected: []Item{doubleQuote(`\\`), tEOF},
		},
		{
			name:     "long string with single quotes",
			input:    []byte("''' ' '' '''"),
			expected: []Item{longString(" ' '' "), tEOF},
		},
		{
			name:  "some nulls",
			input: []byte("null null.bool"),
			// Since the first null doesn't have a period we don't know that it is a null until we parse.
			expected: []Item{symbol("null"), nullItem("null.bool"), tEOF},
		},
		{
			name:  "some nulls in a list",
			input: []byte("[\n\tnull,\n\tnull.bool]"),
			// Since the first null doesn't have a period we don't know that it is a null until we parse.
			expected: []Item{tListStart, symbol("null"), tComma, nullItem("null.bool"), tListEnd, tEOF},
		},
		{
			name:     "quoted null to null",
			input:    []byte("'null.bool':null.bool"),
			expected: []Item{quotedSym("null.bool"), tColon, nullItem("null.bool"), tEOF},
		},
		{
			name:     "we don't know that these are boolean",
			input:    []byte("true false"),
			expected: []Item{symbol("true"), symbol("false"), tEOF},
		},
		{
			name:     "line comment",
			input:    []byte("// Line Comment"),
			expected: []Item{lineComment(" Line Comment"), tEOF},
		},
		{
			name:     "block comment",
			input:    []byte("/* Block\n Comment*/"),
			expected: []Item{blockComment(" Block\n Comment"), tEOF},
		},

		// Numeric.

		{
			name:  "infinity",
			input: []byte("inf +inf -inf"),
			// "inf" must have a plus or minus on it to be considered a number.
			expected: []Item{symbol("inf"), {Type: IonInfinity, Val: []byte("+inf")}, {Type: IonInfinity, Val: []byte("-inf")}, tEOF},
		},
		{
			name:     "integers",
			input:    []byte("0 -1 1_2_3 0xFf 0Xe_d 0b10 0B1_0"),
			expected: []Item{integer("0"), integer("-1"), integer("1_2_3"), hexInt("0xFf"), hexInt("0Xe_d"), binInt("0b10"), binInt("0B1_0"), tEOF},
		},
		{
			name:     "decimals",
			input:    []byte("0. 0.123 -0.12d4 0D-0 0d+0 12_34.56_78"),
			expected: []Item{decimal("0."), decimal("0.123"), decimal("-0.12d4"), decimal("0D-0"), decimal("0d+0"), decimal("12_34.56_78"), tEOF},
		},
		{
			name:     "floats",
			input:    []byte("0E0 0.12e-4 -0e+0"),
			expected: []Item{float("0E0"), float("0.12e-4"), float("-0e+0"), tEOF},
		},
		{
			name:     "dates",
			input:    []byte("2019T 2019-10T 2019-10-30 2019-10-30T"),
			expected: []Item{timestamp("2019T"), timestamp("2019-10T"), timestamp("2019-10-30"), timestamp("2019-10-30T"), tEOF},
		},
		{
			name:     "times",
			input:    []byte("2019-10-30T22:30Z 2019-10-30T12:30:59+02:30 2019-10-30T12:30:59.999-02:30"),
			expected: []Item{timestamp("2019-10-30T22:30Z"), timestamp("2019-10-30T12:30:59+02:30"), timestamp("2019-10-30T12:30:59.999-02:30"), tEOF},
		},

		// Binary.

		{
			name:     "short blob",
			input:    []byte("{{+AB/}}"),
			expected: []Item{tBinaryStart, blob("+AB/"), tBinaryEnd, tEOF},
		},
		{
			name:     "padded blob with whitespace",
			input:    []byte("{{ + A\nB\t/abc= }}"),
			expected: []Item{tBinaryStart, blob("+ A\nB\t/abc= "), tBinaryEnd, tEOF},
		},
		{
			name:     "short clob",
			input:    []byte(`{{ "A\n" }}`),
			expected: []Item{tBinaryStart, clobShort(`A\n`), tBinaryEnd, tEOF},
		},
		{
			name:     "symbol to short clob",
			input:    []byte(`abc : {{ "A\n" }}`),
			expected: []Item{symbol("abc"), tColon, tBinaryStart, clobShort(`A\n`), tBinaryEnd, tEOF},
		},
		{
			name:     "symbol with comments to short clob",
			input:    []byte("abc : // Line\n/* Block */ {{ \"A\\n\" }}"),
			expected: []Item{symbol("abc"), tColon, lineComment(" Line"), blockComment(" Block "), tBinaryStart, clobShort(`A\n`), tBinaryEnd, tEOF},
		},
		{
			name:     "long clob",
			input:    []byte("{{ '''+AB/''' }}"),
			expected: []Item{tBinaryStart, clobLong("+AB/"), tBinaryEnd, tEOF},
		},
		{
			name:     "multiple long clobs",
			input:    []byte("{{ '''A\\nB'''\n'''foo''' }}"),
			expected: []Item{tBinaryStart, clobLong("A\\nB"), clobLong("foo"), tBinaryEnd, tEOF},
		},
		{
			name:     "quotes withing a long clob",
			input:    []byte("{{ ''' ' '' ''' }}"),
			expected: []Item{tBinaryStart, clobLong(" ' '' "), tBinaryEnd, tEOF},
		},

		// Containers

		{
			name:     "symbol to empty list",
			input:    []byte("abc\t:[]"),
			expected: []Item{symbol("abc"), tColon, tListStart, tListEnd, tEOF},
		},
		{
			name:  "list of things",
			input: []byte("[a, 1, ' ', {}, () /* comment */ ]"),
			expected: []Item{
				tListStart, symbol("a"), tComma,
				integer("1"), tComma,
				quotedSym(" "), tComma,
				tStructStart, tStructEnd, tComma,
				tSExpStart, tSExpEnd,
				blockComment(" comment "),
				tListEnd, tEOF},
		},
		{
			name:     "symbol to empty struct",
			input:    []byte("abc:\t{ // comment\n}"),
			expected: []Item{symbol("abc"), tColon, tStructStart, lineComment(" comment"), tStructEnd, tEOF},
		},
		{
			name:  "struct of things",
			input: []byte("{'a' : 1 , s:'', 'st': {}, '''lngstr''': 1,\nlst:[],\"sexp\":()}"),
			expected: []Item{tStructStart,
				quotedSym("a"), tColon, integer("1"), tComma,
				symbol("s"), tColon, quotedSym(""), tComma,
				quotedSym("st"), tColon, tStructStart, tStructEnd, tComma,
				longString("lngstr"), tColon, integer("1"), tComma,
				symbol("lst"), tColon, tListStart, tListEnd, tComma,
				doubleQuote("sexp"), tColon, tSExpStart, tSExpEnd,
				tStructEnd, tEOF},
		},
		{
			name:     "symbol to empty s-expression",
			input:    []byte("abc:\r\n()"),
			expected: []Item{symbol("abc"), tColon, tSExpStart, tSExpEnd, tEOF},
		},
		{
			name:  "s-expression of things",
			input: []byte("(a+b/c--( j * k))"),
			expected: []Item{tSExpStart,
				symbol("a"), operator("+"), symbol("b"), operator("/"), symbol("c"), operator("--"),
				tSExpStart, symbol("j"), operator("*"), symbol("k"), tSExpEnd,
				tSExpEnd, tEOF},
		},

		// Error cases

		{
			name:     "invalid start",
			input:    []byte("  世界"),
			expected: []Item{err("invalid start of a value: U+4E16 '世'")},
		},
		{
			name:     "invalid symbol value",
			input:    []byte("a:世界"),
			expected: []Item{symbol("a"), tColon, err("invalid start of a value: U+4E16 '世'")},
		},
		{
			name:     "unterminated block comment",
			input:    []byte("/* "),
			expected: []Item{err("unexpected end of file while lexing block comment")},
		},
		{
			name:     "rune error in line comment",
			input:    []byte("// \uFFFD"),
			expected: []Item{err("error parsing rune")},
		},
		{
			name:     "rune error in block comment",
			input:    []byte("/* \uFFFD */"),
			expected: []Item{err("error parsing rune")},
		},
		{
			name:     "double struct end",
			input:    []byte("{} a }"),
			expected: []Item{tStructStart, tStructEnd, symbol("a"), err("unexpected closing of container")},
		},
		{
			name:     "double list end",
			input:    []byte("[] a ]"),
			expected: []Item{tListStart, tListEnd, symbol("a"), err("unexpected closing of container")},
		},
		{
			name:     "double sexp end",
			input:    []byte("() a )"),
			expected: []Item{tSExpStart, tSExpEnd, symbol("a"), err("unexpected closing of container")},
		},
		{
			name:     "mismatch: struct list",
			input:    []byte("{]"),
			expected: []Item{tStructStart, err("expected closing of struct but found ]")},
		},
		{
			name:     "mismatch: list sexp",
			input:    []byte("[)"),
			expected: []Item{tListStart, err("expected closing of list but found )")},
		},
		{
			name:     "mismatch: sexp struct",
			input:    []byte("(}"),
			expected: []Item{tSExpStart, err("expected closing of s-expression but found }")},
		},
		{
			name:     "invalid escaped char in long string",
			input:    []byte("'''\\c'''"),
			expected: []Item{err("invalid character after escape: U+0063 'c'")},
		},
		{
			name:     "invalid escaped char in short string",
			input:    []byte("\"\\c\""),
			expected: []Item{err("invalid character after escape: U+0063 'c'")},
		},
		{
			name:     "invalid escaped char in quoted symbol",
			input:    []byte("'\\c'"),
			expected: []Item{err("invalid character after escape: U+0063 'c'")},
		},
		{
			name:     "unterminated long string",
			input:    []byte("'''"),
			expected: []Item{err("unterminated long string")},
		},
		{
			name:     "unterminated string",
			input:    []byte(`"`),
			expected: []Item{err("unterminated quoted string")},
		},
		{
			name:     "unterminated quoted symbol",
			input:    []byte(`'`),
			expected: []Item{err("unterminated quoted symbol")},
		},
		{
			name:     "escaping EOF in string",
			input:    []byte(`"a\`),
			expected: []Item{err("unterminated sequence")},
		},
		{
			name:     "escaping EOF in quoted symbol",
			input:    []byte(`'a\`),
			expected: []Item{err("unterminated sequence")},
		},
		{
			name:     "escaping a non-hex character for hex escape",
			input:    []byte(`'\xAG'`),
			expected: []Item{err("invalid character as part of hex escape: U+0047 'G'")},
		},
		{
			name:     "escaping a non-hex character for unicode escape",
			input:    []byte(`'\u000G'`),
			expected: []Item{err("invalid character as part of unicode escape: U+0047 'G'")},
		},
		{
			name:     "invalid start for a \\U escape",
			input:    []byte(`'\U1000'`),
			expected: []Item{err("invalid character as part of unicode escape: U+0031 '1'")},
		},
		{
			name:     "invalid \\U000 escape",
			input:    []byte(`'\U000G'`),
			expected: []Item{err("invalid character as part of unicode escape: U+0047 'G'")},
		},
		{
			name:     "invalid start for a \\U0010 escape",
			input:    []byte(`'\U001G'`),
			expected: []Item{err("invalid character as part of unicode escape: U+0047 'G'")},
		},
		{
			name:     "invalid \\U escape",
			input:    []byte(`'\U0010G000'`),
			expected: []Item{err("invalid character as part of unicode escape: U+0047 'G'")},
		},
		{
			name:     "invalid character in symbol",
			input:    []byte("null世int"),
			expected: []Item{err("bad character as part of symbol: U+4E16 '世'")},
		},
		{
			name:     "invalid character in quoted symbol",
			input:    []byte("'null\u0007'"),
			expected: []Item{err("bad character as part of quoted symbol: U+0007")},
		},
		{
			name:     "invalid character in quoted string",
			input:    []byte("\"null\u0007\""),
			expected: []Item{err("bad character as part of string: U+0007")},
		},
		{
			name:     "invalid character in long string",
			input:    []byte("'''null\u0007'''"),
			expected: []Item{err("bad character as part of long string: U+0007")},
		},
		{
			name:     "invalid character after long string",
			input:    []byte("'''null'''\u0007"),
			expected: []Item{longString("null"), err("invalid start of a value: U+0007")},
		},
		{
			name:     "int with leading zeros",
			input:    []byte("007"),
			expected: []Item{err("leading zeros are not allowed for decimal integers")},
		},
		{
			name:     "decimal with leading zeros",
			input:    []byte("03.4"),
			expected: []Item{err("leading zeros are not allowed for decimals")},
		},
		{
			name:     "float with leading zeros",
			input:    []byte("03.4e0"),
			expected: []Item{err("leading zeros are not allowed for floats")},
		},
		{
			name:     "decimal with trailing underscore",
			input:    []byte("123.456_"),
			expected: []Item{err("numbers cannot end with an underscore")},
		},
		{
			name:     "hex designator followed by underscore",
			input:    []byte("0x_0"),
			expected: []Item{err("underscore must not be at start of hex or binary number")},
		},
		{
			name:     "hex followed by underscore",
			input:    []byte("0x0_"),
			expected: []Item{err("number span cannot end with an underscore")},
		},
		{
			name:     "underscore before period",
			input:    []byte("1_."),
			expected: []Item{err("number span cannot end with an underscore")},
		},
		{
			name:     "underscore after period",
			input:    []byte("1._"),
			expected: []Item{err("underscore may not follow a period")},
		},
		{
			name:     "underscore after negative sign binary",
			input:    []byte("-_0b1010"),
			expected: []Item{err("underscore must not be after negative sign")},
		},
		{
			name:     "repeated underscores",
			input:    []byte("1__0"),
			expected: []Item{err("number span cannot end with an underscore")},
		},
		{
			name:     "not a numeric stop character",
			input:    []byte("1a"),
			expected: []Item{err("invalid numeric stop character: U+0061 'a'")},
		},
		{
			name:     "year 0000",
			input:    []byte("0000T"),
			expected: []Item{err("year must be greater than zero")},
		},
		{
			name:     "year 0000 with month",
			input:    []byte("0000T-01"),
			expected: []Item{err("year must be greater than zero")},
		},
		{
			name:     "month 20",
			input:    []byte("2019-20T"),
			expected: []Item{err("invalid character as month part of timestamp: U+0032 '2'")},
		},
		{
			name:     "month 13",
			input:    []byte("2019-13T"),
			expected: []Item{err("invalid month 13")},
		},
		{
			name:     "month 0",
			input:    []byte("2019-00T"),
			expected: []Item{err("month must be greater than zero")},
		},
		{
			name:     "year and month must have a T",
			input:    []byte("2019-12 "),
			expected: []Item{err("invalid character after month part of timestamp: U+0020 ' '")},
		},
		{
			name:     "not a numeric stop character after year and month",
			input:    []byte("2019-12Ta"),
			expected: []Item{err("invalid timestamp stop character: U+0061 'a'")},
		},
		{
			name:     "day 40",
			input:    []byte("2019-12-40T"),
			expected: []Item{err("invalid character as day part of timestamp: U+0034 '4'")},
		},
		{
			name:     "day 32",
			input:    []byte("2019-12-32T"),
			expected: []Item{err("invalid day 32")},
		},
		{
			name:     "April 31",
			input:    []byte("2019-04-31T"),
			expected: []Item{err("invalid day 31 for month 4")},
		},
		{
			name:     "day 0",
			input:    []byte("2019-12-00T"),
			expected: []Item{err("day must be greater than zero")},
		},
		{
			name:     "not a numeric stop character after year month and day",
			input:    []byte("2019-12-30a"),
			expected: []Item{err("invalid character after day part of timestamp: U+0061 'a'")},
		},
		{
			name:     "not a numeric character after year month and day",
			input:    []byte("2019-12-30Ta"),
			expected: []Item{err("invalid character as hour/minute part of timestamp: U+0061 'a'")},
		},
		{
			name:     "hour 30",
			input:    []byte("2019-12-30T30:00Z"),
			expected: []Item{err("invalid character as hour/minute part of timestamp: U+0033 '3'")},
		},
		{
			name:     "hour 24",
			input:    []byte("2019-12-30T24:00Z"),
			expected: []Item{err("invalid hour 24")},
		},
		{
			name:     "minute 60",
			input:    []byte("2019-12-30T12:60Z"),
			expected: []Item{err("invalid character as hour/minute part of timestamp: U+0036 '6'")},
		},
		{
			name:     "second 60",
			input:    []byte("2019-12-30T12:34:60Z"),
			expected: []Item{err("invalid character as seconds part of timestamp: U+0036 '6'")},
		},
		{
			name:     "no fractional seconds",
			input:    []byte("2019-12-30T12:34:00.Z"),
			expected: []Item{err("missing fractional seconds value")},
		},
		{
			name:     "timezone offset hour 30",
			input:    []byte("2019-12-30T12:34:59+30:00"),
			expected: []Item{err("invalid character as hour/minute part of timezone: U+0033 '3'")},
		},
		{
			name:     "timezone offset hour 24",
			input:    []byte("2019-12-30T12:34:59+24:00"),
			expected: []Item{err("invalid hour offset 24")},
		},
		{
			name:     "timezone offset minute 60",
			input:    []byte("2019-12-30T12:34:59+10:60"),
			expected: []Item{err("invalid character as hour/minute part of timezone: U+0036 '6'")},
		},
		{
			name:     "invalid timezone offset",
			input:    []byte("2019-12-30T12:34:59a"),
			expected: []Item{err("invalid character as timezone part of timestamp: U+0061 'a'")},
		},
		{
			name:     "invalid character after timezone offset",
			input:    []byte("2019-12-30T12:34:59Za"),
			expected: []Item{err("invalid timestamp stop character: U+0061 'a'")},
		},
		{
			name:     "unterminated blob",
			input:    []byte("{{abcd"),
			expected: []Item{tBinaryStart, err("unterminated blob")},
		},
		{
			name:     "blob with only one ending brace",
			input:    []byte("{{abcd}a"),
			expected: []Item{tBinaryStart, err("invalid end to blob, expected } but found: U+0061 'a'")},
		},
		{
			name:     "blob with padding in the middle",
			input:    []byte("{{ab=cd}}"),
			expected: []Item{tBinaryStart, err("base64 character found after padding character")},
		},
		{
			name:     "invalid blob character",
			input:    []byte("{{abc.}}"),
			expected: []Item{tBinaryStart, err("invalid rune as part of blob string: U+002E '.'")},
		},
		{
			name:     "invalid base64 encoding",
			input:    []byte("{{ab=}}"),
			expected: []Item{tBinaryStart, err("invalid base64 encoding")},
		},
		{
			name:     "unterminated short clob escaping EOF",
			input:    []byte(`{{ "ab\`),
			expected: []Item{tBinaryStart, err("unterminated short clob")},
		},
		{
			name:     "clob escaping c",
			input:    []byte(`{{ "ab\c" }}`),
			expected: []Item{tBinaryStart, err("invalid character after escape: U+0063 'c'")},
		},
		{
			name:     "clob invalid hex escape",
			input:    []byte(`{{ "ab\x0g" }}`),
			expected: []Item{tBinaryStart, err("invalid character as part of hex escape: U+0067 'g'")},
		},
		{
			name:     "clob unicode escape",
			input:    []byte(`{{ "ab\u0067" }}`),
			expected: []Item{tBinaryStart, err("unicode escape is not valid in clob")},
		},
		{
			name:     "unterminated short clob no closing quote",
			input:    []byte(`{{ "ab`),
			expected: []Item{tBinaryStart, err("unterminated short clob")},
		},
		{
			name:     "unterminated short clob no closing brace",
			input:    []byte(`{{ "ab" a`),
			expected: []Item{tBinaryStart, clobShort("ab"), err("invalid end to short clob, expected } but found: U+0061 'a'")},
		},
		{
			name:     "unterminated short clob only one closing brace",
			input:    []byte(`{{ "ab" }a`),
			expected: []Item{tBinaryStart, clobShort("ab"), err("invalid end to short clob, expected second } but found: U+0061 'a'")},
		},
		{
			name:     "invalid short clob text",
			input:    []byte("{{ \"ab\u0007\" }}"),
			expected: []Item{tBinaryStart, err("invalid rune as part of short clob string: U+0007")},
		},
		{
			name:     "unterminated long clob no closing brace",
			input:    []byte(`{{ '''ab''' a`),
			expected: []Item{tBinaryStart, clobLong("ab"), err("expected end of a Clob or start of a long string but found: U+0061 'a'")},
		},
		{
			name:     "unterminated long clob only one closing brace",
			input:    []byte(`{{ '''ab''' }a`),
			expected: []Item{tBinaryStart, clobLong("ab"), err("expected a second } but found: U+0061 'a'")},
		},
		{
			name:     "unterminated long clob escaping EOF",
			input:    []byte(`{{ '''ab\`),
			expected: []Item{tBinaryStart, err("unterminated long clob")},
		},
		{
			name:     "unterminated long clob no closing quotes",
			input:    []byte(`{{ '''ab`),
			expected: []Item{tBinaryStart, err("unterminated long clob")},
		},
		{
			name:     "invalid long clob text",
			input:    []byte("{{ '''ab\u0007''' }}"),
			expected: []Item{tBinaryStart, err("invalid rune as part of long clob string: U+0007")},
		},
	}
	for _, tst := range tests {
		test := tst
		t.Run(test.name, func(t *testing.T) {
			out := runLexer(test.input)
			// Only focusing on the type and value for these tests.
			if diff := cmp.Diff(test.expected, out, cmpopts.EquateEmpty(), cmpopts.IgnoreFields(Item{}, "Pos")); diff != "" {
				t.Log("Expected:", test.expected)
				t.Log("Found:   ", out)
				t.Error("(-expected, +found)", diff)
			}
		})
	}
}

// Gather the items emitted from the Lexer into a slice.
func runLexer(input []byte) []Item {
	x := New(input)
	var items []Item
	for {
		item := x.NextItem()
		items = append(items, item)
		if item.Type == IonEOF || item.Type == IonError {
			break
		}
	}
	return items
}
