/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

package ion

import (
	"bytes"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// This file contains the Timestamp type.

type TimestampPrecision int

const (
	TimestampPrecisionYear TimestampPrecision = iota + 1
	TimestampPrecisionMonth
	TimestampPrecisionDay
	TimestampPrecisionMinute
	TimestampPrecisionSecond
	TimestampPrecisionMillisecond1
	TimestampPrecisionMillisecond2
	TimestampPrecisionMillisecond3
	TimestampPrecisionMillisecond4
	TimestampPrecisionMicrosecond1
	TimestampPrecisionMicrosecond2
	TimestampPrecisionMicrosecond3
	TimestampPrecisionMicrosecond4
)

var timestampPrecisionNameMap = map[TimestampPrecision]string{
	TimestampPrecisionYear: "Year", TimestampPrecisionMonth: "Month", TimestampPrecisionDay: "Day",
	TimestampPrecisionMinute: "Minute", TimestampPrecisionSecond: "Second",
	TimestampPrecisionMillisecond1: "Millisecond Tenths", TimestampPrecisionMillisecond2: "Millisecond Hundredths",
	TimestampPrecisionMillisecond3: "Millisecond Thousandths", TimestampPrecisionMillisecond4: "Millisecond TenThousandths",
	TimestampPrecisionMicrosecond1: "Microsecond Tenths", TimestampPrecisionMicrosecond2: "Microsecond Hundredths",
	TimestampPrecisionMicrosecond3: "Microsecond Thousandths", TimestampPrecisionMicrosecond4: "Microsecond TenThousandths",
}

// String satisfies Stringer.
func (t TimestampPrecision) String() string {
	if s, ok := timestampPrecisionNameMap[t]; ok {
		return s
	}
	return "Unknown"
}

const (
	offsetLocalUnknown = -1
)

/*
	From https://www.w3.org/TR/NOTE-datetime

	Year:
		YYYY (eg 1997)
	Year and month:
		YYYY-MM (eg 1997-07)
	Complete date:
		YYYY-MM-DD (eg 1997-07-16)
	Complete date plus hours and minutes:
		YYYY-MM-DDThh:mmTZD (eg 1997-07-16T19:20+01:00)
	Complete date plus hours, minutes and seconds:
		YYYY-MM-DDThh:mm:ssTZD (eg 1997-07-16T19:20:30+01:00)
	Complete date plus hours, minutes, seconds and a decimal fraction of a second
		YYYY-MM-DDThh:mm:ss.sTZD (eg 1997-07-16T19:20:30.45+01:00)

	where:

	YYYY = four-digit year
	MM   = two-digit month (01=January, etc.)
	DD   = two-digit day of month (01 through 31)
	hh   = two digits of hour (00 through 23) (am/pm NOT allowed)
	mm   = two digits of minute (00 through 59)
	ss   = two digits of second (00 through 59)
	s    = one or more digits representing a decimal fraction of a second
	TZD  = time zone designator (Z or +hh:mm or -hh:mm)
*/
var (
	// precisionFormatMap maps the above valid types to the Go magic time
	// format string "Mon Jan 2 15:04:05 -0700 MST 2006"
	precisionFormatMap = map[TimestampPrecision]string{
		TimestampPrecisionYear:         "2006T",
		TimestampPrecisionMonth:        "2006-01T",
		TimestampPrecisionDay:          "2006-01-02T",
		TimestampPrecisionMinute:       "2006-01-02T15:04",
		TimestampPrecisionSecond:       "2006-01-02T15:04:05",
		TimestampPrecisionMillisecond1: "2006-01-02T15:04:05.0",
		TimestampPrecisionMillisecond2: "2006-01-02T15:04:05.00",
		TimestampPrecisionMillisecond3: "2006-01-02T15:04:05.000",
		TimestampPrecisionMillisecond4: "2006-01-02T15:04:05.0000",
		TimestampPrecisionMicrosecond1: "2006-01-02T15:04:05.00000",
		TimestampPrecisionMicrosecond2: "2006-01-02T15:04:05.000000",
		TimestampPrecisionMicrosecond3: "2006-01-02T15:04:05.0000000",
		TimestampPrecisionMicrosecond4: "2006-01-02T15:04:05.00000000",
	}
)

// Timestamp represents date/time/timezone moments of arbitrary precision.
// Two timestamps are only equivalent if they represent the same instant
// with the same offset and precision.
type Timestamp struct {
	annotations []Symbol
	binary      []byte
	text        []byte
	// offset in minutes.  Use offsetLocalUnknown to denote when an explicit offset
	// is not known.
	offset    time.Duration
	precision TimestampPrecision
	value     time.Time
}

// Precision returns to what precision the timestamp was set to.
func (t Timestamp) Precision() TimestampPrecision {
	if t.precision == 0 && len(t.text) > 0 {
		t.precision = determinePrecision(t.text)
	}
	return t.precision
}

// Value returns the value of the timestamp.
func (t Timestamp) Value() time.Time {
	if t.IsNull() || !t.value.IsZero() {
		return t.value
	}

	if len(t.text) > 0 {
		if t.precision == 0 {
			t.precision = determinePrecision(t.text)
		}

		text := bytes.TrimSuffix(t.text, []byte("Z"))
		// TODO: Handle the unknown local offset case properly.
		// -00:00 is a special offset which means that the offset is local.
		text = bytes.TrimSuffix(text, []byte("-00:00"))

		format := precisionFormatMap[t.precision]
		if bytes.Count(text, []byte("-")) == 3 || bytes.LastIndex(text, []byte("+")) > 0 {
			format += "-07:00"
		}

		// The "T" is optional when the precision is month or day, so add it
		// to the text if it's missing so that the Time parser doesn't fail.
		if (t.precision == TimestampPrecisionMonth || t.precision == TimestampPrecisionDay) && !bytes.HasSuffix(text, []byte("T")) {
			text = append(text, 'T')
		}

		timestamp, err := time.Parse(format, string(text))
		if err != nil {
			panic(errors.Wrap(err, "unable to parse timestamp"))
		}
		t.value = timestamp
	}

	return t.value
}

func determinePrecision(text []byte) TimestampPrecision {
	// There is no real variability in format until we get to dealing
	// with time, so we can handle all of the date-only cases using length.
	switch size := len(text); {
	case size <= 5:
		return TimestampPrecisionYear
	case size <= 8:
		return TimestampPrecisionMonth
	case size <= 11:
		return TimestampPrecisionDay
	}

	// Trim off the date portion.  We only care about time now.
	tim := text[bytes.Index(text, []byte("T"))+1:]

	// Trim off any timezone portion.
	tim = bytes.TrimSuffix(tim, []byte("Z"))
	if plusIndex := bytes.Index(tim, []byte("+")); plusIndex > 0 {
		tim = tim[:plusIndex]
	}
	if minusIndex := bytes.Index(tim, []byte("-")); minusIndex > 0 {
		tim = tim[:minusIndex]
	}

	// Now we can just count characters.
	switch len(tim) {
	case 5:
		return TimestampPrecisionMinute
	case 8:
		return TimestampPrecisionSecond
	case 10:
		return TimestampPrecisionMillisecond1
	case 11:
		return TimestampPrecisionMillisecond2
	case 12:
		return TimestampPrecisionMillisecond3
	case 13:
		return TimestampPrecisionMillisecond4
	case 14:
		return TimestampPrecisionMicrosecond1
	case 15:
		return TimestampPrecisionMicrosecond2
	case 16:
		return TimestampPrecisionMicrosecond3
	case 17:
		return TimestampPrecisionMicrosecond4
	}

	return TimestampPrecisionDay
}

// Annotations satisfies Value.
func (t Timestamp) Annotations() []Symbol {
	return t.annotations
}

// Binary satisfies Value.
func (t Timestamp) Binary() []byte {
	if len(t.binary) > 0 {
		return t.binary
	}

	// TODO
	return t.binary
}

// Text satisfies Value.
func (t Timestamp) Text() []byte {
	if t.IsNull() {
		return []byte(textNullTimestamp)
	}
	if len(t.text) > 0 {
		return t.text
	}

	val := t.Value()
	t.text = []byte(val.Format(precisionFormatMap[t.precision]))

	// If the precision doesn't include time, then we don't
	// need to generate an offset.
	if t.precision <= TimestampPrecisionDay {
		return t.text
	}

	var offset []byte
	switch t.offset {
	case offsetLocalUnknown:
		// Do nothing.
	case 0:
		offset = append(offset, 'Z')
	default:
		if t.offset > 0 {
			offset = append(offset, '+')
		} else {
			offset = append(offset, '-')
		}
		hh := int(t.offset.Hours())
		mm := int(t.offset.Minutes())
		offset = append(offset, []byte(fmt.Sprintf("%02d:%02d", hh, mm))...)
	}
	t.text = append(t.text, offset...)

	return t.text
}

// IsNull satisfies Value.
func (t Timestamp) IsNull() bool {
	return t.binary == nil && t.text == nil
}

// Type satisfies Value.
func (t Timestamp) Type() Type {
	return TypeTimestamp
}
