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
			args:     args{"2000T", Year, Unspecified},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 1, 0, 0, 0, 0, time.UTC), precision: Year},
		},
		{
			testCase: "2000-01T",
			args:     args{"2000-01T", Month, Unspecified},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 1, 0, 0, 0, 0, time.UTC), precision: Month},
		},
		{
			testCase: "2000-01-02T",
			args:     args{"2000-01-02T", Day, Unspecified},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 0, 0, 0, 0, time.UTC), precision: Day},
		},
		{
			testCase: "2000-01-02T03:04Z",
			args:     args{"2000-01-02T03:04Z", Minute, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 0, 0, time.UTC), precision: Minute, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05Z",
			args:     args{"2000-01-02T03:04:05Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 0, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.1Z",
			args:     args{"2000-01-02T03:04:05.1Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 100000000, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.12Z",
			args:     args{"2000-01-02T03:04:05.12Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 120000000, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.123Z",
			args:     args{"2000-01-02T03:04:05.123Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123000000, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.1234Z",
			args:     args{"2000-01-02T03:04:05.1234Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123400000, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.12345Z",
			args:     args{"2000-01-02T03:04:05.12345Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123450000, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.123456Z",
			args:     args{"2000-01-02T03:04:05.123456Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123456000, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.1234567Z",
			args:     args{"2000-01-02T03:04:05.1234567Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123456700, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.12345678Z",
			args:     args{"2000-01-02T03:04:05.12345678Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123456780, time.UTC), precision: Second, kind: UTC},
		},
		{
			testCase: "2000-01-02T03:04:05.123456789Z",
			args:     args{"2000-01-02T03:04:05.123456789Z", Second, UTC},
			expected: Timestamp{dateTime: time.Date(2000, time.Month(1), 2, 3, 4, 5, 123456789, time.UTC), precision: Second, kind: UTC},
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

func TestTimestampFormat(t *testing.T) {
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
			fields:   fields{2000, 1, 1, 1, 0, 0, 0, Year, 0},
			expected: "2000T",
		},
		{
			fields:   fields{2000, 1, 1, 1, 0, 0, 0, Month, 0},
			expected: "2000-01T",
		},
		{
			fields:   fields{2000, 1, 2, 1, 0, 0, 0, Day, 0},
			expected: "2000-01-02T",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 0, 0, Minute, 0},
			expected: "2000-01-02T03:04Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 0, Second, 0},
			expected: "2000-01-02T03:04:05Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 100000000, Nanosecond, 1},
			expected: "2000-01-02T03:04:05.1Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 120000000, Nanosecond, 2},
			expected: "2000-01-02T03:04:05.12Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123000000, Nanosecond, 3},
			expected: "2000-01-02T03:04:05.123Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123400000, Nanosecond, 4},
			expected: "2000-01-02T03:04:05.1234Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123450000, Nanosecond, 5},
			expected: "2000-01-02T03:04:05.12345Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123456000, Nanosecond, 6},
			expected: "2000-01-02T03:04:05.123456Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123456700, Nanosecond, 7},
			expected: "2000-01-02T03:04:05.1234567Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123456780, Nanosecond, 8},
			expected: "2000-01-02T03:04:05.12345678Z",
		},
		{
			fields:   fields{2000, 1, 2, 3, 4, 5, 123456789, Nanosecond, 9},
			expected: "2000-01-02T03:04:05.123456789Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			dateTime := time.Date(tt.fields.year, time.Month(tt.fields.month), tt.fields.day, tt.fields.hour,
				tt.fields.minute, tt.fields.second, tt.fields.nanosecond, time.UTC)

			kind := Unspecified
			if tt.fields.precision >= Minute {
				kind = UTC
			}

			ts := &Timestamp{
				dateTime:             dateTime,
				precision:            tt.fields.precision,
				kind:                 kind,
				numFractionalSeconds: tt.fields.numFractionalSeconds,
			}
			if actual := ts.Format(); actual != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}
