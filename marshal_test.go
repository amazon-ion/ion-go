package ion

import (
	"math"
	"testing"
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

	test("hello\tworld", "\"hello\\tworld\"")

	test(struct{ A, B int }{42, 0}, "{A:42,B:0}")
	test(struct {
		A int `json:"val,ignoreme"`
		B int `json:"-"`
		C int `json:",omitempty"`
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

func TestMarshalNestedStructs(t *testing.T) {
	type gp struct {
		A int `json:"a"`
	}

	type gp2 struct {
		B int `json:"b"`
	}

	type parent struct {
		gp
		*gp2
		C int `json:"c"`
	}

	type root struct {
		parent
		D int `json:"d"`
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
