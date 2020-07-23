package ion

import (
	"fmt"
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

func (tp TimestampPrecision) formatString(kind TimestampKind) string {
	switch tp {
	case NoPrecision:
		return ""
	case Year:
		return "2006T"
	case Month:
		return "2006-01T"
	case Day:
		return "2006-01-02T"
	case Minute:
		if kind == Local {
			return "2006-01-02T15:04Z07:00"
		}
		return "2006-01-02T15:04Z"
	case Second:
		if kind == Local {
			return "2006-01-02T15:04:05Z07:00"
		}
		return "2006-01-02T15:04:05Z"
	case Nanosecond:
		if kind == Local {
			return "2006-01-02T15:04:05.999999999Z07:00"
		}
		return "2006-01-02T15:04:05.999999999Z"
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
	DateTime  time.Time
	precision TimestampPrecision
	kind      TimestampKind
}

// NewSimpleTimestamp constructor
func NewSimpleTimestamp(dateTime time.Time, precision TimestampPrecision) Timestamp {
	return Timestamp{dateTime, precision, Unspecified}
}

// NewTimestamp constructor
func NewTimestamp(dateTime time.Time, precision TimestampPrecision, kind TimestampKind) Timestamp {
	if precision <= Day {
		// Timestamps with Year, Month, or Day precision necessarily have Unspecified kind
		return Timestamp{dateTime, precision, Unspecified}
	}
	return Timestamp{dateTime, precision, kind}
}

// NewTimestampFromStr constructor
func NewTimestampFromStr(dateStr string, precision TimestampPrecision, kind TimestampKind) (Timestamp, error) {
	dateTime, err := time.Parse(precision.formatString(kind), dateStr)
	if err != nil {
		return Timestamp{time.Time{}, NoPrecision, Unspecified}, err
	}

	return NewTimestamp(dateTime, precision, kind), nil
}

func emptyTimestamp() Timestamp {
	return Timestamp{time.Time{}, NoPrecision, Unspecified}
}

// Format returns a formatted Timestamp string.
func (ts *Timestamp) Format() string {
	return ts.DateTime.Format(ts.precision.formatString(ts.kind))
}

// Equal figures out if two timestamps are equal for each component.
func (ts *Timestamp) Equal(ts1 Timestamp) bool {
	return ts.DateTime.Equal(ts1.DateTime) &&
		ts.precision == ts1.precision &&
		ts.kind == ts1.kind
}

// SetLocation sets the location for the internal time object.
func (ts *Timestamp) SetLocation(loc *time.Location) {
	ts.DateTime = ts.DateTime.In(loc)
}
