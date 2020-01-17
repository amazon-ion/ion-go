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

import (
	"bytes"
	"encoding/base64"
)

// This file contains the binary-like types Blob and Clob.

const (
	textNullBlob = "null.blob"
	textNullClob = "null.clob"
)

// Blob is binary data of user-defined encoding.
type Blob struct {
	annotations []Symbol
	binary      []byte
	text        []byte
}

// Annotations satisfies Value.
func (b Blob) Annotations() []Symbol {
	return b.annotations
}

// Value returns the Base64 encoded version of the Blob.
func (b Blob) Value() []byte {
	return b.Text()
}

// Binary returns the raw binary representation of the Blob.  If the
// representation was originally text and there is a problem decoding it,
// then this will panic.  This is because it is assumed that the original
// parsing of the text value will catch improperly formatted encodings.
func (b Blob) Binary() []byte {
	if len(b.binary) != 0 || len(b.text) == 0 {
		return b.binary
	}

	// Trim any whitespace characters from the text representation of
	// the Blob.
	trimmedText := bytes.Map(func(r rune) rune {
		if r == ' ' || r == '\r' || r == '\n' || r == '\t' || r == '\f' || r == '\v' {
			return -1
		}
		return r
	}, b.text)

	b.binary = make([]byte, base64.StdEncoding.DecodedLen(len(trimmedText)))
	if _, err := base64.StdEncoding.Decode(b.binary, trimmedText); err != nil {
		panic(err)
	}
	return b.binary
}

// Text returns the Base64 encoded version of the Blob.
func (b Blob) Text() []byte {
	if b.IsNull() {
		return []byte(textNullBlob)
	}

	if len(b.text) != 0 || len(b.binary) == 0 {
		return b.text
	}

	b.text = make([]byte, base64.StdEncoding.EncodedLen(len(b.binary)))
	base64.StdEncoding.Encode(b.text, b.binary)
	return b.text
}

// IsNull satisfies Value.
func (b Blob) IsNull() bool {
	return b.binary == nil && b.text == nil
}

// Type satisfies Value.
func (b Blob) Type() Type {
	return TypeBlob
}

// Clob is text data of user-defined encoding.  It is a binary type that is
// designed for binary values that are either text encoded in a code page that
// is ASCII compatible or should be octet editable by a human (escaped string
// syntax vs. base64 encoded data).
type Clob struct {
	annotations []Symbol
	text        []byte
}

// Value returns the single string version of the Clob.
func (c Clob) Value() string {
	if c.IsNull() {
		return ""
	}

	return string(c.text)
}

// Annotations satisfies Value.
func (c Clob) Annotations() []Symbol {
	return c.annotations
}

// Binary returns the raw binary representation of the Clob.  Because the binary
// format is represented directly as the octet values this returns the same
// representation as Text().
func (c Clob) Binary() []byte {
	return c.text
}

// Text returns a text representation of the Clob.
func (c Clob) Text() []byte {
	if c.IsNull() {
		return []byte(textNullClob)
	}

	return c.text
}

// IsNull satisfies Value.
func (c Clob) IsNull() bool {
	return c.text == nil
}

// Type satisfies Value.
func (c Clob) Type() Type {
	return TypeClob
}
