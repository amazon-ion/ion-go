package ion

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

var ten = big.NewInt(10)

// TODO: Precision.

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
	a, b := rescale(d, o)
	return &Decimal{
		n:     new(big.Int).Add(a.n, b.n),
		scale: a.scale,
	}
}

// TODO: Maths.

func (d *Decimal) Cmp(o *Decimal) int {
	a, b := rescale(d, o)
	return a.n.Cmp(b.n)
}

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

func (d *Decimal) String() string {
	switch {
	case d.scale == 0:
		// Value is an unscaled integer.
		return d.n.String() + "."

	case d.scale < 0:
		// Value is a scaled integer, nnndsss.
		return d.n.String() + "d" + fmt.Sprintf("%d", -d.scale)

	default:
		// Value is a downscaled integer nn.nnd-ss
		str := d.n.String()
		idx := len(str) - d.scale

		prefix := 1
		if d.n.Sign() < 0 {
			// Account for leading '-'
			prefix++
		}

		if idx >= prefix {
			// Put the decimal point in the middle.
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
