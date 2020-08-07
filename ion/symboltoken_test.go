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

func TestNewSymbolTokenSidAndTextUnknown(t *testing.T) {
	st := NewSymbolToken(nil, UnknownSid, nil)
	if st.text != nil {
		t.Errorf("expected %v, got %v", nil, st.text)
	}
	if st.sid != UnknownSid {
		t.Errorf("expected %v, got %v", UnknownSid, st.sid)
	}
	if st.importLocation != nil {
		t.Errorf("expected %v, got %v", nil, st.importLocation)
	}
}

var boolEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 123},
	{"text2", 456, "text2", 456},
}

func TestBoolEqualsOperator(t *testing.T) {
	for _, testData := range boolEqualsTestData {
		st1 := NewSymbolToken(&testData.text1, testData.sid1, nil)
		st2 := NewSymbolToken(&testData.text2, testData.sid2, nil)

		if !st1.Equal(st2) {
			t.Errorf("expected %v, got %v", true, false)
		}
	}
}

var boolNotEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 456},
	{"text2", 456, "text3", 456},
}

func TestBoolNotEqualsOperator(t *testing.T) {
	for _, testData := range boolNotEqualsTestData {
		st1 := NewSymbolToken(&testData.text1, testData.sid1, nil)
		st2 := NewSymbolToken(&testData.text2, testData.sid2, nil)

		if st1.Equal(st2) {
			t.Errorf("expected %v, got %v", false, true)
		}
	}
}
