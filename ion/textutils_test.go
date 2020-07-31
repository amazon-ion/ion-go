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
	"strings"
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	test := func(str string, eval string, expectedPrecision TimestampPrecision, expectedKind TimezoneKind, expectedFractionSeconds uint8) {
		t.Run(str, func(t *testing.T) {
			val, err := parseTimestamp(str)
			if err != nil {
				t.Fatal(err)
			}

			et, err := time.Parse(time.RFC3339Nano, eval)
			if err != nil {
				t.Fatal(err)
			}

			expectedTimestamp := NewTimestampWithFractionalSeconds(et, expectedPrecision, expectedKind, expectedFractionSeconds)

			if !val.Equal(expectedTimestamp) {
				t.Errorf("expected %v, got %v", expectedTimestamp, val)
			}
		})
	}

	test("1234T", "1234-01-01T00:00:00Z", TimestampPrecisionYear, TimezoneUnspecified, 0)
	test("1234-05T", "1234-05-01T00:00:00Z", TimestampPrecisionMonth, TimezoneUnspecified, 0)
	test("1234-05-06", "1234-05-06T00:00:00Z", TimestampPrecisionDay, TimezoneUnspecified, 0)
	test("1234-05-06T", "1234-05-06T00:00:00Z", TimestampPrecisionDay, TimezoneUnspecified, 0)
	test("1234-05-06T07:08Z", "1234-05-06T07:08:00Z", TimestampPrecisionMinute, TimezoneUTC, 0)
	test("1234-05-06T07:08:09Z", "1234-05-06T07:08:09Z", TimestampPrecisionSecond, TimezoneUTC, 0)
	test("1234-05-06T07:08:09.100Z", "1234-05-06T07:08:09.100Z", TimestampPrecisionNanosecond, TimezoneUTC, 1)
	test("1234-05-06T07:08:09.100100Z", "1234-05-06T07:08:09.100100Z", TimestampPrecisionNanosecond, TimezoneUTC, 4)

	// Test rounding of >=9 fractional seconds.
	test("1234-05-06T07:08:09.000100100Z", "1234-05-06T07:08:09.000100100Z", TimestampPrecisionNanosecond, TimezoneUTC, 7)
	test("1234-05-06T07:08:09.100100100Z", "1234-05-06T07:08:09.100100100Z", TimestampPrecisionNanosecond, TimezoneUTC, 7)
	test("1234-05-06T07:08:09.00010010044Z", "1234-05-06T07:08:09.000100100Z", TimestampPrecisionNanosecond, TimezoneUTC, 7)
	test("1234-05-06T07:08:09.00010010044Z", "1234-05-06T07:08:09.000100100Z", TimestampPrecisionNanosecond, TimezoneUTC, 7)
	test("1234-05-06T07:08:09.00010010055Z", "1234-05-06T07:08:09.000100101Z", TimestampPrecisionNanosecond, TimezoneUTC, 9)
	test("1234-05-06T07:08:09.00010010099Z", "1234-05-06T07:08:09.000100101Z", TimestampPrecisionNanosecond, TimezoneUTC, 9)
	test("1234-05-06T07:08:09.99999999999Z", "1234-05-06T07:08:10.000000000Z", TimestampPrecisionNanosecond, TimezoneUTC, 9)
	test("1234-12-31T23:59:59.99999999999Z", "1235-01-01T00:00:00.000000000Z", TimestampPrecisionNanosecond, TimezoneUTC, 9)
	test("1234-05-06T07:08:09.000100100+09:10", "1234-05-06T07:08:09.000100100+09:10", TimestampPrecisionNanosecond, TimezoneLocal, 7)
	test("1234-05-06T07:08:09.100100100-10:11", "1234-05-06T07:08:09.100100100-10:11", TimestampPrecisionNanosecond, TimezoneLocal, 7)
	test("1234-05-06T07:08:09.00010010044+09:10", "1234-05-06T07:08:09.000100100+09:10", TimestampPrecisionNanosecond, TimezoneLocal, 7)
	test("1234-05-06T07:08:09.00010010055-10:11", "1234-05-06T07:08:09.000100101-10:11", TimestampPrecisionNanosecond, TimezoneLocal, 9)
	test("1234-05-06T07:08:09.00010010099+09:10", "1234-05-06T07:08:09.000100101+09:10", TimestampPrecisionNanosecond, TimezoneLocal, 9)
	test("1234-05-06T07:08:09.99999999999-10:11", "1234-05-06T07:08:10.000000000-10:11", TimestampPrecisionNanosecond, TimezoneLocal, 9)
	test("1234-12-31T23:59:59.99999999999+09:10", "1235-01-01T00:00:00.000000000+09:10", TimestampPrecisionNanosecond, TimezoneLocal, 9)

	test("1234-05-06T07:08+09:10", "1234-05-06T07:08:00+09:10", TimestampPrecisionMinute, TimezoneLocal, 0)
	test("1234-05-06T07:08:09-10:11", "1234-05-06T07:08:09-10:11", TimestampPrecisionSecond, TimezoneLocal, 0)
}

func TestWriteSymbol(t *testing.T) {
	test := func(sym, expected string) {
		t.Run(expected, func(t *testing.T) {
			buf := strings.Builder{}
			if err := writeSymbol(sym, &buf); err != nil {
				t.Fatal(err)
			}
			actual := buf.String()
			if actual != expected {
				t.Errorf("expected \"%v\", got \"%v\"", expected, actual)
			}
		})
	}

	test("", "''")
	test("null", "'null'")
	test("null.null", "'null.null'")

	test("basic", "basic")
	test("_basic_", "_basic_")
	test("$basic$", "$basic$")
	test("$123", "$123")

	test("123", "'123'")
	test("abc'def", "'abc\\'def'")
	test("abc\"def", "'abc\"def'")
}

func TestSymbolNeedsQuoting(t *testing.T) {
	test := func(sym string, expected bool) {
		t.Run(sym, func(t *testing.T) {
			actual := symbolNeedsQuoting(sym)
			if actual != expected {
				t.Errorf("expected %v, got %v", expected, actual)
			}
		})
	}

	test("", true)
	test("null", true)
	test("true", true)
	test("false", true)
	test("nan", true)

	test("basic", false)
	test("_basic_", false)
	test("basic$123", false)
	test("$", false)
	test("$basic", false)
	test("$123", false)

	test("123", true)
	test("abc.def", true)
	test("abc,def", true)
	test("abc:def", true)
	test("abc{def", true)
	test("abc}def", true)
	test("abc[def", true)
	test("abc]def", true)
	test("abc'def", true)
	test("abc\"def", true)
}

func TestIsSymbolRef(t *testing.T) {
	test := func(sym string, expected bool) {
		t.Run(sym, func(t *testing.T) {
			actual := isSymbolRef(sym)
			if actual != expected {
				t.Errorf("expected %v, got %v", expected, actual)
			}
		})
	}

	test("", false)
	test("1", false)
	test("a", false)
	test("$", false)
	test("$1", true)
	test("$1234567890", true)
	test("$a", false)
	test("$1234a567890", false)
}

func TestWriteEscapedSymbol(t *testing.T) {
	test := func(sym, expected string) {
		t.Run(expected, func(t *testing.T) {
			buf := strings.Builder{}
			if err := writeEscapedSymbol(sym, &buf); err != nil {
				t.Fatal(err)
			}
			actual := buf.String()
			if actual != expected {
				t.Errorf("bad encoding of \"%v\": \"%v\"",
					expected, actual)
			}
		})
	}

	test("basic", "basic")
	test("\"basic\"", "\"basic\"")
	test("o'clock", "o\\'clock")
	test("c:\\", "c:\\\\")
}

func TestWriteEscapedChar(t *testing.T) {
	test := func(c byte, expected string) {
		t.Run(expected, func(t *testing.T) {
			buf := strings.Builder{}
			if err := writeEscapedChar(c, &buf); err != nil {
				t.Fatal(err)
			}
			actual := buf.String()
			if actual != expected {
				t.Errorf("bad encoding of '%v': \"%v\"",
					expected, actual)
			}
		})
	}

	test(0, "\\0")
	test('\n', "\\n")
	test(1, "\\x01")
	test('\xFF', "\\xFF")
}
