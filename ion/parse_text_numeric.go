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
