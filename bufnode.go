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
	WriteTo(w io.Writer) error
}

// An atom is a value that has been fully serialized and can be written directly.
type atom []byte

func (a atom) Len() uint64 {
	return uint64(len(a))
}

func (a atom) WriteTo(w io.Writer) error {
	_, err := w.Write(a)
	return err
}

// A fieldname is the symbol id of a field name inside a struct.
type fieldname uint64

func (f fieldname) Len() uint64 {
	return varUintLen(uint64(f))
}

func (f fieldname) WriteTo(w io.Writer) error {
	_, err := w.Write(packVarUint(uint64(f)))
	return err
}

// A container holds multiple child values and serializes them together with a
// tag and length on demand.
type container struct {
	code     byte
	len      uint64
	children []bufnode
}

func (c *container) Add(n bufnode) {
	c.len += n.Len()
	c.children = append(c.children, n)
}

func (c *container) Len() uint64 {
	if c.len < 0x0E {
		// Short tag
		return c.len + 1
	}
	// Long tag.
	return c.len + varUintLen(c.len) + 1
}

func (c *container) WriteTo(w io.Writer) error {
	if err := writeTag(w, c.code, c.len); err != nil {
		return nil
	}

	for _, child := range c.children {
		if err := child.WriteTo(w); err != nil {
			return err
		}
	}

	return nil
}

func writeTag(w io.Writer, code byte, len uint64) error {
	if len < 0x0E {
		// Short form, with length embedded in code byte.
		_, err := w.Write([]byte{code | byte(len)})
		return err
	}

	// Long form, with separate length.
	if _, err := w.Write([]byte{code | 0x0E}); err != nil {
		return err
	}
	_, err := w.Write(packVarUint(uint64(len)))
	return err
}
