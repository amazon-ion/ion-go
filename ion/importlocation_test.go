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

func TestNewImportLocationSidAndTextUnknown(t *testing.T) {
	st := NewImportLocation(nil, UnknownSid)
	if st.importName != nil {
		t.Errorf("expected %v, got %v", nil, st.importName)
	}
	if st.sid != UnknownSid {
		t.Errorf("expected %v, got %v", UnknownSid, st.sid)
	}
}

var importLocationBoolEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 123},
	{"text2", 456, "text2", 456},
}

func TestImportLocationBoolEqualsOperator(t *testing.T) {
	for _, testData := range importLocationBoolEqualsTestData {
		st1 := NewImportLocation(&testData.text1, testData.sid1)
		st2 := NewImportLocation(&testData.text2, testData.sid2)

		if !st1.Equal(st2) {
			t.Errorf("expected %v, got %v", true, false)
		}
	}
}

var importLocationBoolNotEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 456},
	{"text2", 456, "text3", 456},
}

func TestImportLocationBoolNotEqualsOperator(t *testing.T) {
	for _, testData := range importLocationBoolNotEqualsTestData {
		st1 := NewImportLocation(&testData.text1, testData.sid1)
		st2 := NewImportLocation(&testData.text2, testData.sid2)

		if st1.Equal(st2) {
			t.Errorf("expected %v, got %v", false, true)
		}
	}
}
