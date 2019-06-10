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
