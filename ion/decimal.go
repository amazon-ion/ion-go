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
	"math"
	"math/big"
	"strconv"
	"strings"
)

// A ParseError is returned if ParseDecimal is called with a parameter that
// cannot be parsed as a Decimal.
type ParseError struct {
	Num string
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("ion: ParseDecimal(%v): %v", e.Num, e.Msg)
}

// https://github.com/amazon-ion/ion-go/issues/119

// Decimal is an arbitrary-precision decimal value.
type Decimal struct {
	n         *big.Int
	scale     int32
	isNegZero bool
}

// NewDecimal creates a new decimal whose value is equal to n * 10^exp.
func NewDecimal(n *big.Int, exp int32, negZero bool) *Decimal {
	return &Decimal{
		n:         n,
		scale:     -exp,
		isNegZero: negZero,
	}
}

// NewDecimalInt creates a new decimal whose value is equal to n.
func NewDecimalInt(n int64) *Decimal {
	return NewDecimal(big.NewInt(n), 0, false)
}

// MustParseDecimal parses the given string into a decimal object,
// panicking on error.
func MustParseDecimal(in string) *Decimal {
	d, err := ParseDecimal(in)
	if err != nil {
		panic(err)
	}
	return d
}

// ParseDecimal parses the given string into a decimal object,
// returning an error on failure.
func ParseDecimal(in string) (*Decimal, error) {
	if len(in) == 0 {
		return nil, &ParseError{in, "empty string"}
	}

	exponent := int32(0)

	d := strings.IndexAny(in, "Dd")
	if d != -1 {
		// There's an explicit exponent.
		exp := in[d+1:]
		if len(exp) == 0 {
			return nil, &ParseError{in, "unexpected end of input after d"}
		}

		tmp, err := strconv.ParseInt(exp, 10, 32)
		if err != nil {
			return nil, &ParseError{in, err.Error()}
		}

		exponent = int32(tmp)
		in = in[:d]
	}

	d = strings.Index(in, ".")
	if d != -1 {
		// There's zero or more decimal places.
		ipart := in[:d]
		fpart := in[d+1:]

		exponent -= int32(len(fpart))
		in = ipart + fpart
	}

	n, ok := new(big.Int).SetString(in, 10)
	if !ok {
		// Unfortunately this is all we get?
		return nil, &ParseError{in, "cannot parse coefficient"}
	}

	isNegZero := n.Sign() == 0 && len(in) > 0 && in[0] == '-'

	return NewDecimal(n, exponent, isNegZero), nil
}

// CoEx returns this decimal's coefficient and exponent.
func (d *Decimal) CoEx() (*big.Int, int32) {
	return d.n, -d.scale
}

// Abs returns the absolute value of this Decimal.
func (d *Decimal) Abs() *Decimal {
	return &Decimal{
		n:     new(big.Int).Abs(d.n),
		scale: d.scale,
	}
}

// Add returns the result of adding this Decimal to another Decimal.
func (d *Decimal) Add(o *Decimal) *Decimal {
	// a*10^x + b*10^y = (a*10^(x-y) + b) * 10^y
	dd, oo := rescale(d, o)
	return &Decimal{
		n:     new(big.Int).Add(dd.n, oo.n),
		scale: dd.scale,
	}
}

// Sub returns the result of substrating another Decimal from this Decimal.
func (d *Decimal) Sub(o *Decimal) *Decimal {
	dd, oo := rescale(d, o)
	return &Decimal{
		n:     new(big.Int).Sub(dd.n, oo.n),
		scale: dd.scale,
	}
}

// Neg returns the negative of this Decimal.
func (d *Decimal) Neg() *Decimal {
	return &Decimal{
		n:     new(big.Int).Neg(d.n),
		scale: d.scale,
	}
}

// Mul multiplies two decimals and returns the result.
func (d *Decimal) Mul(o *Decimal) *Decimal {
	// a*10^x * b*10^y = (a*b) * 10^(x+y)
	scale := int64(d.scale) + int64(o.scale)
	if scale > math.MaxInt32 || scale < math.MinInt32 {
		panic("exponent out of bounds")
	}

	return &Decimal{
		n:     new(big.Int).Mul(d.n, o.n),
		scale: int32(scale),
	}
}

// ShiftL returns a new decimal shifted the given number of decimal
// places to the left. It's a computationally-cheap way to compute
// d * 10^shift.
func (d *Decimal) ShiftL(shift int) *Decimal {
	scale := int64(d.scale) - int64(shift)
	if scale > math.MaxInt32 || scale < math.MinInt32 {
		panic("exponent out of bounds")
	}

	return &Decimal{
		n:     d.n,
		scale: int32(scale),
	}
}

// ShiftR returns a new decimal shifted the given number of decimal
// places to the right. It's a computationally-cheap way to compute
// d / 10^shift.
func (d *Decimal) ShiftR(shift int) *Decimal {
	scale := int64(d.scale) + int64(shift)
	if scale > math.MaxInt32 || scale < math.MinInt32 {
		panic("exponent out of bounds")
	}

	return &Decimal{
		n:     d.n,
		scale: int32(scale),
	}
}

// https://github.com/amazon-ion/ion-go/issues/118

// Sign returns -1 if the value is less than 0, 0 if it is equal to zero,
// and +1 if it is greater than zero.
func (d *Decimal) Sign() int {
	return d.n.Sign()
}

// Cmp compares two decimals, returning -1 if d is smaller, +1 if d is
// larger, and 0 if they are equal (ignoring precision).
func (d *Decimal) Cmp(o *Decimal) int {
	dd, oo := rescale(d, o)
	return dd.n.Cmp(oo.n)
}

// Equal determines if two decimals are equal (discounting precision,
// at least for now).
func (d *Decimal) Equal(o *Decimal) bool {
	return d.Cmp(o) == 0
}

func rescale(a, b *Decimal) (*Decimal, *Decimal) {
	if a.scale < b.scale {
		return a.upscale(b.scale), b
	} else if a.scale > b.scale {
		return a, b.upscale(a.scale)
	} else {
		return a, b
	}
}

// Make 'n' bigger by making 'scale' smaller, since we know we can
// do that. (1d100 -> 10d99). Makes comparisons and math easier, at the
// expense of more storage space. Technically speaking implies adding
// more precision, but we're not tracking that too closely.
func (d *Decimal) upscale(scale int32) *Decimal {
	diff := int64(scale) - int64(d.scale)
	if diff < 0 {
		panic("can't upscale to a smaller scale")
	}

	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(diff), nil)
	n := new(big.Int).Mul(d.n, pow)

	return &Decimal{
		n:     n,
		scale: scale,
	}
}

// Check to upscale a decimal which means to make 'n' bigger by making 'scale' smaller.
// Makes comparisons and math easier, at the expense of more storage space.
func (d *Decimal) checkToUpscale() (*Decimal, error) {
	if d.scale < 0 {
		// Don't even bother trying this with numbers that *definitely* too big to represent
		// as an int64, because upscale(0) will consume a bunch of memory.
		if d.scale < -20 {
			return d, &strconv.NumError{
				Func: "ParseInt",
				Num:  d.String(),
				Err:  strconv.ErrRange,
			}
		}
		return d.upscale(0), nil
	}
	return d, nil
}

// Trunc attempts to truncate this decimal to an int64, dropping any fractional bits.
func (d *Decimal) trunc() (int64, error) {
	ud, err := d.checkToUpscale()
	if err != nil {
		return 0, err
	}
	str := ud.n.String()

	truncateTo := len(str) - int(ud.scale)
	if truncateTo <= 0 {
		return 0, nil
	}

	return strconv.ParseInt(str[:truncateTo], 10, 64)
}

// Round attempts to truncate this decimal to an int64, rounding any fractional bits.
func (d *Decimal) round() (int64, error) {
	ud, err := d.checkToUpscale()
	if err != nil {
		return 0, err
	}

	floatValue := float64(ud.n.Int64()) / math.Pow10(int(ud.scale))
	roundedValue := math.Round(floatValue)
	return int64(roundedValue), nil
}

// Truncate returns a new decimal, truncated to the given number of
// decimal digits of precision. It does not round, so 19.Truncate(1)
// = 1d1.
func (d *Decimal) Truncate(precision int) *Decimal {
	if precision <= 0 {
		panic("precision must be positive")
	}

	// Is there a better way to calculate precision? It really
	// seems like there should be...

	str := d.n.String()
	if str[0] == '-' {
		// Cheating a bit.
		precision++
	}

	diff := len(str) - precision
	if diff <= 0 {
		// Already small enough, nothing to truncate.
		return d
	}

	// Lazy man's division by a power of 10.
	n, ok := new(big.Int).SetString(str[:precision], 10)
	if !ok {
		// Should never happen, since we started with a valid int.
		panic("failed to parse integer")
	}

	scale := int64(d.scale) - int64(diff)
	if scale < math.MinInt32 {
		panic("exponent out of range")
	}

	return &Decimal{
		n:     n,
		scale: int32(scale),
	}
}

// String formats the decimal as a string in Ion text format.
func (d *Decimal) String() string {
	switch {
	case d.scale == 0:
		// Value is an unscaled integer. Just mark it as a decimal.
		if d.isNegZero {
			return "-0."
		}
		return d.n.String() + "."

	case d.scale < 0:
		// Value is a upscaled integer, nn'd'ss
		if d.isNegZero {
			return "-0d" + fmt.Sprintf("%d", -d.scale)
		}
		return d.n.String() + "d" + fmt.Sprintf("%d", -d.scale)

	default:
		// Value is a downscaled integer nn.nn('d'-ss)?
		var str string
		if d.isNegZero {
			str = "-0"
		} else {
			str = d.n.String()
		}

		idx := len(str) - int(d.scale)

		prefix := 1
		if len(str) > 0 && str[0] == '-' {
			// Account for leading '-'.
			prefix++
		}

		if idx >= prefix {
			// Put the decimal point in the middle, no exponent.
			return str[:idx] + "." + str[idx:]
		}

		// Put the decimal point at the beginning and
		// add a (negative) exponent.
		b := strings.Builder{}
		b.WriteString(str[:prefix])

		if len(str) > prefix {
			b.WriteString(".")
			b.WriteString(str[prefix:])
		}

		b.WriteString("d")
		b.WriteString(fmt.Sprintf("%d", idx-prefix))

		return b.String()
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Decimal) UnmarshalJSON(decimalBytes []byte) error {
	str := string(decimalBytes)
	if str == "null" {
		return nil
	}
	str = strings.Replace(str, "E", "D", 1)
	str = strings.Replace(str, "e", "d", 1)
	parsed, err := ParseDecimal(str)
	if err != nil {
		return fmt.Errorf("error unmarshalling decimal '%s': %w", str, err)
	}
	*d = *parsed
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (d *Decimal) MarshalJSON() ([]byte, error) {
	absN := new(big.Int).Abs(d.n).String()
	scale := int(-d.scale)
	sign := d.n.Sign()

	var str string
	if scale == 0 {
		str = absN
	} else if scale > 0 {
		// add zeroes to the right
		str = absN + strings.Repeat("0", scale)
	} else {
		// add zeroes to the left
		absScale := -scale
		nLen := len(absN)

		if absScale >= nLen {
			str = "0." + strings.Repeat("0", absScale-nLen) + absN
		} else {
			str = absN[:nLen-absScale] + "." + absN[nLen-absScale:]
		}
		str = strings.TrimRight(str, "0")
		str = strings.TrimSuffix(str, ".")
	}

	if sign == -1 {
		str = "-" + str
	}
	return []byte(str), nil
}
