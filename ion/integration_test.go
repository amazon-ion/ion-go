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

package ion

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodPath = "../ion-tests/iontestdata/good"
const badPath = "../ion-tests/iontestdata/bad"
const equivsPath = "../ion-tests/iontestdata/good/equivs"
const nonEquivsPath = "../ion-tests/iontestdata/good/non-equivs"

type testingFunc func(t *testing.T, path string)

type ionItem struct {
	ionType     Type
	annotations []SymbolToken
	value       []interface{}
	fieldName   SymbolToken
}

func (i *ionItem) equal(o ionItem) bool {
	if i.ionType != o.ionType {
		return false
	}
	if !cmpAnnotations(i.annotations, o.annotations) {
		return false
	}

	switch i.ionType {
	case FloatType:
		return cmpFloats(i.value[0], o.value[0])
	case DecimalType:
		return cmpDecimals(i.value[0], o.value[0])
	case TimestampType:
		return cmpTimestamps(i.value[0], o.value[0])
	case ListType, SexpType:
		return cmpValueSlices(i.value, o.value)
	case StructType:
		return cmpStruct(i.value, o.value)
	case SymbolType:
		return cmpSymbols(i.value[0], o.value[0])
	default:
		return reflect.DeepEqual(i.value, o.value)
	}
}

var readGoodFilesSkipList = []string{
	"T7-large.10n",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
}

var binaryRoundTripSkipList = []string{
	"T7-large.10n",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
}

var textRoundTripSkipList = []string{
	"T7-large.10n",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
}

var malformedIonsSkipList = []string{
	"invalidVersionMarker_ion_0_0.ion",
	"invalidVersionMarker_ion_1234_0.ion",
	"invalidVersionMarker_ion_1_1.ion",
	"invalidVersionMarker_ion_2_0.ion",
	"minLongWithLenTooSmall.10n",
	"nopPadTooShort.10n",
	"nullDotCommentInt.ion",
	"surrogate_1.ion",
	"surrogate_10.ion",
	"surrogate_2.ion",
	"surrogate_4.ion",
	"surrogate_5.ion",
	"surrogate_6.ion",
	"surrogate_7.ion",
	"surrogate_8.ion",
	"surrogate_9.ion",
}

var equivsSkipList = []string{
	"nonIVMNoOps.ion",
	"stringUtf8.ion", // fails on utf-16 surrogate https://github.com/amzn/ion-go/issues/75
}

var nonEquivsSkipList = []string{}

func TestLoadGood(t *testing.T) {
	readFilesAndTest(t, goodPath, readGoodFilesSkipList, func(t *testing.T, path string) {
		testLoadFile(t, false, path)
	})
}

func TestBinaryRoundTrip(t *testing.T) {
	readFilesAndTest(t, goodPath, binaryRoundTripSkipList, func(t *testing.T, path string) {
		testBinaryRoundTrip(t, path)
	})
}

func TestTextRoundTrip(t *testing.T) {
	readFilesAndTest(t, goodPath, textRoundTripSkipList, func(t *testing.T, path string) {
		testTextRoundTrip(t, path)
	})
}

func TestLoadBad(t *testing.T) {
	readFilesAndTest(t, badPath, malformedIonsSkipList, func(t *testing.T, path string) {
		testLoadFile(t, true, path)
	})
}

func TestEquivalency(t *testing.T) {
	readFilesAndTest(t, equivsPath, equivsSkipList, func(t *testing.T, path string) {
		testEquivalency(t, path, true)
	})
}

func TestNonEquivalency(t *testing.T) {
	readFilesAndTest(t, nonEquivsPath, nonEquivsSkipList, func(t *testing.T, path string) {
		testEquivalency(t, path, false)
	})
}

// Re-encodes the provided file as binary Ion, reads that binary Ion and writes it as text Ion, then
// constructs Readers over the binary and text encodings to verify that the streams are equivalent.
func testBinaryRoundTrip(t *testing.T, fp string) {
	fileBytes := loadFile(t, fp)
	symbolTable := getSymbolTable(fileBytes)

	// Make a binary writer from the file
	buf := encodeAsBinaryIon(t, fileBytes, symbolTable.Imports()...)

	// Re-encode binWriter's stream as text into a string builder
	str := encodeAsTextIon(t, buf.Bytes(), symbolTable.Imports()...)

	reader1 := NewReader(bytes.NewReader(buf.Bytes()))
	reader2 := NewReader(strings.NewReader(str.String()))

	for reader1.Next() {
		i1 := readCurrentValue(t, reader1)
		reader2.Next()
		i2 := readCurrentValue(t, reader2)

		assert.True(t, i1.equal(i2), "Failed on %s round trip. Binary reader has %v "+
			"where the value in Text reader is %v", fp, i1.value, i2.value)
	}
}

// Re-encodes the provided file as text Ion, reads that text Ion and writes it as binary Ion, then
// constructs Readers over the text and binary encodings to verify that the streams are equivalent.
func testTextRoundTrip(t *testing.T, fp string) {
	fileBytes := loadFile(t, fp)
	symbolTable := getSymbolTable(fileBytes)

	// Make a text writer from the file
	str := encodeAsTextIon(t, fileBytes, symbolTable.Imports()...)

	// Re-encode txtWriter's stream as binary into a bytes.Buffer
	buf := encodeAsBinaryIon(t, []byte(str.String()), symbolTable.Imports()...)

	reader1 := NewReader(strings.NewReader(str.String()))
	reader2 := NewReader(bytes.NewReader(buf.Bytes()))

	for reader1.Next() {
		i1 := readCurrentValue(t, reader1)
		reader2.Next()
		i2 := readCurrentValue(t, reader2)

		assert.True(t, i1.equal(i2), "Failed on %s round trip. Text reader has %v "+
			"where the value in Binary reader is %v", fp, i1.value, i2.value)
	}
}

// Re-encode the provided Ion data as a text Ion string.
func encodeAsTextIon(t *testing.T, data []byte, st ...SharedSymbolTable) strings.Builder {
	reader := NewReader(bytes.NewReader(data))
	str := strings.Builder{}
	txtWriter := NewTextWriterOpts(&str, 0, st...)
	writeFromReaderToWriter(t, reader, txtWriter)
	require.NoError(t, txtWriter.Finish())
	return str
}

// Re-encode the provided Ion data as a binary Ion buffer.
func encodeAsBinaryIon(t *testing.T, data []byte, st ...SharedSymbolTable) bytes.Buffer {
	reader := NewReaderCat(bytes.NewReader(data), NewCatalog(st...))
	buf := bytes.Buffer{}
	binWriter := NewBinaryWriter(&buf, st...)
	writeFromReaderToWriter(t, reader, binWriter)
	require.NoError(t, binWriter.Finish())
	return buf
}

// Reads Ion values from the provided file, verifying that an
// error is or is not encountered as indicated by errorExpected.
func testLoadFile(t *testing.T, errorExpected bool, fp string) {
	file, err := os.Open(fp)
	require.NoError(t, err)

	r := NewReader(file)
	err = testInvalidReader(r)

	if errorExpected {
		require.True(t, r.Err() != nil || err != nil, "Should have failed loading \""+fp+"\".")

		errMsg := "no"
		if r.Err() != nil {
			errMsg = r.Err().Error()
		} else if err != nil {
			errMsg = err.Error()
		}
		t.Log("Test passed for " + fp + " with \"" + errMsg + "\" error.")
	} else {
		require.NoError(t, r.Err(), "Failed loading \""+fp+"\"")
		require.NoError(t, err, "Failed loading \""+fp+"\"")
	}

	require.NoError(t, file.Close())
}

// Traverse the reader and check if it is an invalid reader, containing malformed Ion values.
func testInvalidReader(r Reader) error {
	for r.Next() {
		switch r.Type() {
		case StructType, ListType, SexpType:
			if r.IsNull() { // null.list, null.struct, null.sexp
				continue
			}
			err := r.StepIn()
			if err != nil {
				return err
			}
			err = testInvalidReader(r)
			if err != nil {
				return err
			}
			err = r.StepOut()
			if err != nil {
				return err
			}
		}
	}

	return r.Err()
}

// Execute equivalency and non-equivalency tests, where true for eq means
// equivalency and false denotes non-equivalency test.
func testEquivalency(t *testing.T, fp string, eq bool) {
	file, err := os.Open(fp)
	require.NoError(t, err)

	r := NewReader(file)
	topLevelCounter := 0
	for r.Next() {
		annotations, err := r.Annotations()
		if err != nil {
			t.Fatal(err)
		}
		embDoc := isEmbeddedDoc(annotations)
		ionType := r.Type()
		switch ionType {
		case StructType, ListType, SexpType:
			fmt.Printf("Checking values of top level %s #%d ...\n", ionType.String(), topLevelCounter)
			require.NoError(t, r.StepIn())

			var values [][]ionItem
			if embDoc {
				values = handleEmbeddedDoc(t, r)
			} else {
				for r.Next() {
					values = append(values, []ionItem{readCurrentValue(t, r)})
				}
			}

			equivalencyAssertion(t, values, eq)

			require.NoError(t, r.StepOut())

			topLevelCounter++
		}
	}
	assert.NoError(t, r.Err())
	require.NoError(t, file.Close())
}

// Handle equivalency tests with embedded_documents annotation
func handleEmbeddedDoc(t *testing.T, r Reader) [][]ionItem {
	var values [][]ionItem
	for r.Next() {
		str, err := r.StringValue()
		assert.NoError(t, err, "Must be string value.")

		if str != nil {
			newReader := NewReaderString(*str)
			var ionItems []ionItem
			for newReader.Next() {
				ionItems = append(ionItems, readCurrentValue(t, newReader))
			}
			values = append(values, ionItems)
		}
	}
	return values
}

func isEmbeddedDoc(an []SymbolToken) bool {
	if len(an) >= 1 && an[0].Text != nil && *an[0].Text == "embedded_documents" {
		return true
	}
	return false
}

func equivalencyAssertion(t *testing.T, values [][]ionItem, eq bool) {
	// Nested for loops to evaluate each ionItem value with all the other values in the list/struct/sexp
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if i == j {
				continue
			}

			res, idx := compareIonItemSlices(values[i], values[j])
			if eq {
				assert.True(t, res, "Equivalency test failed. All values should be interpreted as "+
					"equal for:\nrow %d = %v\nrow %d = %v", i, values[i][idx].value, j, values[j][idx].value)
			} else {
				assert.False(t, res, "Non-Equivalency test failed. Values should not be interpreted as "+
					"equal for:\nrow %d = %v\nrow %d = %v", i, values[i][idx].value, j, values[j][idx].value)
			}
		}
	}
}

// Read and load the files in testing path and pass them to testing functions.
func readFilesAndTest(t *testing.T, path string, skipList []string, testingFunc testingFunc) {
	files, err := ioutil.ReadDir(path)
	require.NoError(t, err)

	for _, file := range files {
		fp := filepath.Join(path, file.Name())
		if file.IsDir() {
			readFilesAndTest(t, fp, skipList, testingFunc)
		} else if skipFile(skipList, file.Name()) {
			continue
		} else {
			t.Run(fp, func(t *testing.T) {
				testingFunc(t, fp)
			})
		}
	}
}

func loadFile(t *testing.T, path string) []byte {
	b, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	return b
}

// Files with extensions other than "ion" or "10n", or in skip list. Return True to skip the file, false otherwise.
func skipFile(skipList []string, fn string) bool {
	ionFile := strings.HasSuffix(fn, "ion") ||
		strings.HasSuffix(fn, "10n")

	return !ionFile || isInSkipList(skipList, fn)
}

func isInSkipList(skipList []string, fn string) bool {
	for _, a := range skipList {
		if a == fn {
			return true
		}
	}
	return false
}

// Read all the values in the reader and write them in the writer
func writeFromReaderToWriter(t *testing.T, reader Reader, writer Writer) {
	for reader.Next() {
		fns, err := reader.FieldName()
		if err == nil && reader.IsInStruct() && fns != nil {
			require.NoError(t, writer.FieldNameSymbol(*fns))
		}

		an, err := reader.Annotations()
		require.NoError(t, err)
		if len(an) > 0 {
			require.NoError(t, writer.Annotations(an...))
		}

		currentType := reader.Type()
		if reader.IsNull() {
			require.NoError(t, writer.WriteNullType(currentType))
			continue
		}

		switch currentType {
		case NullType:
			assert.NoError(t, writer.WriteNullType(NullType), "Something went wrong while writing a Null value")

		case BoolType:
			val, err := reader.BoolValue()
			assert.NoError(t, err, "Something went wrong while reading a Boolean value")

			if val == nil {
				assert.NoError(t, writer.WriteNullType(BoolType))
			} else {
				assert.NoError(t, writer.WriteBool(*val), "Something went wrong while writing a Boolean value")
			}

		case IntType:
			intSize, err := reader.IntSize()
			require.NoError(t, err, "Something went wrong while retrieving the Int size")

			switch intSize {
			case Int32, Int64:
				val, err := reader.Int64Value()
				assert.NoError(t, err, "Something went wrong while reading an Int value")

				assert.NoError(t, writer.WriteInt(*val), "Something went wrong while writing an Int value")

			case BigInt:
				val, err := reader.BigIntValue()
				assert.NoError(t, err, "Something went wrong while reading a Big Int value")

				assert.NoError(t, writer.WriteBigInt(val), "Something went wrong while writing a Big Int value")

			case NullInt:
				assert.NoError(t, writer.WriteNullType(IntType))

			default:
				t.Error("Expected intSize to be one of Int32, Int64, Uint64, or BigInt")
			}

		case FloatType:
			val, err := reader.FloatValue()
			assert.NoError(t, err, "Something went wrong while reading a Float value")

			if val == nil {
				assert.NoError(t, writer.WriteNullType(FloatType))
			} else {
				assert.NoError(t, writer.WriteFloat(*val), "Something went wrong while writing a Float value")
			}

		case DecimalType:
			val, err := reader.DecimalValue()
			assert.NoError(t, err, "Something went wrong while reading a Decimal value")

			if val == nil {
				assert.NoError(t, writer.WriteNullType(DecimalType))
			} else {
				assert.NoError(t, writer.WriteDecimal(val), "Something went wrong while writing a Decimal value")
			}

		case TimestampType:
			val, err := reader.TimestampValue()
			assert.NoError(t, err, "Something went wrong while reading a Timestamp value")

			if val == nil {
				assert.NoError(t, writer.WriteNullType(TimestampType))
			} else {
				assert.NoError(t, writer.WriteTimestamp(*val), "Something went wrong while writing a Timestamp value")
			}

		case SymbolType:
			val, err := reader.SymbolValue()
			assert.NoError(t, err, "Something went wrong while reading a Symbol value")

			if val == nil {
				assert.NoError(t, writer.WriteNullType(SymbolType))
			} else {
				assert.NoError(t, writer.WriteSymbol(*val), "Something went wrong while writing a Symbol value")
			}

		case StringType:
			val, err := reader.StringValue()
			assert.NoError(t, err, "Something went wrong while reading a String value")

			if val == nil {
				assert.NoError(t, writer.WriteNullType(StringType))
			} else {
				assert.NoError(t, writer.WriteString(*val), "Something went wrong while writing a String value")
			}

		case ClobType:
			val, err := reader.ByteValue()
			assert.NoError(t, err, "Something went wrong while reading a Clob value")

			assert.NoError(t, writer.WriteClob(val), "Something went wrong while writing a Clob value")

		case BlobType:
			val, err := reader.ByteValue()
			assert.NoError(t, err, "Something went wrong while reading a Blob value")

			assert.NoError(t, writer.WriteBlob(val), "Something went wrong while writing a Blob value")

		case SexpType:
			require.NoError(t, reader.StepIn())
			require.NoError(t, writer.BeginSexp())

			writeFromReaderToWriter(t, reader, writer)

			require.NoError(t, reader.StepOut())
			require.NoError(t, writer.EndSexp())

		case ListType:
			require.NoError(t, reader.StepIn())
			require.NoError(t, writer.BeginList())

			writeFromReaderToWriter(t, reader, writer)

			require.NoError(t, reader.StepOut())
			require.NoError(t, writer.EndList())

		case StructType:
			require.NoError(t, reader.StepIn())
			require.NoError(t, writer.BeginStruct())

			writeFromReaderToWriter(t, reader, writer)

			require.NoError(t, reader.StepOut())
			require.NoError(t, writer.EndStruct())
		}
	}

	assert.NoError(t, reader.Err(), "Something went wrong executing reader.Next()")
}

// Read the current value in the reader and put that in an ionItem struct (defined in this file).
func readCurrentValue(t *testing.T, reader Reader) ionItem {
	var ionItem ionItem

	an, err := reader.Annotations()
	require.NoError(t, err, "Something went wrong when reading annotations")

	if len(an) > 0 {
		ionItem.annotations = an
	}

	fn, err := reader.FieldName()
	require.NoError(t, err, "Something went wrong when reading field name")

	if fn != nil {
		ionItem.fieldName = *fn
	}

	currentType := reader.Type()
	switch currentType {
	case NullType:
		ionItem.value = append(ionItem.value, nil)
		ionItem.ionType = NullType

	case BoolType:
		val, err := reader.BoolValue()
		assert.NoError(t, err, "Something went wrong when reading Boolean value")

		if val == nil {
			ionItem.value = append(ionItem.value, nil)
		} else {
			ionItem.value = append(ionItem.value, *val)
		}
		ionItem.ionType = BoolType

	case IntType:
		val, err := reader.BigIntValue()
		assert.NoError(t, err, "Something went wrong when reading Int value")

		if val == nil {
			ionItem.value = append(ionItem.value, nil)
		} else {
			ionItem.value = append(ionItem.value, *val)
		}
		ionItem.ionType = IntType

	case FloatType:
		val, err := reader.FloatValue()
		assert.NoError(t, err, "Something went wrong when reading Float value")

		if val == nil {
			ionItem.value = append(ionItem.value, nil)
		} else {
			ionItem.value = append(ionItem.value, *val)
		}
		ionItem.ionType = FloatType

	case DecimalType:
		val, err := reader.DecimalValue()
		assert.NoError(t, err, "Something went wrong when reading Decimal value")

		if val == nil {
			ionItem.value = append(ionItem.value, nil)
		} else {
			ionItem.value = append(ionItem.value, val)
		}
		ionItem.ionType = DecimalType

	case TimestampType:
		val, err := reader.TimestampValue()
		assert.NoError(t, err, "Something went wrong when reading Timestamp value")

		if val == nil {
			ionItem.value = append(ionItem.value, nil)
		} else {
			ionItem.value = append(ionItem.value, *val)
		}
		ionItem.ionType = TimestampType

	case SymbolType:
		val, err := reader.SymbolValue()
		assert.NoError(t, err, "Something went wrong when reading Symbol value")

		if val == nil {
			ionItem.value = append(ionItem.value, nil)
		} else {
			ionItem.value = append(ionItem.value, *val)
		}
		ionItem.ionType = SymbolType

	case StringType:
		val, err := reader.StringValue()
		assert.NoError(t, err, "Something went wrong when reading String value")

		if val == nil {
			ionItem.value = append(ionItem.value, nil)
		} else {
			ionItem.value = append(ionItem.value, *val)
		}
		ionItem.ionType = StringType

	case ClobType:
		val, err := reader.ByteValue()
		assert.NoError(t, err, "Something went wrong when reading Clob value")

		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = ClobType

	case BlobType:
		val, err := reader.ByteValue()
		assert.NoError(t, err, "Something went wrong when reading Blob value")

		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = BlobType

	case SexpType:
		if reader.IsNull() {
			ionItem.value = append(ionItem.value, nil)
			ionItem.ionType = SexpType
			break
		}
		require.NoError(t, reader.StepIn())

		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = SexpType

		require.NoError(t, reader.StepOut())

	case ListType:
		if reader.IsNull() {
			ionItem.value = append(ionItem.value, nil)
			ionItem.ionType = ListType
			break
		}
		require.NoError(t, reader.StepIn())

		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = ListType

		require.NoError(t, reader.StepOut())

	case StructType:
		if reader.IsNull() {
			ionItem.value = append(ionItem.value, nil)
			ionItem.ionType = StructType
			break
		}
		require.NoError(t, reader.StepIn())

		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = StructType

		require.NoError(t, reader.StepOut())
	}

	return ionItem
}

func compareIonItemSlices(this, that []ionItem) (bool, int) {
	if len(this) != len(that) {
		return false, 0
	}

	idx := 0
	for i := 0; i < len(this); i++ {
		idx = i
		if !this[i].equal(that[i]) {
			return false, idx
		}
	}
	return true, idx
}

func getSymbolTable(fileBytes []byte) SymbolTable {
	reader := NewReader(bytes.NewReader(fileBytes))
	reader.Next()
	return reader.SymbolTable()
}
