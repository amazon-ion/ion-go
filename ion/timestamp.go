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
	"strconv"
	"strings"
	"time"
)

// TimestampPrecision is for tracking the precision of a timestamp
type TimestampPrecision uint8

// Possible TimestampPrecision values
const (
	TimestampNoPrecision TimestampPrecision = iota
	TimestampPrecisionYear
	TimestampPrecisionMonth
	TimestampPrecisionDay
	TimestampPrecisionMinute
	TimestampPrecisionSecond
	TimestampPrecisionNanosecond
)

const maxFractionalPrecision = 9

func (tp TimestampPrecision) String() string {
	switch tp {
	case TimestampNoPrecision:
		return "<no precision>"
	case TimestampPrecisionYear:
		return "Year"
	case TimestampPrecisionMonth:
		return "Month"
	case TimestampPrecisionDay:
		return "Day"
	case TimestampPrecisionMinute:
		return "Minute"
	case TimestampPrecisionSecond:
		return "Second"
	case TimestampPrecisionNanosecond:
		return "Nanosecond"
	default:
		return fmt.Sprintf("<unknown precision %v>", uint8(tp))
	}
}

// Layout returns a suitable format string to be used in time.Parse() or time.Format().
// The idea of the format string is to format the particular date Mon Jan 2 15:04:05 MST 2006 (Unix time 1136239445)
// in the desired layout we want to use to format other dates.
func (tp TimestampPrecision) Layout(kind TimezoneKind, precisionUnits uint8) string {
	switch tp {
	case TimestampPrecisionYear:
		return "2006T"
	case TimestampPrecisionMonth:
		return "2006-01T"
	case TimestampPrecisionDay:
		return "2006-01-02T"
	case TimestampPrecisionMinute:
		if kind == TimezoneUnspecified {
			return "2006-01-02T15:04-07:00"
		}
		return "2006-01-02T15:04Z07:00"
	case TimestampPrecisionSecond:
		if kind == TimezoneUnspecified {
			return "2006-01-02T15:04:05-07:00"
		}
		return "2006-01-02T15:04:05Z07:00"
	case TimestampPrecisionNanosecond:
		layout := strings.Builder{}
		layout.WriteString("2006-01-02T15:04:05")

		if precisionUnits > 9 {
			precisionUnits = 9
		}

		if precisionUnits > 0 {
			layout.WriteByte('.')
			for i := uint8(0); i < precisionUnits; i++ {
				layout.WriteByte('9')
			}
		}

		if kind == TimezoneUnspecified {
			layout.WriteString("-07:00")
		} else {
			layout.WriteString("Z07:00")
		}

		return layout.String()
	}

	return time.RFC3339Nano
}

// TimezoneKind tracks the type of timezone.
type TimezoneKind uint8

const (
	// TimezoneUnspecified is for timestamps without a timezone such as dates with no time component (ie. Year/Month/Day precision).
	// Negative zero offsets (ie. yyyy-mm-ddThh:mm-00:00) are also considered Unspecified.
	TimezoneUnspecified TimezoneKind = iota

	// TimezoneUTC is for UTC timestamps and they are usually denoted with a trailing 'Z' (ie. yyyy-mm-ddThh:mmZ).
	// Timestamps with a positive zero offset (ie. yyyy-mm-ddThh:mm+00:00) are also considered UTC.
	TimezoneUTC

	// TimezoneLocal is for timestamps that have a non-zero offset from UTC (ie. 2001-02-03T04:05+08:30, 2009-05-18T16:20-04:00).
	TimezoneLocal
)

// Timestamp struct
type Timestamp struct {
	dateTime             time.Time
	precision            TimestampPrecision
	kind                 TimezoneKind
	numFractionalSeconds uint8
}

// NewDateTimestamp constructor meant for timestamps that only have a date portion (ie. no time portion).
func NewDateTimestamp(dateTime time.Time, precision TimestampPrecision) Timestamp {
	numDecimalPlacesOfFractionalSeconds := uint8(0)
	if precision >= TimestampPrecisionNanosecond {
		numDecimalPlacesOfFractionalSeconds = maxFractionalPrecision
	}
	return Timestamp{dateTime, precision, TimezoneUnspecified, numDecimalPlacesOfFractionalSeconds}
}

// NewTimestamp constructor
func NewTimestamp(dateTime time.Time, precision TimestampPrecision, kind TimezoneKind) Timestamp {
	numDecimalPlacesOfFractionalSeconds := uint8(0)

	if precision <= TimestampPrecisionDay {
		// Timestamps with Year, Month, or Day precision necessarily have TimezoneUnspecified timezone.
		kind = TimezoneUnspecified
	} else if precision >= TimestampPrecisionNanosecond {
		numDecimalPlacesOfFractionalSeconds = maxFractionalPrecision
	}
	return Timestamp{dateTime, precision, kind, numDecimalPlacesOfFractionalSeconds}
}

// NewTimestampWithFractionalSeconds constructor
func NewTimestampWithFractionalSeconds(dateTime time.Time, precision TimestampPrecision, kind TimezoneKind, fractionPrecision uint8) Timestamp {
	if fractionPrecision > maxFractionalPrecision {
		// 9 is the max precision supported
		fractionPrecision = maxFractionalPrecision
	}
	if precision < TimestampPrecisionNanosecond {
		fractionPrecision = 0
	}
	return Timestamp{dateTime, precision, kind, fractionPrecision}
}

// NewTimestampFromStr constructor
func NewTimestampFromStr(dateStr string, precision TimestampPrecision, kind TimezoneKind) (Timestamp, error) {
	// Count number of fractional seconds units.
	fractionUnits := uint8(0)
	if precision >= TimestampPrecisionNanosecond {
		pointIdx := strings.LastIndex(dateStr, ".")
		if pointIdx != -1 {
			idx := pointIdx + 1
			for idx < len(dateStr) && isDigit(int(dateStr[idx])) {
				fractionUnits++
				idx++
			}

			if idx == len(dateStr) {
				return Timestamp{}, fmt.Errorf("ion: invalid date string '%v'", dateStr)
			}
		}
	}

	dateTime, err := time.Parse(precision.Layout(kind, fractionUnits), dateStr)
	if err != nil {
		return Timestamp{}, err
	}

	return NewTimestampWithFractionalSeconds(dateTime, precision, kind, fractionUnits), nil
}

func invalidTimestamp(val string) (Timestamp, error) {
	return Timestamp{}, fmt.Errorf("ion: invalid timestamp: %v", val)
}

func tryCreateDateTimestamp(year, month, day int, precision TimestampPrecision) (Timestamp, error) {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// time.Date converts 2000-01-32 input to 2000-02-01
	if year != date.Year() || time.Month(month) != date.Month() || day != date.Day() {
		return Timestamp{}, fmt.Errorf("ion: invalid timestamp")
	}

	return NewDateTimestamp(date, precision), nil
}

func tryCreateTimestamp(ts []int, nsecs int, overflow bool, offset, sign int64, precision TimestampPrecision, fractionPrecision uint8) (Timestamp, error) {
	date := time.Date(ts[0], time.Month(ts[1]), ts[2], ts[3], ts[4], ts[5], nsecs, time.UTC)
	// time.Date converts 2000-01-32 input to 2000-02-01
	if ts[0] != date.Year() || time.Month(ts[1]) != date.Month() || ts[2] != date.Day() {
		return Timestamp{}, fmt.Errorf("ion: invalid timestamp")
	}

	if precision <= TimestampPrecisionDay {
		return NewDateTimestamp(date, precision), nil
	}

	if overflow {
		date = date.Add(time.Second)
	}

	if offset == 0 {
		if sign == -1 {
			// Negative zero timezone offset is Unspecified
			return NewTimestampWithFractionalSeconds(date, precision, TimezoneUnspecified, fractionPrecision), nil
		}

		// Positive zero timezone offset is UTC
		return NewTimestampWithFractionalSeconds(date, precision, TimezoneUTC, fractionPrecision), nil
	}

	date = date.In(time.FixedZone("fixed", int(offset)*60))

	// Non-zero offset is Local
	return NewTimestampWithFractionalSeconds(date, precision, TimezoneLocal, fractionPrecision), nil
}

// MustParseTimestamp parses the given string into an ion timestamp object,
// panicking on error.
func MustParseTimestamp(dateStr string) Timestamp {
	ts, err := ParseTimestamp(dateStr)
	if err != nil {
		panic(err)
	}
	return ts
}

// ParseTimestamp parses a timestamp string and returns an ion timestamp.
func ParseTimestamp(dateStr string) (Timestamp, error) {
	if len(dateStr) < 5 {
		return invalidTimestamp(dateStr)
	}

	year, err := strconv.ParseInt(dateStr[:4], 10, 32)
	if err != nil || year < 1 {
		return invalidTimestamp(dateStr)
	}

	if len(dateStr) == 5 && (dateStr[4] == 't' || dateStr[4] == 'T') {
		// yyyyT
		return tryCreateDateTimestamp(int(year), 1, 1, TimestampPrecisionYear)
	}

	if dateStr[4] != '-' {
		return invalidTimestamp(dateStr)
	}

	if len(dateStr) < 8 {
		return invalidTimestamp(dateStr)
	}

	month, err := strconv.ParseInt(dateStr[5:7], 10, 32)
	if err != nil {
		return invalidTimestamp(dateStr)
	}

	if len(dateStr) == 8 && (dateStr[7] == 't' || dateStr[7] == 'T') {
		// yyyy-mmT
		return tryCreateDateTimestamp(int(year), int(month), 1, TimestampPrecisionMonth)
	}

	if dateStr[7] != '-' {
		return invalidTimestamp(dateStr)
	}

	if len(dateStr) < 10 {
		return invalidTimestamp(dateStr)
	}

	day, err := strconv.ParseInt(dateStr[8:10], 10, 32)
	if err != nil {
		return invalidTimestamp(dateStr)
	}

	if len(dateStr) == 10 || (len(dateStr) == 11 && (dateStr[10] == 't' || dateStr[10] == 'T')) {
		// yyyy-mm-dd or yyyy-mm-ddT
		return tryCreateDateTimestamp(int(year), int(month), int(day), TimestampPrecisionDay)
	}

	if dateStr[10] != 't' && dateStr[10] != 'T' {
		return invalidTimestamp(dateStr)
	}

	// At this point timestamp must have hour:minute
	if len(dateStr) < 17 {
		return invalidTimestamp(dateStr)
	}

	switch dateStr[16] {
	case 'z', 'Z', '+', '-':
		// yyyy-mm-ddThh:mm
		kind, err := computeTimezoneKind(dateStr, 16)
		if err != nil {
			return Timestamp{}, err
		}

		return NewTimestampFromStr(dateStr, TimestampPrecisionMinute, kind)
	case ':':
		// yyyy-mm-ddThh:mm:ss
		if len(dateStr) < 20 {
			break
		}

		idx := 19
		if dateStr[idx] == '.' {
			idx++
			for idx < len(dateStr) && isDigit(int(dateStr[idx])) {
				idx++
			}
		}

		kind, err := computeTimezoneKind(dateStr, idx)
		if err != nil {
			return Timestamp{}, err
		}

		if idx <= 20 {
			return NewTimestampFromStr(dateStr, TimestampPrecisionSecond, kind)
		} else if idx <= 28 {
			return NewTimestampFromStr(dateStr, TimestampPrecisionNanosecond, kind)
		}

		// Greater than 9 fractional seconds.
		return roundFractionalSeconds(dateStr, idx, kind)
	}

	return invalidTimestamp(dateStr)
}

func computeOffset(val string, idx int) (int64, int64, error) {
	// +hh:mm
	if idx+5 > len(val) || val[idx+3] != ':' {
		return 0, 0, fmt.Errorf("ion: invalid offset: '%v'", val)
	}

	hourOffset, err := strconv.ParseInt(val[idx+1:idx+3], 10, 32)
	if err != nil {
		return 0, 0, err
	}

	minuteOffset, err := strconv.ParseInt(val[idx+4:], 10, 32)
	if err != nil {
		return 0, 0, err
	}

	return hourOffset, minuteOffset, nil
}

func computeTimezoneKind(val string, idx int) (TimezoneKind, error) {
	switch val[idx] {
	case 'z', 'Z':
		// 'Z' zulu time means UTC timezone.
		return TimezoneUTC, nil
	case '+', '-':
		hourOffset, minuteOffset, err := computeOffset(val, idx)
		if err != nil {
			return TimezoneUnspecified, err
		}

		if hourOffset >= 24 || minuteOffset >= 60 {
			return TimezoneUnspecified, fmt.Errorf("ion: invalid offset %v:%v", hourOffset, minuteOffset)
		} else if hourOffset == 0 && minuteOffset == 0 {
			// Negative zero offset is Unspecified timezone.
			if val[idx] == '-' {
				return TimezoneUnspecified, nil
			}

			// Positive zero offset is UTC.
			return TimezoneUTC, nil
		}

		// Valid non-zero offset is Local timezone.
		return TimezoneLocal, nil
	}

	return TimezoneUnspecified, fmt.Errorf("ion: invalid character: '%v' at position %v in %v", val[idx], idx, val)
}

func roundFractionalSeconds(val string, idx int, kind TimezoneKind) (Timestamp, error) {
	// Convert to float to perform rounding.
	floatValue, err := strconv.ParseFloat(val[18:idx], 64)
	if err != nil {
		return invalidTimestamp(val)
	}

	roundedStringValue := fmt.Sprintf("%.9f", floatValue)
	roundedFloatValue, err := strconv.ParseFloat(roundedStringValue, 64)
	if err != nil {
		return invalidTimestamp(val)
	}

	// Microsecond overflow 9.9999999999 -> 10.00000000.
	if roundedFloatValue == 10 {
		roundedStringValue := "9.000000000"
		val = val[:18] + roundedStringValue + val[idx:]
		timeValue, err := time.Parse(TimestampPrecisionNanosecond.Layout(kind, 9), val)
		if err != nil {
			return invalidTimestamp(val)
		}

		timeValue = timeValue.Add(time.Second)
		return NewTimestampWithFractionalSeconds(timeValue, TimestampPrecisionNanosecond, kind, 9), err
	}

	val = val[:18] + roundedStringValue + val[idx:]

	return NewTimestampFromStr(val, TimestampPrecisionNanosecond, kind)
}

// GetDateTime returns the timestamps date time.
func (ts Timestamp) GetDateTime() time.Time {
	return ts.dateTime
}

// GetPrecision returns the timestamp's precision.
func (ts Timestamp) GetPrecision() TimestampPrecision {
	return ts.precision
}

// GetTimezoneKind returns the kind of timezone.
func (ts Timestamp) GetTimezoneKind() TimezoneKind {
	return ts.kind
}

// GetNumberOfFractionalSeconds returns the number of precision units in the timestamp's fractional seconds.
func (ts Timestamp) GetNumberOfFractionalSeconds() uint8 {
	return ts.numFractionalSeconds
}

// String returns a formatted Timestamp string.
func (ts Timestamp) String() string {
	layout := ts.precision.Layout(ts.kind, ts.numFractionalSeconds)
	format := ts.dateTime.Format(layout)

	// The above time.Format() does not produce the format we want in some scenarios.
	// So we may need to make some adjustments.

	// Add back removed trailing zeros from fractional seconds (ie. ".000")
	if ts.precision >= TimestampPrecisionNanosecond && ts.numFractionalSeconds > 0 {
		// Find the position of 'T'
		tIndex := strings.Index(format, "T")
		if tIndex == -1 {
			tIndex = strings.Index(format, "t")
			if tIndex == -1 {
				return format
			}
		}

		timeZoneIndex := strings.LastIndex(format, "Z")
		if timeZoneIndex == -1 || timeZoneIndex < tIndex {
			timeZoneIndex = strings.LastIndex(format, "+")
			if timeZoneIndex == -1 || timeZoneIndex < tIndex {
				timeZoneIndex = strings.LastIndex(format, "-")
			}
		}

		// This position better be right of 'T'
		if timeZoneIndex != -1 && tIndex < timeZoneIndex {
			zeros := strings.Builder{}
			numZerosNeeded := 0

			// Specify trailing zeros if fractional precision is less than the nanoseconds.
			// e.g. A timestamp: 2021-05-25T13:41:31.00001234 with fractional precision: 2 will return "2021-05-25 13:41:31.00"
			ns := ts.dateTime.Nanosecond()
			if ns == 0 || maxFractionalPrecision-len(strconv.Itoa(ns)) >= int(ts.numFractionalSeconds) {
				zeros.WriteByte('.')
				numZerosNeeded = int(ts.numFractionalSeconds)
			} else {
				decimalPlaceIndex := strings.LastIndex(format, ".")
				if decimalPlaceIndex != -1 {
					decimalPlacesOccupied := timeZoneIndex - decimalPlaceIndex - 1
					numZerosNeeded = int(ts.numFractionalSeconds) - decimalPlacesOccupied
				}
			}

			// Add trailing zeros until the fractional seconds component is the correct length
			for i := 0; i < numZerosNeeded; i++ {
				zeros.WriteByte('0')
			}

			format = format[0:timeZoneIndex] + zeros.String() + format[timeZoneIndex:]
		}
	}

	// A timestamp with time precision (ie. Minute/Second/Nanosecond) and Unspecified timezone
	// should have a "-00:00" offset but time.String() is returning a "+00:00" offset.
	if ts.precision >= TimestampPrecisionMinute && ts.kind == TimezoneUnspecified {
		index := strings.LastIndex(format, "+00:00")
		if index != -1 {
			format = format[0:index] + "-00:00"
		}
	}

	return format
}

// Equal figures out if two timestamps are equal for each component.
func (ts Timestamp) Equal(ts1 Timestamp) bool {
	_, offset := ts.dateTime.Zone()
	_, offset1 := ts1.dateTime.Zone()

	return ts.dateTime.Equal(ts1.dateTime) &&
		offset == offset1 &&
		ts.precision == ts1.precision &&
		ts.kind == ts1.kind &&
		ts.numFractionalSeconds == ts1.numFractionalSeconds
}

// TruncatedNanoseconds returns nanoseconds with trailing values removed up to the difference of max fractional precision - time stamp's fractional precision
// e.g. 123456000 with fractional precision: 3 will get truncated to 123.
func (ts Timestamp) TruncatedNanoseconds() int {
	nsecs := ts.dateTime.Nanosecond()

	for i := uint8(0); i < (maxFractionalPrecision-ts.numFractionalSeconds) && nsecs > 0; i++ {
		nsecs /= 10
	}
	return nsecs
}
