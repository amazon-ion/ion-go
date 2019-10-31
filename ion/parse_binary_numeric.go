/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

package ion

import (
	"bytes"
	"io"
	"math"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

// This file contains binary parsers for Int, Float, Decimal, and Timestamp.

// parseBinaryInt parses the magnitude and optional length portion of an an Int.
// The magnitude is a UInt so we need to be told what the sign is.
func parseBinaryInt(ann []Symbol, isNegative bool, lengthByte byte, r io.Reader) (Value, error) {
	if lengthByte == 0 {
		if isNegative {
			return nil, errors.New("negative zero is invalid")
		}
		return Int{annotations: ann, isSet: true, binary: []byte{}, value: &big.Int{}}, nil
	}

	numBytes, errLength := determineLength32(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of int")
	}

	buf := make([]byte, numBytes)
	if n, err := r.Read(buf); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read int - read %d bytes of %d with err: %v", n, numBytes, err)
	}

	// Negative zero is not valid.
	if isNegative {
		isZero := true
		for _, b := range buf {
			if b != 0 {
				isZero = false
				break
			}
		}
		if isZero {
			return nil, errors.Errorf("negative zero is invalid")
		}
	}

	return Int{annotations: ann, isSet: true, isNegative: isNegative, binary: buf}, nil
}

// parseBinaryFloat parses either the 32-bit or 64-bit version of of an IEEE-754 floating
// point number.
func parseBinaryFloat(ann []Symbol, numBytes byte, r io.Reader) (Value, error) {
	// Represents 0e0.
	if numBytes == 0 {
		return Float{annotations: ann, isSet: true, binary: []byte{}}, nil
	}
	if numBytes != 4 && numBytes != 8 {
		return nil, errors.Errorf("invalid float length %d", numBytes)
	}
	buf := make([]byte, numBytes)
	if n, err := r.Read(buf); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read float - read %d bytes of %d with err: %v", n, numBytes, err)
	}
	return Float{annotations: ann, isSet: true, binary: buf}, nil
}

// parseBinaryDecimal parses a variable length Decimal with exponent and coefficient components.
func parseBinaryDecimal(ann []Symbol, lengthByte byte, r io.Reader) (Value, error) {
	// Represents 0d0.
	if lengthByte == 0 {
		return Decimal{annotations: ann, isSet: true, binary: []byte{}}, nil
	}

	numBytes, errLength := determineLength16(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of decimal")
	}

	// Read in the entirety of the Decimal value from the stream, then farm out those
	// bytes to read the exponent and coefficient to ensure that we have a valid decimal.
	data := make([]byte, numBytes)
	if n, err := r.Read(data); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read decimal - read %d bytes of %d with err: %v", n, numBytes, err)
	}

	dataReader := bytes.NewReader(data)
	expBytes, errExp := readVarNumber(numBytes, dataReader)
	if errExp != nil {
		return nil, errors.WithMessage(errExp, "unable to read exponent part of decimal")
	}

	coefficientLength := numBytes - uint16(len(expBytes))
	if coefficientLength <= 0 {
		return nil, errors.Errorf("invalid decimal - total length %d with exponent length %d", numBytes, len(expBytes))
	}

	return Decimal{annotations: ann, isSet: true, binary: data}, nil
}

// parseBinaryTimestamp parses a timestamp comprised of a required year and offset with
// optional month, day, hour, minute, second, and fractional sub-second components.
func parseBinaryTimestamp(ann []Symbol, lengthByte byte, r io.Reader) (Value, error) {
	numBytes, errLength := determineLength16(lengthByte, r)
	if errLength != nil {
		return nil, errors.WithMessage(errLength, "unable to parse length of timestamp")
	}

	/*
		Timestamp value |    6    |    L    |
		                +---------+---------+========+
		                :      length [VarUInt]      :
		                +----------------------------+
		                |      offset [VarInt]       |
		                +----------------------------+
		                |       year [VarUInt]       |
		                +----------------------------+
		                :       month [VarUInt]      :
		                +============================+
		                :         day [VarUInt]      :
		                +============================+
		                :        hour [VarUInt]      :
		                +====                    ====+
		                :      minute [VarUInt]      :
		                +============================+
		                :      second [VarUInt]      :
		                +============================+
		                : fraction_exponent [VarInt] :
		                +============================+
		                : fraction_coefficient [Int] :
		                +============================+
	*/
	// Sanity check.  Don't try to parse a timestamp of unreasonable length.
	// offset (2) year (2) month (1) day (1) hour (1) minute (1) second (1) exponent (1) coefficient (2).
	maxLength := 2 + 2 + 1 + 1 + 1 + 1 + 1 + 1 + 2
	if numBytes > uint16(maxLength) {
		return nil, errors.Errorf("timestamp length of %d exceeds expected maximum of %d", numBytes, maxLength)
	}

	// Offset = at least 1 byte, Year = at least 1 byte.
	if numBytes < 2 {
		return nil, errors.Errorf("timestamp must have a length of at least two bytes")
	}

	// Read in the entirety of the Timestamp value from the stream, then farm out those
	// bytes to read the constituent parts to ensure that we have a valid timestamp.
	data := make([]byte, numBytes)
	if n, err := r.Read(data); err != nil || n != int(numBytes) {
		return nil, errors.Errorf("unable to read timestamp - read %d bytes of %d with err: %v", n, numBytes, err)
	}

	dataReader := bytes.NewReader(data)
	offset, errOffset := readVarInt64(dataReader)
	if errOffset != nil {
		return nil, errors.WithMessage(errOffset, "unable to determine timestamp offset")
	}
	if offset >= 1440 || offset <= -1440 {
		return nil, errors.Errorf("invalid timestamp offset %d", offset)
	}

	year, errYear := readVarUInt16(dataReader)
	if errYear != nil {
		return nil, errors.WithMessage(errYear, "unable to determine timestamp year")
	}
	if year > 9999 {
		return nil, errors.Errorf("invalid year %d", year)
	}
	precision := TimestampPrecisionYear
	month, day, hour, minute, sec, nsec := uint8(1), uint8(1), uint8(0), uint8(0), uint8(0), uint32(0)

	var err error
	if dataReader.Len() > 0 {
		precision = TimestampPrecisionMonth
		if month, err = readVarUInt8(dataReader); err != nil {
			return nil, errors.WithMessage(err, "unable to determine timestamp month")
		}
		if month > 12 {
			return nil, errors.Errorf("invalid month %d", month)
		}
	}

	if dataReader.Len() > 0 {
		precision = TimestampPrecisionDay
		if day, err = readVarUInt8(dataReader); err != nil {
			return nil, errors.WithMessage(err, "unable to determine timestamp day")
		}
		if day > 31 {
			return nil, errors.Errorf("invalid day %d", day)
		}
	}

	if dataReader.Len() == 1 {
		// "The hour and minute is considered as a single component, that is, it is illegal
		// to have hour but not minute (and vice versa)."
		return nil, errors.New("invalid timestamp - cannot specify hours without minutes")
	}

	if dataReader.Len() > 0 {
		precision = TimestampPrecisionMinute
		if hour, err = readVarUInt8(dataReader); err != nil {
			return nil, errors.WithMessage(err, "unable to determine timestamp hour")
		}
		if hour > 23 {
			return nil, errors.Errorf("invalid hour %d", hour)
		}

		if minute, err = readVarUInt8(dataReader); err != nil {
			return nil, errors.WithMessage(err, "unable to determine timestamp minute")
		}
		if minute > 59 {
			return nil, errors.Errorf("invalid minute %d", minute)
		}
	}

	if dataReader.Len() > 0 {
		precision = TimestampPrecisionSecond
		if sec, err = readVarUInt8(dataReader); err != nil || sec > 59 {
			return nil, errors.WithMessage(err, "unable to determine timestamp second")
		}
		if sec > 59 {
			return nil, errors.Errorf("invalid second %d", sec)
		}
	}

	var exponent int8
	if dataReader.Len() > 0 {
		// "The fraction_exponent and fraction_coefficient denote the fractional seconds
		// of the timestamp as a decimal value. The fractional seconds’ value is
		// coefficient * 10 ^ exponent. It must be greater than or equal to zero and less
		// than 1. A missing coefficient defaults to zero. Fractions whose coefficient is
		// zero and exponent is greater than -1 are ignored."
		//
		// We expect the exponent to be a single byte.  That is able to cover a precision
		// of up to 63 digits which is excessive.
		exp, errExp := readVarUInt8(dataReader)
		if errExp != nil {
			return nil, errors.WithMessage(errExp, "unable to determine timestamp fractional second exponent")
		}
		exponent = int8(exp) & 0x3F
		if exp&0x40 != 0 {
			exponent *= -1
		}

		switch {
		case exponent > -1:
			// "Fractions whose coefficient is zero and exponent is greater than -1 are ignored."
			if dataReader.Len() == 0 {
				precision = TimestampPrecisionSecond
			}
		case exponent == -1:
			precision = TimestampPrecisionMillisecond1
		case exponent == -2:
			precision = TimestampPrecisionMillisecond2
		case exponent == -3:
			precision = TimestampPrecisionMillisecond3
		case exponent == -4:
			precision = TimestampPrecisionMillisecond4
		case exponent == -5:
			precision = TimestampPrecisionMicrosecond1
		case exponent == -6:
			precision = TimestampPrecisionMicrosecond2
		case exponent == -7:
			precision = TimestampPrecisionMicrosecond3
		case exponent == -8:
			precision = TimestampPrecisionMicrosecond4
		default:
			return nil, errors.Errorf("invalid exponent for timestamp fractional second: %#x", exp)
		}
	}

	if dataReader.Len() > 0 {
		coBytes := make([]byte, dataReader.Len())
		// We've already verified lengths and are basically performing a copy to
		// a pre-allocated byte slice.  There is no error to catch.
		_, _ = dataReader.Read(coBytes)

		var coefficient int16
		switch len(coBytes) {
		case 1:
			coefficient = int16(coBytes[0] & 0x7F)
		case 2:
			coefficient = int16(coBytes[0] & 0x7F)
			coefficient <<= 8
			coefficient |= int16(coBytes[1])
		}
		if coBytes[0]&0x80 != 0 {
			coefficient *= -1
		}

		switch {
		case coefficient < 0:
			// "It must be greater than or equal to zero and less than 1."
			// A negative coefficient can't be greater than or equal to zero.
			return nil, errors.Errorf("negative coefficient is not legal")
		case coefficient == 0 && exponent > -1:
			// "Fractions whose coefficient is zero and exponent is greater than -1 are ignored."
			precision = TimestampPrecisionSecond
		default:
			fraction := math.Pow10(int(exponent)) * float64(coefficient)
			if fraction >= 1 || fraction < 0 {
				return nil, errors.Errorf("invalid fractional seconds: %F", fraction)
			}
			nsec = uint32(fraction * float64(time.Second))
		}
	}

	// Ignore the offset if we don't have a time component.
	if precision <= TimestampPrecisionDay {
		offset = 0
	}

	loc := time.FixedZone("", int(offset))
	timestamp := time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(sec), int(nsec), loc)
	// time.Date does a translation, with invalid month/day combinations (e.g. April 31) being
	// reflected as overflows into the next month (e.g. May 1).
	if timestamp.Month() != time.Month(month) {
		return nil, errors.Errorf("invalid year / month / day combination: %d %d %d", year, month, day)
	}

	return Timestamp{annotations: ann, precision: precision, binary: data, value: timestamp}, nil
}
