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
	"io"
)

// Writing binary ion is a bit tricky: values are preceded by their length,
// which can be hard to predict until we've actually written out the value.
// To make matters worse, we can't predict the length of the /length/ ahead
// of time in order to reserve space for it, because it uses a variable-length
// encoding. To avoid copying bytes around all over the place, we write into
// an in-memory tree structure, which we then blast out to the actual io.Writer
// once all the relevant lengths are known.

// A bufnode is a node in the partially-serialized tree.
type bufnode interface {
	Len() uint64
	EmitTo(w io.Writer) error
}

// A bufseq is a bufnode that's also an appendable sequence of bufnodes.
type bufseq interface {
	bufnode
	Append(n bufnode)
}

var _ bufnode = atom([]byte{})
var _ bufseq = &datagram{}
var _ bufseq = &container{}

// An atom is a value that has been fully serialized and can be emitted directly.
type atom []byte

func (a atom) Len() uint64 {
	return uint64(len(a))
}

func (a atom) EmitTo(w io.Writer) error {
	_, err := w.Write(a)
	return err
}

// A datagram is a sequence of nodes that will be emitted one
// after another. Most notably, used to buffer top-level values
// when we haven't yet finalized the local symbol table.
type datagram struct {
	len      uint64
	children []bufnode
}

func (d *datagram) Append(n bufnode) {
	d.len += n.Len()
	d.children = append(d.children, n)
}

func (d *datagram) Len() uint64 {
	return d.len
}

func (d *datagram) EmitTo(w io.Writer) error {
	for _, child := range d.children {
		if err := child.EmitTo(w); err != nil {
			return err
		}
	}

	return nil
}

// A container is a datagram that's preceded by a code+length tag.
type container struct {
	code byte
	datagram
}

func (c *container) Len() uint64 {
	if c.len < 0x0E {
		return c.len + 1
	}
	return c.len + (varUintLen(c.len) + 1)
}

func (c *container) EmitTo(w io.Writer) error {
	var arr [11]byte
	buf := arr[:0]
	buf = appendTag(buf, c.code, c.len)

	if _, err := w.Write(buf); err != nil {
		return err
	}
	return c.datagram.EmitTo(w)
}

// A bufstack is a stack of bufseqs, more or less matching the
// stack of BeginList/Sexp/Struct calls made on a binaryWriter.
// The top of the stack is the sequence we're currently writing
// values into; when it's popped off, it will be appended to the
// bufseq below it.
type bufstack struct {
	arr []bufseq
}

func (s *bufstack) peek() bufseq {
	if len(s.arr) == 0 {
		return nil
	}
	return s.arr[len(s.arr)-1]
}

func (s *bufstack) push(b bufseq) {
	s.arr = append(s.arr, b)
}

func (s *bufstack) pop() {
	if len(s.arr) == 0 {
		panic("pop called on an empty stack")
	}
	s.arr = s.arr[:len(s.arr)-1]
}
