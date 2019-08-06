package ion

type ctxType byte

const (
	ctxAtTopLevel ctxType = iota
	ctxInStruct
	ctxInList
	ctxInSexp
)

// ctx is a context stack.
type ctx struct {
	stack []ctxType
}

// peek returns the current context.
func (c *ctx) peek() ctxType {
	if len(c.stack) == 0 {
		return ctxAtTopLevel
	}
	return c.stack[len(c.stack)-1]
}

// push pushes a new context onto the stack.
func (c *ctx) push(ctx ctxType) {
	c.stack = append(c.stack, ctx)
}

// pop pops the top context off the stack.
func (c *ctx) pop() {
	if len(c.stack) == 0 {
		panic("pop called at top level")
	}
	c.stack = c.stack[:len(c.stack)-1]
}
