/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

package ion

// parseDecimal parses a decimal value and counts the number of characters
// in the coefficient.
func (p *parser) parseDecimal(annotations []Symbol) Decimal {
	text := p.next().Val

	// TODO: Properly calculate and track the precision of the decimal.

	return Decimal{annotations: annotations, isSet: true, text: text}
}

// parseFloat parses the next value as a float.
func (p *parser) parseFloat(annotations []Symbol) Float {
	return Float{annotations: annotations, isSet: true, text: p.next().Val}
}

// parseInt parses an int for the given base.
func (p *parser) parseInt(annotations []Symbol, base intBase) Int {
	text := p.next().Val
	// An empty slice of bytes is not a valid int, so we're going to make the assumption
	// that we can check the first element of the text slice.
	return Int{annotations: annotations, isSet: true, base: base, isNegative: text[0] == '-', text: text}
}

// parseTimestamp parses the next value as a Timestamp.
func (p *parser) parseTimestamp(annotations []Symbol) Timestamp {
	return Timestamp{annotations: annotations, text: p.next().Val}
}
