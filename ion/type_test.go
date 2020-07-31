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

func TestTypeToString(t *testing.T) {
	for i := NoType; i <= StructType+1; i++ {
		str := i.String()
		if str == "" {
			t.Errorf("expected a non-empty string for type %v", uint8(i))
		}
	}
}

func TestIntSizeToString(t *testing.T) {
	for i := NullInt; i <= BigInt+1; i++ {
		str := i.String()
		if str == "" {
			t.Errorf("expected a non-empty string for size %v", uint8(i))
		}
	}
}

func TestIsScalar(t *testing.T) {
	scalarTypes := []Type{NullType, BoolType, IntType, FloatType, DecimalType,
		TimestampType, SymbolType, StringType, ClobType, BlobType}

	for _, ionType := range scalarTypes {
		if !IsScalar(ionType) {
			t.Errorf("Expected IsScalar() to return true for type %s", ionType.String())
		}
	}

	nonScalarTypes := []Type{NoType, ListType, SexpType, StructType}

	for _, ionType := range nonScalarTypes {
		if IsScalar(ionType) {
			t.Errorf("Expected IsScalar() to return false for type %s", ionType.String())
		}
	}
}

func TestIsContainer(t *testing.T) {
	containerTypes := []Type{ListType, SexpType, StructType}

	for _, ionType := range containerTypes {
		if !IsContainer(ionType) {
			t.Errorf("Expected IsContainer() to return true for type %s", ionType.String())
		}
	}

	nonContainerTypes := []Type{NoType, NullType, BoolType, IntType, FloatType, DecimalType,
		TimestampType, SymbolType, StringType, ClobType, BlobType}

	for _, ionType := range nonContainerTypes {
		if IsContainer(ionType) {
			t.Errorf("Expected IsContainer() to return false for type %s", ionType.String())
		}
	}
}
