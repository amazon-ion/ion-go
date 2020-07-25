package ion

import (
	"fmt"
	"strings"
	"time"
)

// TimestampPrecision is for tracking the precision of a timestamp
type TimestampPrecision uint8

// Possible TimestampPrecision values
const (
	NoPrecision TimestampPrecision = iota
	Year
	Month
	Day
	Minute
	Second
	Nanosecond
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
	case Nanosecond:
		return "Nanosecond"
	default:
		return fmt.Sprintf("<unknown precision %v>", uint8(tp))
	}
}

func (tp TimestampPrecision) formatString(kind TimestampKind, precisionUnits uint8) string {
	switch tp {
	case Year:
		return "2006T"
	case Month:
		return "2006-01T"
	case Day:
		return "2006-01-02T"
	case Minute:
		if kind == Unspecified {
			return "2006-01-02T15:04-00:00"
		}
		return "2006-01-02T15:04Z07:00"
	case Second:
		if kind == Unspecified {
			return "2006-01-02T15:04:05-00:00"
		}
		return "2006-01-02T15:04:05Z07:00"
	case Nanosecond:
		formatStr := "2006-01-02T15:04:05"

		if precisionUnits > 9 {
			precisionUnits = 9
		}

		if precisionUnits > 0 {
			formatStr += "."
			for i := uint8(0); i < precisionUnits; i++ {
				formatStr += "9"
			}
		}

		if kind == Unspecified {
			formatStr += "-00:00"
		} else {
			formatStr += "Z07:00"
		}

		return formatStr
	}

	return time.RFC3339Nano
}

// TimestampKind is for tracking the kind of timestamp
type TimestampKind uint8

// Possible TimestampPrecision values
const (
	Unspecified TimestampKind = iota
	UTC
	Local
)

// Timestamp struct
type Timestamp struct {
	DateTime          time.Time
	precision         TimestampPrecision
	kind              TimestampKind
	fractionPrecision uint8
}

// NewSimpleTimestamp constructor
func NewSimpleTimestamp(dateTime time.Time, precision TimestampPrecision) Timestamp {
	return Timestamp{dateTime, precision, Unspecified, 0}
}

// NewTimestamp constructor
func NewTimestamp(dateTime time.Time, precision TimestampPrecision, kind TimestampKind) Timestamp {
	if precision <= Day {
		// Timestamps with Year, Month, or Day precision necessarily have Unspecified kind
		kind = Unspecified
	}
	return Timestamp{dateTime, precision, kind, 0}
}

// NewTimestampWithFractionalPrecision constructor
func NewTimestampWithFractionalPrecision(dateTime time.Time, precision TimestampPrecision, kind TimestampKind, fractionPrecision uint8) Timestamp {
	if fractionPrecision > 9 {
		// 9 is the max precision supported
		fractionPrecision = 9
	}
	return Timestamp{dateTime, precision, kind, fractionPrecision}
}

// NewTimestampFromStr constructor
func NewTimestampFromStr(dateStr string, precision TimestampPrecision, kind TimestampKind) (Timestamp, error) {
	precisionUnits := uint8(0)

	if precision >= Nanosecond {
		idx := strings.LastIndex(dateStr, ".")
		if idx != -1 {
			idx++
			for idx < len(dateStr) && isDigit(int(dateStr[idx])) {
				precisionUnits++
				idx++
			}
		}
	}

	dateTime, err := time.Parse(precision.formatString(kind, precisionUnits), dateStr)
	if err != nil {
		return Timestamp{time.Time{}, NoPrecision, Unspecified, 0}, err
	}

	return NewTimestampWithFractionalPrecision(dateTime, precision, kind, precisionUnits), nil
}

func emptyTimestamp() Timestamp {
	return Timestamp{time.Time{}, NoPrecision, Unspecified, 0}
}

func tryCreateTimestampWithNSecAndOffset(ts []int, nsecs int, overflow bool, offset int64, sign int64, precision TimestampPrecision, fractionPrecision uint8) (Timestamp, error) {
	date := time.Date(ts[0], time.Month(ts[1]), ts[2], ts[3], ts[4], ts[5], nsecs, time.UTC)
	// time.Date converts 2000-01-32 input to 2000-02-01
	if ts[0] != date.Year() || time.Month(ts[1]) != date.Month() || ts[2] != date.Day() {
		return emptyTimestamp(), fmt.Errorf("ion: invalid timestamp")
	}

	if overflow {
		date = date.Add(time.Second)
	}

	date = date.In(time.FixedZone("fixed", int(offset)*60))

	if precision <= Day {
		return NewSimpleTimestamp(date, precision), nil
	} else if offset == 0 {
		if sign == -1 {
			return NewTimestampWithFractionalPrecision(date, precision, Unspecified, fractionPrecision), nil
		}
		return NewTimestampWithFractionalPrecision(date, precision, UTC, fractionPrecision), nil
	}

	return NewTimestampWithFractionalPrecision(date, precision, Local, fractionPrecision), nil
}

func tryCreateTimestamp(val string, year int64, month int64, day int64, precision TimestampPrecision) (Timestamp, error) {
	date := time.Date(int(year), time.Month(month), int(day), 0, 0, 0, 0, time.UTC)

	// time.Date converts 2000-01-32 input to 2000-02-01
	if int(year) != date.Year() || time.Month(month) != date.Month() || int(day) != date.Day() {
		return invalidTimestamp(val)
	}

	return NewSimpleTimestamp(date, precision), nil
}

func invalidTimestamp(val string) (Timestamp, error) {
	return emptyTimestamp(), fmt.Errorf("ion: invalid timestamp: %v", val)
}

// Format returns a formatted Timestamp string.
func (ts *Timestamp) Format() string {
	format := ts.DateTime.Format(ts.precision.formatString(ts.kind, ts.fractionPrecision))

	if ts.fractionPrecision > 0 && ts.DateTime.Nanosecond() == 0 {
		var index int
		if ts.kind == Unspecified {
			index = strings.LastIndex(format, "-")
		} else {
			index = strings.LastIndex(format, "Z")
		}

		if index != -1 {
			zeros := "."
			for i := uint8(0); i < ts.fractionPrecision; i++ {
				zeros += "0"
			}

			format = format[0:index] + zeros + format[index:]
		}
	}

	return format
}

// Equal figures out if two timestamps are equal for each component.
func (ts *Timestamp) Equal(ts1 Timestamp) bool {
	return ts.DateTime.Equal(ts1.DateTime) &&
		ts.precision == ts1.precision &&
		ts.kind == ts1.kind &&
		ts.fractionPrecision == ts1.fractionPrecision
}

// SetLocation sets the location for the internal time object.
func (ts *Timestamp) SetLocation(loc *time.Location) {
	ts.DateTime = ts.DateTime.In(loc)
}
