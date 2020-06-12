package ion

import (
	"bytes"
	"math"
	"testing"
	"time"
)

func TestMarshalText(t *testing.T) {
	test := func(v interface{}, eval string) {
		t.Run(eval, func(t *testing.T) {
			val, err := MarshalText(v)
			if err != nil {
				t.Fatal(err)
			}
			if string(val) != eval {
				t.Errorf("expected '%v', got '%v'", eval, string(val))
			}
		})
	}

	test(nil, "null")
	test(true, "true")
	test(false, "false")

	test(byte(42), "42")
	test(-42, "-42")
	test(uint64(math.MaxUint64), "18446744073709551615")
	test(math.MinInt64, "-9223372036854775808")

	test(42.0, "4.2e+1")
	test(math.Inf(1), "+inf")
	test(math.Inf(-1), "-inf")
	test(math.NaN(), "nan")

	test(MustParseDecimal("1.20"), "1.20")
	test(time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), "2010-01-01T00:00:00Z")

	test("hello\tworld", "\"hello\\tworld\"")

	test(struct{ A, B int }{42, 0}, "{A:42,B:0}")
	test(struct {
		A int `ion:"val,ignoreme"`
		B int `ion:"-"`
		C int `ion:",omitempty"`
		d int
	}{42, 0, 0, 0}, "{val:42}")

	test(struct{ V interface{} }{}, "{V:null}")
	test(struct{ V interface{} }{"42"}, "{V:\"42\"}")

	fourtytwo := 42

	test(struct{ V *int }{}, "{V:null}")
	test(struct{ V *int }{&fourtytwo}, "{V:42}")

	test(map[string]int{"b": 2, "a": 1}, "{a:1,b:2}")

	test(struct{ V []int }{}, "{V:null}")
	test(struct{ V []int }{[]int{4, 2}}, "{V:[4,2]}")

	test(struct{ V []byte }{}, "{V:null}")
	test(struct{ V []byte }{[]byte{4, 2}}, "{V:{{BAI=}}}")

	test(struct{ V [2]byte }{[2]byte{4, 2}}, "{V:[4,2]}")
}

func TestMarshalBinary(t *testing.T) {
	test := func(v interface{}, name string, eval []byte) {
		t.Run(name, func(t *testing.T) {
			val, err := MarshalBinary(v)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(val, eval) {
				t.Errorf("expected '%v', got '%v'", fmtbytes(eval), fmtbytes(val))
			}
		})
	}

	test(nil, "null", []byte{0xE0, 0x01, 0x00, 0xEA, 0x0F})
	test(struct{ A, B int }{42, 0}, "{A:42,B:0}", []byte{
		0xE0, 0x01, 0x00, 0xEA,
		0xE9, 0x81, 0x83, 0xD6, 0x87, 0xB4, 0x81, 'A', 0x81, 'B',
		0xD5,
		0x8A, 0x21, 0x2A,
		0x8B, 0x20,
	})
}

func TestMarshalBinaryLST(t *testing.T) {
	lsta := NewLocalSymbolTable(nil, nil)
	lstb := NewLocalSymbolTable(nil, []string{
		"A", "B",
	})

	test := func(v interface{}, name string, lst SymbolTable, eval []byte) {
		t.Run(name, func(t *testing.T) {
			val, err := MarshalBinaryLST(v, lst)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(val, eval) {
				t.Errorf("expected '%v', got '%v'", fmtbytes(eval), fmtbytes(val))
			}
		})
	}

	test(nil, "null", lsta, []byte{0xE0, 0x01, 0x00, 0xEA, 0x0F})
	test(struct{ A, B int }{42, 0}, "{A:42,B:0}", lstb, []byte{
		0xE0, 0x01, 0x00, 0xEA,
		0xE9, 0x81, 0x83, 0xD6, 0x87, 0xB4, 0x81, 'A', 0x81, 'B',
		0xD5,
		0x8A, 0x21, 0x2A,
		0x8B, 0x20,
	})
}

func TestMarshalNestedStructs(t *testing.T) {
	type gp struct {
		A int `ion:"a"`
	}

	type gp2 struct {
		B int `ion:"b"`
	}

	type parent struct {
		gp
		*gp2
		C int `ion:"c"`
	}

	type root struct {
		parent
		D int `ion:"d"`
	}

	v := root{
		parent: parent{
			gp: gp{
				A: 1,
			},
			gp2: &gp2{
				B: 2,
			},
			C: 3,
		},
		D: 4,
	}

	val, err := MarshalText(v)
	if err != nil {
		t.Fatal(err)
	}

	eval := "{a:1,b:2,c:3,d:4}"
	if string(val) != eval {
		t.Errorf("expected %v, got %v", eval, string(val))
	}
}
