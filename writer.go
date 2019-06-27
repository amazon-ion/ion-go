package ion

import (
	"errors"
	"io"
)

// writer holds shared stuff for all writers.
type writer struct {
	out io.Writer
	ctx ctx
	err error

	fieldName       string
	typeAnnotations []string
}

// InStruct returns true if we're currently writing a struct.
func (w *writer) InStruct() bool {
	return w.ctx.peek() == ctxInStruct
}

// InList returns true if we're currently writing a list.
func (w *writer) InList() bool {
	return w.ctx.peek() == ctxInList
}

// InSexp returns true if we're currently writing an s-expression.
func (w *writer) InSexp() bool {
	return w.ctx.peek() == ctxInSexp
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
