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

package main

import (
	"math/big"

	"github.com/amzn/ion-go/ion"
)

type nopwriter struct{}

// NewNopWriter returns a no-op Ion writer.
func NewNopWriter() ion.Writer {
	return nopwriter{}
}

func (nopwriter) FieldName(ion.SymbolToken) error {
	return nil
}

func (nopwriter) Annotation(ion.SymbolToken) error {
	return nil
}

func (nopwriter) Annotations(...ion.SymbolToken) error {
	return nil
}

func (nopwriter) WriteNull() error {
	return nil
}

func (nopwriter) WriteNullType(ion.Type) error {
	return nil
}

func (nopwriter) WriteBool(bool) error {
	return nil
}

func (nopwriter) WriteInt(int64) error {
	return nil
}

func (nopwriter) WriteUint(uint64) error {
	return nil
}

func (nopwriter) WriteBigInt(*big.Int) error {
	return nil
}

func (nopwriter) WriteFloat(float64) error {
	return nil
}

func (nopwriter) WriteDecimal(*ion.Decimal) error {
	return nil
}

func (nopwriter) WriteTimestamp(ion.Timestamp) error {
	return nil
}

func (nopwriter) WriteSymbol(ion.SymbolToken) error {
	return nil
}

func (nopwriter) WriteSymbolFromString(string) error {
	return nil
}

func (nopwriter) WriteString(string) error {
	return nil
}

func (nopwriter) WriteClob([]byte) error {
	return nil
}

func (nopwriter) WriteBlob([]byte) error {
	return nil
}

func (nopwriter) BeginList() error {
	return nil
}

func (nopwriter) EndList() error {
	return nil
}

func (nopwriter) BeginSexp() error {
	return nil
}

func (nopwriter) EndSexp() error {
	return nil
}

func (nopwriter) BeginStruct() error {
	return nil
}

func (nopwriter) EndStruct() error {
	return nil
}

func (nopwriter) Finish() error {
	return nil
}

func (nopwriter) IsInStruct() bool {
	return false
}
