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
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTextTopLevelFieldName(t *testing.T) {
	writeText(func(w Writer) {
		assert.Error(t, w.FieldName(NewSymbolTokenFromString("foo")))
	})
}

func TestWriteTextEmptyStruct(t *testing.T) {
	testTextWriter(t, "{}", func(w Writer) {
		require.NoError(t, w.BeginStruct())

		require.NoError(t, w.EndStruct())

		require.Error(t, w.EndStruct())
	})
}

func TestWriteTextAnnotatedStruct(t *testing.T) {
	testTextWriter(t, "foo::$bar::'.baz'::{}", func(w Writer) {
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("foo")))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("$bar")))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString(".baz")))
		assert.NoError(t, w.BeginStruct())
		require.NoError(t, w.EndStruct())
	})
}

func TestWriteTextNestedStruct(t *testing.T) {
	testTextWriter(t, "{foo:'true'::{},'null':{}}", func(w Writer) {
		assert.NoError(t, w.BeginStruct())

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("foo")))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("true")))
		assert.NoError(t, w.BeginStruct())
		assert.NoError(t, w.EndStruct())

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("null")))
		assert.NoError(t, w.BeginStruct())
		assert.NoError(t, w.EndStruct())

		assert.NoError(t, w.EndStruct())
	})
}

func TestWriteTextEmptyList(t *testing.T) {
	testTextWriter(t, "[]", func(w Writer) {
		require.NoError(t, w.BeginList())
		require.NoError(t, w.EndList())
		require.Error(t, w.EndList())
	})
}

func TestWriteTextNestedLists(t *testing.T) {
	testTextWriter(t, "[{},foo::{},'null'::[]]", func(w Writer) {
		assert.NoError(t, w.BeginList())

		assert.NoError(t, w.BeginStruct())
		assert.NoError(t, w.EndStruct())

		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("foo")))
		assert.NoError(t, w.BeginStruct())
		assert.NoError(t, w.EndStruct())

		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("null")))
		assert.NoError(t, w.BeginList())
		assert.NoError(t, w.EndList())

		assert.NoError(t, w.EndList())
	})
}

func TestWriteTextSexps(t *testing.T) {
	testTextWriter(t, "()\n(())\n(() ())", func(w Writer) {
		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.EndSexp())

		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.EndSexp())
		assert.NoError(t, w.EndSexp())

		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.EndSexp())
		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.EndSexp())
		assert.NoError(t, w.EndSexp())
	})
}

func TestWriteTextNulls(t *testing.T) {
	expected := "[null,foo::null.null,null.bool,null.int,null.float,null.decimal," +
		"null.timestamp,null.symbol,null.string,null.clob,null.blob," +
		"null.list,'null'::null.sexp,null.struct]"

	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.BeginList())

		assert.NoError(t, w.WriteNull())
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("foo")))
		assert.NoError(t, w.WriteNullType(NullType))
		assert.NoError(t, w.WriteNullType(BoolType))
		assert.NoError(t, w.WriteNullType(IntType))
		assert.NoError(t, w.WriteNullType(FloatType))
		assert.NoError(t, w.WriteNullType(DecimalType))
		assert.NoError(t, w.WriteNullType(TimestampType))
		assert.NoError(t, w.WriteNullType(SymbolType))
		assert.NoError(t, w.WriteNullType(StringType))
		assert.NoError(t, w.WriteNullType(ClobType))
		assert.NoError(t, w.WriteNullType(BlobType))
		assert.NoError(t, w.WriteNullType(ListType))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("null")))
		assert.NoError(t, w.WriteNullType(SexpType))
		assert.NoError(t, w.WriteNullType(StructType))

		assert.NoError(t, w.EndList())
	})
}

func TestWriteTextBool(t *testing.T) {
	expected := "true\n(false '123'::true)\n'false'::false"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.WriteBool(true))

		assert.NoError(t, w.BeginSexp())

		assert.NoError(t, w.WriteBool(false))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("123")))
		assert.NoError(t, w.WriteBool(true))

		assert.NoError(t, w.EndSexp())

		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("false")))
		assert.NoError(t, w.WriteBool(false))
	})
}

func TestWriteTextInt(t *testing.T) {
	expected := "(zero::0 1 -1 (9223372036854775807 -9223372036854775808))"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.BeginSexp())

		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("zero")))
		assert.NoError(t, w.WriteInt(0))
		assert.NoError(t, w.WriteInt(1))
		assert.NoError(t, w.WriteInt(-1))

		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.WriteInt(math.MaxInt64))
		assert.NoError(t, w.WriteInt(math.MinInt64))
		assert.NoError(t, w.EndSexp())

		assert.NoError(t, w.EndSexp())
	})
}

func TestWriteTextBigInt(t *testing.T) {
	expected := "[0,big::18446744073709551616]"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.BeginList())

		assert.NoError(t, w.WriteBigInt(big.NewInt(0)))

		var val, max, one big.Int
		max.SetUint64(math.MaxUint64)
		one.SetInt64(1)
		val.Add(&max, &one)

		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("big")))
		assert.NoError(t, w.WriteBigInt(&val))

		assert.NoError(t, w.EndList())
	})
}

func TestWriteTextFloat(t *testing.T) {
	expected := "{z:0e+0,nz:-0e+0,s:1.234e+1,l:1.234e-55,n:nan,i:+inf,ni:-inf}"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.BeginStruct())

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("z")))
		assert.NoError(t, w.WriteFloat(0.0))
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("nz")))
		assert.NoError(t, w.WriteFloat(-1.0/math.Inf(1)))

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("s")))
		assert.NoError(t, w.WriteFloat(12.34))
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("l")))
		assert.NoError(t, w.WriteFloat(12.34e-56))

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("n")))
		assert.NoError(t, w.WriteFloat(math.NaN()))
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("i")))
		assert.NoError(t, w.WriteFloat(math.Inf(1)))
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("ni")))
		assert.NoError(t, w.WriteFloat(math.Inf(-1)))

		assert.NoError(t, w.EndStruct())
	})
}

func TestWriteTextDecimal(t *testing.T) {
	expected := "0.\n-1.23d-98"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("0")))
		assert.NoError(t, w.WriteDecimal(MustParseDecimal("-123d-100")))
	})
}

func TestWriteTextTimestamp(t *testing.T) {
	expected := "1970-01-01T00:00:00.001Z\n1970-01-01T01:23:00+01:23"
	testTextWriter(t, expected, func(w Writer) {
		dateTime := time.Unix(0, 1000000).In(time.UTC)
		assert.NoError(t, w.WriteTimestamp(NewTimestampWithFractionalSeconds(dateTime, TimestampPrecisionNanosecond, TimezoneUTC, 3)))
		dateTime = time.Unix(0, 0).In(time.FixedZone("foo", 4980))
		assert.NoError(t, w.WriteTimestamp(NewTimestamp(dateTime, TimestampPrecisionSecond, TimezoneLocal)))
	})
}

func TestWriteTextSymbol(t *testing.T) {
	expected := "{foo:bar,empty:'','null':'null',f:a::b::u::'lo🇺🇸',$123:$456}"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.BeginStruct())

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("foo")))
		assert.NoError(t, w.WriteSymbolFromString("bar"))
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("empty")))
		assert.NoError(t, w.WriteSymbolFromString(""))
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("null")))
		assert.NoError(t, w.WriteSymbolFromString("null"))

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("f")))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("a")))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("b")))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("u")))
		assert.NoError(t, w.WriteSymbolFromString("lo🇺🇸"))

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("$123")))
		assert.NoError(t, w.WriteSymbolFromString("$456"))

		assert.NoError(t, w.EndStruct())
	})
}

func TestWriteTextString(t *testing.T) {
	expected := `("hello" "" ("\\\"\n\"\\" zany::"🤪"))`
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.WriteString("hello"))
		assert.NoError(t, w.WriteString(""))

		assert.NoError(t, w.BeginSexp())
		assert.NoError(t, w.WriteString("\\\"\n\"\\"))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("zany")))
		assert.NoError(t, w.WriteString("🤪"))
		assert.NoError(t, w.EndSexp())

		assert.NoError(t, w.EndSexp())
	})
}

func TestWriteTextBlob(t *testing.T) {
	expected := "{{AAEC/f7/}}\n{{SGVsbG8gV29ybGQ=}}\nempty::{{}}"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.WriteBlob([]byte{0, 1, 2, 0xFD, 0xFE, 0xFF}))
		assert.NoError(t, w.WriteBlob([]byte("Hello World")))
		assert.NoError(t, w.Annotation(NewSymbolTokenFromString("empty")))
		assert.NoError(t, w.WriteBlob(nil))
	})
}

func TestWriteTextClob(t *testing.T) {
	expected := "{hello:{{\"world\"}},bits:{{\"\\0\\x01\\xFE\\xFF\"}}}"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.BeginStruct())
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("hello")))
		assert.NoError(t, w.WriteClob([]byte("world")))
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("bits")))
		assert.NoError(t, w.WriteClob([]byte{0, 1, 0xFE, 0xFF}))
		assert.NoError(t, w.EndStruct())
	})
}

func TestWriteTextFinish(t *testing.T) {
	expected := "1\nfoo\n\"bar\"\n{}\n"
	testTextWriter(t, expected, func(w Writer) {
		assert.NoError(t, w.WriteInt(1))
		assert.NoError(t, w.WriteSymbolFromString("foo"))
		assert.NoError(t, w.WriteString("bar"))
		assert.NoError(t, w.BeginStruct())
		assert.NoError(t, w.EndStruct())
		require.NoError(t, w.Finish())
	})
}

func TestWriteTextBadFinish(t *testing.T) {
	buf := strings.Builder{}
	w := NewTextWriter(&buf)

	assert.NoError(t, w.BeginStruct())
	require.Error(t, w.Finish())
}

func TestWriteTextPretty(t *testing.T) {
	buf := strings.Builder{}
	w := NewTextWriterOpts(&buf, TextWriterPretty)

	assert.NoError(t, w.BeginStruct())
	{
		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("struct")))
		assert.NoError(t, w.BeginStruct())
		assert.NoError(t, w.EndStruct())

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("list")))
		assert.NoError(t, w.Annotations(
			NewSymbolTokenFromString("i"),
			NewSymbolTokenFromString("am"),
			NewSymbolTokenFromString("a"),
			NewSymbolTokenFromString("list")))
		assert.NoError(t, w.BeginList())
		{
			assert.NoError(t, w.WriteString("value"))
			assert.NoError(t, w.WriteNullType(StringType))
			assert.NoError(t, w.BeginStruct())
			{
				assert.NoError(t, w.FieldName(NewSymbolTokenFromString("1")))
				assert.NoError(t, w.WriteString("one"))
				assert.NoError(t, w.FieldName(NewSymbolTokenFromString("2")))
				assert.NoError(t, w.WriteString("two"))
			}
			assert.NoError(t, w.EndStruct())
		}
		assert.NoError(t, w.EndList())

		assert.NoError(t, w.FieldName(NewSymbolTokenFromString("sexp")))
		assert.NoError(t, w.BeginSexp())
		{
			assert.NoError(t, w.WriteSymbolFromString("+"))
			assert.NoError(t, w.WriteInt(123))
			assert.NoError(t, w.BeginSexp())
			{
				assert.NoError(t, w.WriteSymbolFromString("*"))
				assert.NoError(t, w.WriteInt(456))
				assert.NoError(t, w.WriteInt(789))
			}
			assert.NoError(t, w.EndSexp())
		}
		assert.NoError(t, w.EndSexp())
	}
	assert.NoError(t, w.EndStruct())

	require.NoError(t, w.Finish())

	actual := buf.String()
	expected := `{
	struct: {},
	list: i::am::a::list::[
		"value",
		null.string,
		{
			'1': "one",
			'2': "two"
		}
	],
	sexp: (
		'+'
		123
		(
			'*'
			456
			789
		)
	)
}
`
	assert.Equal(t, expected, actual)
}

func testTextWriter(t *testing.T, expected string, f func(Writer)) {
	actual := writeText(f)
	assert.Equal(t, expected, actual)
}

func writeText(f func(Writer)) string {
	buf := strings.Builder{}
	w := NewTextWriter(&buf)

	f(w)

	return buf.String()
}
