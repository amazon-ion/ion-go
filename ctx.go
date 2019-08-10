package ion

import "fmt"

// ctx is the current reader or writer context.
type ctx uint8

const (
	ctxAtTopLevel ctx = iota
	ctxInStruct
	ctxInList
	ctxInSexp
)

func ctxToContainerType(c ctx) Type {
	switch c {
	case ctxInList:
		return ListType
	case ctxInSexp:
		return SexpType
	case ctxInStruct:
		return StructType
	default:
		return NoType
	}
}

func containerTypeToCtx(t Type) ctx {
	switch t {
	case ListType:
		return ctxInList
	case SexpType:
		return ctxInSexp
	case StructType:
		return ctxInStruct
	default:
		panic(fmt.Sprintf("type %v is not a container type", t))
	}
}

// ctxstack is a context stack.
type ctxstack struct {
	arr []ctx
}

// peek returns the current context.
func (c *ctxstack) peek() ctx {
	if len(c.arr) == 0 {
		return ctxAtTopLevel
	}
	return c.arr[len(c.arr)-1]
}

// push pushes a new context onto the stack.
func (c *ctxstack) push(ctx ctx) {
	c.arr = append(c.arr, ctx)
}

// pop pops the top context off the stack.
func (c *ctxstack) pop() {
	if len(c.arr) == 0 {
		panic("pop called at top level")
	}
	c.arr = c.arr[:len(c.arr)-1]
}
