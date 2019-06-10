package ion

import (
	"math/big"
	"testing"
)

func TestDecimalToString(t *testing.T) {
	test := func(n int64, scale int, expected string) {
		t.Run(expected, func(t *testing.T) {
			d := Decimal{
				n:     big.NewInt(n),
				scale: scale,
			}
			actual := d.String()
			if actual != expected {
				t.Errorf("expected '%v', got '%v'", expected, actual)
			}
		})
	}

	test(0, 0, "0.")
	test(0, -1, "0d1")
	test(0, 1, "0d-1")

	test(1, 0, "1.")
	test(1, -1, "1d1")
	test(1, 1, "1d-1")

	test(-1, 0, "-1.")
	test(-1, -1, "-1d1")
	test(-1, 1, "-1d-1")

	test(123, 0, "123.")
	test(-456, 0, "-456.")

	test(123, -5, "123d5")
	test(-456, -5, "-456d5")

	test(123, 1, "12.3")
	test(123, 2, "1.23")
	test(123, 3, "1.23d-1")
	test(123, 4, "1.23d-2")

	test(-456, 1, "-45.6")
	test(-456, 2, "-4.56")
	test(-456, 3, "-4.56d-1")
	test(-456, 4, "-4.56d-2")
}

func TestParseDecimal(t *testing.T) {
	test := func(in string, n *big.Int, scale int) {
		t.Run(in, func(t *testing.T) {
			d, err := ParseDecimal(in)
			if err != nil {
				t.Fatal(err)
			}

			if n.Cmp(d.n) != 0 {
				t.Errorf("wrong n; expected %v, got %v", n, d.n)
			}
			if scale != d.scale {
				t.Errorf("wrong scale; expected %v, got %v", scale, d.scale)
			}
		})
	}

	test("0", big.NewInt(0), 0)
	test("-0", big.NewInt(0), 0)
	test("0D0", big.NewInt(0), 0)
	test("-0d-1", big.NewInt(0), 1)

	test("1.", big.NewInt(1), 0)
	test("1.0", big.NewInt(10), 1)
	test("0.123", big.NewInt(123), 3)

	test("1d0", big.NewInt(1), 0)
	test("1d1", big.NewInt(1), -1)
	test("1d+1", big.NewInt(1), -1)
	test("1d-1", big.NewInt(1), 1)

	test("-0.12d4", big.NewInt(-12), -2)
}

func TestAbs(t *testing.T) {
	t.Run("0", func(t *testing.T) {
		d := NewDecimal(big.NewInt(0))
		actual := d.Abs().String()
		if actual != "0." {
			t.Errorf("expected 0., got %v", actual)
		}
	})

	t.Run("-1d100", func(t *testing.T) {
		d, _ := ParseDecimal("-1d100")
		actual := d.Abs().String()
		if actual != "1d100" {
			t.Errorf("expected 1d100, got %v", actual)
		}
	})

	t.Run("-1.2d-3", func(t *testing.T) {
		d, _ := ParseDecimal("-1.2d-3")
		actual := d.Abs().String()
		if actual != "1.2d-3" {
			t.Errorf("expected 1.2d-3, got %v", actual)
		}
	})
}

func TestAdd(t *testing.T) {
	test := func(a, b, expected string) {
		t.Run("("+a+"+"+b+")", func(t *testing.T) {
			aa, _ := ParseDecimal(a)
			bb, _ := ParseDecimal(b)
			ee, _ := ParseDecimal(expected)

			actual := aa.Add(bb)
			if !actual.Equal(ee) {
				t.Errorf("expected %v, got %v", ee, actual)
			}
		})
	}

	test("1", "1", "2")
	test("1", "0.1", "1.1")
	test("0.3", "0.06", "0.36")
	test("1", "100", "101")
	test("1d100", "1d98", "101d98")
	test("1d-100", "1d-98", "1.01d-98")
}

func TestCmp(t *testing.T) {
	test := func(a, b string, expected int) {
		t.Run("("+a+","+b+")", func(t *testing.T) {
			ad, _ := ParseDecimal(a)
			bd, _ := ParseDecimal(b)
			actual := ad.Cmp(bd)
			if actual != expected {
				t.Errorf("expected %v, got %v", expected, actual)
			}
		})
	}

	test("0", "0", 0)
	test("0", "1", -1)
	test("0", "-1", 1)

	test("1d2", "100", 0)
	test("100", "1d2", 0)
	test("1d2", "10", 1)
	test("10", "1d2", -1)

	test("0.01", "1d-2", 0)
	test("1d-2", "0.01", 0)
	test("0.01", "1d-3", 1)
	test("1d-3", "0.01", -1)
}

func TestUpscale(t *testing.T) {
	d, _ := ParseDecimal("1d1")
	actual := d.upscale(4).String()
	if actual != "10.0000" {
		t.Errorf("expected 10.0000, got %v", actual)
	}
}
