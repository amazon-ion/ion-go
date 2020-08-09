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

import "testing"

var text1 = "text1"
var text2 = "text2"
var text3 = "text3"

func TestNewSymbolTokenSidAndTextUnknown(t *testing.T) {
	st := SymbolToken{Text: nil, SID: UnknownSid}
	if st.Text != nil {
		t.Errorf("expected %v, got %v", nil, st.Text)
	}
	if st.SID != UnknownSid {
		t.Errorf("expected %v, got %v", UnknownSid, st.SID)
	}
}

var boolEqualsTestData = []struct {
	text1 *string
	sid1  int64
	text2 *string
	sid2  int64
}{
	{nil, 456, nil, 456},
	{&text1, 123, &text1, 123},
	{&text2, 456, &text2, 456},
}

func TestBoolEqualsOperator(t *testing.T) {
	for _, testData := range boolEqualsTestData {
		st1 := SymbolToken{Text: testData.text1, SID: testData.sid1}
		st2 := SymbolToken{Text: testData.text2, SID: testData.sid2}

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
	{nil, 456, &text1, 456},
	{&text1, 123, &text1, 456},
	{&text2, 456, &text3, 456},
}

func TestBoolNotEqualsOperator(t *testing.T) {
	for _, testData := range boolNotEqualsTestData {
		st1 := SymbolToken{Text: testData.text1, SID: testData.sid1}
		st2 := SymbolToken{Text: testData.text2, SID: testData.sid2}

		if st1.Equal(&st2) {
			t.Errorf("expected %v, got %v", false, true)
		}
	}
}
