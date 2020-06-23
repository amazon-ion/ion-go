package main

import (
	"io"
	"os"

	"github.com/amzn/ion-go/ion"
)

type stdin struct{}

func (stdin) Read(bs []byte) (int, error) { return os.Stdin.Read(bs) }
func (stdin) Close() error                { return nil }

// OpenInput opens an input stream.
func OpenInput(in string) (io.ReadCloser, error) {
	r, err := os.Open(in)
	if err != nil {
		return nil, err
	}
	return r, nil
}

type uncloseable struct {
	w io.Writer
}

func (u uncloseable) Write(bs []byte) (int, error) {
	return u.w.Write(bs)
}

func (u uncloseable) Close() error {
	return nil
}

// OpenOutput opens the output stream.
func OpenOutput(outf string) (io.WriteCloser, error) {
	if outf == "" {
		return uncloseable{os.Stdout}, nil
	}
	return os.OpenFile(outf, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
}

// OpenError opens the error stream.
func OpenError(errf string) (io.WriteCloser, error) {
	if errf == "" {
		return uncloseable{os.Stderr}, nil
	}
	return os.OpenFile(errf, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
}

// ErrorReport is a (serialized) report of errors that occur during processing.
type ErrorReport struct {
	w *ion.Encoder
}

// NewErrorReport creates a new ErrorReport.
func NewErrorReport(w io.Writer) *ErrorReport {
	return &ErrorReport{
		w: ion.NewTextEncoder(w),
	}
}

// Append appends an error to this report.
func (r *ErrorReport) Append(typ errortype, msg, loc string, idx int) {
	if err := r.w.Encode(errordescription{typ, msg, loc, idx}); err != nil {
		panic(err)
	}
}

// Finish finishes writing this report.
func (r *ErrorReport) Finish() {
	r.w.Finish()
}
