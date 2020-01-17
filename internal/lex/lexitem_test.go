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
)

func TestLexItem_String(t *testing.T) {
	tests := []struct {
		item     Item
		expected string
	}{
		{},
		{item: Item{Type: IonEOF}, expected: "EOF"},
		{item: Item{Type: itemType(100)}, expected: "Unknown itemType 100 <>"},
		{item: Item{Type: IonIllegal, Val: []byte("illegal")}, expected: "illegal"},
		{item: Item{Type: IonCommentLine, Val: []byte("comment")}, expected: `<comment>`},
		{
			item: Item{
				Type: IonString,
				Val:  []byte("12345678901234567890123456789012345678901234567890123456789012345678901234567890"),
			},
			expected: `<123456789012345678901234567890123456789012345678901234567890123456789012345>...`,
		},
		{
			item: Item{
				Type: IonString,
				Val:  []byte("123456789012345678901234567890123456789012345678901234567890123456789012345"),
			},
			expected: `<123456789012345678901234567890123456789012345678901234567890123456789012345>`,
		},
	}

	for _, tst := range tests {
		test := tst
		t.Run(test.expected, func(t *testing.T) {
			if diff := cmp.Diff(test.expected, test.item.String()); diff != "" {
				t.Error("(-expected, +found)", diff)
			}
		})
	}
}
