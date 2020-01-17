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
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Ensure that all of our types satisfy the Value interface.
var _ Value = Blob{}
var _ Value = Bool{}
var _ Value = Clob{}
var _ Value = Decimal{}
var _ Value = Float{}
var _ Value = Int{}
var _ Value = List{}
var _ Value = Null{}
var _ Value = padding{}
var _ Value = SExp{}
var _ Value = String{}
var _ Value = Struct{}
var _ Value = Symbol{}
var _ Value = Timestamp{}

func TestBool(t *testing.T) {
	tests := []struct {
		b             Bool
		isNull        bool
		expectedValue bool
		expectedText  string
	}{
		{isNull: true, expectedText: "null.bool"},
		{b: Bool{value: true}, isNull: true, expectedText: "null.bool"},
		{b: Bool{isSet: true}, expectedText: "false"},
		{b: Bool{isSet: true, value: true}, expectedValue: true, expectedText: "true"},
	}

	for _, tst := range tests {
		test := tst
		t.Run(fmt.Sprintf("%#v", test.b), func(t *testing.T) {
			if isNull := test.b.IsNull(); isNull != test.isNull {
				t.Error("expected IsNull", test.isNull, "but found", isNull)
			}
			if found := test.b.Value(); found != test.expectedValue {
				t.Error("expected value", test.expectedValue, "but found", found)
			}
			if diff := cmp.Diff(test.expectedText, string(test.b.Text())); diff != "" {
				t.Error("(-expected, +found)", diff)
			}
			if typ := test.b.Type(); typ != TypeBool {
				t.Error("expected TypeBool", TypeBool, "but found", typ)
			}
		})
	}
}

func TestNull(t *testing.T) {
	tests := []struct {
		typ          Type
		expectedText string
	}{
		{expectedText: "null.null"},
		{typ: TypeBlob, expectedText: "null.blob"},
		{typ: TypeBool, expectedText: "null.bool"},
		{typ: TypeClob, expectedText: "null.clob"},
		{typ: TypeDecimal, expectedText: "null.decimal"},
		{typ: TypeFloat, expectedText: "null.float"},
		{typ: TypeInt, expectedText: "null.int"},
		{typ: TypeList, expectedText: "null.list"},
		{typ: TypeLongString, expectedText: "null.string"},
		{typ: TypeSExp, expectedText: "null.sexp"},
		{typ: TypeString, expectedText: "null.string"},
		{typ: TypeStruct, expectedText: "null.struct"},
		{typ: TypeSymbol, expectedText: "null.symbol"},
		{typ: TypeTimestamp, expectedText: "null.timestamp"},
	}

	for _, tst := range tests {
		test := tst
		t.Run(strconv.Itoa(int(test.typ)), func(t *testing.T) {
			null := &Null{typ: test.typ}
			if diff := cmp.Diff(test.expectedText, string(null.Text())); diff != "" {
				t.Error("(-expected, +found)", diff)
			}
			if diff := cmp.Diff(test.typ, null.Type()); diff != "" {
				t.Error("(-expected, +found)", diff)
			}
			if !null.IsNull() {
				t.Error("expected IsNull to be true")
			}
		})
	}
}
