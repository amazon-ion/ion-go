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
	"bufio"
	"bytes"
	"io"
	"strings"
)

type Reader interface {
	// Positions this Reader on the next position in the current value stream, returning the Event.
	// Once positioned, the contents of this Event can be accessed.
	Next() Event

	// Steps into the current value if it is a container. It returns an error if there is no current value or if
	// the value is not a container. On success, the Reader is positioned before the first value in the container.
	StepIn() error

	// Steps out of the current container value being read. It returns an error if this Reader is not currently
	// stepped into a container. On success, the Reader is positioned after the end of the container, but before any
	// subsequent values in the stream.
	StepOut() error
}

// Creates a new Ion reader of the appropriate type by peeking
// at the first several bytes of input for a binary version marker.
func NewReader(in io.Reader) Reader {
	return NewReaderCatalog(in, nil)
}

// Creates a new reader from a string.
func NewReaderStr(str string) Reader {
	return NewReader(strings.NewReader(str))
}

// Creates a new reader for the given bytes.
func NewReaderBytes(in []byte) Reader {
	return NewReader(bytes.NewReader(in))
}

// Creates a new reader with the given catalog.
func NewReaderCatalog(in io.Reader, cat Catalog) Reader {
	br := bufio.NewReader(in)

	// Determine if binary data, so check the binary version marker.
	bs, err := br.Peek(4)
	if err == nil && bs[0] == 0xE0 && bs[1] == 0x01 && bs[2] == 0x00 && bs[3] == 0xEA {
		return newBinaryReader(br, cat)
	}

	return newTextReader(br)
}
