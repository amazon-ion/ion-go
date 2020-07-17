package ion

import (
	"fmt"
	"time"
)

type TimestampPrecision uint8

const (
	NoPrecision TimestampPrecision = iota
	Year
	Month
	Day
	Minute
	Second
)

func (tp TimestampPrecision) String() string {
	switch tp {
	case NoPrecision:
		return "<no precision>"
	case Year:
		return "Year"
	case Month:
		return "Month"
	case Day:
		return "Day"
	case Minute:
		return "Minute"
	case Second:
		return "Second"
	default:
		return fmt.Sprintf("<unknown precision %v>", uint8(tp))
	}
}

type Timestamp struct {
	dateTime  time.Time
	precision TimestampPrecision
}

func emptyTimestamp() Timestamp {
	return Timestamp{time.Time{}, NoPrecision}
}
