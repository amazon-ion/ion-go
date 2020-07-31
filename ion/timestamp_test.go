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
	"testing"
	"time"
)

func TestNewTimestampFromStr(t *testing.T) {
	type args struct {
		dateStr   string
		precision TimestampPrecision
		kind      TimezoneKind
	}
	tests := []struct {
		testCase string
		args     args
		expected Timestamp
	}{
		{
			testCase: "2000T",
			args:     args{"2000T", TimestampPrecisionYear, TimezoneUnspecified},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 1, 0, 0, 0, 0, time.UTC), precision: TimestampPrecisionYear},
		},
		{
			testCase: "2000-01T",
			args:     args{"2000-01T", TimestampPrecisionMonth, TimezoneUnspecified},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 1, 0, 0, 0, 0, time.UTC), precision: TimestampPrecisionMonth},
		},
		{
			testCase: "2000-01-02T",
			args:     args{"2000-01-02T", TimestampPrecisionDay, TimezoneUnspecified},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 0, 0, 0, 0, time.UTC), precision: TimestampPrecisionDay},
		},
		{
			testCase: "2000-01-02T03:04Z",
			args:     args{"2000-01-02T03:04Z", TimestampPrecisionMinute, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 0, 0, time.UTC), precision: TimestampPrecisionMinute, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05Z",
			args:     args{"2000-01-02T03:04:05Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 0, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.1Z",
			args:     args{"2000-01-02T03:04:05.1Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 100000000, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.12Z",
			args:     args{"2000-01-02T03:04:05.12Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 120000000, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.123Z",
			args:     args{"2000-01-02T03:04:05.123Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123000000, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.1234Z",
			args:     args{"2000-01-02T03:04:05.1234Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123400000, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.12345Z",
			args:     args{"2000-01-02T03:04:05.12345Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123450000, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.123456Z",
			args:     args{"2000-01-02T03:04:05.123456Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123456000, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.1234567Z",
			args:     args{"2000-01-02T03:04:05.1234567Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123456700, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.12345678Z",
			args:     args{"2000-01-02T03:04:05.12345678Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123456780, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
		{
			testCase: "2000-01-02T03:04:05.123456789Z",
			args:     args{"2000-01-02T03:04:05.123456789Z", TimestampPrecisionSecond, TimezoneUTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123456789, time.UTC), precision: TimestampPrecisionSecond, kind: TimezoneUTC},
		},
	}
	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			actual, err := NewTimestampFromStr(tt.args.dateStr, tt.args.precision, tt.args.kind)
			if err != nil {
				t.Fatal(err)
			}
			if !actual.Equal(tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestTimestampString(t *testing.T) {
	type fields struct {
		year                 int
		month                int
		day                  int
		hour                 int
		minute               int
		second               int
		nanosecond           int
		precision            TimestampPrecision
		numFractionalSeconds uint8
	}

	tests := []struct {
		fields   fields
		expected string
	}{
		{
			fields:   fields{2000, 1, 1, 1, 0, 0, 0, TimestampPrecisionYear, 0},
			expected: "2000T",
		},
		{
			fields:   fields{2000, 1, 1, 1, 0, 0, 0, TimestampPrecisionMonth, 0},
			expected: "2000-01T",
		},
		{
			fields:   fields{2000, 1, 2, 1, 0, 0, 0, TimestampPrecisionDay, 0},
			expected: "2000-01-02T",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 0, 0, TimestampPrecisionMinute, 0},
			expected: "2000-01-02T03:04Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 0, TimestampPrecisionSecond, 0},
			expected: "2000-01-02T03:04:05Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 100000000, TimestampPrecisionNanosecond, 1},
			expected: "2000-01-02T03:04:05.1Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 120000000, TimestampPrecisionNanosecond, 2},
			expected: "2000-01-02T03:04:05.12Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123000000, TimestampPrecisionNanosecond, 3},
			expected: "2000-01-02T03:04:05.123Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123400000, TimestampPrecisionNanosecond, 4},
			expected: "2000-01-02T03:04:05.1234Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123450000, TimestampPrecisionNanosecond, 5},
			expected: "2000-01-02T03:04:05.12345Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123456000, TimestampPrecisionNanosecond, 6},
			expected: "2000-01-02T03:04:05.123456Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123456700, TimestampPrecisionNanosecond, 7},
			expected: "2000-01-02T03:04:05.1234567Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123456780, TimestampPrecisionNanosecond, 8},
			expected: "2000-01-02T03:04:05.12345678Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123456789, TimestampPrecisionNanosecond, 9},
			expected: "2000-01-02T03:04:05.123456789Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			dateTime := time.Date(tt.fields.year, time.Month(tt.fields.month), tt.fields.day, tt.fields.hour,
				tt.fields.minute, tt.fields.second, tt.fields.nanosecond, time.UTC)

			kind := TimezoneUnspecified
			if tt.fields.precision >= TimestampPrecisionMinute {
				kind = TimezoneUTC
			}

			ts := &Timestamp{
				dateTime:             dateTime,
				precision:            tt.fields.precision,
				kind:                 kind,
				numFractionalSeconds: tt.fields.numFractionalSeconds,
			}
			if actual := ts.String(); actual != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}
