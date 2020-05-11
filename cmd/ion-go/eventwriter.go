package main

import (
	"io"
	"math/big"
	"time"

	"github.com/amzn/ion-go/ion"
)

type eventwriter struct {
	w *ion.Encoder

	depth       int
	fieldname   string
	annotations []string
}

// NewEventWriter creates an ion.Writer that writes out a sequence
// of ion-test-driver events.
func NewEventWriter(out io.Writer) ion.Writer {
	w := ion.NewTextWriter(out)
	w.WriteSymbol("$ion_event_stream")

	e := ion.NewEncoder(w)
	return &eventwriter{w: e}
}

func (e *eventwriter) FieldName(val string) error {
	e.fieldname = val
	return nil
}

func (e *eventwriter) Annotation(val string) error {
	e.annotations = append(e.annotations, val)
	return nil
}

func (e *eventwriter) Annotations(vals ...string) error {
	e.annotations = append(e.annotations, vals...)
	return nil
}

// TODO: Implement these.

func (e *eventwriter) WriteNull() error {
	return nil
}

func (eventwriter) WriteNullType(ion.Type) error {
	return nil
}

func (eventwriter) WriteBool(bool) error {
	return nil
}

func (eventwriter) WriteInt(int64) error {
	return nil
}

func (eventwriter) WriteUint(uint64) error {
	return nil
}

func (eventwriter) WriteBigInt(*big.Int) error {
	return nil
}

func (eventwriter) WriteFloat(float64) error {
	return nil
}

func (eventwriter) WriteDecimal(*ion.Decimal) error {
	return nil
}

func (eventwriter) WriteTimestamp(time.Time) error {
	return nil
}

func (eventwriter) WriteSymbol(string) error {
	return nil
}

func (eventwriter) WriteString(string) error {
	return nil
}

func (eventwriter) WriteClob([]byte) error {
	return nil
}

func (eventwriter) WriteBlob([]byte) error {
	return nil
}

func (eventwriter) BeginList() error {
	return nil
}

func (eventwriter) EndList() error {
	return nil
}

func (eventwriter) BeginSexp() error {
	return nil
}

func (eventwriter) EndSexp() error {
	return nil
}

func (eventwriter) BeginStruct() error {
	return nil
}

func (eventwriter) EndStruct() error {
	return nil
}

func (eventwriter) Finish() error {
	return nil
}
