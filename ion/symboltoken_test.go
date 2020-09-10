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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var symbolTokenEqualsTestData = []struct {
	text1   *string
	sid1    int64
	source1 *ImportSource
	text2   *string
	sid2    int64
	source2 *ImportSource
	equals  bool
}{
	{nil, 123, nil, nil, 123, nil, true},
	{nil, 123, newSource("table", 1), nil, 456, newSource("table", 1), true},
	{newString("text1"), 123, nil, newString("text1"), 123, nil, true},
	{newString("text2"), 123, newSource("table", 1), newString("text2"), 456, newSource("table", 1), true},
	{nil, 123, newSource("table", 1), nil, 123, nil, false},
	{nil, 123, nil, newString("text1"), 456, nil, false},
	{nil, 123, newSource("table", 1), nil, 123, newSource("table2", 1), false},
	{nil, 123, newSource("table", 1), nil, 456, newSource("table", 2), false},
	{newString("text2"), 123, nil, newString("text3"), 123, nil, false},
}

func TestSymbolTokenEqualsOperator(t *testing.T) {
	for _, testData := range symbolTokenEqualsTestData {
		st1 := SymbolToken{Text: testData.text1, LocalSID: testData.sid1, Source: testData.source1}
		st2 := SymbolToken{Text: testData.text2, LocalSID: testData.sid2, Source: testData.source2}

		if testData.equals {
			if !st1.Equal(&st2) {
				t.Errorf("expected %v, got %v", true, false)
			}
		} else {
			if st1.Equal(&st2) {
				t.Errorf("expected %v, got %v", true, false)
			}
		}
	}
}

// Make sure SymbolToken conforms to Stringer
var _ fmt.Stringer = &SymbolToken{}

func TestSymbolToken_String(t *testing.T) {
	cases := []struct {
		desc     string
		token    SymbolToken
		expected string
	}{
		{
			desc: "Text and SID",
			token: SymbolToken{
				Text:     newString("hello"),
				LocalSID: 10,
				Source:   nil,
			},
			expected: `{"hello" 10 nil}`,
		},
		{
			desc: "nil Text",
			token: SymbolToken{
				Text:     nil,
				LocalSID: 11,
				Source:   nil,
			},
			expected: `{nil 11 nil}`,
		},
		{
			desc: "Text and SID with Import",
			token: SymbolToken{
				Text:     newString("world"),
				LocalSID: 12,
				Source:   newSource("foobar", 3),
			},
			expected: `{"world" 12 {"foobar" 3}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if diff := cmp.Diff(c.expected, c.token.String()); diff != "" {
				t.Errorf("Token String() differs (-expected, +actual):\n%s", diff)
			}
		})
	}
}

func TestNewImportSource(t *testing.T) {
	is := newSource("table", 1)
	if is.Table != "table" {
		t.Errorf("expected %v, got %v", "table", is.Table)
	}
	if is.SID != 1 {
		t.Errorf("expected %v, got %v", 1, is.SID)
	}
}

var importSourceEqualsTestData = []struct {
	text1  string
	sid1   int64
	text2  string
	sid2   int64
	equals bool
}{
	{"text1", 123, "text1", 123, true},
	{"text2", 456, "text2", 456, true},
	{"text1", 123, "text1", 456, false},
	{"text2", 456, "text3", 456, false},
}

func TestImportSourceEqualsOperator(t *testing.T) {
	for _, testData := range importSourceEqualsTestData {
		is1 := newSource(testData.text1, testData.sid1)
		is2 := newSource(testData.text2, testData.sid2)

		if testData.equals {
			if !is1.Equal(is2) {
				t.Errorf("expected %v, got %v", true, false)
			}
		} else {
			if is1.Equal(is2) {
				t.Errorf("expected %v, got %v", false, true)
			}
		}
	}
}

func TestNewSymbolTokenThatAlreadyExistInSymbolTable(t *testing.T) {
	expectedSymbolToken := SymbolToken{Text: newString("$ion"), LocalSID: SymbolIDUnknown}

	actualSymbolToken, err := NewSymbolToken(V1SystemSymbolTable, "$ion")
	if err != nil {
		t.Errorf("expected NewSymbolToken() to execute without errors; %s", err.Error())
	}

	if !actualSymbolToken.Equal(&expectedSymbolToken) {
		t.Errorf("expected %v, got %v", expectedSymbolToken, actualSymbolToken)
	}
}

func TestNewSymbolTokenThatDoesNotExistInSymbolTable(t *testing.T) {
	expectedSymbolToken := SymbolToken{Text: newString("newToken"), LocalSID: SymbolIDUnknown}

	actualSymbolToken, err := NewSymbolToken(V1SystemSymbolTable, "newToken")
	if err != nil {
		t.Errorf("expected NewSymbolToken() to execute without errors; %s", err.Error())
	}

	if !actualSymbolToken.Equal(&expectedSymbolToken) {
		t.Errorf("expected %v, got %v", expectedSymbolToken, actualSymbolToken)
	}
}

func TestNewSymbolTokensThatAlreadyExistInSymbolTable(t *testing.T) {
	expectedSymbolTokens := []SymbolToken{
		{Text: newString("$ion"), LocalSID: SymbolIDUnknown},
		{Text: newString("$ion_1_0"), LocalSID: SymbolIDUnknown}}

	actualSymbolTokens, err := NewSymbolTokens(V1SystemSymbolTable, []string{"$ion", "$ion_1_0"})
	if err != nil {
		t.Errorf("expected NewSymbolTokens() to execute without errors; %s", err.Error())
	}

	for index, actualSymbolToken := range actualSymbolTokens {
		if !actualSymbolToken.Equal(&expectedSymbolTokens[index]) {
			t.Errorf("expected %v, got %v", &expectedSymbolTokens[index], actualSymbolToken)
		}
	}
}

func TestNewSymbolTokensThatDoNotExistInSymbolTable(t *testing.T) {
	expectedSymbolTokens := []SymbolToken{
		{Text: newString("newToken1"), LocalSID: SymbolIDUnknown},
		{Text: newString("newToken2"), LocalSID: SymbolIDUnknown}}

	actualSymbolTokens, err := NewSymbolTokens(V1SystemSymbolTable, []string{"newToken1", "newToken2"})
	if err != nil {
		t.Errorf("expected NewSymbolTokens() to execute without errors; %s", err.Error())
	}

	for index, actualSymbolToken := range actualSymbolTokens {
		if !actualSymbolToken.Equal(&expectedSymbolTokens[index]) {
			t.Errorf("expected %v, got %v", &expectedSymbolTokens[index], actualSymbolToken)
		}
	}
}

func TestSymbolIdentifier(t *testing.T) {
	test := func(sym string, expectedSID int64, expectedOK bool) {
		t.Run(sym, func(t *testing.T) {
			sid, ok := symbolIdentifier(sym)

			if ok != expectedOK {
				t.Errorf("expected %v, got %v", expectedOK, ok)
			}

			if expectedOK && sid != expectedSID {
				t.Errorf("expected %v, got %v", expectedSID, sid)
			}
		})
	}

	test("", SymbolIDUnknown, false)
	test("1", SymbolIDUnknown, false)
	test("a", SymbolIDUnknown, false)
	test("$", SymbolIDUnknown, false)
	test("$1", 1, true)
	test("$1234567890", 1234567890, true)
	test("$a", SymbolIDUnknown, false)
	test("$1234a567890", SymbolIDUnknown, false)
}
