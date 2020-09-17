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
)

func TestWriteTextTopLevelFieldName(t *testing.T) {
	writeText(func(w Writer) {
		if err := w.FieldName("foo"); err == nil {
			t.Error("expected an error")
		}
	})
}

func TestWriteTextEmptyStruct(t *testing.T) {
	testTextWriter(t, "{}", func(w Writer) {
		if err := w.BeginStruct(); err != nil {
			t.Fatal(err)
		}

		if err := w.EndStruct(); err != nil {
			t.Fatal(err)
		}

		if err := w.EndStruct(); err == nil {
			t.Fatal("no error from ending struct too many times")
		}
	})
}

func TestWriteTextAnnotatedStruct(t *testing.T) {
	testTextWriter(t, "foo::$bar::'.baz'::{}", func(w Writer) {
		w.Annotation(SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown})
		w.Annotation(SymbolToken{Text: newString("$bar"), LocalSID: SymbolIDUnknown})
		w.Annotation(SymbolToken{Text: newString(".baz"), LocalSID: SymbolIDUnknown})
		w.BeginStruct()
		err := w.EndStruct()

		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestWriteTextNestedStruct(t *testing.T) {
	testTextWriter(t, "{foo:'true'::{},'null':{}}", func(w Writer) {
		w.BeginStruct()

		w.FieldName("foo")
		w.Annotation(SymbolToken{Text: newString("true"), LocalSID: SymbolIDUnknown})
		w.BeginStruct()
		w.EndStruct()

		w.FieldName("null")
		w.BeginStruct()
		w.EndStruct()

		w.EndStruct()
	})
}

func TestWriteTextEmptyList(t *testing.T) {
	testTextWriter(t, "[]", func(w Writer) {
		if err := w.BeginList(); err != nil {
			t.Fatal(err)
		}

		if err := w.EndList(); err != nil {
			t.Fatal(err)
		}

		if err := w.EndList(); err == nil {
			t.Error("no error calling endlist at top level")
		}
	})
}

func TestWriteTextNestedLists(t *testing.T) {
	testTextWriter(t, "[{},foo::{},'null'::[]]", func(w Writer) {
		w.BeginList()

		w.BeginStruct()
		w.EndStruct()

		w.Annotation(SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown})
		w.BeginStruct()
		w.EndStruct()

		w.Annotation(SymbolToken{Text: newString("null"), LocalSID: SymbolIDUnknown})
		w.BeginList()
		w.EndList()

		w.EndList()
	})
}

func TestWriteTextSexps(t *testing.T) {
	testTextWriter(t, "()\n(())\n(() ())", func(w Writer) {
		w.BeginSexp()
		w.EndSexp()

		w.BeginSexp()
		w.BeginSexp()
		w.EndSexp()
		w.EndSexp()

		w.BeginSexp()
		w.BeginSexp()
		w.EndSexp()
		w.BeginSexp()
		w.EndSexp()
		w.EndSexp()
	})
}

func TestWriteTextNulls(t *testing.T) {
	expected := "[null,foo::null.null,null.bool,null.int,null.float,null.decimal," +
		"null.timestamp,null.symbol,null.string,null.clob,null.blob," +
		"null.list,'null'::null.sexp,null.struct]"

	testTextWriter(t, expected, func(w Writer) {
		w.BeginList()

		w.WriteNull()
		w.Annotation(SymbolToken{Text: newString("foo"), LocalSID: SymbolIDUnknown})
		w.WriteNullType(NullType)
		w.WriteNullType(BoolType)
		w.WriteNullType(IntType)
		w.WriteNullType(FloatType)
		w.WriteNullType(DecimalType)
		w.WriteNullType(TimestampType)
		w.WriteNullType(SymbolType)
		w.WriteNullType(StringType)
		w.WriteNullType(ClobType)
		w.WriteNullType(BlobType)
		w.WriteNullType(ListType)
		w.Annotation(SymbolToken{Text: newString("null"), LocalSID: SymbolIDUnknown})
		w.WriteNullType(SexpType)
		w.WriteNullType(StructType)

		w.EndList()
	})
}

func TestWriteTextBool(t *testing.T) {
	expected := "true\n(false '123'::true)\n'false'::false"
	testTextWriter(t, expected, func(w Writer) {
		w.WriteBool(true)

		w.BeginSexp()

		w.WriteBool(false)
		w.Annotation(SymbolToken{Text: newString("123"), LocalSID: SymbolIDUnknown})
		w.WriteBool(true)

		w.EndSexp()

		w.Annotation(SymbolToken{Text: newString("false"), LocalSID: SymbolIDUnknown})
		w.WriteBool(false)
	})
}

func TestWriteTextInt(t *testing.T) {
	expected := "(zero::0 1 -1 (9223372036854775807 -9223372036854775808))"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginSexp()

		w.Annotation(SymbolToken{Text: newString("zero"), LocalSID: SymbolIDUnknown})
		w.WriteInt(0)
		w.WriteInt(1)
		w.WriteInt(-1)

		w.BeginSexp()
		w.WriteInt(math.MaxInt64)
		w.WriteInt(math.MinInt64)
		w.EndSexp()

		w.EndSexp()
	})
}

func TestWriteTextBigInt(t *testing.T) {
	expected := "[0,big::18446744073709551616]"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginList()

		w.WriteBigInt(big.NewInt(0))

		var val, max, one big.Int
		max.SetUint64(math.MaxUint64)
		one.SetInt64(1)
		val.Add(&max, &one)

		w.Annotation(SymbolToken{Text: newString("big"), LocalSID: SymbolIDUnknown})
		w.WriteBigInt(&val)

		w.EndList()
	})
}

func TestWriteTextFloat(t *testing.T) {
	expected := "{z:0e+0,nz:-0e+0,s:1.234e+1,l:1.234e-55,n:nan,i:+inf,ni:-inf}"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginStruct()

		w.FieldName("z")
		w.WriteFloat(0.0)
		w.FieldName("nz")
		w.WriteFloat(-1.0 / math.Inf(1))

		w.FieldName("s")
		w.WriteFloat(12.34)
		w.FieldName("l")
		w.WriteFloat(12.34e-56)

		w.FieldName("n")
		w.WriteFloat(math.NaN())
		w.FieldName("i")
		w.WriteFloat(math.Inf(1))
		w.FieldName("ni")
		w.WriteFloat(math.Inf(-1))

		w.EndStruct()
	})
}

func TestWriteTextDecimal(t *testing.T) {
	expected := "0.\n-1.23d-98"
	testTextWriter(t, expected, func(w Writer) {
		w.WriteDecimal(MustParseDecimal("0"))
		w.WriteDecimal(MustParseDecimal("-123d-100"))
	})
}

func TestWriteTextTimestamp(t *testing.T) {
	expected := "1970-01-01T00:00:00.001Z\n1970-01-01T01:23:00+01:23"
	testTextWriter(t, expected, func(w Writer) {
		dateTime := time.Unix(0, 1000000).In(time.UTC)
		w.WriteTimestamp(NewTimestampWithFractionalSeconds(dateTime, TimestampPrecisionNanosecond, TimezoneUTC, 3))
		dateTime = time.Unix(0, 0).In(time.FixedZone("foo", 4980))
		w.WriteTimestamp(NewTimestamp(dateTime, TimestampPrecisionSecond, TimezoneLocal))
	})
}

func TestWriteTextSymbol(t *testing.T) {
	expected := "{foo:bar,empty:'','null':'null',f:a::b::u::'lo🇺🇸',$123:$456}"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginStruct()

		w.FieldName("foo")
		w.WriteSymbol("bar")
		w.FieldName("empty")
		w.WriteSymbol("")
		w.FieldName("null")
		w.WriteSymbol("null")

		w.FieldName("f")
		w.Annotation(SymbolToken{Text: newString("a"), LocalSID: SymbolIDUnknown})
		w.Annotation(SymbolToken{Text: newString("b"), LocalSID: SymbolIDUnknown})
		w.Annotation(SymbolToken{Text: newString("u"), LocalSID: SymbolIDUnknown})
		w.WriteSymbol("lo🇺🇸")

		w.FieldName("$123")
		w.WriteSymbol("$456")

		w.EndStruct()
	})
}

func TestWriteTextString(t *testing.T) {
	expected := `("hello" "" ("\\\"\n\"\\" zany::"🤪"))`
	testTextWriter(t, expected, func(w Writer) {
		w.BeginSexp()
		w.WriteString("hello")
		w.WriteString("")

		w.BeginSexp()
		w.WriteString("\\\"\n\"\\")
		w.Annotation(SymbolToken{Text: newString("zany"), LocalSID: SymbolIDUnknown})
		w.WriteString("🤪")
		w.EndSexp()

		w.EndSexp()
	})
}

func TestWriteTextBlob(t *testing.T) {
	expected := "{{AAEC/f7/}}\n{{SGVsbG8gV29ybGQ=}}\nempty::{{}}"
	testTextWriter(t, expected, func(w Writer) {
		w.WriteBlob([]byte{0, 1, 2, 0xFD, 0xFE, 0xFF})
		w.WriteBlob([]byte("Hello World"))
		w.Annotation(SymbolToken{Text: newString("empty"), LocalSID: SymbolIDUnknown})
		w.WriteBlob(nil)
	})
}

func TestWriteTextClob(t *testing.T) {
	expected := "{hello:{{\"world\"}},bits:{{\"\\0\\x01\\xFE\\xFF\"}}}"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginStruct()
		w.FieldName("hello")
		w.WriteClob([]byte("world"))
		w.FieldName("bits")
		w.WriteClob([]byte{0, 1, 0xFE, 0xFF})
		w.EndStruct()
	})
}

func TestWriteTextFinish(t *testing.T) {
	expected := "1\nfoo\n\"bar\"\n{}\n"
	testTextWriter(t, expected, func(w Writer) {
		w.WriteInt(1)
		w.WriteSymbol("foo")
		w.WriteString("bar")
		w.BeginStruct()
		w.EndStruct()
		if err := w.Finish(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestWriteTextBadFinish(t *testing.T) {
	buf := strings.Builder{}
	w := NewTextWriter(&buf)

	w.BeginStruct()
	err := w.Finish()

	if err == nil {
		t.Error("should not be able to finish in the middle of a struct")
	}
}

func TestWriteTextPretty(t *testing.T) {
	buf := strings.Builder{}
	w := NewTextWriterOpts(&buf, TextWriterPretty)

	w.BeginStruct()
	{
		w.FieldName("struct")
		w.BeginStruct()
		w.EndStruct()

		w.FieldName("list")
		w.Annotations(SymbolToken{Text: newString("i"), LocalSID: SymbolIDUnknown}, SymbolToken{Text: newString("am"), LocalSID: SymbolIDUnknown}, SymbolToken{Text: newString("a"), LocalSID: SymbolIDUnknown}, SymbolToken{Text: newString("list"), LocalSID: SymbolIDUnknown})
		w.BeginList()
		{
			w.WriteString("value")
			w.WriteNullType(StringType)
			w.BeginStruct()
			{
				w.FieldName("1")
				w.WriteString("one")
				w.FieldName("2")
				w.WriteString("two")
			}
			w.EndStruct()
		}
		w.EndList()

		w.FieldName("sexp")
		w.BeginSexp()
		{
			w.WriteSymbol("+")
			w.WriteInt(123)
			w.BeginSexp()
			{
				w.WriteSymbol("*")
				w.WriteInt(456)
				w.WriteInt(789)
			}
			w.EndSexp()
		}
		w.EndSexp()
	}
	w.EndStruct()

	if err := w.Finish(); err != nil {
		t.Fatal(err)
	}

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
	if actual != expected {
		t.Errorf("expected:\n%v\ngot:\n%v", expected, actual)
	}
}

func testTextWriter(t *testing.T, expected string, f func(Writer)) {
	actual := writeText(f)
	if actual != expected {
		t.Errorf("expected: %v, actual: %v", expected, actual)
	}
}

func writeText(f func(Writer)) string {
	buf := strings.Builder{}
	w := NewTextWriter(&buf)

	f(w)

	return buf.String()
}
