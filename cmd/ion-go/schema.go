package main

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

type eventtype string

const (
	containerStart = eventtype("CONTAINER_START")
	containerEnd   = eventtype("CONTAINER_END")
	scalar         = eventtype("SCALAR")
	symbolTable    = eventtype("SYMBOL_TABLE")
	streamEnd      = eventtype("STREAM_END")
)

// event describes an Ion processing event.
type event struct {
	EventType   eventtype          `json:"event_type,symbol"`
	IonType     string             `json:"ion_type,symbol,omitempty"`
	FieldName   *token             `json:"field_name,omitempty"`
	Annotations []token            `json:"annotations,omitempty"`
	ValueText   string             `json:"value_text,omitempty"`
	ValueBinary []int              `json:"value_binary,omitempty"`
	Imports     []importdescriptor `json:"imports,omitempty"`
	Depth       int                `json:"depth"`
}

type errortype string

const (
	read  errortype = "READ"
	write errortype = "WRITE"
	state errortype = "STATE"
)

// errordescription describes an error during Ion processing.
type errordescription struct {
	ErrorType errortype `json:"error_type,symbol"`
	Message   string    `json:"message"`
	Location  string    `json:"location"`
	Index     int       `json:"event_index"`
}
