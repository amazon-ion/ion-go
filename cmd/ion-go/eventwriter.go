package main

import (
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/amzn/ion-go/ion"
)

type eventwriter struct {
	enc *ion.Encoder

	depth       int
	fieldname   string
	annotations []string
}

// NewEventWriter creates an ion.Writer that writes out a sequence
// of ion-test-driver events.
func NewEventWriter(out io.Writer) ion.Writer {
	w := ion.NewTextWriter(out)
	w.WriteSymbol("$ion_event_stream")

	return &eventwriter{enc: ion.NewEncoder(w)}
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

func (e *eventwriter) WriteNull() error {
	return e.write(event{
		EventType: scalar,
		IonType:   "NULL",
		ValueText: "null",
	})
}

func (e *eventwriter) WriteNullType(val ion.Type) error {
	ts := val.String()
	return e.write(event{
		EventType: scalar,
		IonType:   strings.ToUpper(ts),
		ValueText: "null." + ts,
	})
}

func (e *eventwriter) WriteBool(val bool) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "BOOL",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) WriteInt(val int64) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "INT",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) WriteUint(val uint64) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "INT",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) WriteBigInt(val *big.Int) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "INT",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) WriteFloat(val float64) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "FLOAT",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) WriteDecimal(val *ion.Decimal) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "DECIMAL",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) WriteTimestamp(val time.Time) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "TIMESTAMP",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) WriteSymbol(val string) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "SYMBOL",
		ValueText: symbolify(val),
	})
}

func (e *eventwriter) WriteString(val string) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "STRING",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) WriteClob(val []byte) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "CLOB",
		ValueText: clobify(val),
	})
}

func (e *eventwriter) WriteBlob(val []byte) error {
	return e.write(event{
		EventType: scalar,
		IonType:   "BLOB",
		ValueText: stringify(val),
	})
}

func (e *eventwriter) BeginList() error {
	err := e.write(event{
		EventType: containerStart,
		IonType:   "LIST",
	})
	if err != nil {
		return err
	}
	e.depth++
	return nil
}

func (e *eventwriter) EndList() error {
	e.depth--
	return e.write(event{
		EventType: containerEnd,
		IonType:   "LIST",
	})
}

func (e *eventwriter) BeginSexp() error {
	err := e.write(event{
		EventType: containerStart,
		IonType:   "SEXP",
	})
	if err != nil {
		return err
	}
	e.depth++
	return nil
}

func (e *eventwriter) EndSexp() error {
	e.depth--
	return e.write(event{
		EventType: containerEnd,
		IonType:   "SEXP",
	})
}

func (e *eventwriter) BeginStruct() error {
	err := e.write(event{
		EventType: containerStart,
		IonType:   "STRUCT",
	})
	if err != nil {
		return err
	}
	e.depth++
	return nil
}

func (e *eventwriter) EndStruct() error {
	e.depth--
	return e.write(event{
		EventType: containerEnd,
		IonType:   "STRUCT",
	})
}

func (e *eventwriter) Finish() error {
	if err := e.write(event{EventType: streamEnd}); err != nil {
		return err
	}
	return e.enc.Finish()
}

func stringify(val interface{}) string {
	bs, err := ion.MarshalText(val)
	if err != nil {
		panic(err)
	}
	return string(bs)
}

func symbolify(val string) string {
	buf := strings.Builder{}
	w := ion.NewTextWriterOpts(&buf, ion.TextWriterQuietFinish)

	w.WriteSymbol(val)
	if err := w.Finish(); err != nil {
		panic(err)
	}

	return buf.String()
}

func clobify(val []byte) string {
	buf := strings.Builder{}
	w := ion.NewTextWriterOpts(&buf, ion.TextWriterQuietFinish)

	w.WriteClob(val)
	if err := w.Finish(); err != nil {
		panic(err)
	}

	return buf.String()
}

func (e *eventwriter) write(ev event) error {
	name := e.fieldname
	e.fieldname = ""
	annos := e.annotations
	e.annotations = nil

	if name != "" {
		ev.FieldName = &token{Text: name}
	}

	if len(annos) > 0 {
		asyms := make([]token, len(annos))
		for i, a := range annos {
			asyms[i] = token{Text: a}
		}
		ev.Annotations = asyms
	}

	ev.Depth = e.depth

	return e.enc.Encode(ev)
}
