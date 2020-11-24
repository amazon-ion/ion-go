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
	"math/big"
	"reflect"
	"time"
)

var binaryNulls = func() []byte {
	ret := make([]byte, StructType+1)
	ret[NoType] = 0x0F
	ret[NullType] = 0x0F
	ret[BoolType] = 0x1F
	ret[IntType] = 0x2F
	ret[FloatType] = 0x4F
	ret[DecimalType] = 0x5F
	ret[TimestampType] = 0x6F
	ret[SymbolType] = 0x7F
	ret[StringType] = 0x8F
	ret[ClobType] = 0x9F
	ret[BlobType] = 0xAF
	ret[ListType] = 0xBF
	ret[SexpType] = 0xCF
	ret[StructType] = 0xDF
	return ret
}()

var textNulls = func() []string {
	ret := make([]string, StructType+1)
	ret[NoType] = "null"
	ret[NullType] = "null.null"
	ret[BoolType] = "null.bool"
	ret[IntType] = "null.int"
	ret[FloatType] = "null.float"
	ret[DecimalType] = "null.decimal"
	ret[TimestampType] = "null.timestamp"
	ret[SymbolType] = "null.symbol"
	ret[StringType] = "null.string"
	ret[ClobType] = "null.clob"
	ret[BlobType] = "null.blob"
	ret[ListType] = "null.list"
	ret[SexpType] = "null.sexp"
	ret[StructType] = "null.struct"
	return ret
}()

var hexChars = []byte{
	'0', '1', '2', '3', '4', '5', '6', '7',
	'8', '9', 'A', 'B', 'C', 'D', 'E', 'F',
}

var timestampType = reflect.TypeOf(Timestamp{})
var nativeTimeType = reflect.TypeOf(time.Time{})
var decimalType = reflect.TypeOf(Decimal{})
var bigIntType = reflect.TypeOf(big.Int{})
var symbolType = reflect.TypeOf(SymbolToken{})
