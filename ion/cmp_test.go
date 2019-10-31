/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

package ion

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// This file contains all of the specialized comparison functions we use for tests.

func assertEquivalentValues(values []Value, t *testing.T) {
	t.Helper()

	for i, value := range values {
		next := values[i+1]
		if diff := cmpValueResults(value, next); diff != "" {
			t.Logf("value %d: %#v", i, value)
			t.Logf("value %d: %#v", i+1, next)
			t.Error("values", i, "and", i+1, ": (-expected, +found)", diff)
		}
		if i >= len(values)-2 {
			break
		}
	}
}

func assertNonEquivalentValues(values []Value, t *testing.T) {
	t.Helper()

	for i, value := range values {
		for n := i + 1; n < len(values); n++ {
			next := values[n]
			fmt.Println("comparing values", i, "and", n)
			if diff := cmpValueResults(value, next); diff == "" {
				t.Logf("value %d: %#v", i, value)
				t.Logf("value %d: %#v", n, next)
				t.Error("values", i, "and", n, "are equivalent")
			}
		}

		if i >= len(values)-2 {
			break
		}
	}
}

// cmpDigests compares nil-ness of the Digests, then calls cmpValueSlices on the two
// if they are not nil.
func cmpDigests(expected, found *Digest) string {
	if (expected == nil) != (found == nil) {
		return fmt.Sprintf("nil mis-match: expected is %v and found is %v", expected, found)
	}
	if expected == nil {
		return ""
	}

	return cmpValueSlices(expected.values, found.values)
}

// cmpValueSlices compares the results of the calls to Binary(), Text(), and Value() of each
// element of the given Value slices.
func cmpValueSlices(expected, found []Value) string {
	if len(expected) != len(found) {
		return fmt.Sprintf("length mis-match: expected number of values to be %v, but found %v", len(expected), len(found))
	}

	for i, exp := range expected {
		fnd := found[i]
		if diff := cmp.Diff(exp.Type(), fnd.Type()); diff != "" {
			return diff
		}

		if exp.IsNull() != fnd.IsNull() {
			return fmt.Sprintf("item %d expected IsNull %v but found %v", i, exp.IsNull(), fnd.IsNull())
		}

		if diff := cmp.Diff(exp.Binary(), fnd.Binary()); diff != "" {
			return fmt.Sprintf("item %d of Type %s Binary() %s", i, fnd.Type(), diff)
		}

		if diff := cmp.Diff(string(exp.Text()), string(fnd.Text())); diff != "" {
			return fmt.Sprintf("item %d of type %s Text() %s", i, fnd.Type(), diff)
		}

		if diff := cmpAnnotations(exp.Annotations(), fnd.Annotations()); diff != "" {
			return diff
		}

		if diff := cmpValueResults(exp, fnd); diff != "" {
			return fmt.Sprintf("item %d of type %s: %s", i, exp.Type(), diff)
		}
	}
	return ""
}

// cmpValueResults compares the results of calling Value() on the two given Values.
// If there is a difference, then that difference is returned.  Otherwise the empty
// string is returned.
func cmpValueResults(expected, found Value) string {
	if expected.IsNull() != found.IsNull() {
		return fmt.Sprintf("expected is null %v and found is null %v", expected.IsNull(), found.IsNull())
	}

	if expected.Type() != found.Type() {
		return fmt.Sprintf("expected type is %s and found type is %s", expected.Type(), found.Type())
	}

	if diff := cmpAnnotations(expected.Annotations(), found.Annotations()); diff != "" {
		return diff
	}

	switch expected.(type) {
	case Blob:
		return cmp.Diff(string(expected.(Blob).Value()), string(found.(Blob).Value()))
	case Bool:
		return cmp.Diff(expected.(Bool).Value(), found.(Bool).Value())
	case Clob:
		return cmp.Diff(expected.(Clob).Value(), found.(Clob).Value())
	case Decimal:
		return cmpDecimals(expected.(Decimal), found.(Decimal))
	case Float:
		return cmp.Diff(expected.(Float).Value(), found.(Float).Value())
	case Int:
		return cmpInts(expected.(Int), found.(Int))
	case List:
		return cmpValueSlices(expected.(List).values, found.(List).values)
	case SExp:
		return cmpValueSlices(expected.(SExp).values, found.(SExp).values)
	case String:
		return cmp.Diff(expected.(String).Value(), found.(String).Value())
	case Struct:
		return cmpStructFields(expected.(Struct).fields, found.(Struct).fields)
	case Symbol:
		return cmp.Diff(expected.(Symbol).Value(), found.(Symbol).Value())
	case Timestamp:
		return cmpTimestamps(expected.(Timestamp), found.(Timestamp))
	}

	return ""
}

func cmpAnnotations(expected, found []Symbol) string {
	if len(expected) != len(found) {
		return fmt.Sprintf("length mis-match: expected annotation length is %v and found is %v", len(expected), len(found))
	}

	for i, exp := range expected {
		fnd := found[i]
		if exp.id != fnd.id {
			return fmt.Sprintf("expected annotation at index %d to have id %d but found %d", i, exp.id, fnd.id)
		}
		// TODO: Support symbol tables.
		expText, fndText := exp.Text(), fnd.Text()
		if bytes.HasPrefix(expText, []byte{'$'}) || bytes.HasPrefix(fndText, []byte{'$'}) {
			continue
		}
		if diff := cmp.Diff(expText, fndText); diff != "" {
			return fmt.Sprintf("expected annotation at index %d to have text %q but found %q", i, expText, fndText)
		}
	}

	return ""
}

func cmpDecimals(expected, found Decimal) string {
	expVal, fndVal := expected.Value(), found.Value()
	if !expVal.Equal(fndVal) {
		return fmt.Sprintf("value differs: %q %q", expVal, fndVal)
	}

	// TODO: Do a comparison that tracks precision.

	return ""
}

// cmpErrs calls cmp.Diff on a string representation of the two errors.
func cmpErrs(expected, found error) string {
	expectedStr := "nil"
	if expected != nil {
		expectedStr = expected.Error()
	}
	foundStr := "nil"
	if found != nil {
		foundStr = found.Error()
	}
	return cmp.Diff(expectedStr, foundStr)
}

func cmpInts(expected, found Int) string {
	exp, fnd := expected.Value(), found.Value()
	if (exp == nil) != (fnd == nil) {
		return fmt.Sprintf("nil Value() mis-match for Int: expected is %v and found is %v", exp, fnd)
	}
	if exp == nil {
		return ""
	}

	if exp.Cmp(fnd) != 0 {
		return fmt.Sprintf("int values differ: %q %q", exp.String(), fnd.String())
	}

	return ""
}

func cmpStructFields(expected, found []StructField) string {
	if len(expected) != len(found) {
		return fmt.Sprintf("length mis-match: expected struct field length is %v and found is %v", len(expected), len(found))
	}

	for i, exp := range expected {
		fnd := found[i]
		if exp.Symbol.id != fnd.Symbol.id {
			return fmt.Sprintf("field %d: expected symbolID %d but found %d", i, exp.Symbol.id, fnd.Symbol.id)
		}
		expText, fndText := exp.Symbol.Text(), fnd.Symbol.Text()
		if diff := cmp.Diff(expText, fndText); diff != "" {
			return fmt.Sprintf("field %d: diff %s", i, diff)
		}
		if diff := cmpValueSlices([]Value{exp.Value}, []Value{fnd.Value}); diff != "" {
			return diff
		}
	}

	return ""
}

func cmpTimestamps(expected, found Timestamp) string {
	exp, fnd := expected.Value(), found.Value()
	if diff := exp.Sub(fnd); diff != 0 {
		return fmt.Sprintf("timestamps differ by %v: %q %q", diff, exp, fnd)
	}
	if diff := cmp.Diff(expected.Precision(), found.Precision()); diff != "" {
		return diff
	}
	return ""
}
