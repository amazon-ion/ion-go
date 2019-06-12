package ion

import (
	"errors"
	"io"
)

type ctxType byte

const (
	atTopLevelCtx ctxType = iota
	inStructCtx
	inListCtx
	inSexpCtx
)

// writer holds shared stuff for all writers.
type writer struct {
	out    io.Writer
	ctxArr []ctxType
	err    error

	fieldName       string
	typeAnnotations []string
}

// InStruct returns true if we're currently writing a struct.
func (w *writer) InStruct() bool {
	return w.ctx() == inStructCtx
}

// InList returns true if we're currently writing a list.
func (w *writer) InList() bool {
	return w.ctx() == inListCtx
}

// InSexp returns true if we're currently writing an s-expression.
func (w *writer) InSexp() bool {
	return w.ctx() == inSexpCtx
}

// Err returns the current error, or nil if there are none yet.
func (w *writer) Err() error {
	return w.err
}

// FieldName sets the field name for the next value written.
// It may only be called while writing a struct.
func (w *writer) FieldName(val string) {
	if w.err != nil {
		return
	}
	if !w.InStruct() {
		w.err = errors.New("FieldName() called but not writing a struct")
		return
	}
	w.fieldName = val
}

// TypeAnnotation adds a type annotation to the next value written.
func (w *writer) TypeAnnotation(val string) {
	if w.err != nil {
		return
	}
	w.typeAnnotations = append(w.typeAnnotations, val)
}

// TypeAnnotations adds one or more type annotations to the next value
// written.
func (w *writer) TypeAnnotations(val ...string) {
	if w.err != nil {
		return
	}
	w.typeAnnotations = append(w.typeAnnotations, val...)
}

// ctx returns the current writing context
func (w *writer) ctx() ctxType {
	if len(w.ctxArr) == 0 {
		return atTopLevelCtx
	}
	return w.ctxArr[len(w.ctxArr)-1]
}

// push pushes a new writing context when a new container is begun.
func (w *writer) push(ctx ctxType) {
	w.ctxArr = append(w.ctxArr, ctx)
}

// pop pops the writing context when a container is ended.
func (w *writer) pop() {
	if len(w.ctxArr) == 0 {
		panic("pop called at top level")
	}
	w.ctxArr = w.ctxArr[:len(w.ctxArr)-1]
}
