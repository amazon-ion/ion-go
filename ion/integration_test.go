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
)

const goodPath = "../ion-tests/iontestdata/good"
const badPath = "../ion-tests/iontestdata/bad"
const equivsPath = "../ion-tests/iontestdata/good/equivs"
const nonEquivsPath = "../ion-tests/iontestdata/good/non-equivs"

type testingFunc func(t *testing.T, path string)

type ionItem struct {
	ionType     Type
	annotations []string
	value       []interface{}
}

func (i *ionItem) equal(o ionItem) bool {
	return reflect.DeepEqual(i.value, o.value) &&
		reflect.DeepEqual(i.annotations, o.annotations) &&
		reflect.DeepEqual(i.ionType, o.ionType)
}

var readGoodFilesSkipList = []string{
	"T7-large.10n",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
}

var binaryRoundTripSkipList = []string{
	"clobNewlines.ion",
	"clobWithNonAsciiCharacter.10n",
	"clobs.ion",
	"float32.10n",
	"floatSpecials.ion",
	"floats.ion",
	"item1.10n",
	"leapDay.ion",
	"leapDayRollover.ion",
	"lists.ion",
	"localSymbolTableImportZeroMaxId.ion",
	"nonNulls.ion",
	"nonNulls.ion",
	"structWhitespace.ion",
	"subfieldInt.ion",
	"subfieldVarInt.ion",
	"symbolEmpty.ion",
	"symbols.ion",
	"T6-large.10n",
	"T6-small.10n",
	"T7-large.10n",
	"T9.10n",
	"T10.10n",
	"testfile25.ion",
	"testfile33.ion",
	"testfile35.ion",
	"timestamp2011-02-20.10n",
	"timestamp2011-02-20T19_30_59_100-08_00.10n",
	"timestamp2011-02.10n",
	"timestamp2011.10n",
	"timestampFractions.10n",
	"timestampFractions.ion",
	"timestampSuperfluousOffset.10n",
	"timestampWithTerminatingEof.ion",
	"timestamps.ion",
	"timestamps.ion",
	"timestamps.ion",
	"timestamps.ion",
	"timestampsLargeFractionalPrecision.ion",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
}

var textRoundTripSkipList = []string{
	"annotations.ion",
	"clobNewlines.ion",
	"clobs.ion",
	"float32.10n",
	"floatSpecials.ion",
	"floats.ion",
	"item1.10n",
	"leapDay.ion",
	"leapDayRollover.ion",
	"lists.ion",
	"localSymbolTableImportZeroMaxId.ion",
	"nonNulls.ion",
	"notVersionMarkers.ion",
	"structWhitespace.ion",
	"subfieldInt.ion",
	"subfieldVarInt.ion",
	"subfieldVarUInt.ion",
	"subfieldVarUInt15bit.ion",
	"subfieldVarUInt16bit.ion",
	"subfieldVarUInt32bit.ion",
	"symbolEmpty.ion",
	"symbols.ion",
	"symbols.ion",
	"symbols.ion",
	"systemSymbols.ion",
	"systemSymbolsAsAnnotations.ion",
	"T6-large.10n",
	"T6-small.10n",
	"T7-large.10n",
	"T9.10n",
	"T10.10n",
	"testfile23.ion",
	"testfile25.ion",
	"testfile31.ion",
	"testfile33.ion",
	"testfile35.ion",
	"testfile37.ion",
	"timestamp2011-02-20.10n",
	"timestamp2011-02-20T19_30_59_100-08_00.10n",
	"timestamp2011-02.10n",
	"timestamp2011.10n",
	"timestampFractions.10n",
	"timestampFractions.ion",
	"timestampSuperfluousOffset.10n",
	"timestampWithTerminatingEof.ion",
	"timestamps.ion",
	"timestamps.ion",
	"timestamps.ion",
	"timestamps.ion",
	"timestampsLargeFractionalPrecision.ion",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
}

var malformedIonsSkipList = []string{
	"annotationLengthTooLongContainer.10n",
	"annotationLengthTooLongScalar.10n",
	"annotationLengthTooShortContainer.10n",
	"annotationLengthTooShortScalar.10n",
	"annotationNested.10n",
	"annotationSymbolIDUnmapped.10n",
	"annotationSymbolIDUnmapped.ion",
	"binaryIntWithMultipleUnderscores.ion",
	"binaryIntWithTrailingUnderscore.ion",
	"binaryIntWithUnderscoreAfterRadixPrefix.ion",
	"blobLenTooLarge.10n",
	"clobLenTooLarge.10n",
	"clobWithLongLiteralBlockCommentAtEnd.ion",
	"clobWithLongLiteralCommentsInMiddle.ion",
	"clobWithNonAsciiCharacter.ion",
	"clobWithNonAsciiCharacterMultiline.ion",
	"clobWithNullCharacter.ion",
	"clobWithValidUtf8ButNonAsciiCharacter.ion",
	"clob_2.ion",
	"clob_U0000003F.ion",
	"clob_U00000080.ion",
	"clob_U0000013F.ion",
	"clob_u0020.ion",
	"clob_u00FF.ion",
	"clob_u01FF.ion",
	"decimalLenTooLarge.10n",
	"decimalWithMultipleUnderscores.ion",
	"decimalWithTrailingUnderscore.ion",
	"decimalWithUnderscoreBeforeDecimalPoint.ion",
	"emptyAnnotatedInt.10n",
	"fieldNameSymbolIDUnmapped.10n",
	"fieldNameSymbolIDUnmapped.ion",
	"floatLenTooLarge.10n",
	"hexIntWithMultipleUnderscores.ion",
	"hexIntWithTrailingUnderscore.ion",
	"hexIntWithUnderscoreAfterRadixPrefix.ion",
	"intWithMultipleUnderscores.ion",
	"intWithTrailingUnderscore.ion",
	"invalidVersionMarker_ion_0_0.ion",
	"invalidVersionMarker_ion_1234_0.ion",
	"invalidVersionMarker_ion_1_1.ion",
	"invalidVersionMarker_ion_2_0.ion",
	"localSymbolTableImportNegativeMaxId.ion",
	"localSymbolTableImportNonIntegerMaxId.ion",
	"localSymbolTableImportNullMaxId.ion",
	"localSymbolTableWithMultipleImportsFields.ion",
	"localSymbolTableWithMultipleSymbolsAndImportsFields.ion",
	"localSymbolTableWithMultipleSymbolsFields.10n",
	"localSymbolTableWithMultipleSymbolsFields.ion",
	"minLongWithLenTooLarge.10n",
	"minLongWithLenTooSmall.10n",
	"negativeIntZero.10n",
	"negativeIntZeroLn.10n",
	"nopPadTooShort.10n",
	"nopPadWithAnnotations.10n",
	"nullDotCommentInt.ion",
	"sexpOperatorAnnotation.ion",
	"stringWithLatinEncoding.10n",
	"structOrderedEmpty.10n",
	"surrogate_1.ion",
	"surrogate_10.ion",
	"surrogate_2.ion",
	"surrogate_4.ion",
	"surrogate_5.ion",
	"surrogate_6.ion",
	"surrogate_7.ion",
	"surrogate_8.ion",
	"surrogate_9.ion",
	"symbolIDUnmapped.10n",
	"symbolIDUnmapped.ion",
	"symbolLenTooLarge.10n",
	"timestampLenTooLarge.10n",
	"type_3_length_0.10n",
}

var equivsSkipList = []string{
	"annotatedIvms.ion",
	"localSymbolTableAppend.ion",
	"localSymbolTableNullSlots.ion",
	"localSymbolTableWithAnnotations.ion",
	"localSymbolTables.ion",
	"localSymbolTablesValuesWithAnnotations.ion",
	"nonIVMNoOps.ion",
	"sexps.ion",
	"stringUtf8.ion",
	"structsFieldsDiffOrder.ion",
	"structsFieldsRepeatedNames.ion",
	"systemSymbols.ion",
	"systemSymbolsAsAnnotations.ion",
	"timestampSuperfluousOffset.10n",
	"timestamps.ion",
	"timestampsLargeFractionalPrecision.ion",
}

var nonEquivsSkipList = []string{
	"decimals.ion",
	"documents.ion",
	"floats.ion",
	"floatsVsDecimals.ion",
	"localSymbolTableWithAnnotations.ion",
	"structs.ion",
	"symbolTables.ion",
	"symbolTablesUnknownText.ion",
	"symbols.ion",
	"timestamps.ion",
}

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

	// Make a binary writer from the file
	buf := encodeAsBinaryIon(t, fileBytes)

	// Re-encode binWriter's stream as text into a string builder
	str := encodeAsTextIon(t, buf.Bytes())

	reader1 := NewReader(bytes.NewReader(buf.Bytes()))
	reader2 := NewReader(strings.NewReader(str.String()))

	for reader1.Next() {
		i1 := readCurrentValue(t, reader1)
		reader2.Next()
		i2 := readCurrentValue(t, reader2)

		if !i1.equal(i2) {
			t.Errorf("Failed on %s round trip. Binary reader has %v "+
				"where the value in Text reader is %v", fp, i1.value, i2.value)
		}
	}
}

// Re-encodes the provided file as text Ion, reads that text Ion and writes it as binary Ion, then
// constructs Readers over the text and binary encodings to verify that the streams are equivalent.
func testTextRoundTrip(t *testing.T, fp string) {
	fileBytes := loadFile(t, fp)

	// Make a text writer from the file
	str := encodeAsTextIon(t, fileBytes)

	// Re-encode txtWriter's stream as binary into a bytes.Buffer
	buf := encodeAsBinaryIon(t, []byte(str.String()))

	reader1 := NewReader(strings.NewReader(str.String()))
	reader2 := NewReader(bytes.NewReader(buf.Bytes()))

	for reader1.Next() {
		i1 := readCurrentValue(t, reader1)
		reader2.Next()
		i2 := readCurrentValue(t, reader2)

		if !i1.equal(i2) {
			t.Errorf("Failed on %s round trip. Text reader has %v "+
				"where the value in Binary reader is %v", fp, i1.value, i2.value)
		}
	}
}

// Re-encode the provided Ion data as a text Ion string.
func encodeAsTextIon(t *testing.T, data []byte) strings.Builder {
	reader := NewReader(bytes.NewReader(data))
	str := strings.Builder{}
	txtWriter := NewTextWriter(&str)
	writeToWriterFromReader(t, reader, txtWriter)
	err := txtWriter.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return str
}

// Re-encode the provided Ion data as a binary Ion buffer.
func encodeAsBinaryIon(t *testing.T, data []byte) bytes.Buffer {
	reader := NewReader(bytes.NewReader(data))
	buf := bytes.Buffer{}
	binWriter := NewBinaryWriter(&buf)
	writeToWriterFromReader(t, reader, binWriter)
	err := binWriter.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return buf
}

// Reads Ion values from the provided file, verifying that an
// error is or is not encountered as indicated by errorExpected.
func testLoadFile(t *testing.T, errorExpected bool, fp string) {
	file, er := os.Open(fp)
	if er != nil {
		t.Fatal(er)
	}

	r := NewReader(file)
	err := testInvalidReader(t, r)

	if errorExpected && r.Err() == nil && err == nil {
		t.Fatal("Should have failed loading \"" + fp + "\".")
	} else if !errorExpected && (r.Err() != nil || err != nil) {
		t.Fatal("Failed loading \"" + fp + "\" : " + r.Err().Error())
	} else {
		errMsg := "no"
		if r.Err() != nil {
			errMsg = r.Err().Error()
		}
		t.Log("Test passed for " + fp + " with \"" + errMsg + "\" error.")
	}

	err = file.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// Traverse the reader and check if it is an invalid reader, containing malformed Ion values.
func testInvalidReader(t *testing.T, r Reader) error {
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
			err = testInvalidReader(t, r)
			if err != nil {
				return err
			}
			err = r.StepOut()
			if err != nil {
				return err
			}
		}
	}
	if r.Err() != nil {
		return r.Err()
	}

	return nil
}

// Execute equivalency and non-equivalency tests, where true for eq means
// equivalency and false denotes non-equivalency test.
func testEquivalency(t *testing.T, fp string, eq bool) {
	file, er := os.Open(fp)
	if er != nil {
		t.Fatal(er)
	}

	r := NewReader(file)
	topLevelCounter := 0
	for r.Next() {
		embDoc := isEmbeddedDoc(r.Annotations())
		ionType := r.Type()
		switch ionType {
		case StructType, ListType, SexpType:
			fmt.Printf("Checking values of top level %s #%d ...\n", ionType.String(), topLevelCounter)
			var values []ionItem
			err := r.StepIn()
			if err != nil {
				t.Fatal(err)
			}
			if embDoc {
				values = handleEmbeddedDoc(t, r)
			} else {
				for r.Next() {
					values = append(values, readCurrentValue(t, r))
				}
			}
			equivalencyAssertion(t, values, eq)
			err = r.StepOut()
			topLevelCounter++
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if r.Err() != nil {
		t.Error()
	}
	err := file.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// Handle equivalency tests with embedded_documents annotation
func handleEmbeddedDoc(t *testing.T, r Reader) []ionItem {
	var values []ionItem
	for r.Next() {
		str, err := r.StringValue()
		if err != nil {
			t.Error("Must be string value.")
		}
		newReader := NewReaderStr(str)
		for newReader.Next() {
			values = append(values, readCurrentValue(t, newReader))
		}
	}
	return values
}

func isEmbeddedDoc(an []string) bool {
	for _, a := range an {
		if a == "embedded_documents" {
			return true
		}
	}
	return false
}

func equivalencyAssertion(t *testing.T, values []ionItem, eq bool) {
	// Nested for loops to evaluate each ionItem value with all the other values in the list/struct/sexp
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if i == j {
				continue
			}
			if eq {
				if !values[i].equal(values[j]) {
					t.Errorf("Equivalency test failed. All values should be interpreted as "+
						"equal for:\nrow %d = %v\nrow %d = %v", i, values[i].value, j, values[j].value)
				}
			} else {
				if values[i].equal(values[j]) {
					t.Errorf("Non-Equivalency test failed. Values should not be interpreted as "+
						"equal for:\nrow %d = %v\nrow %d = %v", i, values[i].value, j, values[j].value)
				}
			}
		}
	}
}

// Read and load the files in testing path and pass them to testing functions.
func readFilesAndTest(t *testing.T, path string, skipList []string, testingFunc testingFunc) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}
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
func writeToWriterFromReader(t *testing.T, reader Reader, writer Writer) {
	for reader.Next() {
		name := reader.FieldName()
		if name != "" {
			err := writer.FieldName(name)
			if err != nil {
				t.Fatal(err)
			}
		}

		an := reader.Annotations()
		if len(an) > 0 {
			err := writer.Annotations(an...)
			if err != nil {
				t.Fatal(err)
			}
		}

		currentType := reader.Type()
		if reader.IsNull() {
			err := writer.WriteNullType(currentType)
			if err != nil {
				t.Fatal(err)
			}
			return
		}

		switch currentType {
		case BoolType:
			val, err := reader.BoolValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Boolean value. " + err.Error())
			}
			err = writer.WriteBool(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Boolean value. " + err.Error())
			}

		case IntType:
			val, err := reader.BigIntValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Int value. " + err.Error())
			}
			err = writer.WriteBigInt(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Int value. " + err.Error())
			}

		case FloatType:
			val, err := reader.FloatValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Float value. " + err.Error())
			}
			err = writer.WriteFloat(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Float value. " + err.Error())
			}

		case DecimalType:
			val, err := reader.DecimalValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Decimal value. " + err.Error())
			}
			err = writer.WriteDecimal(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Decimal value. " + err.Error())
			}

		case TimestampType:
			val, err := reader.TimeValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Timestamp value. " + err.Error())
			}
			err = writer.WriteTimestamp(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Timestamp value. " + err.Error())
			}

		case SymbolType:
			val, err := reader.StringValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Symbol value. " + err.Error())
			}
			err = writer.WriteSymbol(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Symbol value. " + err.Error())
			}

		case StringType:
			val, err := reader.StringValue()
			if err != nil {
				t.Errorf("Something went wrong when reading String value. " + err.Error())
			}
			err = writer.WriteString(val)
			if err != nil {
				t.Errorf("Something went wrong when writing String value. " + err.Error())
			}

		case ClobType:
			val, err := reader.ByteValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Clob value. " + err.Error())
			}
			err = writer.WriteClob(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Clob value. " + err.Error())
			}

		case BlobType:
			val, err := reader.ByteValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Blob value. " + err.Error())
			}
			err = writer.WriteBlob(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Blob value. " + err.Error())
			}

		case SexpType:
			err := reader.StepIn()
			if err != nil {
				t.Fatal(err)
			}
			err = writer.BeginSexp()
			if err != nil {
				t.Fatal(err)
			}
			writeToWriterFromReader(t, reader, writer)
			err = reader.StepOut()
			if err != nil {
				t.Fatal(err)
			}
			err = writer.EndSexp()
			if err != nil {
				t.Fatal(err)
			}

		case ListType:
			err := reader.StepIn()
			if err != nil {
				t.Fatal(err)
			}
			err = writer.BeginList()
			if err != nil {
				t.Fatal(err)
			}
			writeToWriterFromReader(t, reader, writer)
			err = reader.StepOut()
			if err != nil {
				t.Fatal(err)
			}
			err = writer.EndList()
			if err != nil {
				t.Fatal(err)
			}

		case StructType:
			err := reader.StepIn()
			if err != nil {
				t.Fatal(err)
			}
			err = writer.BeginStruct()
			if err != nil {
				t.Fatal(err)
			}
			writeToWriterFromReader(t, reader, writer)
			err = reader.StepOut()
			if err != nil {
				t.Fatal(err)
			}
			err = writer.EndStruct()
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	if reader.Err() != nil {
		t.Errorf(reader.Err().Error())
	}
}

// Read the current value in the reader and put that in an ionItem struct (defined in this file).
func readCurrentValue(t *testing.T, reader Reader) ionItem {
	var ionItem ionItem

	an := reader.Annotations()
	if len(an) > 0 {
		ionItem.annotations = an
	}

	currentType := reader.Type()
	if reader.IsNull() {
		ionItem.value = append(ionItem.value, textNulls[currentType])
		ionItem.ionType = currentType

		return ionItem
	}

	switch currentType {
	case BoolType:
		val, err := reader.BoolValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Boolean value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = BoolType

	case IntType:
		val, err := reader.BigIntValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Int value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = IntType

	case FloatType:
		val, err := reader.FloatValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Float value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = FloatType

	case DecimalType:
		val, err := reader.DecimalValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Decimal value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = DecimalType

	case TimestampType:
		val, err := reader.TimeValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Timestamp value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = TimestampType

	case SymbolType:
		val, err := reader.StringValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Symbol value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = SymbolType

	case StringType:
		val, err := reader.StringValue()
		if err != nil {
			t.Errorf("Something went wrong when reading String value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = StringType

	case ClobType:
		val, err := reader.ByteValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Clob value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = ClobType

	case BlobType:
		val, err := reader.ByteValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Blob value. " + err.Error())
		}
		ionItem.value = append(ionItem.value, val)
		ionItem.ionType = BlobType

	case SexpType:
		err := reader.StepIn()
		if err != nil {
			t.Fatal(err)
		}
		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = SexpType
		err = reader.StepOut()
		if err != nil {
			t.Fatal(err)
		}

	case ListType:
		err := reader.StepIn()
		if err != nil {
			t.Fatal(err)
		}
		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = ListType
		err = reader.StepOut()
		if err != nil {
			t.Fatal(err)
		}

	case StructType:
		err := reader.StepIn()
		if err != nil {
			t.Fatal(err)
		}
		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = StructType
		err = reader.StepOut()
		if err != nil {
			t.Fatal(err)
		}
	}
	return ionItem
}
