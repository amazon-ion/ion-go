package ion

import (
	"math"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestTopLevelFieldName(t *testing.T) {
	writeText(func(w Writer) {
		w.FieldName("foo")
		if w.Err() == nil {
			t.Error("expected an error")
		}
	})
}

func TestEmptyStruct(t *testing.T) {
	testTextWriter(t, "{}", func(w Writer) {
		if w.InStruct() {
			t.Error("already in struct")
		}

		w.BeginStruct()
		if w.Err() != nil {
			t.Fatal(w.Err())
		}

		if !w.InStruct() {
			t.Error("not in struct after begin")
		}

		w.EndStruct()
		if w.Err() != nil {
			t.Fatal(w.Err())
		}

		if w.InStruct() {
			t.Error("still in struct after end")
		}

		w.EndStruct()
		if w.Err() == nil {
			t.Fatal("no error from ending struct too many times")
		}
	})
}

func TestAnnotatedStruct(t *testing.T) {
	testTextWriter(t, "foo::$bar::'.baz'::{}", func(w Writer) {
		w.TypeAnnotation("foo")
		w.TypeAnnotation("$bar")
		w.TypeAnnotation(".baz")
		w.BeginStruct()
		w.EndStruct()

		if w.Err() != nil {
			t.Fatal(w.Err())
		}
	})
}

func TestNestedStruct(t *testing.T) {
	testTextWriter(t, "{foo:'true'::{},'null':{}}", func(w Writer) {
		w.BeginStruct()

		w.FieldName("foo")
		w.TypeAnnotation("true")
		w.BeginStruct()
		w.EndStruct()

		w.FieldName("null")
		w.BeginStruct()
		w.EndStruct()

		w.EndStruct()
	})
}

func TestEmptyList(t *testing.T) {
	testTextWriter(t, "[]", func(w Writer) {
		w.BeginList()
		if w.Err() != nil {
			t.Fatal(w.Err())
		}

		if w.InStruct() {
			t.Error("instruct returns true in a list")
		}

		w.EndList()
		if w.Err() != nil {
			t.Fatal(w.Err())
		}

		w.EndList()
		if w.Err() == nil {
			t.Error("no error calling endlist at top level")
		}
	})
}

func TestNestedLists(t *testing.T) {
	testTextWriter(t, "[{},foo::{},'null'::[]]", func(w Writer) {
		w.BeginList()

		w.BeginStruct()
		w.EndStruct()

		w.TypeAnnotation("foo")
		w.BeginStruct()
		w.EndStruct()

		w.TypeAnnotation("null")
		w.BeginList()
		w.EndList()

		w.EndList()
	})
}

func TestSexps(t *testing.T) {
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

func TestNull(t *testing.T) {
	expected := "[null,foo::null,null.int,bar::null.sexp]"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginList()

		w.WriteNull()
		w.TypeAnnotation("foo")
		w.WriteNullWithType(NullType)
		w.WriteNullWithType(IntType)
		w.TypeAnnotation("bar")
		w.WriteNullWithType(SexpType)

		w.EndList()
	})
}

func TestBool(t *testing.T) {
	expected := "true\n(false '123'::true)\n'false'::false"
	testTextWriter(t, expected, func(w Writer) {
		w.WriteBool(true)

		w.BeginSexp()

		w.WriteBool(false)
		w.TypeAnnotation("123")
		w.WriteBool(true)

		w.EndSexp()

		w.TypeAnnotation("false")
		w.WriteBool(false)
	})
}

func TestInt(t *testing.T) {
	expected := "(zero::0 1 -1 (9223372036854775807 -9223372036854775808))"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginSexp()

		w.TypeAnnotation("zero")
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

func TestBigInt(t *testing.T) {
	expected := "[0,big::18446744073709551616]"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginList()

		w.WriteBigInt(big.NewInt(0))

		var val, max, one big.Int
		max.SetUint64(math.MaxUint64)
		one.SetInt64(1)
		val.Add(&max, &one)

		w.TypeAnnotation("big")
		w.WriteBigInt(&val)

		w.EndList()
	})
}

func TestFloat(t *testing.T) {
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

func TestDecimal(t *testing.T) {
	expected := "0.\n-1.23d-98"
	testTextWriter(t, expected, func(w Writer) {
		w.WriteDecimal(MustParseDecimal("0"))
		w.WriteDecimal(MustParseDecimal("-123d-100"))
	})
}

func TestTimestamp(t *testing.T) {
	expected := "1970-01-01T00:00:00.001Z\n1970-01-01T01:23:00+01:23"
	testTextWriter(t, expected, func(w Writer) {
		w.WriteTimestamp(time.Unix(0, 1000000).In(time.UTC))
		w.WriteTimestamp(time.Unix(0, 0).In(time.FixedZone("wtf", 4980)))
	})
}

func TestSymbol(t *testing.T) {
	expected := "{foo:bar,empty:'','null':'null',f:a::b::u::'lo🇺🇸'}"
	testTextWriter(t, expected, func(w Writer) {
		w.BeginStruct()

		w.FieldName("foo")
		w.WriteSymbol("bar")
		w.FieldName("empty")
		w.WriteSymbol("")
		w.FieldName("null")
		w.WriteSymbol("null")

		w.FieldName("f")
		w.TypeAnnotation("a")
		w.TypeAnnotation("b")
		w.TypeAnnotation("u")
		w.WriteSymbol("lo🇺🇸")

		w.EndStruct()
	})
}

func TestString(t *testing.T) {
	expected := `("hello" "" ("\\\"\n\"\\" zany::"🤪"))`
	testTextWriter(t, expected, func(w Writer) {
		w.BeginSexp()
		w.WriteString("hello")
		w.WriteString("")

		w.BeginSexp()
		w.WriteString("\\\"\n\"\\")
		w.TypeAnnotation("zany")
		w.WriteString("🤪")
		w.EndSexp()

		w.EndSexp()
	})
}

func TestBlob(t *testing.T) {
	expected := "{{AAEC/f7/}}\n{{SGVsbG8gV29ybGQ=}}\nempty::{{}}"
	testTextWriter(t, expected, func(w Writer) {
		w.WriteBlob([]byte{0, 1, 2, 0xFD, 0xFE, 0xFF})
		w.WriteBlob([]byte("Hello World"))
		w.TypeAnnotation("empty")
		w.WriteBlob(nil)
	})
}

func TestClob(t *testing.T) {
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

func TestFinish(t *testing.T) {
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

func TestBadFinish(t *testing.T) {
	buf := strings.Builder{}
	w := NewTextWriter(&buf)

	w.BeginStruct()
	err := w.Finish()

	if err == nil {
		t.Error("should not be able to finish in the middle of a struct")
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
