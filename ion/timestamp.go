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

func (tp TimestampPrecision) formatString(kind TimezoneKind, precisionUnits uint8) string {
	switch tp {
	case Year:
		return "2006T"
	case Month:
		return "2006-01T"
	case Day:
		return "2006-01-02T"
	case Minute:
		if kind == Unspecified {
			return "2006-01-02T15:04-07:00"
		}
		return "2006-01-02T15:04Z07:00"
	case Second:
		if kind == Unspecified {
			return "2006-01-02T15:04:05-07:00"
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
			formatStr += "-07:00"
		} else {
			formatStr += "Z07:00"
		}

		return formatStr
	}

	return time.RFC3339Nano
}

// TimezoneKind tracks the type of timezone.
type TimezoneKind uint8

const (
	// Unspecified is for dates without a timezone such as imprecise dates with only year, month, or day precision.
	// Also dates with a negative zero offset (ie. yyyy-mm-ddThh:mm-00:00) are Unspecified.
	Unspecified TimezoneKind = iota

	// UTC is for UTC dates and they are usually denoted with a trailing 'Z' (ie. yyyy-mm-ddThh:mmZ).
	// Dates with a positive zero offset (ie. yyyy-mm-ddThh:mm+00:00) are also considered UTC.
	UTC

	// Local is for dates that have a non-zero offset from UTC (ie. 2001-02-03T04:05+08:30, 2009-05-18T16:20-04:00)
	Local
)

// Timestamp struct
type Timestamp struct {
	DateTime             time.Time
	precision            TimestampPrecision
	kind                 TimezoneKind
	numFractionalSeconds uint8
}

// NewSimpleTimestamp constructor
func NewSimpleTimestamp(dateTime time.Time, precision TimestampPrecision) Timestamp {
	return Timestamp{dateTime, precision, Unspecified, 0}
}

// NewTimestamp constructor
func NewTimestamp(dateTime time.Time, precision TimestampPrecision, kind TimezoneKind) Timestamp {
	if precision <= Day {
		// Timestamps with Year, Month, or Day precision necessarily have Unspecified timezone.
		kind = Unspecified
	}
	return Timestamp{dateTime, precision, kind, 0}
}

// NewTimestampWithFractionalSeconds constructor
func NewTimestampWithFractionalSeconds(dateTime time.Time, precision TimestampPrecision, kind TimezoneKind, fractionPrecision uint8) Timestamp {
	if fractionPrecision > 9 {
		// 9 is the max precision supported
		fractionPrecision = 9
	}
	return Timestamp{dateTime, precision, kind, fractionPrecision}
}

// NewTimestampFromStr constructor
func NewTimestampFromStr(dateStr string, precision TimestampPrecision, kind TimezoneKind) (Timestamp, error) {
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

	return NewTimestampWithFractionalSeconds(dateTime, precision, kind, precisionUnits), nil
}

func emptyTimestamp() Timestamp {
	return Timestamp{time.Time{}, NoPrecision, Unspecified, 0}
}

func tryCreateTimestampWithNSecAndOffset(ts []int, nsecs int, overflow bool, offset, sign int64, precision TimestampPrecision, fractionPrecision uint8) (Timestamp, error) {
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
			// Negative zero timezone offset is Unspecified
			return NewTimestampWithFractionalSeconds(date, precision, Unspecified, fractionPrecision), nil
		}
		// Positive zero timezone offset is UTC
		return NewTimestampWithFractionalSeconds(date, precision, UTC, fractionPrecision), nil
	}

	// Non-zero offset is Local
	return NewTimestampWithFractionalSeconds(date, precision, Local, fractionPrecision), nil
}

func tryCreateTimestamp(year, month, day int, precision TimestampPrecision) (Timestamp, error) {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// time.Date converts 2000-01-32 input to 2000-02-01
	if year != date.Year() || time.Month(month) != date.Month() || day != date.Day() {
		return emptyTimestamp(), fmt.Errorf("ion: invalid timestamp")
	}

	return NewSimpleTimestamp(date, precision), nil
}

func invalidTimestamp(val string) (Timestamp, error) {
	return emptyTimestamp(), fmt.Errorf("ion: invalid timestamp: %v", val)
}

// Format returns a formatted Timestamp string.
func (ts *Timestamp) Format() string {
	format := ts.DateTime.Format(ts.precision.formatString(ts.kind, ts.numFractionalSeconds))

	// The above time.Format() removes trailing zeros from fractional seconds (ie. ".000")
	// so we're adding them back if necessary.
	if ts.precision >= Minute && ts.numFractionalSeconds > 0 && ts.DateTime.Nanosecond() == 0 {
		// Find the position of 'T'
		tIndex := strings.Index(format, "T")
		if tIndex == -1 {
			tIndex = strings.Index(format, "t")
			if tIndex == -1 {
				return format
			}
		}

		index := strings.LastIndex(format, "Z")
		if index == -1 || index < tIndex {
			index = strings.LastIndex(format, "+")
			if index == -1 || index < tIndex {
				index = strings.LastIndex(format, "-")
			}
		}

		// This position better be right of 'T'
		if index != -1 && tIndex < index {
			zeros := "."
			for i := uint8(0); i < ts.numFractionalSeconds; i++ {
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
		ts.numFractionalSeconds == ts1.numFractionalSeconds
}

// Equivalent figures out if two timestamps have equal dateTime and precision.
func (ts *Timestamp) Equivalent(ts1 Timestamp) bool {
	return ts.DateTime.Equal(ts1.DateTime) && ts.precision == ts1.precision
}

// SetLocation sets the location for the internal time object.
func (ts *Timestamp) SetLocation(loc *time.Location) {
	ts.DateTime = ts.DateTime.In(loc)
}

// TruncateNS returns nanoseconds with 0's removed and the fractional precision indicator.
func (ts *Timestamp) TruncateNS() (int, uint8) {
	nsecs := ts.DateTime.Nanosecond()
	for i := uint8(0); i < (9-ts.numFractionalSeconds) && nsecs > 0 && (nsecs%10) == 0; i++ {
		nsecs /= 10
	}

	return nsecs, ts.numFractionalSeconds | 0xC0
}
