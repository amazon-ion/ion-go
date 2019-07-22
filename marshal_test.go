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
	}{42, 0}, "{val:42}")

	test(struct{ v interface{} }{}, "{v:null}")
	test(struct{ v interface{} }{"42"}, "{v:\"42\"}")

	fourtytwo := 42

	test(struct{ v *int }{}, "{v:null}")
	test(struct{ v *int }{&fourtytwo}, "{v:42}")

	test(map[string]int{"b": 2, "a": 1}, "{a:1,b:2}")

	test(struct{ v []int }{}, "{v:null}")
	test(struct{ v []int }{[]int{4, 2}}, "{v:[4,2]}")

	test(struct{ v []byte }{}, "{v:null}")
	test(struct{ v []byte }{[]byte{4, 2}}, "{v:{{BAI=}}}")

	test(struct{ v [2]byte }{[2]byte{4, 2}}, "{v:[4,2]}")
}
