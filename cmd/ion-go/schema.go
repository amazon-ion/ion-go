package main

type importlocation struct {
	ImportName string `json:"import_name"`
	Location   int    `json:"location"`
}

type token struct {
	Text           string         `json:"text,omitempty"`
	ImportLocation importlocation `json:"import_location,omitempty"`
}

type importdescriptor struct {
	ImportName string `json:"import_name"`
	Version    int    `json:"version"`
	MaxID      int    `json:"max_id"`
}

type event struct {
	EventType   string             `json:"event_type"`
	IonType     string             `json:"ion_type"`
	FieldName   token              `json:"field_name"`
	Annotations []token            `json:"annotations"`
	ValueText   string             `json:"value_text"`
	ValueBinary []int              `json:"value_binary"`
	Imports     []importdescriptor `json:"imports"`
	Depth       int                `json:"depth"`
}

// errordescription describes an error during Ion processing.
type errordescription struct {
	ErrorType string `json:"error_type"`
	Message   string `json:"message"`
	Location  string `json:"location"`
	Index     int    `json:"event_index"`
}
