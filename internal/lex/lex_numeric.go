/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

package lex

import (
	"bytes"
)

const (
	// One of these runes must follow a decimal, float, int, or timestamp.
	numericStopRunes = ", \t\n\r{}[]()\"'\v\f"

	binaryDigits  = "_01"
	decimalDigits = "_0123456789"
	hexDigits     = "_0123456789abcdefABCDEF"
)

// This file contains the state functions for lexing all numeric types: decimal, float,
// int, and timestamp.

// lexNumber scans a number: decimal, float, int (base 10, 16, or 2), or timestamp.
// Returns lexValue.
func lexNumber(x *Lexer) stateFn {
	// Optional leading sign.
	hasSign := x.accept("-")
	runeCount := 0
	hasDot := false

	// Handle infinity: "+inf" or "-inf".
	if x.acceptString("+inf") || (hasSign && x.acceptString("inf")) {
		x.emit(IonInfinity)
		return lexValue
	}

	if hasSign && x.accept("_") {
		return x.error("underscore must not be after negative sign")
	}

	// Default to base10, but look for different potential valid character sets based
	// on whether or not the first number is a 0.
	validRunes := decimalDigits
	it := IonInt
	if x.accept("0") {
		runeCount++
		if x.accept("xX") {
			validRunes = hexDigits
			it = IonIntHex
			runeCount++
		} else if x.accept("bB") {
			validRunes = binaryDigits
			it = IonIntBinary
			runeCount++
		}
	}

	// We started with 0x or some variant, so the next character must not be an underscore.
	if validRunes != decimalDigits && x.accept("_") {
		return x.error("underscore must not be at start of hex or binary number")
	}

	// Hit the first stop point.  Make sure that the end of the stop point wasn't
	// an underscore.  If we get to this point then we have read at least one rune
	// so it is safe to backup.
	runeCount += x.acceptRun(validRunes, "_")

	x.backup()
	if ch := x.next(); ch == '_' {
		return x.error("number span cannot end with an underscore")
	}

	// We can only continue on to further stop points if we are dealing with decimal digits.
	if validRunes == decimalDigits {
		// We can have a period and then E or D, but we can't have E or D then a period.
		if x.accept(".") {
			hasDot = true
			runeCount++
			it = IonDecimal
			if x.peek() == '_' {
				return x.error("underscore may not follow a period")
			}
			x.acceptRun(validRunes, "_")
		}
		// Finally attempt to pull in everything after a float or decimal designator.
		if x.accept("eE") {
			it = IonFloat
			// Exponents are allowed to have a no sign, a plus sign, or a minus sign.
			x.accept("+-")
			x.acceptRun(validRunes, "_")
		} else if x.accept("dD") {
			it = IonDecimal
			// Exponents are allowed to have a no sign, a plus sign, or a minus sign.
			x.accept("+-")
			x.acceptRun(validRunes, "_")
		}
	}

	// If you're an int, use decimal digits, don't have a sign or a dot and you have
	// exactly four numbers before hitting a stop character, you might be a timestamp.
	mightBeTimestamp := it == IonInt && validRunes == decimalDigits && runeCount == 4 && !hasSign && !hasDot
	switch ch := x.next(); {
	case ch == 'T' && mightBeTimestamp:
		// Four numbers followed by a T is most likely a year-only timestamp.
		if x.input[x.pos-5] == '0' && x.input[x.pos-4] == '0' && x.input[x.pos-3] == '0' && x.input[x.pos-2] == '0' {
			return x.error("year must be greater than zero")
		}
		x.emit(IonTimestamp)
		return lexValue
	case ch == '-' && mightBeTimestamp:
		// Four numbers followed by a - is most likely a timestamp.
		if x.input[x.pos-5] == '0' && x.input[x.pos-4] == '0' && x.input[x.pos-3] == '0' && x.input[x.pos-2] == '0' {
			return x.error("year must be greater than zero")
		}
		return lexTimestamp
	case isNumericStop(ch) || ch == eof:
	// Do nothing, number terminated as expected.
	default:
		return x.errorf("invalid numeric stop character: %#U", ch)
	}

	// We consumed one character past the number, so back up.
	x.backup()

	if x.input[x.pos-1] == '_' {
		return x.error("numbers cannot end with an underscore")
	}
	if it == IonInt && x.input[x.itemStart] == '0' && x.itemStart < x.pos-1 {
		return x.error("leading zeros are not allowed for decimal integers")
	}
	if it == IonFloat && x.input[x.itemStart] == '0' && !bytes.ContainsRune([]byte(".eE"), rune(x.input[x.itemStart+1])) {
		return x.error("leading zeros are not allowed for floats")
	}
	if it == IonDecimal && x.input[x.itemStart] == '0' && !bytes.ContainsRune([]byte(".dD"), rune(x.input[x.itemStart+1])) {
		return x.error("leading zeros are not allowed for decimals")
	}

	x.emit(it)
	return lexValue
}

// lexTimestamp lexes everything past the first "-" in a timestamp.  It is assumed that
// the year and dash have been consumed.
func lexTimestamp(x *Lexer) stateFn {
	// Set defaults for all our our values so that we can safely check them all
	// later without worrying about which ones were set.
	year := [4]byte{x.input[x.pos-5], x.input[x.pos-4], x.input[x.pos-3], x.input[x.pos-2]}
	month := [2]byte{'0', '1'}
	day, hour, hourOffset := month, month, month

	// Overall form can be some subset of yyyy-mm-ddThh:mm:ss.sssTZD.  The "yyyy-"
	// has already been lexed.  Comments will show the progression in parsing.
	// yyyy-mm
	if !isMonthStart(x.next()) || !isNumber(x.next()) {
		x.backup()
		return x.errorf("invalid character as month part of timestamp: %#U", x.next())
	}
	month[0], month[1] = x.input[x.pos-2], x.input[x.pos-1]

	// yyyy-mmT or yyyy-mm-
	switch ch := x.next(); {
	case ch == 'T':
		if pk := x.peek(); !isNumericStop(pk) && pk != eof {
			return x.errorf("invalid timestamp stop character: %#U", pk)
		}
		return validateDateAndEmit(x, year, month, day, hour, hourOffset)
	case ch == '-':
	// Do nothing, a dash means we're going into days.
	default:
		return x.errorf("invalid character after month part of timestamp: %#U", ch)
	}

	// yyyy-mm-dd
	if !isDayStart(x.next()) || !isNumber(x.next()) {
		x.backup()
		return x.errorf("invalid character as day part of timestamp: %#U", x.next())
	}
	day[0], day[1] = x.input[x.pos-2], x.input[x.pos-1]

	// The day portion does not need to be terminated by a 'T' to be a valid timestamp.
	// yyyy-mm-dd or yyyy-mm-ddT
	switch ch := x.next(); {
	case ch == 'T':
		if pk := x.peek(); isNumericStop(pk) || pk == eof {
			return validateDateAndEmit(x, year, month, day, hour, hourOffset)
		}
	case isNumericStop(ch):
		x.backup()
		return validateDateAndEmit(x, year, month, day, hour, hourOffset)
	default:
		return x.errorf("invalid character after day part of timestamp: %#U", ch)
	}

	// yyyy-mm-ddThh:mm
	if !isHourStart(x.next()) || !isNumber(x.next()) || x.next() != ':' || !isMinuteStart(x.next()) || !isNumber(x.next()) {
		x.backup()
		return x.errorf("invalid character as hour/minute part of timestamp: %#U", x.next())
	}
	hour[0], hour[1] = x.input[x.pos-5], x.input[x.pos-4]

	// yyyy-mm-ddThh:mm:ss(.sss)?
	if x.peek() == ':' {
		x.next()
		// yyyy-mm-ddThh:mm:ss
		if !isMinuteStart(x.next()) || !isNumber(x.next()) {
			x.backup()
			return x.errorf("invalid character as seconds part of timestamp: %#U", x.next())
		}
		// yyyy-mm-ddThh:mm:ss.sss (can be any number of digits)
		if x.peek() == '.' {
			x.next()
			// There must be at least one number after the period.
			if !isNumber(x.peek()) {
				return x.error("missing fractional seconds value")
			}
			for isNumber(x.peek()) {
				x.next()
			}
		}
	}

	// If the time is included, then there must be a timezone component.
	// https://www.w3.org/TR/NOTE-datetime
	// yyyy-mm-ddThh:mm:ss(.sss)?TZD
	switch ch := x.next(); {
	case ch == '+' || ch == '-':
		// TZD == +hh:mm or -hh:mm
		if !isHourStart(x.next()) || !isNumber(x.next()) || x.next() != ':' || !isMinuteStart(x.next()) || !isNumber(x.next()) {
			x.backup()
			return x.errorf("invalid character as hour/minute part of timezone: %#U", x.next())
		}
		hourOffset[0], hourOffset[1] = x.input[x.pos-5], x.input[x.pos-4]
	case ch == 'Z':
		// Do nothing. 'Z' is a great way to end a timestamp.
	default:
		return x.errorf("invalid character as timezone part of timestamp: %#U", ch)
	}

	if ch := x.peek(); !isNumericStop(ch) && ch != eof {
		return x.errorf("invalid timestamp stop character: %#U", ch)
	}

	return validateDateAndEmit(x, year, month, day, hour, hourOffset)
}

func validateDateAndEmit(x *Lexer, year [4]byte, month, day, hour, hourOffset [2]byte) stateFn {
	monthInt := ((month[0] - '0') * 10) + (month[1] - '0')
	if monthInt > 12 {
		return x.errorf("invalid month %d", monthInt)
	}
	if monthInt == 0 {
		return x.error("month must be greater than zero")
	}

	dayInt := ((day[0] - '0') * 10) + (day[1] - '0')
	if dayInt > 31 {
		return x.errorf("invalid day %d", dayInt)
	}
	if dayInt == 0 {
		return x.error("day must be greater than zero")
	}
	if (monthInt == 4 || monthInt == 6 || monthInt == 9 || monthInt == 11) && dayInt == 31 {
		return x.errorf("invalid day %d for month %d", dayInt, monthInt)
	}
	// Only care about the year if we are in February so that we can calculate whether
	// or not it is a leap year.
	if monthInt == 2 {
		yearInt := (int(year[0]-'0') * 1000) + (int(year[1]-'0') * 100) + (int(year[2]-'0') * 10) + int(year[3]-'0')
		isLeapYear := (yearInt%4 == 0 && yearInt%100 != 0) || (yearInt%400 == 0)
		if (isLeapYear && dayInt >= 30) || (!isLeapYear && dayInt >= 29) {
			return x.errorf("invalid day %d for month %d in year %d", dayInt, monthInt, yearInt)
		}
	}

	if hour[0] == '2' && hour[1] > '3' {
		return x.errorf("invalid hour %s", hour)
	}
	if hourOffset[0] == '2' && hourOffset[1] > '3' {
		return x.errorf("invalid hour offset %s", hourOffset)
	}

	x.emit(IonTimestamp)

	return lexValue
}

// isNumericStart returns if the given rune is a valid start of an decimal,
// float, int, or timestamp
func isNumericStart(ch rune) bool {
	return isNumber(ch) || ch == '-' || ch == '+'
}

// isNumber returns if the given rune is a number between 0-9.
func isNumber(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

// isMonthStart returns if the given rune is a 0 or a 1.
func isMonthStart(ch rune) bool {
	return ch == '0' || ch == '1'
}

// isHourStart returns if the given rune is a number between 0-2.
func isHourStart(ch rune) bool {
	return '0' <= ch && ch <= '2'
}

// isDayStart returns if the given rune is a number between 0-3.
func isDayStart(ch rune) bool {
	return '0' <= ch && ch <= '3'
}

// isMinuteStart returns if the given rune is a number between 0-5.
func isMinuteStart(ch rune) bool {
	return '0' <= ch && ch <= '5'
}

// isNumericStop returns true if the given rune is one of the numeric/timestamp stop chars.
func isNumericStop(ch rune) bool {
	return bytes.ContainsRune([]byte(numericStopRunes), ch)
}
