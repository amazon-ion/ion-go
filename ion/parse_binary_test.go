/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

package ion

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseBinaryBlob(t *testing.T) {
	tests := []struct {
		name        string
		blob        []byte
		expected    *Digest
		expectedErr error
	}{
		{
			name:        "no bytes",
			blob:        []byte{},
			expectedErr: errors.New("read 0 bytes of binary version marker with err: EOF"),
		},
		{
			name:        "only three bytes",
			blob:        ion10BVM[:3],
			expectedErr: errors.New("read 3 bytes of binary version marker with err: <nil>"),
		},
		{
			name:        "byte version marker for Ion 2.0",
			blob:        []byte{0xE0, 0x02, 0x00, 0xEA},
			expectedErr: errors.New("invalid binary version marker: 0xe0 0x02 0x00 0xea"),
		},
		{
			name:     "no data after BVM",
			blob:     ion10BVM,
			expected: &Digest{},
		},
		{
			name:        "unsupported type",
			blob:        append(ion10BVM, 0xF0),
			expectedErr: errors.New("invalid header combination - high: 15 low: 0"),
		},

		// Null.

		{
			name:     "null.null",
			blob:     append(ion10BVM, 0x0F),
			expected: &Digest{values: []Value{Null{}}},
		},
		{
			name:     "null.bool",
			blob:     append(ion10BVM, 0x1F),
			expected: &Digest{values: []Value{Null{typ: TypeBool}}},
		},
		{
			name:     "null.int",
			blob:     append(ion10BVM, 0x2F),
			expected: &Digest{values: []Value{Null{typ: TypeInt}}},
		},
		{
			name:     "null.int (negative)",
			blob:     append(ion10BVM, 0x3F),
			expected: &Digest{values: []Value{Null{typ: TypeInt}}},
		},
		{
			name:     "null.float",
			blob:     append(ion10BVM, 0x4F),
			expected: &Digest{values: []Value{Null{typ: TypeFloat}}},
		},
		{
			name:     "null.decimal",
			blob:     append(ion10BVM, 0x5F),
			expected: &Digest{values: []Value{Null{typ: TypeDecimal}}},
		},
		{
			name:     "null.timestamp",
			blob:     append(ion10BVM, 0x6F),
			expected: &Digest{values: []Value{Null{typ: TypeTimestamp}}},
		},
		{
			name:     "null.symbol",
			blob:     append(ion10BVM, 0x7F),
			expected: &Digest{values: []Value{Null{typ: TypeSymbol}}},
		},
		{
			name:     "null.string",
			blob:     append(ion10BVM, 0x8F),
			expected: &Digest{values: []Value{Null{typ: TypeString}}},
		},
		{
			name:     "null.clob",
			blob:     append(ion10BVM, 0x9F),
			expected: &Digest{values: []Value{Null{typ: TypeClob}}},
		},
		{
			name:     "null.blob",
			blob:     append(ion10BVM, 0xAF),
			expected: &Digest{values: []Value{Null{typ: TypeBlob}}},
		},
		{
			name:     "null.list",
			blob:     append(ion10BVM, 0xBF),
			expected: &Digest{values: []Value{Null{typ: TypeList}}},
		},
		{
			name:     "null.sexp",
			blob:     append(ion10BVM, 0xCF),
			expected: &Digest{values: []Value{Null{typ: TypeSExp}}},
		},
		{
			name:     "null.struct",
			blob:     append(ion10BVM, 0xDF),
			expected: &Digest{values: []Value{Null{typ: TypeStruct}}},
		},

		// Padding and Bool.

		{
			name:     "zero length padding",
			blob:     append(ion10BVM, 0x00),
			expected: &Digest{values: []Value{padding{}}},
		},
		{
			name:     "two bytes of padding",
			blob:     append(ion10BVM, 0x01, 0xFF),
			expected: &Digest{values: []Value{padding{[]byte{0xFF}}}},
		},
		{
			name:     "sixteen bytes of padding",
			blob:     append(ion10BVM, 0x0E, 0x8E, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF),
			expected: &Digest{values: []Value{padding{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}}}},
		},
		{
			name:     "bool false",
			blob:     append(ion10BVM, 0x10),
			expected: &Digest{values: []Value{Bool{isSet: true}}},
		},
		{
			name:     "bool true",
			blob:     append(ion10BVM, 0x11),
			expected: &Digest{values: []Value{Bool{isSet: true, value: true}}},
		},
		{
			name:        "bool invalid representation",
			blob:        append(ion10BVM, 0x12),
			expectedErr: errors.New("invalid bool representation 0x2"),
		},

		// Symbol and String.

		{
			name:     "zero length symbol",
			blob:     append(ion10BVM, 0x70),
			expected: &Digest{values: []Value{Symbol{}}},
		},
		{
			name:        "symbol - length is too big",
			blob:        append(ion10BVM, 0x75),
			expectedErr: errors.New("symbol ID length of 5 bytes exceeds expected maximum of 4"),
		},
		{
			name:        "symbolID too large",
			blob:        append(ion10BVM, 0x74, 0x80, 0x00, 0x00, 0x00),
			expectedErr: errors.New("uint32 value 2147483648 overflows int32"),
		},
		{
			name:     "symbolID max value",
			blob:     append(ion10BVM, 0x74, 0x7F, 0xFF, 0xFF, 0xFF),
			expected: &Digest{values: []Value{Symbol{id: math.MaxInt32}}},
		},
		{
			name:     "zero length string",
			blob:     append(ion10BVM, 0x80),
			expected: &Digest{values: []Value{String{text: []byte{}}}},
		},
		{
			name:     "short string",
			blob:     append(ion10BVM, 0x83, 0x62, 0x6f, 0x6f),
			expected: &Digest{values: []Value{String{text: []byte{'b', 'o', 'o'}}}},
		},
		{
			name:     "long string",
			blob:     append(ion10BVM, 0x8E, 0x8E, 0x62, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f),
			expected: &Digest{values: []Value{String{text: []byte{'b', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o'}}}},
		},

		// Blob and Clob

		{
			name:     "zero length blob",
			blob:     append(ion10BVM, 0xA0),
			expected: &Digest{values: []Value{Blob{binary: []byte{}}}},
		},
		{
			name:     "zero length clob",
			blob:     append(ion10BVM, 0x90),
			expected: &Digest{values: []Value{Clob{text: []byte{}}}},
		},
		{
			name:     "short blob",
			blob:     append(ion10BVM, 0xA3, 0x62, 0x6f, 0x6f),
			expected: &Digest{values: []Value{Blob{binary: []byte{'b', 'o', 'o'}}}},
		},
		{
			name:     "short clob",
			blob:     append(ion10BVM, 0x93, 0x62, 0x6f, 0x6f),
			expected: &Digest{values: []Value{Clob{text: []byte{'b', 'o', 'o'}}}},
		},
		{
			name:     "long blob",
			blob:     append(ion10BVM, 0xAE, 0x8E, 0x62, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f),
			expected: &Digest{values: []Value{Blob{binary: []byte{'b', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o'}}}},
		},
		{
			name:     "long clob",
			blob:     append(ion10BVM, 0x9E, 0x8E, 0x62, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f, 0x6f),
			expected: &Digest{values: []Value{Clob{text: []byte{'b', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o'}}}},
		},

		// Positive and negative Int.

		{
			name: "zero length positive int",
			blob: append(ion10BVM, 0x20),
			// Expected text value is not verified.
			expected: &Digest{values: []Value{Int{isSet: true, binary: []byte{}, text: []byte{0x30}}}},
		},
		{
			name:        "zero length negative int",
			blob:        append(ion10BVM, 0x30),
			expectedErr: errors.New("negative zero is invalid"),
		},
		{
			name:        "int - negative zero",
			blob:        append(ion10BVM, 0x31, 0x00),
			expectedErr: errors.New("negative zero is invalid"),
		},
		{
			name:        "int - length is too high",
			blob:        append(ion10BVM, 0x2E, 0x10, 0x00, 0x00, 0x00, 0x80),
			expectedErr: errors.New("unable to parse length of int: number is too big to fit into uint32: 0x10 0x00 0x00 0x00 0x80"),
		},
		{
			name:     "short positive int",
			blob:     append(ion10BVM, 0x21, 0x42),
			expected: &Digest{values: []Value{Int{isSet: true, binary: []byte{0x42}}}},
		},
		{
			name:     "short negative int",
			blob:     append(ion10BVM, 0x31, 0x42),
			expected: &Digest{values: []Value{Int{isSet: true, isNegative: true, binary: []byte{0x42}}}},
		},
		{
			name:     "long positive int",
			blob:     append(ion10BVM, 0x2E, 0x8E, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e),
			expected: &Digest{values: []Value{Int{isSet: true, binary: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e}}}},
		},
		{
			name:     "long negative int",
			blob:     append(ion10BVM, 0x3E, 0x8E, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e),
			expected: &Digest{values: []Value{Int{isSet: true, isNegative: true, binary: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e}}}},
		},

		// Float and Decimal.

		{
			name:     "zero length float",
			blob:     append(ion10BVM, 0x40),
			expected: &Digest{values: []Value{Float{isSet: true, binary: []byte{}}}},
		},
		{
			name:        "float - invalid length",
			blob:        append(ion10BVM, 0x42),
			expectedErr: errors.New("invalid float length 2"),
		},
		{
			name:     "float - 4 byte value",
			blob:     append(ion10BVM, 0x44, 0x01, 0x02, 0x03, 0x04),
			expected: &Digest{values: []Value{Float{isSet: true, binary: []byte{0x01, 0x02, 0x03, 0x04}}}},
		},
		{
			name:     "float - 8 byte value",
			blob:     append(ion10BVM, 0x48, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08),
			expected: &Digest{values: []Value{Float{isSet: true, binary: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}}},
		},
		{
			name:     "zero length decimal",
			blob:     append(ion10BVM, 0x50),
			expected: &Digest{values: []Value{Decimal{isSet: true, binary: []byte{}}}},
		},
		{
			name:        "decimal - exponent takes up all the bytes",
			blob:        append(ion10BVM, 0x52, 0x00, 0x80),
			expectedErr: errors.New("invalid decimal - total length 2 with exponent length 2"),
		},
		{
			name:        "decimal - exponent isn't terminated",
			blob:        append(ion10BVM, 0x5E, 0x8E, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e),
			expectedErr: errors.New("unable to read exponent part of decimal: number not terminated after 14 bytes"),
		},
		{
			name:        "decimal - length is too high",
			blob:        append(ion10BVM, 0x5E, 0x04, 0x00, 0x80),
			expectedErr: errors.New("unable to parse length of decimal: number is too big to fit into uint16: 0x04 0x00 0x80"),
		},
		{
			name:     "short decimal",
			blob:     append(ion10BVM, 0x52, 0x80, 0x08),
			expected: &Digest{values: []Value{Decimal{isSet: true, binary: []byte{0x80, 0x08}}}},
		},
		{
			name:     "long decimal",
			blob:     append(ion10BVM, 0x5E, 0x8E, 0x01, 0x02, 0x03, 0x80, 0x05, 0x06, 0x07, 0x80, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e),
			expected: &Digest{values: []Value{Decimal{isSet: true, binary: []byte{0x01, 0x02, 0x03, 0x80, 0x05, 0x06, 0x07, 0x80, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e}}}},
		},

		// Timestamp.

		{
			name:        "timestamp - too short",
			blob:        append(ion10BVM, 0x61),
			expectedErr: errors.New("timestamp must have a length of at least two bytes"),
		},
		{
			name:        "timestamp - length exceeds maximum",
			blob:        append(ion10BVM, 0x6D),
			expectedErr: errors.New("timestamp length of 13 exceeds expected maximum of 12"),
		},
		{
			name:        "timestamp - offset isn't terminated",
			blob:        append(ion10BVM, 0x62, 0x00, 0x00),
			expectedErr: errors.New("unable to determine timestamp offset: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name: "timestamp - year only",
			blob: append(ion10BVM, 0x63, 0x80, 0x0F, 0xD0),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionYear,
				binary:    []byte{0x80, 0x0F, 0xD0},
				value:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			}}},
		},
		{
			name:        "timestamp - year isn't terminated",
			blob:        append(ion10BVM, 0x63, 0x80, 0x00, 0x00),
			expectedErr: errors.New("unable to determine timestamp year: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name: "timestamp - year and month",
			blob: append(ion10BVM, 0x64, 0x80, 0x0F, 0xD0, 0x81),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMonth,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81},
				value:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			}}},
		},
		{
			name:        "timestamp - month isn't terminated",
			blob:        append(ion10BVM, 0x64, 0x80, 0x0F, 0xD0, 0x01),
			expectedErr: errors.New("unable to determine timestamp month: number not terminated after 1 bytes"),
		},
		{
			name: "timestamp - year, month, and day",
			blob: append(ion10BVM, 0x65, 0x80, 0x0F, 0xD0, 0x81, 0x82),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionDay,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82},
				value:     time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
			}}},
		},
		{
			name:        "timestamp - day isn't terminated",
			blob:        append(ion10BVM, 0x65, 0x80, 0x0F, 0xD0, 0x81, 0x01),
			expectedErr: errors.New("unable to determine timestamp day: number not terminated after 1 bytes"),
		},
		{
			name:        "timestamp - hour without minutes",
			blob:        append(ion10BVM, 0x66, 0x80, 0x0F, 0xD0, 0x81, 0x81, 0x81),
			expectedErr: errors.New("invalid timestamp - cannot specify hours without minutes"),
		},
		{
			name: "timestamp - year, month, day, hour, and minute",
			blob: append(ion10BVM, 0x67, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMinute,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84},
				value:     time.Date(2000, 1, 2, 3, 4, 0, 0, time.UTC),
			}}},
		},
		{
			name:        "timestamp - hour isn't terminated",
			blob:        append(ion10BVM, 0x67, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x03, 0x84),
			expectedErr: errors.New("unable to determine timestamp hour: number not terminated after 1 bytes"),
		},
		{
			name:        "timestamp - minute isn't terminated",
			blob:        append(ion10BVM, 0x67, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x04),
			expectedErr: errors.New("unable to determine timestamp minute: number not terminated after 1 bytes"),
		},
		{
			name: "timestamp - year, month, day, hour, minute, and second",
			blob: append(ion10BVM, 0x68, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionSecond,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name:        "timestamp - second isn't terminated",
			blob:        append(ion10BVM, 0x68, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x05),
			expectedErr: errors.New("unable to determine timestamp second: number not terminated after 1 bytes"),
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and millisecond1",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC1),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMillisecond1,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC1},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and millisecond2",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC2),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMillisecond2,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC2},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and millisecond3",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC3),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMillisecond3,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC3},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and millisecond4",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC4),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMillisecond4,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC4},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and microsecond1",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC5),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMicrosecond1,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC5},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and microsecond2",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC6),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMicrosecond2,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC6},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and microsecond3",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC7),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMicrosecond3,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC7},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and microsecond4",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC8),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMicrosecond4,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC8},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and microsecond5",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC9),
			// Reading the exponent removes the stop bit of the variable int which turns
			// the C into a 4.
			expectedErr: errors.New("invalid exponent for timestamp fractional second: 0x49"),
		},
		{
			name:        "timestamp - exponent isn't terminated",
			blob:        append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0x04),
			expectedErr: errors.New("unable to determine timestamp fractional second exponent: number not terminated after 1 bytes"),
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, exponent, and coefficient",
			blob: append(ion10BVM, 0x6A, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC1, 0x06),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMillisecond1,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC1, 0x06},
				value:     time.Date(2000, 1, 2, 3, 4, 5, int(600*time.Millisecond), time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, exponent, and 2 byte coefficient",
			blob: append(ion10BVM, 0x6B, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC1, 0x00, 0x06),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMillisecond1,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC1, 0x00, 0x06},
				value:     time.Date(2000, 1, 2, 3, 4, 5, int(600*time.Millisecond), time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, and exponent1 - exponent is < C1",
			blob: append(ion10BVM, 0x69, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC0),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionSecond,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC0},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp - year, month, day, hour, minute, second, exponent, and coefficient - exponent is < C1 and coefficient is 0",
			blob: append(ion10BVM, 0x6A, 0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC0, 0x00),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionSecond,
				binary:    []byte{0x80, 0x0F, 0xD0, 0x81, 0x82, 0x83, 0x84, 0x85, 0xC0, 0x00},
				value:     time.Date(2000, 1, 2, 3, 4, 5, 0, time.UTC),
			}}},
		},
		{
			name: "timestamp2011-02-20T19_30_59_100-08_00.10n",
			blob: append(ion10BVM, 0x6B, 0x43, 0xE0, 0x0F, 0xDB, 0x82, 0x94, 0x93, 0x9E, 0xBB, 0xC3, 0x64),
			expected: &Digest{values: []Value{Timestamp{
				precision: TimestampPrecisionMillisecond3,
				binary:    []byte{0x43, 0xE0, 0x0F, 0xDB, 0x82, 0x94, 0x93, 0x9E, 0xBB, 0xC3, 0x64},
				value:     time.Date(2011, 2, 20, 19, 30, 59, int(100*time.Millisecond), time.FixedZone("-8:00", -480)),
			}}},
		},

		// List and S-Expression.

		{
			name:     "zero length list",
			blob:     append(ion10BVM, 0xB0),
			expected: &Digest{values: []Value{List{values: []Value{}}}},
		},
		{
			name:     "zero length sexp",
			blob:     append(ion10BVM, 0xC0),
			expected: &Digest{values: []Value{SExp{values: []Value{}}}},
		},
		{
			name:        "list - invalid bool",
			blob:        append(ion10BVM, 0xB1, 0x12),
			expectedErr: errors.New("unable to parse list: invalid bool representation 0x2"),
		},
		{
			name:        "sexp - invalid bool",
			blob:        append(ion10BVM, 0xC1, 0x12),
			expectedErr: errors.New("unable to parse list: invalid bool representation 0x2"),
		},
		{
			name:        "list - valid bool, invalid float",
			blob:        append(ion10BVM, 0xB2, 0x11, 0x42),
			expectedErr: errors.New("unable to parse list: invalid float length 2"),
		},
		{
			name:        "sexp - valid bool, invalid float",
			blob:        append(ion10BVM, 0xC2, 0x11, 0x42),
			expectedErr: errors.New("unable to parse list: invalid float length 2"),
		},
		{
			name:        "nested list - invalid bool",
			blob:        append(ion10BVM, 0xB2, 0xB1, 0x12),
			expectedErr: errors.New("unable to parse list: unable to parse list: invalid bool representation 0x2"),
		},
		{
			name:        "nested sexp - invalid bool",
			blob:        append(ion10BVM, 0xC2, 0xB1, 0x12),
			expectedErr: errors.New("unable to parse list: unable to parse list: invalid bool representation 0x2"),
		},
		{
			name: "list - valid bool",
			blob: append(ion10BVM, 0xB1, 0x11),
			expected: &Digest{
				values: []Value{List{
					values: []Value{Bool{isSet: true, value: true}},
				}},
			},
		},
		{
			name: "sexp - valid bool",
			blob: append(ion10BVM, 0xC1, 0x11),
			expected: &Digest{
				values: []Value{SExp{
					values: []Value{Bool{isSet: true, value: true}},
				}},
			},
		},
		{
			name: "sexp in list - valid bool",
			blob: append(ion10BVM, 0xB2, 0xC1, 0x11),
			expected: &Digest{
				values: []Value{List{
					values: []Value{SExp{
						values: []Value{Bool{isSet: true, value: true}},
					}},
				}},
			},
		},
		{
			name: "list in sexp - valid bool",
			blob: append(ion10BVM, 0xC2, 0xB1, 0x11),
			expected: &Digest{
				values: []Value{SExp{
					values: []Value{List{
						values: []Value{Bool{isSet: true, value: true}},
					}},
				}},
			},
		},

		// Struct.

		{
			name:     "zero length struct",
			blob:     append(ion10BVM, 0xD0),
			expected: &Digest{values: []Value{Struct{fields: []StructField{}}}},
		},
		{
			name:        "struct - symbol isn't terminated",
			blob:        append(ion10BVM, 0xD2, 0x00, 0x00),
			expectedErr: errors.New("unable to read struct field symbol: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "struct - invalid bool field value",
			blob:        append(ion10BVM, 0xD2, 0x80, 0x12),
			expectedErr: errors.New("unable to read struct field value: invalid bool representation 0x2"),
		},
		{
			name:        "struct - no value",
			blob:        append(ion10BVM, 0xD2, 0x00, 0x84),
			expectedErr: errors.New("unable to read struct field value: EOF"),
		},
		{
			name: "struct - valid bool",
			blob: append(ion10BVM, 0xD2, 0x80, 0x11),
			expected: &Digest{values: []Value{Struct{fields: []StructField{
				{Symbol: Symbol{}, Value: Bool{isSet: true, value: true}},
			}}}},
		},
		{
			name:     "struct - valid padding",
			blob:     append(ion10BVM, 0xD3, 0x80, 0x01, 0xFF),
			expected: &Digest{values: []Value{Struct{fields: []StructField{}}}},
		},

		// Annotation.

		{
			name:        "annotation - length too short",
			blob:        append(ion10BVM, 0xE1),
			expectedErr: errors.New("length must be at least 3 for an annotation wrapper, found 1"),
		},
		{
			name:        "annotation - annotation length not terminated",
			blob:        append(ion10BVM, 0xE3, 0x00, 0x00, 0x11),
			expectedErr: errors.New("unable to determine annotation symbol length: number not terminated after 3 bytes"),
		},
		{
			name:        "annotation - annotation length equal to numBytes",
			blob:        append(ion10BVM, 0xE3, 0x83, 0x84, 0x11),
			expectedErr: errors.New("invalid lengths for annotation - field length is 3 while annotation symbols length is 3"),
		},
		{
			name:        "annotation - annotation not terminated",
			blob:        append(ion10BVM, 0xE3, 0x81, 0x00, 0x11),
			expectedErr: errors.New("unable to read annotation symbol: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "annotation - invalid bool",
			blob:        append(ion10BVM, 0xE3, 0x81, 0x84, 0x12),
			expectedErr: errors.New("unable to read annotation value: invalid bool representation 0x2"),
		},
		{
			name:        "annotation - length exceeds single value",
			blob:        append(ion10BVM, 0xE4, 0x81, 0x84, 0x11, 0x11),
			expectedErr: errors.New("annotation declared 4 bytes but there are 1 bytes left"),
		},
		{
			name:        "annotation on noop padding",
			blob:        append(ion10BVM, 0xE3, 0x81, 0x84, 0x00),
			expectedErr: errors.New("annotation on padding is not legal"),
		},
		{
			name: "annotation - valid bool",
			blob: append(ion10BVM, 0xE3, 0x81, 0x84, 0x11),
			expected: &Digest{values: []Value{Bool{
				annotations: []Symbol{{id: 4}},
				isSet:       true,
				value:       true,
			}}},
		},
		{
			name: "two annotations - valid bool",
			blob: append(ion10BVM, 0xE4, 0x82, 0x84, 0x87, 0x11),
			expected: &Digest{values: []Value{Bool{
				annotations: []Symbol{{id: 4}, {id: 7}},
				isSet:       true,
				value:       true,
			}}},
		},

		// Error - EOF while reading length.

		{
			name:        "padding - EOF while reading length",
			blob:        append(ion10BVM, 0x0E, 0x0E),
			expectedErr: errors.New("unable to parse length of padding: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "pos int - EOF while reading length",
			blob:        append(ion10BVM, 0x2E, 0x0E),
			expectedErr: errors.New("unable to parse length of int: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "neg int - EOF while reading length",
			blob:        append(ion10BVM, 0x3E, 0x0E),
			expectedErr: errors.New("unable to parse length of int: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "decimal - EOF while reading length",
			blob:        append(ion10BVM, 0x5E, 0x0E),
			expectedErr: errors.New("unable to parse length of decimal: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "timestamp - EOF while reading length",
			blob:        append(ion10BVM, 0x6E, 0x0E),
			expectedErr: errors.New("unable to parse length of timestamp: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "symbol - EOF while reading length",
			blob:        append(ion10BVM, 0x7E, 0x0E),
			expectedErr: errors.New("unable to parse length of symbol: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "string - EOF while reading length",
			blob:        append(ion10BVM, 0x8E, 0x0E),
			expectedErr: errors.New("unable to parse length of string: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "clob - EOF while reading length",
			blob:        append(ion10BVM, 0x9E, 0x0E),
			expectedErr: errors.New("unable to parse length of bytes: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "blob - EOF while reading length",
			blob:        append(ion10BVM, 0xAE, 0x0E),
			expectedErr: errors.New("unable to parse length of bytes: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "list - EOF while reading length",
			blob:        append(ion10BVM, 0xBE, 0x0E),
			expectedErr: errors.New("unable to parse length of list: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "sexp - EOF while reading length",
			blob:        append(ion10BVM, 0xCE, 0x0E),
			expectedErr: errors.New("unable to parse length of list: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "struct - EOF while reading length",
			blob:        append(ion10BVM, 0xDE, 0x0E),
			expectedErr: errors.New("unable to parse length of struct: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "struct - EOF while reading length special length 1 case",
			blob:        append(ion10BVM, 0xD1, 0x0E),
			expectedErr: errors.New("unable to parse length of struct: read 0 bytes (wanted one) of number with err: EOF"),
		},
		{
			name:        "annotation - EOF while reading length",
			blob:        append(ion10BVM, 0xEE, 0x0E),
			expectedErr: errors.New("unable to parse length of annotation: read 0 bytes (wanted one) of number with err: EOF"),
		},

		// Error - EOF while reading value.

		{
			name:        "padding - EOF while reading padding",
			blob:        append(ion10BVM, 0x0E, 0x8E, 0xFF),
			expectedErr: errors.New("read 1 of expected 14 padding bytes with err: <nil>"),
		},
		{
			name:        "pos int - EOF while reading value",
			blob:        append(ion10BVM, 0x22, 0x08),
			expectedErr: errors.New("unable to read int - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "neg int - EOF while reading value",
			blob:        append(ion10BVM, 0x32, 0x08),
			expectedErr: errors.New("unable to read int - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "float - EOF while reading value",
			blob:        append(ion10BVM, 0x44, 0x08),
			expectedErr: errors.New("unable to read float - read 1 bytes of 4 with err: <nil>"),
		},
		{
			name:        "decimal - EOF while reading value",
			blob:        append(ion10BVM, 0x52, 0x08),
			expectedErr: errors.New("unable to read decimal - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "timestamp - EOF while reading value",
			blob:        append(ion10BVM, 0x62, 0x08),
			expectedErr: errors.New("unable to read timestamp - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "symbol - EOF while reading value",
			blob:        append(ion10BVM, 0x72, 0x08),
			expectedErr: errors.New("unable to read symbol ID - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "string - EOF while reading value",
			blob:        append(ion10BVM, 0x82, 0x08),
			expectedErr: errors.New("unable to read string - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "clob - EOF while reading bytes",
			blob:        append(ion10BVM, 0x92, 0x08),
			expectedErr: errors.New("unable to read bytes - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "blob - EOF while reading bytes",
			blob:        append(ion10BVM, 0xA2, 0x08),
			expectedErr: errors.New("unable to read bytes - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "list - EOF while reading values",
			blob:        append(ion10BVM, 0xB2, 0x08),
			expectedErr: errors.New("unable to read list - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "sexp - EOF while reading values",
			blob:        append(ion10BVM, 0xC2, 0x08),
			expectedErr: errors.New("unable to read list - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "struct - EOF while reading values",
			blob:        append(ion10BVM, 0xD2, 0x08),
			expectedErr: errors.New("unable to read struct - read 1 bytes of 2 with err: <nil>"),
		},
		{
			name:        "annotation - EOF while reading values",
			blob:        append(ion10BVM, 0xE3, 0x81),
			expectedErr: errors.New("unable to read annotation - read 1 bytes of 3 with err: <nil>"),
		},
	}

	for _, tst := range tests {
		test := tst
		t.Run(test.name, func(t *testing.T) {
			out, err := parseBinaryBlob(test.blob)
			if diff := cmpDigests(test.expected, out); diff != "" {
				t.Error("out: (-expected, +found)", diff)
			}
			if diff := cmpErrs(test.expectedErr, err); diff != "" {
				t.Error("err: (-expected, +found)", diff)
			}
		})
	}
}

func TestBinaryStream(t *testing.T) {
	// Test the cases that aren't covered by TestParseBinaryBlob.
	tests := []struct {
		name        string
		blob        []byte
		expected    []Value
		expectedErr error
	}{
		{
			name:     "two digests, two booleans",
			blob:     []byte{0xE0, 0x01, 0x00, 0xEA, 0x11, 0xE0, 0x01, 0x00, 0xEA, 0x11},
			expected: []Value{Bool{isSet: true, value: true}, Bool{isSet: true, value: true}},
		},
		{
			name:        "EOF reading second BVM",
			blob:        []byte{0xE0, 0x01, 0x00, 0xEA, 0x11, 0xE0, 0x01, 0x00},
			expectedErr: errors.New("unable to read binary version marker - read 2 bytes of 3 with err: <nil>"),
		},
		{
			name:        "invalid second BVM",
			blob:        []byte{0xE0, 0x01, 0x00, 0xEA, 0x11, 0xE0, 0x02, 0x00, 0xEA, 0x11},
			expectedErr: errors.New("invalid binary version marker: 0xe0 0x02 0x00 0xea"),
		},
		{
			name:        "invalid boolean in second Digest",
			blob:        []byte{0xE0, 0x01, 0x00, 0xEA, 0x11, 0xE0, 0x01, 0x00, 0xEA, 0x12},
			expected:    []Value{Bool{isSet: true, value: true}},
			expectedErr: errors.New("invalid bool representation 0x2"),
		},
	}

	for _, tst := range tests {
		test := tst
		t.Run(test.name, func(t *testing.T) {
			ch := parseBinaryStream(bytes.NewReader(test.blob))

			var values []Value
			var err error
		Loop:
			for {
				select {
				case item, ok := <-ch:
					if !ok {
						break Loop
					}
					if item.Error != nil {
						err = item.Error
						break Loop
					}
					values = append(values, item.Digest.values...)
				case <-time.After(1 * time.Second):
					t.Fatal("timed out")
				}
			}

			if diff := cmpValueSlices(test.expected, values); diff != "" {
				t.Error("(-expected, +found)", diff)
			}
			if diff := cmpErrs(test.expectedErr, err); diff != "" {
				t.Error("err: (-expected, +found)", diff)
			}
		})
	}
}

func TestIonTests_Binary_Good(t *testing.T) {
	testFilePath := "../ion-tests/iontestdata/good"
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".10n") {
			return nil
		}

		t.Run(strings.TrimPrefix(path, testFilePath), func(t *testing.T) {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			out, err := parseBinaryBlob(data)
			if err != nil {
				t.Fatal(err)
			}
			if out == nil || len(out.Value()) == 0 {
				t.Error("expected out to have at least one value")
			}

			// TODO: If we are in the equivs directory, then verify that each top-level
			//       Value in the Digest is comprised of equivalent sub-elements.  Need
			//       the Value() functions to be able to pull that off.
		})
		return nil
	}
	if err := filepath.Walk(testFilePath, walkFn); err != nil {
		t.Fatal(err)
	}
}

func TestIonTests_Binary_Equivalents(t *testing.T) {
	testFilePath := "../ion-tests/iontestdata/good/equivs"
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".10n") {
			return nil
		}

		t.Run(strings.TrimPrefix(path, testFilePath), func(t *testing.T) {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println("parsing file", path)
			out, err := parseBinaryBlob(data)
			if err != nil {
				t.Fatal(err)
			}

			if out == nil || len(out.Value()) == 0 {
				t.Error("expected out to have at least one value")
			}

			for i, value := range out.Value() {
				t.Log("collection", i, "of", info.Name())
				switch value.Type() {
				case TypeList:
					assertEquivalentValues(value.(List).values, t)
				case TypeSExp:
					assertEquivalentValues(value.(SExp).values, t)
				default:
					t.Error("top-element item is", value.Type(), "for", info.Name())
				}
			}
		})
		return nil
	}
	if err := filepath.Walk(testFilePath, walkFn); err != nil {
		t.Fatal(err)
	}
}

func TestIonTests_Binary_Bad(t *testing.T) {
	filesToSkip := map[string]bool{
		// TODO: Deal with symbol tables and verification of SymbolIDs.
		"annotationSymbolIDUnmapped.10n":                          true,
		"fieldNameSymbolIDUnmapped.10n":                           true,
		"localSymbolTableWithMultipleImportsFields.10n":           true,
		"localSymbolTableWithMultipleSymbolsAndImportsFields.10n": true,
		"localSymbolTableWithMultipleSymbolsFields.10n":           true,
		"symbolIDUnmapped.10n":                                    true,
		// Not performing timestamp verification on parse.
		"leapDayNonLeapYear_1.10n": true,
		"leapDayNonLeapYear_2.10n": true,
		"timestampSept31.10n":      true,
		// Not performing string verification on parse.
		"stringWithLatinEncoding.10n": true,
	}

	testFilePath := "../ion-tests/iontestdata/bad"
	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".10n") {
			return nil
		}

		name := info.Name()
		if _, ok := filesToSkip[name]; ok {
			t.Log("skipping", name)
			return nil
		}

		t.Run(strings.TrimPrefix(path, testFilePath), func(t *testing.T) {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			out, err := parseBinaryBlob(data)
			if err == nil {
				t.Error("expected error but found none")
			}
			if out != nil {
				t.Errorf("%#v", out)
			}
		})
		return nil
	}
	if err := filepath.Walk(testFilePath, walkFn); err != nil {
		t.Fatal(err)
	}
}

func Test_parseBinaryNull(t *testing.T) {
	// It's not possible to hit this case going through the parser, but we want
	// to demonstrate that if something gets broken this is what it looks like.
	if val, err := parseBinaryNull(0xFF); val != nil || err == nil {
		t.Errorf("expected nil value and error but found %#v and %+v", val, err)
	}
}
