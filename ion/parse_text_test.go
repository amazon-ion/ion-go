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
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TODO: Need to wire up tests that call some of the underlying parse functions individually
//       so that we can bypass the checks that the lexer makes and trigger the panics.

func TestParseText(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		expected    *Digest
		expectedErr error
	}{
		// Strings.

		{
			name: "short and long strings",
			text: "\"short string\"\n'''long'''\n'''string'''",
			expected: &Digest{values: []Value{
				String{text: []byte("short string")},
				String{text: []byte("longstring")},
			}},
		},

		// Numeric

		{
			name: "infinity",
			text: "inf +inf -inf",
			expected: &Digest{values: []Value{
				// "inf" must have a plus or minus on it to be considered a number.
				Symbol{text: []byte("inf")},
				Float{isSet: true, text: []byte("+inf")},
				Float{isSet: true, text: []byte("-inf")},
			}},
		},
		{
			name: "integers",
			text: "0 -1 1_2_3 0xFf -0xFf 0Xe_d 0b10 -0b10 0B1_0",
			expected: &Digest{values: []Value{
				Int{isSet: true, text: []byte("0")},
				Int{isSet: true, isNegative: true, text: []byte("-1")},
				Int{isSet: true, text: []byte("1_2_3")},
				Int{isSet: true, base: intBase16, text: []byte("0xFf")},
				Int{isSet: true, isNegative: true, base: intBase16, text: []byte("-0xFf")},
				Int{isSet: true, base: intBase16, text: []byte("0Xe_d")},
				Int{isSet: true, base: intBase2, text: []byte("0b10")},
				Int{isSet: true, isNegative: true, base: intBase2, text: []byte("-0b10")},
				Int{isSet: true, base: intBase2, text: []byte("0B1_0")},
			}},
		},
		{
			name: "decimals",
			text: "0. 0.123 -0.12d4 0D-0 0d+0 12_34.56_78",
			expected: &Digest{values: []Value{
				Decimal{isSet: true, text: []byte("0.")},
				Decimal{isSet: true, text: []byte("0.123")},
				Decimal{isSet: true, text: []byte("-0.12d4")},
				Decimal{isSet: true, text: []byte("0D-0")},
				Decimal{isSet: true, text: []byte("0d+0")},
				Decimal{isSet: true, text: []byte("12_34.56_78")},
			}},
		},
		{
			name: "floats",
			text: "0E0 0.12e-4 -0e+0",
			expected: &Digest{values: []Value{
				Float{isSet: true, text: []byte("0E0")},
				Float{isSet: true, text: []byte("0.12e-4")},
				Float{isSet: true, text: []byte("-0e+0")},
			}},
		},

		{
			name: "dates",
			text: "2019T 2019-10T 2019-10-30 2019-10-30T",
			expected: &Digest{values: []Value{
				Timestamp{precision: TimestampPrecisionYear, text: []byte("2019T")},
				Timestamp{precision: TimestampPrecisionMonth, text: []byte("2019-10T")},
				Timestamp{precision: TimestampPrecisionDay, text: []byte("2019-10-30")},
				Timestamp{precision: TimestampPrecisionDay, text: []byte("2019-10-30T")},
			}},
		},
		{
			name: "times",
			text: "2019-10-30T22:30Z 2019-10-30T12:30:59+02:30 2019-10-30T12:30:59.999-02:30",
			expected: &Digest{values: []Value{
				Timestamp{precision: TimestampPrecisionMinute, text: []byte("2019-10-30T22:30Z")},
				Timestamp{precision: TimestampPrecisionSecond, text: []byte("2019-10-30T12:30:59+02:30")},
				Timestamp{precision: TimestampPrecisionMillisecond3, text: []byte("2019-10-30T12:30:59.999-02:30")},
			}},
		},

		// Binary.

		{
			name:     "short blob",
			text:     "{{+AB/}}",
			expected: &Digest{values: []Value{Blob{text: []byte("+AB/")}}},
		},
		{
			name:     "padded blob with whitespace",
			text:     "{{ + A\nB\t/abc= }}",
			expected: &Digest{values: []Value{Blob{text: []byte("+AB/abc=")}}},
		},
		{
			name:     "short clob",
			text:     `{{ "A\n" }}`,
			expected: &Digest{values: []Value{Clob{text: []byte("A\n")}}},
		},
		{
			name:     "long clob",
			text:     "{{ '''+AB/''' }}",
			expected: &Digest{values: []Value{Clob{text: []byte("+AB/")}}},
		},
		{
			name:     "multiple long clobs",
			text:     "{{ '''A\\nB'''\n'''foo''' }}",
			expected: &Digest{values: []Value{Clob{text: []byte("A\nBfoo")}}},
		},

		// Containers

		{
			name: "struct with symbol to symbol",
			text: `{symbol1: 'symbol', 'symbol2': symbol}`,
			expected: &Digest{values: []Value{
				Struct{fields: []StructField{
					{Symbol: Symbol{text: []byte("symbol1")}, Value: Symbol{quoted: true, text: []byte("symbol")}},
					{Symbol: Symbol{quoted: true, text: []byte("symbol2")}, Value: Symbol{text: []byte("symbol")}},
				}},
			}},
		},
		{
			name: "struct with annotated field",
			text: `{symbol1: ann::'symbol'}`,
			expected: &Digest{values: []Value{
				Struct{fields: []StructField{
					{Symbol: Symbol{text: []byte("symbol1")}, Value: Symbol{annotations: []Symbol{{text: []byte("ann")}}, quoted: true, text: []byte("symbol")}},
				}},
			}},
		},
		{
			name: "struct with doubly-annotated field",
			text: `{symbol1: ann1::ann2::'symbol'}`,
			expected: &Digest{values: []Value{
				Struct{fields: []StructField{
					{Symbol: Symbol{text: []byte("symbol1")}, Value: Symbol{annotations: []Symbol{{text: []byte("ann1")}, {text: []byte("ann2")}}, quoted: true, text: []byte("symbol")}},
				}},
			}},
		},
		{
			name: "struct with comments between symbol and value",
			text: "{abc : // Line\n/* Block */ {{ \"A\\n\" }}}",
			expected: &Digest{values: []Value{
				Struct{fields: []StructField{
					{Symbol: Symbol{text: []byte("abc")}, Value: Clob{text: []byte("A\n")}},
				}},
			}},
		},

		{
			name: "struct with empty list, struct, and sexp",
			text: "{a:[], b:{}, c:()}",
			expected: &Digest{values: []Value{
				Struct{fields: []StructField{
					{Symbol: Symbol{text: []byte("a")}, Value: List{}},
					{Symbol: Symbol{text: []byte("b")}, Value: Struct{}},
					{Symbol: Symbol{text: []byte("c")}, Value: SExp{}},
				}},
			}},
		},
		{
			name: "list with empty list, struct, and sexp",
			text: "[[], {}, ()]",
			expected: &Digest{values: []Value{
				List{values: []Value{List{}, Struct{}, SExp{}}},
			}},
		},
		{
			name: "list of things",
			text: "[a, 1, ' ', {}, () /* comment */ ]",
			expected: &Digest{values: []Value{
				List{values: []Value{
					Symbol{text: []byte("a")},
					Int{isSet: true, text: []byte("1")},
					Symbol{text: []byte(" ")},
					Struct{},
					SExp{},
				}},
			}},
		},
		{
			name: "struct of things",
			text: "{'a' : 1 , s:'', 'st': {}, \n/* comment */lst:[],\"sexp\":()}",
			expected: &Digest{values: []Value{
				Struct{fields: []StructField{
					{Symbol: Symbol{text: []byte("a")}, Value: Int{isSet: true, text: []byte("1")}},
					{Symbol: Symbol{text: []byte("s")}, Value: Symbol{text: []byte("")}},
					{Symbol: Symbol{text: []byte("st")}, Value: Struct{}},
					{Symbol: Symbol{text: []byte("lst")}, Value: List{}},
					{Symbol: Symbol{text: []byte("sexp")}, Value: SExp{}},
				}},
			}},
		},
		{
			name: "s-expression of things",
			text: "(a+b/c<( j * k))",
			expected: &Digest{values: []Value{
				SExp{values: []Value{
					Symbol{text: []byte("a")},
					Symbol{text: []byte("+")},
					Symbol{text: []byte("b")},
					Symbol{text: []byte("/")},
					Symbol{text: []byte("c")},
					Symbol{text: []byte("<")},
					SExp{values: []Value{
						Symbol{text: []byte("j")},
						Symbol{text: []byte("*")},
						Symbol{text: []byte("k")},
					}},
				}},
			}},
		},

		// Error cases

		{
			name:        "list starts with comma",
			text:        "[, [], {}, ()]",
			expectedErr: errors.New("parsing line 1 - list may not start with a comma"),
		},
		{
			name:        "struct starts with comma",
			text:        "{, a:1}",
			expectedErr: errors.New("parsing line 1 - struct may not start with a comma"),
		},
		{
			name:        "list without commas",
			text:        "[[] {} ()]",
			expectedErr: errors.New("parsing line 1 - list items must be separated by commas"),
		},
		{
			name:        "struct without commas",
			text:        "{a:1 b:2}",
			expectedErr: errors.New("parsing line 1 - struct fields must be separated by commas"),
		},
	}
	for _, tst := range tests {
		test := tst
		t.Run(test.name, func(t *testing.T) {
			digest, err := ParseText(strings.NewReader(test.text))
			if diff := cmpDigests(test.expected, digest); diff != "" {
				t.Logf("expected: %#v", test.expected)
				t.Logf("found:    %#v", digest)
				t.Error("(-expected, +found)", diff)
			}
			if diff := cmpErrs(test.expectedErr, err); diff != "" {
				t.Error("err: (-expected, +found)", diff)
			}
		})
	}
}

func TestIonTests_Text_Good(t *testing.T) {
	// We don't support UTF-16 or UTF-32 so skip those two test files.
	filesToSkip := map[string]bool{
		"utf16.ion": true,
		"utf32.ion": true,
		// TODO amzn/ion-go#3 (newline normalization in CLOB)
		"clobNewlines.ion": true,
	}

	testFilePath := "../ion-tests/iontestdata/good"
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".ion") {
			return nil
		}

		name := info.Name()
		if _, ok := filesToSkip[name]; ok {
			t.Log("skipping", name)
			return nil
		}

		t.Run(strings.TrimPrefix(path, testFilePath), func(t *testing.T) {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			out, err := ParseText(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}

			// There are a couple of files where correct parsing yields an empty Digest.
			if strings.HasSuffix(path, "blank.ion") || strings.HasSuffix(path, "empty.ion") {
				if out == nil {
					t.Error("expected out to not be nil")
				}
			} else if out == nil || len(out.Value()) == 0 {
				t.Error("expected out to have at least one value")
			}
		})
		return nil
	}
	if err := filepath.Walk(testFilePath, walkFn); err != nil {
		t.Fatal(err)
	}
}

func TestIonTests_Text_Equivalents(t *testing.T) {
	// We have some use-cases that are not yet supported.
	filesToSkip := map[string]bool{
		// TODO: Deal with symbol tables and verification of SymbolIDs.
		"annotatedIvms.ion":                          true,
		"keywordPrefixes.ion":                        true,
		"localSymbolTableAppend.ion":                 true,
		"localSymbolTableNullSlots.ion":              true,
		"localSymbolTableWithAnnotations.ion":        true,
		"localSymbolTables.ion":                      true,
		"localSymbolTablesValuesWithAnnotations.ion": true,
		"nonIVMNoOps.ion":                            true,
		"systemSymbols.ion":                          true,
		// "Structures are unordered collections of name/value pairs."  Comparing
		// the structs for equivalency requires specialized logic that is not part
		// of the spec.
		"structsFieldsDiffOrder.ion":     true,
		"structsFieldsRepeatedNames.ion": true,
		// We don't support arbitrary precision for timestamps.  Once you get
		// past microseconds it's pretty meaningless.
		"timestampsLargeFractionalPrecision.ion": true,
		// These files contain UTF16 and UTF32 which we do not support.
		"stringU0001D11E.ion": true,
		"stringUtf8.ion":      true,
		// TODO amzn/ion-go#3 (newline normalization in CLOB)
		"clobNewlines.ion": true,
	}

	testFilePath := "../ion-tests/iontestdata/good/equivs"
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".ion") {
			return nil
		}

		name := info.Name()
		if _, ok := filesToSkip[name]; ok {
			t.Log("skipping", name)
			return nil
		}

		t.Run(strings.TrimPrefix(path, testFilePath), func(t *testing.T) {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			out, err := ParseText(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}

			if out == nil || len(out.Value()) == 0 {
				t.Error("expected out to have at least one value")
			}

			for i, value := range out.Value() {
				t.Log("collection", i, "of", info.Name())
				switch value.Type() {
				case TypeList:
					assertEquivalentValues(value.(List).values, t)
				case TypeSExp:
					assertEquivalentValues(value.(SExp).values, t)
				default:
					t.Error("top-element item is", value.Type(), "for", info.Name())
				}
			}
		})
		return nil
	}
	if err := filepath.Walk(testFilePath, walkFn); err != nil {
		t.Fatal(err)
	}
}

func TestIonTests_Text_NonEquivalents(t *testing.T) {
	// We have some use-cases that are not yet supported.
	filesToSkip := map[string]bool{
		// TODO: Deal with symbol tables and verification of SymbolIDs.
		"annotations.ion": true,
		"symbols.ion":     true,
		// Not properly tracking decimal precision yet.
		"decimals.ion": true,
		"nonNulls.ion": true,
		// Not handling negative zero.
		"floats.ion":           true,
		"floatsVsDecimals.ion": true,
		// Not properly handling unknown local offset yet.
		"timestamps.ion": true,
	}

	testFilePath := "../ion-tests/iontestdata/good/non-equivs"
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".ion") {
			return nil
		}

		name := info.Name()
		if _, ok := filesToSkip[name]; ok {
			t.Log("skipping", name)
			return nil
		}

		t.Run(strings.TrimPrefix(path, testFilePath), func(t *testing.T) {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			out, err := ParseText(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}

			if out == nil || len(out.Value()) == 0 {
				t.Error("expected out to have at least one value")
			}

			for i, value := range out.Value() {
				t.Log("collection", i, "of", info.Name())
				switch value.Type() {
				case TypeList:
					assertNonEquivalentValues(value.(List).values, t)
				case TypeSExp:
					assertNonEquivalentValues(value.(SExp).values, t)
				default:
					t.Error("top-element item is", value.Type(), "for", info.Name())
				}
			}
		})
		return nil
	}
	if err := filepath.Walk(testFilePath, walkFn); err != nil {
		t.Fatal(err)
	}
}

func TestIonTests_Text_Bad(t *testing.T) {
	filesToSkip := map[string]bool{
		// TODO: Deal with symbol tables and verification of SymbolIDs.
		"annotationSymbolIDUnmapped.ion":                          true,
		"localSymbolTableImportNegativeMaxId.ion":                 true,
		"localSymbolTableImportNonIntegerMaxId.ion":               true,
		"localSymbolTableImportNullMaxId.ion":                     true,
		"localSymbolTableWithMultipleImportsFields.ion":           true,
		"localSymbolTableWithMultipleSymbolsAndImportsFields.ion": true,
		"localSymbolTableWithMultipleSymbolsFields.ion":           true,
		"symbolIDUnmapped.ion":                                    true,
		// We only support UTF-8
		"fieldNameSymbolIDUnmapped.ion": true,
		"longStringSplitEscape_2.ion":   true,
		"surrogate_1.ion":               true,
		"surrogate_2.ion":               true,
		"surrogate_3.ion":               true,
		"surrogate_4.ion":               true,
		"surrogate_5.ion":               true,
		"surrogate_6.ion":               true,
		"surrogate_7.ion":               true,
		"surrogate_8.ion":               true,
		"surrogate_9.ion":               true,
		"surrogate_10.ion":              true,
	}

	testFilePath := "../ion-tests/iontestdata/bad"
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".ion") {
			return nil
		}

		name := info.Name()
		if _, ok := filesToSkip[name]; ok {
			t.Log("skipping", name)
			return nil
		}

		t.Run(strings.TrimPrefix(path, testFilePath), func(t *testing.T) {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			out, err := ParseText(bytes.NewReader(data))
			if err == nil {
				t.Error("expected error but found none")
			}
			if out != nil {
				t.Errorf("%#v", out.values)
			}
		})
		return nil
	}
	if err := filepath.Walk(testFilePath, walkFn); err != nil {
		t.Fatal(err)
	}
}
