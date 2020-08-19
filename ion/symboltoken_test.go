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

func newString(value string) *string {
	return &value
}

var boolEqualsTestData = []struct {
	text1   *string
	sid1    int64
	source1 *ImportSource
	text2   *string
	sid2    int64
	source2 *ImportSource
}{
	{nil, 123, nil,
		nil, 123, nil},
	{nil, 123, newSource("table", 1),
		nil, 123, newSource("table", 1)},
	{newString("text1"), 123, nil,
		newString("text1"), 123, nil},
	{newString("text2"), 123, newSource("table", 1),
		newString("text2"), 123, newSource("table", 1)},
}

func TestBoolEqualsOperator(t *testing.T) {
	for _, testData := range boolEqualsTestData {
		st1 := SymbolToken{Text: testData.text1, LocalSID: testData.sid1, Source: testData.source1}
		st2 := SymbolToken{Text: testData.text2, LocalSID: testData.sid2, Source: testData.source2}

		if !st1.Equal(&st2) {
			t.Errorf("expected %v, got %v", true, false)
		}
	}
}

var boolNotEqualsTestData = []struct {
	text1   *string
	sid1    int64
	source1 *ImportSource
	text2   *string
	sid2    int64
	source2 *ImportSource
}{
	{nil, 123, newSource("table", 1),
		nil, 123, nil},
	{nil, 123, nil,
		newString("text1"), 123, nil},
	{nil, 123, newSource("table", 1),
		nil, 123, newSource("table2", 1)},
	{nil, 123, newSource("table", 1),
		nil, 123, newSource("table", 2)},
	{newString("text2"), 123, nil,
		newString("text3"), 123, nil},
}

func TestBoolNotEqualsOperator(t *testing.T) {
	for _, testData := range boolNotEqualsTestData {
		st1 := SymbolToken{Text: testData.text1, LocalSID: testData.sid1, Source: testData.source1}
		st2 := SymbolToken{Text: testData.text2, LocalSID: testData.sid2, Source: testData.source2}

		if st1.Equal(&st2) {
			t.Errorf("expected %v, got %v", false, true)
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

var ImportSourceBoolEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 123},
	{"text2", 456, "text2", 456},
}

func TestImportSourceBoolEqualsOperator(t *testing.T) {
	for _, testData := range ImportSourceBoolEqualsTestData {
		is1 := newSource(testData.text1, testData.sid1)
		is2 := newSource(testData.text2, testData.sid2)

		if !is1.Equal(is2) {
			t.Errorf("expected %v, got %v", true, false)
		}
	}
}

var ImportSourceBoolNotEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 456},
	{"text2", 456, "text3", 456},
}

func TestImportSourceBoolNotEqualsOperator(t *testing.T) {
	for _, testData := range ImportSourceBoolNotEqualsTestData {
		is1 := newSource(testData.text1, testData.sid1)
		is2 := newSource(testData.text2, testData.sid2)

		if is1.Equal(is2) {
			t.Errorf("expected %v, got %v", false, true)
		}
	}
}

func TestNewSymbolTokenThatAlreadyExistInSymbolTable(t *testing.T) {
	expectedSymbolToken := SymbolToken{Text: newString("$ion"), LocalSID: SymbolIDUnknown}

	actualSymbolToken := NewSymbolToken(V1SystemSymbolTable, "$ion")

	if !actualSymbolToken.Equal(&expectedSymbolToken) {
		t.Errorf("expected %v, got %v", expectedSymbolToken, actualSymbolToken)
	}
}

func TestNewSymbolTokenThatDoesNotExistInSymbolTable(t *testing.T) {
	expectedSymbolToken := SymbolToken{Text: newString("newToken"), LocalSID: SymbolIDUnknown}

	actualSymbolToken := NewSymbolToken(V1SystemSymbolTable, "newToken")

	if !actualSymbolToken.Equal(&expectedSymbolToken) {
		t.Errorf("expected %v, got %v", expectedSymbolToken, actualSymbolToken)
	}
}

func TestNewSymbolTokensThatAlreadyExistInSymbolTable(t *testing.T) {
	expectedSymbolTokens := []SymbolToken{
		{Text: newString("$ion"), LocalSID: SymbolIDUnknown},
		{Text: newString("$ion_1_0"), LocalSID: SymbolIDUnknown}}

	actualSymbolTokens := NewSymbolTokens(V1SystemSymbolTable, []string{"$ion", "$ion_1_0"})

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

	actualSymbolTokens := NewSymbolTokens(V1SystemSymbolTable, []string{"newToken1", "newToken2"})

	for index, actualSymbolToken := range actualSymbolTokens {
		if !actualSymbolToken.Equal(&expectedSymbolTokens[index]) {
			t.Errorf("expected %v, got %v", &expectedSymbolTokens[index], actualSymbolToken)

		}
	}
}
