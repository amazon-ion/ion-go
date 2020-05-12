package main

import (
	"fmt"
	"strings"

	"github.com/amzn/ion-go/ion"
)

type importlocation struct {
	ImportName string `json:"import_name"`
	Location   int    `json:"location"`
}

// token describes an Ion symbol token.
type token struct {
	Text           string          `json:"text,omitempty"`
	ImportLocation *importlocation `json:"import_location,omitempty"`
}

type importdescriptor struct {
	ImportName string `json:"import_name"`
	Version    int    `json:"version"`
	MaxID      int    `json:"max_id"`
}

type eventtype uint8

const (
	containerStart eventtype = iota
	containerEnd
	scalar
	symbolTable
	streamEnd
)

func (e eventtype) String() string {
	switch e {
	case containerStart:
		return "CONTAINER_START"
	case containerEnd:
		return "CONTAINER_END"
	case scalar:
		return "SCALAR"
	case symbolTable:
		return "SYMBOL_TABLE"
	case streamEnd:
		return "STREAM_END"
	default:
		panic(fmt.Sprintf("unknown eventtype %d", e))
	}
}

func (e eventtype) MarshalIon(w ion.Writer) error {
	return w.WriteSymbol(e.String())
}

type iontype ion.Type

func (i iontype) MarshalIon(w ion.Writer) error {
	return w.WriteSymbol(strings.ToUpper(ion.Type(i).String()))
}

// event describes an Ion processing event.
type event struct {
	EventType   eventtype          `json:"event_type"`
	IonType     iontype            `json:"ion_type,omitempty"`
	FieldName   *token             `json:"field_name,omitempty"`
	Annotations []token            `json:"annotations,omitempty"`
	ValueText   string             `json:"value_text,omitempty"`
	ValueBinary []int              `json:"value_binary,omitempty"`
	Imports     []importdescriptor `json:"imports,omitempty"`
	Depth       int                `json:"depth"`
}

type errortype uint8

const (
	read errortype = iota
	write
	state
)

func (e errortype) String() string {
	switch e {
	case read:
		return "READ"
	case write:
		return "WRITE"
	case state:
		return "STATE"
	default:
		panic(fmt.Sprintf("unknown errortype %d", e))
	}
}

func (e errortype) MarshalIon(w ion.Writer) error {
	return w.WriteSymbol(e.String())
}

// errordescription describes an error during Ion processing.
type errordescription struct {
	ErrorType errortype `json:"error_type"`
	Message   string    `json:"message"`
	Location  string    `json:"location"`
	Index     int       `json:"event_index"`
}
