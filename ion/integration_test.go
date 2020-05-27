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

var binaryRoundTripSkipList = []string{
	"allNulls.ion",
	"bigInts.ion",
	"clobNewlines.ion",
	"clobWithNonAsciiCharacter.10n",
	"clobs.ion",
	"clobs.ion",
	"decimal64BitBoundary.ion",
	"decimals.ion",
	"float32.10n",
	"floatSpecials.ion",
	"floats.ion",
	"intBigSize1201.10n",
	"intBigSize13.10n",
	"intBigSize14.10n",
	"intBigSize16.10n",
	"intBigSize256.10n",
	"intBigSize256.ion",
	"intBigSize512.ion",
	"intLongMaxValuePlusOne.10n",
	"item1.10n",
	"leapDay.ion",
	"leapDayRollover.ion",
	"lists.ion",
	"localSymbolTableImportZeroMaxId.ion",
	"nonNulls.ion",
	"nonNulls.ion",
	"nullBlob.10n",
	"nullClob.10n",
	"nullDecimal.10n",
	"nullTimestamp.10n",
	"nulls.ion",
	"structWhitespace.ion",
	"subfieldInt.ion",
	"subfieldUInt.ion",
	"subfieldVarInt.ion",
	"symbolEmpty.ion",
	"symbols.ion",
	"T10.10n",
	"T2.10n",
	"T3.10n",
	"T5.10n",
	"T6-large.10n",
	"T6-small.10n",
	"T7-large.10n",
	"T9.10n",
	"testfile22.ion",
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
	"timestampsLargeFractionalPrecision.ion",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
}

var textRoundTripSkipList = []string{
	"allNulls.ion",
	"annotations.ion",
	"bigInts.ion",
	"clobNewlines.ion",
	"clobs.ion",
	"clobs.ion",
	"decimal64BitBoundary.ion",
	"decimal_values.ion",
	"decimals.ion",
	"decimalsWithUnderscores.ion",
	"float32.10n",
	"floatSpecials.ion",
	"floats.ion",
	"intBigSize1201.10n",
	"intBigSize13.10n",
	"intBigSize14.10n",
	"intBigSize16.10n",
	"intBigSize256.10n",
	"intBigSize256.ion",
	"intBigSize512.ion",
	"intLongMaxValuePlusOne.10n",
	"item1.10n",
	"leapDay.ion",
	"leapDayRollover.ion",
	"lists.ion",
	"localSymbolTableImportZeroMaxId.ion",
	"nonNulls.ion",
	"nonNulls.ion",
	"notVersionMarkers.ion",
	"nullBlob.10n",
	"nullClob.10n",
	"nullDecimal.10n",
	"nullTimestamp.10n",
	"nulls.ion",
	"structWhitespace.ion",
	"subfieldInt.ion",
	"subfieldUInt.ion",
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
	"T2.10n",
	"T3.10n",
	"T5.10n",
	"T6-large.10n",
	"T6-small.10n",
	"T7-large.10n",
	"T9.10n",
	"T10.10n",
	"testfile22.ion",
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
	"dateDaysInMonth_1.ion",
	"dateDaysInMonth_2.ion",
	"dateDaysInMonth_3.ion",
	"dateDaysInMonth_4.ion",
	"dateDaysInMonth_5.ion",
	"day_1.ion",
	"day_2.ion",
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
	"leapDayNonLeapYear_1.10n",
	"leapDayNonLeapYear_1.ion",
	"leapDayNonLeapYear_2.10n",
	"localSymbolTableImportNegativeMaxId.ion",
	"localSymbolTableImportNonIntegerMaxId.ion",
	"localSymbolTableImportNullMaxId.ion",
	"localSymbolTableWithMultipleImportsFields.ion",
	"localSymbolTableWithMultipleSymbolsAndImportsFields.ion",
	"localSymbolTableWithMultipleSymbolsFields.10n",
	"localSymbolTableWithMultipleSymbolsFields.ion",
	"longStringRawControlCharacter.ion",
	"minLongWithLenTooLarge.10n",
	"minLongWithLenTooSmall.10n",
	"month_1.ion",
	"month_2.ion",
	"negativeIntZero.10n",
	"negativeIntZeroLn.10n",
	"nopPadTooShort.10n",
	"nopPadWithAnnotations.10n",
	"nullDotCommentInt.ion",
	"offsetHours_1.ion",
	"offsetHours_2.ion",
	"offsetMinutes_1.ion",
	"offsetMinutes_2.ion",
	"offsetMinutes_3.ion",
	"sexpOperatorAnnotation.ion",
	"stringLenTooLarge.10n",
	"stringRawControlCharacter.ion",
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
	"timestampHourWithoutMinute.10n",
	"timestampLenTooLarge.10n",
	"timestampSept31.10n",
	"timestamp_0000-00-00.ion",
	"timestamp_0000-00-00T.ion",
	"timestamp_0000-00-01.ion",
	"timestamp_0000-00-01T.ion",
	"timestamp_0000-00T.ion",
	"timestamp_0000-01-00.ion",
	"timestamp_0000-01-00T.ion",
	"timestamp_0000-01-01.ion",
	"timestamp_0000-01-01T.ion",
	"timestamp_0000-01T.ion",
	"timestamp_0000-12-31.ion",
	"timestamp_0000T.ion",
	"timestamp_0001-00-00.ion",
	"timestamp_0001-00-00T.ion",
	"timestamp_0001-00-01.ion",
	"timestamp_0001-00-01T.ion",
	"timestamp_0001-00T.ion",
	"timestamp_0001-01-00.ion",
	"timestamp_0001-01-00T.ion",
	"timestamp_10.ion",
	"timestamp_11.ion",
	"timestamp_5.ion",
	"timestamp_6.ion",
	"timestamp_7.ion",
	"timestamp_8.ion",
	"timestamp_9.ion",
	"type_3_length_0.10n",
	"year_3.ion",
}

var equivsSkipList = []string{
	"annotatedIvms.ion",
	"bigInts.ion",
	"clobs.ion",
	"localSymbolTableAppend.ion",
	"localSymbolTableNullSlots.ion",
	"localSymbolTableWithAnnotations.ion",
	"localSymbolTables.ion",
	"localSymbolTablesValuesWithAnnotations.ion",
	"nonIVMNoOps.ion",
	"sexps.ion",
	"stringUtf8.ion",
	"strings.ion",
	"structsFieldsDiffOrder.ion",
	"structsFieldsRepeatedNames.ion",
	"systemSymbols.ion",
	"systemSymbolsAsAnnotations.ion",
	"timestampSuperfluousOffset.10n",
	"timestamps.ion",
	"timestampsLargeFractionalPrecision.ion",
}

var nonEquivsSkipList = []string{
	"bools.ion",
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
		testLoadBad(t, path)
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

// Execute round trip testing for BinaryWriter. Create a BinaryWriter, using the BinaryWriter create
// a TextWriter and back to BinaryWriter again. Validate that the first and last Writers are equal.
func testBinaryRoundTrip(t *testing.T, fp string) {
	fileBytes := loadFile(t, fp)

	// Make a binary writer from the file
	buf := encodeAsBinaryIon(t, fileBytes)

	// Re-encode binWriter's stream as text into a string builder
	str := encodeAsTextIon(t, buf.String())

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

// Execute round trip testing for TextWriter. Create a TextWriter, using the TextWriter creat a
// BinaryWriter and back to TextWriter again. Validate that the first and last Writers are equal.
func testTextRoundTrip(t *testing.T, fp string) {
	fileBytes := loadFile(t, fp)

	// Make a text writer from the file
	str := encodeAsTextIon(t, string(fileBytes))

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

// Create a TextWriter from data parameter. Return the writer and string builder containing writer's contents
func encodeAsTextIon(t *testing.T, data string) strings.Builder {
	reader := NewReader(strings.NewReader(data))
	str := strings.Builder{}
	txtWriter := NewTextWriter(&str)
	writeToWriterFromReader(t, reader, txtWriter)
	txtWriter.Finish()
	return str
}

// Create a BinaryWriter from data parameter. Return the writer and buffer containing writer's contents
func encodeAsBinaryIon(t *testing.T, data []byte) bytes.Buffer {
	reader := NewReader(bytes.NewReader(data))
	buf2 := bytes.Buffer{}
	binWriter2 := NewBinaryWriter(&buf2)
	writeToWriterFromReader(t, reader, binWriter2)
	binWriter2.Finish()
	return buf2
}

// Execute loading malformed Ion values into a Reader and validate the Reader.
func testLoadBad(t *testing.T, fp string) {
	file, er := os.Open(fp)
	if er != nil {
		t.Fatal(er)
	}
	defer file.Close()

	r := NewReader(file)

	err := testInvalidReader(t, r)

	if r.Err() == nil && err == nil {
		t.Fatal("Should have failed loading \"" + fp + "\".")
	} else {
		t.Log("expectedly failed loading " + r.Err().Error())
	}
}

// Traverse the reader and check if it is an invalid reader, containing malformed Ion values.
func testInvalidReader(t *testing.T, r Reader) error {
	for r.Next() {
		switch r.Type() {
		case StructType, ListType, SexpType:
			r.StepIn()
			testInvalidReader(t, r)
			r.StepOut()
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
	defer file.Close()

	r := NewReader(file)
	for r.Next() {
		embDoc := isEmbeddedDoc(r.Annotations())
		switch r.Type() {
		case StructType, ListType, SexpType:
			var values []ionItem
			r.StepIn()
			if embDoc {
				values = handleEmbeddedDoc(t, r)
			} else {
				for r.Next() {
					values = append(values, readCurrentValue(t, r))
				}
			}
			equivalencyAssertion(t, values, eq)
			r.StepOut()
		}
	}
	if r.Err() != nil {
		t.Error()
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
						"equal for %v and %v", values[i].value, values[j].value)
				}
			} else {
				if values[i].equal(values[j]) {
					t.Errorf("Non-Equivalency test failed. Values should not be interpreted as "+
						"equal for %v and %v", values[i].value, values[j].value)
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
			writer.FieldName(name)
		}

		an := reader.Annotations()
		if len(an) > 0 {
			writer.Annotations(an...)
		}

		switch reader.Type() {
		case NullType:
			err := writer.WriteNull()
			if err != nil {
				t.Errorf("Something went wrong when writing Null value. " + err.Error())
			}

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
			reader.StepIn()
			writer.BeginSexp()
			writeToWriterFromReader(t, reader, writer)
			reader.StepOut()
			writer.EndSexp()

		case ListType:
			reader.StepIn()
			writer.BeginList()
			writeToWriterFromReader(t, reader, writer)
			reader.StepOut()
			writer.EndList()

		case StructType:
			reader.StepIn()
			writer.BeginStruct()
			writeToWriterFromReader(t, reader, writer)
			reader.StepOut()
			writer.EndStruct()
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

	switch reader.Type() {
	case NullType:
		ionItem.value = append(ionItem.value, textNulls[NoType])
		ionItem.ionType = NullType

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
		reader.StepIn()
		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = SexpType
		reader.StepOut()

	case ListType:
		reader.StepIn()
		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = ListType
		reader.StepOut()

	case StructType:
		reader.StepIn()
		for reader.Next() {
			ionItem.value = append(ionItem.value, readCurrentValue(t, reader))
		}
		ionItem.ionType = StructType
		reader.StepOut()
	}
	return ionItem
}
