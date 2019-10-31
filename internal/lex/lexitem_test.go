/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

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
