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
	text1 *string
	sid1  int64
	text2 *string
	sid2  int64
}{
	{nil, 456, nil, 456},
	{newString("text1"), 123, newString("text1"), 123},
	{newString("text2"), 456, newString("text2"), 456},
}

func TestBoolEqualsOperator(t *testing.T) {
	for _, testData := range boolEqualsTestData {
		st1 := SymbolToken{Text: testData.text1, LocalSID: testData.sid1}
		st2 := SymbolToken{Text: testData.text2, LocalSID: testData.sid2}

		if !st1.Equal(&st2) {
			t.Errorf("expected %v, got %v", true, false)
		}
	}
}

var boolNotEqualsTestData = []struct {
	text1 *string
	sid1  int64
	text2 *string
	sid2  int64
}{
	{nil, 123, nil, 456},
	{nil, 456, newString("text1"), 456},
	{newString("text1"), 123, newString("text1"), 456},
	{newString("text2"), 456, newString("text3"), 456},
}

func TestBoolNotEqualsOperator(t *testing.T) {
	for _, testData := range boolNotEqualsTestData {
		st1 := SymbolToken{Text: testData.text1, LocalSID: testData.sid1}
		st2 := SymbolToken{Text: testData.text2, LocalSID: testData.sid2}

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
