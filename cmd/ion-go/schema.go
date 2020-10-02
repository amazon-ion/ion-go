/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package main

import (
	"fmt"
	"strings"

	"github.com/amzn/ion-go/ion"
)

type importdescriptor struct {
	ImportName string `ion:"import_name"`
	Version    int    `ion:"version"`
	MaxID      int    `ion:"max_id"`
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
	return w.WriteSymbolFromString(e.String())
}

type iontype ion.Type

func (i iontype) MarshalIon(w ion.Writer) error {
	return w.WriteSymbolFromString(strings.ToUpper(ion.Type(i).String()))
}

// event describes an Ion processing event.
type event struct {
	EventType   eventtype          `ion:"event_type"`
	IonType     iontype            `ion:"ion_type,omitempty"`
	FieldName   *ion.SymbolToken   `ion:"field_name,omitempty"`
	Annotations []ion.SymbolToken  `ion:"annotations,omitempty"`
	ValueText   string             `ion:"value_text,omitempty"`
	ValueBinary []int              `ion:"value_binary,omitempty"`
	Imports     []importdescriptor `ion:"imports,omitempty"`
	Depth       int                `ion:"depth"`
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
	return w.WriteSymbolFromString(e.String())
}

// errordescription describes an error during Ion processing.
type errordescription struct {
	ErrorType errortype `ion:"error_type"`
	Message   string    `ion:"message"`
	Location  string    `ion:"location"`
	Index     int       `ion:"event_index"`
}
