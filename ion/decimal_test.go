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
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecimalToString(t *testing.T) {
	test := func(n int64, scale int32, expected string) {
		t.Run(expected, func(t *testing.T) {
			d := Decimal{
				n:     big.NewInt(n),
				scale: scale,
			}
			actual := d.String()
			assert.Equal(t, expected, actual)
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
	test := func(in string, n *big.Int, scale int32) {
		t.Run(in, func(t *testing.T) {
			d, err := ParseDecimal(in)
			require.NoError(t, err)

			assert.True(t, n.Cmp(d.n) == 0, "wrong n; expected %v, got %v", n, d.n)
			assert.Equal(t, scale, d.scale)
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

func absF(d *Decimal) *Decimal { return d.Abs() }
func negF(d *Decimal) *Decimal { return d.Neg() }

type unaryop struct {
	sym string
	fun func(d *Decimal) *Decimal
}

var abs = &unaryop{"abs", absF}
var neg = &unaryop{"neg", negF}

func testUnaryOp(t *testing.T, a, e string, op *unaryop) {
	t.Run(op.sym+"("+a+")="+e, func(t *testing.T) {
		aa, _ := ParseDecimal(a)
		ee, _ := ParseDecimal(e)
		actual := op.fun(aa)
		assert.True(t, actual.Equal(ee), "expected %v, got %v", ee, actual)
	})
}

func TestAbs(t *testing.T) {
	test := func(a, e string) {
		testUnaryOp(t, a, e, abs)
	}

	test("0", "0")
	test("1d100", "1d100")
	test("-1d100", "1d100")
	test("1.2d-3", "1.2d-3")
	test("-1.2d-3", "1.2d-3")
}

func TestNeg(t *testing.T) {
	test := func(a, e string) {
		testUnaryOp(t, a, e, neg)
	}

	test("0", "0")
	test("1d100", "-1d100")
	test("-1d100", "1d100")
	test("1.2d-3", "-1.2d-3")
	test("-1.2d-3", "1.2d-3")
}

func TestTrunc(t *testing.T) {
	test := func(a string, eval int64) {
		t.Run(fmt.Sprintf("trunc(%v)=%v", a, eval), func(t *testing.T) {
			aa := MustParseDecimal(a)
			val, err := aa.trunc()
			require.NoError(t, err)
			assert.Equal(t, eval, val)
		})
	}

	test("0.", 0)
	test("0.01", 0)
	test("1.", 1)
	test("-1.", -1)
	test("1.01", 1)
	test("-1.01", -1)
	test("101", 101)
	test("1d3", 1000)
}

func TestRound(t *testing.T) {
	test := func(a string, eval int64) {
		t.Run(fmt.Sprintf("trunc(%v)=%v", a, eval), func(t *testing.T) {
			aa := MustParseDecimal(a)
			val, err := aa.round()
			require.NoError(t, err)
			assert.Equal(t, eval, val)
		})
	}

	test("0.", 0)
	test("0.01", 0)
	test("1.", 1)
	test("-1.", -1)
	test("1.01", 1)
	test("-1.01", -1)
	test("1.4", 1)
	test("1.5", 2)
	test("1.6", 2)
	test("0.4", 0)
	test("0.5", 1)
	test("0.9999999999", 1)
	test("0.099", 0)
	test("101", 101)
	test("1d3", 1000)
}

func addF(a, b *Decimal) *Decimal { return a.Add(b) }
func subF(a, b *Decimal) *Decimal { return a.Sub(b) }
func mulF(a, b *Decimal) *Decimal { return a.Mul(b) }

type binop struct {
	sym string
	fun func(a, b *Decimal) *Decimal
}

func TestShiftL(t *testing.T) {
	test := func(a string, b int, e string) {
		aa, _ := ParseDecimal(a)
		ee, _ := ParseDecimal(e)
		actual := aa.ShiftL(b)
		assert.True(t, actual.Equal(ee), "expected %v, got %v", ee, actual)
	}

	test("0", 10, "0")
	test("1", 0, "1")
	test("123", 1, "1230")
	test("123", 100, "123d100")
	test("1.23d-100", 102, "123")
}

func TestShiftR(t *testing.T) {
	test := func(a string, b int, e string) {
		aa, _ := ParseDecimal(a)
		ee, _ := ParseDecimal(e)
		actual := aa.ShiftR(b)
		assert.True(t, actual.Equal(ee), "expected %v, got %v", ee, actual)
	}

	test("0", 10, "0")
	test("1", 0, "1")
	test("123", 1, "12.3")
	test("123", 100, "1.23d-98")
	test("1.23d100", 98, "123")
}

var add = &binop{"+", addF}
var sub = &binop{"-", subF}
var mul = &binop{"*", mulF}

func testBinaryOp(t *testing.T, a, b, e string, op *binop) {
	t.Run(a+op.sym+b+"="+e, func(t *testing.T) {
		aa, _ := ParseDecimal(a)
		bb, _ := ParseDecimal(b)
		ee, _ := ParseDecimal(e)

		actual := op.fun(aa, bb)
		assert.True(t, actual.Equal(ee), "expected %v, got %v", ee, actual)
	})
}

func TestAdd(t *testing.T) {
	test := func(a, b, e string) {
		testBinaryOp(t, a, b, e, add)
	}

	test("1", "0", "1")
	test("1", "1", "2")
	test("1", "0.1", "1.1")
	test("0.3", "0.06", "0.36")
	test("1", "100", "101")
	test("1d100", "1d98", "101d98")
	test("1d-100", "1d-98", "1.01d-98")
}

func TestSub(t *testing.T) {
	test := func(a, b, e string) {
		testBinaryOp(t, a, b, e, sub)
	}

	test("1", "0", "1")
	test("1", "1", "0")
	test("1", "0.1", "0.9")
	test("0.3", "0.06", "0.24")
	test("1", "100", "-99")
	test("1d100", "1d98", "99d98")
	test("1d-100", "1d-98", "-99d-100")
}

func TestMul(t *testing.T) {
	test := func(a, b, e string) {
		testBinaryOp(t, a, b, e, mul)
	}

	test("1", "0", "0")
	test("1", "1", "1")
	test("2", "-1", "-2")
	test("7", "6", "42")
	test("10", "0.3", "3")
	test("3d100", "2d50", "6d150")
	test("3d-100", "2d-50", "6d-150")
	test("2d100", "4d-98", "8d2")
}

func TestTruncate(t *testing.T) {
	test := func(a string, p int, expected string) {
		t.Run(fmt.Sprintf("trunc(%v,%v)", a, p), func(t *testing.T) {
			aa := MustParseDecimal(a)
			actual := aa.Truncate(p).String()
			assert.Equal(t, expected, actual)
		})
	}

	test("1", 1, "1.")
	test("1", 10, "1.")
	test("10", 1, "1d1")
	test("1999", 1, "1d3")
	test("1.2345", 3, "1.23")
	test("100d100", 2, "10d101")
	test("1.2345d-100", 2, "1.2d-100")
}

func TestCmp(t *testing.T) {
	test := func(a, b string, expected int) {
		t.Run("("+a+","+b+")", func(t *testing.T) {
			ad, _ := ParseDecimal(a)
			bd, _ := ParseDecimal(b)
			actual := ad.Cmp(bd)
			assert.Equal(t, expected, actual)
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
	assert.Equal(t, "10.0000", actual)
}
