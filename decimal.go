package ion

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// TODO: Explicitly track precision?

// Decimal is an arbitrary-precision decimal value.
type Decimal struct {
	n     *big.Int
	scale int
}

// NewDecimal creates a new decimal whose value is equal to the given
// (big) integer.
func NewDecimal(n *big.Int) *Decimal {
	return NewDecimalWithScale(n, 0)
}

// NewDecimalWithScale creates a new scaled decimal whose value is
// equal to n * 10^-scale.
func NewDecimalWithScale(n *big.Int, scale int) *Decimal {
	return &Decimal{
		n:     n,
		scale: scale,
	}
}

// MustParseDecimal parses the given string into a decimal object,
// panicing on error.
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
		return nil, errors.New("empty string")
	}

	shift := 0

	d := strings.IndexAny(in, "Dd")
	if d != -1 {
		// There's an explicit exponent.
		exp := in[d+1:]
		if len(exp) == 0 {
			return nil, errors.New("unexpected end of input after d")
		}

		tmp, err := strconv.ParseInt(exp, 10, 32)
		if err != nil {
			return nil, err
		}

		shift = int(tmp)
		in = in[:d]
	}

	d = strings.Index(in, ".")
	if d != -1 {
		// There's zero or more decimal places.
		ipart := in[:d]
		fpart := in[d+1:]

		shift -= len(fpart)
		in = ipart + fpart
	}

	n, ok := new(big.Int).SetString(in, 10)
	if !ok {
		// Unfortunately this is all we get?
		return nil, errors.New("not a valid number")
	}

	return NewDecimalWithScale(n, -shift), nil
}

func (d *Decimal) Abs() *Decimal {
	return &Decimal{
		n:     new(big.Int).Abs(d.n),
		scale: d.scale,
	}
}

func (d *Decimal) Add(o *Decimal) *Decimal {
	// a*10^x + b*10^y = (a*10^(x-y) + b) * 10^y
	dd, oo := rescale(d, o)
	return &Decimal{
		n:     new(big.Int).Add(dd.n, oo.n),
		scale: dd.scale,
	}
}

func (d *Decimal) Sub(o *Decimal) *Decimal {
	dd, oo := rescale(d, o)
	return &Decimal{
		n:     new(big.Int).Sub(dd.n, oo.n),
		scale: dd.scale,
	}
}

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
		scale: int(scale),
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
		scale: int(scale),
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
		scale: int(scale),
	}
}

// TODO: Div, Exp, etc?

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

var ten = big.NewInt(10)

// Make 'n' bigger by making 'scale' smaller, since we know we can
// do that. (1d100 -> 10d99). Makes comparisons and math easier, at the
// expense of more storage space. Technically speaking implies adding
// more precision, but we're not tracking that too closely.
func (d *Decimal) upscale(scale int) *Decimal {
	diff := int64(scale) - int64(d.scale)
	if diff < 0 {
		panic("can't upscale to a smaller scale")
	}

	pow := new(big.Int).Exp(ten, big.NewInt(diff), nil)
	n := new(big.Int).Mul(d.n, pow)

	return &Decimal{
		n:     n,
		scale: scale,
	}
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
		scale: int(scale),
	}
}

// String formats the decimal as a string in Ion text format.
func (d *Decimal) String() string {
	switch {
	case d.scale == 0:
		// Value is an unscaled integer. Just mark it as a decimal.
		// TODO: If there are enough trailing zeros should we knock them
		// off and do nnn'd'sss here? That'd technically erase precision.
		return d.n.String() + "."

	case d.scale < 0:
		// Value is a upscaled integer, nn'd'ss
		return d.n.String() + "d" + fmt.Sprintf("%d", -d.scale)

	default:
		// Value is a downscaled integer nn.nn('d'-ss)?
		str := d.n.String()
		idx := len(str) - d.scale

		prefix := 1
		if d.n.Sign() < 0 {
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
