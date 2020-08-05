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
	"testing"
)

func TestSkipNumber(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipNumber)

	test("", -1)
	test("0", -1)
	test("-1234567890,", ',')
	test("1.2 ", ' ')
	test("1d45\n", '\n')
	test("1.4e-12//", '/')

	testErr("1.2d3d", "ion: unexpected rune 'd' (offset 5)")
}

func TestSkipBinary(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipBinary)

	test("0b0", -1)
	test("-0b10 ", ' ')
	test("0b010101,", ',')

	testErr("0b2", "ion: unexpected rune '2' (offset 2)")
}

func TestSkipHex(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipHex)

	test("0x0", -1)
	test("-0x0F ", ' ')
	test("0x1234567890abcdefABCDEF,", ',')

	testErr("0x0G", "ion: unexpected rune 'G' (offset 3)")
}

func TestSkipTimestamp(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipTimestamp)

	test("2001T", -1)
	test("2001-01T,", ',')
	test("2001-01-02}", '}')
	test("2001-01-02T ", ' ')
	test("2001-01-02T+00:00\t", '\t')
	test("2001-01-02T-00:00\n", '\n')
	test("2001-01-02T03:04+00:00 ", ' ')
	test("2001-01-02T03:04-00:00 ", ' ')
	test("2001-01-02T03:04Z ", ' ')
	test("2001-01-02T03:04z ", ' ')
	test("2001-01-02T03:04:05Z ", ' ')
	test("2001-01-02T03:04:05+00:00 ", ' ')
	test("2001-01-02T03:04:05.666Z ", ' ')
	test("2001-01-02T03:04:05.666666z ", ' ')

	testErr("", "ion: unexpected end of input (offset 0)")
	testErr("2001", "ion: unexpected end of input (offset 4)")
	testErr("2001z", "ion: unexpected rune 'z' (offset 4)")
	testErr("20011", "ion: unexpected rune '1' (offset 4)")
	testErr("2001-0", "ion: unexpected end of input (offset 6)")
	testErr("2001-01", "ion: unexpected end of input (offset 7)")
	testErr("2001-01-02Tz", "ion: unexpected rune 'z' (offset 11)")
	testErr("2001-01-02T03", "ion: unexpected end of input (offset 13)")
	testErr("2001-01-02T03z", "ion: unexpected rune 'z' (offset 13)")
	testErr("2001-01-02T03:04x ", "ion: unexpected rune 'x' (offset 16)")
	testErr("2001-01-02T03:04:05x ", "ion: unexpected rune 'x' (offset 19)")
}

func TestSkipSymbol(t *testing.T) {
	test, _ := testSkip(t, (*tokenizer).skipSymbol)

	test("f", -1)
	test("foo:", ':')
	test("foo,", ',')
	test("foo ", ' ')
	test("foo\n", '\n')
	test("foo]", ']')
	test("foo}", '}')
	test("foo)", ')')
	test("foo\\n", '\\')
}

func TestSkipSymbolQuoted(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipSymbolQuoted)

	test("'", -1)
	test("foo',", ',')
	test("foo\\'bar':", ':')
	test("foo\\\nbar',", ',')

	testErr("foo", "ion: unexpected end of input (offset 3)")
	testErr("foo\n", "ion: unexpected rune '\\n' (offset 3)")
}

func TestSkipSymbolOperator(t *testing.T) {
	test, _ := testSkip(t, (*tokenizer).skipSymbolOperator)

	test("+", -1)
	test("++", -1)
	test("+= ", ' ')
	test("%b", 'b')
}

func TestSkipString(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipString)

	test("\"", -1)
	test("\",", ',')
	test("foo\\\"bar\"], \"\"", ']')
	test("foo\\\nbar\" \t\t\t", ' ')

	testErr("foobar", "ion: unexpected end of input (offset 6)")
	testErr("foobar\n", "ion: unexpected rune '\\n' (offset 6)")
}

func TestSkipLongString(t *testing.T) {
	test, _ := testSkip(t, (*tokenizer).skipLongString)

	test("'''", -1)
	test("''',", ',')
	test("abc''',", ',')
	test("abc'''   }", '}')
	test("abc''' /*more*/ '''def'''\t//more\r\n]", ']')
}

func TestSkipBlob(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipBlob)

	test("}}", -1)
	test("oogboog}},{{}}", ',')
	test("'''not encoded'''}}\n", '\n')

	testErr("", "ion: unexpected end of input (offset 1)")
	testErr("oogboog", "ion: unexpected end of input (offset 7)")
	testErr("oogboog}", "ion: unexpected end of input (offset 8)")
	testErr("oog}{boog", "ion: unexpected rune '{' (offset 4)")
}

func TestSkipList(t *testing.T) {
	test, testErr := testSkip(t, (*tokenizer).skipList)

	test("]", -1)
	test("[]],", ',')
	test("[123, \"]\", ']']] ", ' ')

	testErr("abc, def, ", "ion: unexpected end of input (offset 10)")
}

type skipFunc func(*tokenizer) (int, error)
type skipTestFunc func(string, int)
type skipTestErrFunc func(string, string)

func testSkip(t *testing.T, f skipFunc) (skipTestFunc, skipTestErrFunc) {
	test := func(str string, ec int) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			c, err := f(tok)
			if err != nil {
				t.Fatal(err)
			}
			if c != ec {
				t.Errorf("expected '%c', got '%c'", ec, c)
			}
		})
	}
	testErr := func(str string, e string) {
		t.Run(str, func(t *testing.T) {
			tok := tokenizeString(str)
			_, err := f(tok)
			if err == nil || err.Error() != e {
				t.Errorf("expected err=%v, got err=%v", e, err)
			}
		})
	}
	return test, testErr
}
