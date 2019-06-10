package ion

import (
	"fmt"
	"math/big"
	"strings"
)

// TODO: Precision.

// Decimal is an arbitrary-precision decimal value.
type Decimal struct {
	n     *big.Int
	scale int
}

// NewDecimal creates a new (big-integer) decimal.
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

// TODO: Maths.

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
