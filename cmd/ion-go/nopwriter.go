package main

import (
	"math/big"

	"github.com/amzn/ion-go/ion"
)

type nopwriter struct{}

// NewNopWriter returns a no-op Ion writer.
func NewNopWriter() ion.Writer {
	return nopwriter{}
}

func (nopwriter) FieldName(string) error {
	return nil
}

func (nopwriter) Annotation(string) error {
	return nil
}

func (nopwriter) Annotations(...string) error {
	return nil
}

func (nopwriter) WriteNull() error {
	return nil
}

func (nopwriter) WriteNullType(ion.Type) error {
	return nil
}

func (nopwriter) WriteBool(bool) error {
	return nil
}

func (nopwriter) WriteInt(int64) error {
	return nil
}

func (nopwriter) WriteUint(uint64) error {
	return nil
}

func (nopwriter) WriteBigInt(*big.Int) error {
	return nil
}

func (nopwriter) WriteFloat(float64) error {
	return nil
}

func (nopwriter) WriteDecimal(*ion.Decimal) error {
	return nil
}

func (nopwriter) WriteTimestamp(ion.Timestamp) error {
	return nil
}

func (nopwriter) WriteSymbol(string) error {
	return nil
}

func (nopwriter) WriteString(string) error {
	return nil
}

func (nopwriter) WriteClob([]byte) error {
	return nil
}

func (nopwriter) WriteBlob([]byte) error {
	return nil
}

func (nopwriter) BeginList() error {
	return nil
}

func (nopwriter) EndList() error {
	return nil
}

func (nopwriter) BeginSexp() error {
	return nil
}

func (nopwriter) EndSexp() error {
	return nil
}

func (nopwriter) BeginStruct() error {
	return nil
}

func (nopwriter) EndStruct() error {
	return nil
}

func (nopwriter) Finish() error {
	return nil
}

func (nopwriter) IsInStruct() bool {
	return false
}
