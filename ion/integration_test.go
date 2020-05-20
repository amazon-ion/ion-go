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
	"regexp"
	"strings"
	"testing"
)

// To debug/run one specific file in any of below paths, put the
// file name with its extension. Works even if the file is listed in skip lists.
// For example: const debugFile = "ints.ion"
const debugFile = ""
const goodPath = "../ion-tests/iontestdata/good"
const badPath = "../ion-tests/iontestdata/bad"
const equivsPath = "../ion-tests/iontestdata/good/equivs"
const nonEquivsPath = "../ion-tests/iontestdata/good/non-equivs"

type testingFunc func(t *testing.T, path string)

type item struct {
	ionType     Type
	annotations []string
	value       []interface{}
}

var binaryRoundTripSkipList = []string{
	"allNulls.ion",
	"bigInts.ion",
	"clobWithNonAsciiCharacter.10n",
	"clobs.ion",
	"decimal64BitBoundary.ion",
	"decimals.ion",
	"float32.10n",
	"floats.ion",
	"intBigSize1201.10n",
	"intBigSize13.10n",
	"intBigSize14.10n",
	"intBigSize16.10n",
	"intBigSize256.10n",
	"intBigSize256.ion",
	"intBigSize512.ion",
	"intLongMaxValuePlusOne.10n",
	"localSymbolTableImportZeroMaxId.ion",
	"nullDecimal.10n",
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
	"T2.10n",
	"T3.10n",
	"T5.10n",
	"T7-large.10n",
	"T9.10n",
	"testfile22.ion",
	"testfile23.ion",
	"testfile31.ion",
	"testfile35.ion",
	"testfile37.ion",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
}

var textRoundTripSkipList = []string{
	"allNulls.ion",
	"annotations.ion",
	"bigInts.ion",
	"clobWithNonAsciiCharacter.10n",
	"clobs.ion",
	"decimal64BitBoundary.ion",
	"decimal_values.ion",
	"decimals.ion",
	"decimalsWithUnderscores.ion",
	"float_zeros.ion",
	"float32.10n",
	"floats.ion",
	"floatsVsDecimals.ion",
	"intBigSize1201.10n",
	"intBigSize13.10n",
	"intBigSize14.10n",
	"intBigSize16.10n",
	"intBigSize256.10n",
	"intBigSize256.ion",
	"intBigSize512.ion",
	"intLongMaxValuePlusOne.10n",
	"localSymbolTableImportZeroMaxId.ion",
	"notVersionMarkers.ion",
	"nullDecimal.10n",
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
	"T7-large.10n",
	"T9.10n",
	"testfile22.ion",
	"testfile23.ion",
	"testfile24.ion",
	"testfile31.ion",
	"testfile35.ion",
	"testfile37.ion",
	"utf16.ion",
	"utf32.ion",
	"whitespace.ion",
	"zeroFloats.ion",
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
		binaryRoundTrip(t, path)
	})
}

func TestTextRoundTrip(t *testing.T) {
	readFilesAndTest(t, goodPath, textRoundTripSkipList, func(t *testing.T, path string) {
		textRoundTrip(t, path)
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

func binaryRoundTrip(t *testing.T, fp string) {
	b := loadFile(t, fp)

	// Make a binary writer from the file
	r := NewReaderBytes(b)
	buf := bytes.Buffer{}
	bw := NewBinaryWriter(&buf)
	writeToWriterFromReader(t, r, bw)
	bw.Finish()

	// Make a text writer from the binary writer
	r = NewReaderBytes(buf.Bytes())
	str := strings.Builder{}
	tw := NewTextWriter(&str)
	writeToWriterFromReader(t, r, tw)
	tw.Finish()

	// Make another binary writer using the text writer
	r = NewReaderStr(str.String())
	buf2 := bytes.Buffer{}
	bw2 := NewBinaryWriter(&buf2)
	writeToWriterFromReader(t, r, bw2)
	bw2.Finish()

	// Compare the 2 binary writers
	if !reflect.DeepEqual(bw, bw2) {
		t.Errorf("Round trip test failed on: " + fp)
	}
}

func textRoundTrip(t *testing.T, fp string) {
	b := loadFile(t, fp)

	// Make a text writer from the file
	r := NewReaderBytes(b)
	str := strings.Builder{}
	tw := NewTextWriter(&str)
	writeToWriterFromReader(t, r, tw)
	tw.Finish()

	// Make a binary writer from the text writer
	r = NewReaderStr(str.String())
	buf := bytes.Buffer{}
	bw := NewBinaryWriter(&buf)
	writeToWriterFromReader(t, r, bw)
	bw.Finish()

	// Make another text writer using the binary writer
	r = NewReaderBytes(buf.Bytes())
	str2 := strings.Builder{}
	tw2 := NewTextWriter(&str2)
	writeToWriterFromReader(t, r, tw2)
	tw2.Finish()

	//compare the 2 text writers
	if !reflect.DeepEqual(tw, tw2) {
		t.Errorf("Round trip test failed on: " + fp)
	}
}

func testLoadBad(t *testing.T, fp string) {
	file, er := os.Open(fp)
	if er != nil {
		t.Fatal(er)
	}
	defer file.Close()

	r := NewReader(file)

	err := testBrokenReader(t, r)

	if r.Err() == nil && err == nil {
		t.Fatal("Should have failed loading \"" + fp + "\".")
	} else {
		t.Log("expectedly failed loading " + r.Err().Error())
	}
}

func testBrokenReader(t *testing.T, r Reader) error {
	for r.Next() {
		switch r.Type() {
		case StructType, ListType, SexpType:
			r.StepIn()
			testBrokenReader(t, r)
			r.StepOut()
		}
	}
	if r.Err() != nil {
		return r.Err()
	}

	return nil
}

func testEquivalency(t *testing.T, fp string, eq bool) {
	file, er := os.Open(fp)
	if er != nil {
		t.Fatal(er)
	}
	defer file.Close()

	r := NewReader(file)
	for r.Next() {
		embDoc := embeddedDoc(r.Annotations())
		switch r.Type() {
		case StructType, ListType, SexpType:
			var values []item
			r.StepIn()
			if embDoc {
				values = handleEmbDoc(t, r)
			} else {
				for r.Next() {
					values = append(values, eqv(t, r))
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

func handleEmbDoc(t *testing.T, r Reader) []item {
	var values []item
	for r.Next() {
		str, err := r.StringValue()
		if err != nil {
			t.Error("Must be string value.")
		}
		newReader := NewReaderStr(str)
		for newReader.Next() {
			values = append(values, eqv(t, newReader))
		}
	}
	return values
}

func embeddedDoc(an []string) bool {
	for _, a := range an {
		if a == "embedded_documents" {
			return true
		}
	}
	return false
}

func equivalencyAssertion(t *testing.T, values []item, eq bool) {
	// Nested for loops to evaluate each item with all the other items in the list/struct/sexp
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if i == j {
				continue
			}
			if eq {
				if !reflect.DeepEqual(values[i].value, values[j].value) ||
					!reflect.DeepEqual(values[i].annotations, values[j].annotations) ||
					!reflect.DeepEqual(values[i].ionType, values[j].ionType) {
					t.Error("Equivalency test failed. All values should interpret equal.")
				}
			} else {
				if reflect.DeepEqual(values[i].value, values[j].value) &&
					reflect.DeepEqual(values[i].annotations, values[j].annotations) &&
					reflect.DeepEqual(values[i].ionType, values[j].ionType) {
					t.Error("Non-Equivalency test failed. Values should not interpret equal.")
				}
			}
		}
	}
}

func readFilesAndTest(t *testing.T, path string, skipList []string, tf testingFunc) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}

	if debugFile != "" {
		fp := filepath.Join(path, debugFile)
		t.Run(fp, func(t *testing.T) {
			tf(t, fp)
		})
	} else {
		for _, file := range files {
			fp := filepath.Join(path, file.Name())
			if file.IsDir() {
				readFilesAndTest(t, fp, skipList, tf)
			} else if skipFile(skipList, file.Name()) {
				continue
			} else {
				t.Run(fp, func(t *testing.T) {
					tf(t, fp)
				})
			}
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

func skipFile(skipList []string, fn string) bool {
	ion, _ := regexp.MatchString(`.ion$`, fn)
	bin, _ := regexp.MatchString(`.10n$`, fn)

	return !ion && !bin || isInSkipList(skipList, fn)
}

func isInSkipList(skipList []string, fn string) bool {
	for _, a := range skipList {
		if a == fn {
			return true
		}
	}
	return false
}

func writeToWriterFromReader(t *testing.T, r Reader, w Writer) {
	for r.Next() {
		name := r.FieldName()
		if name != "" {
			w.FieldName(name)
		}

		an := r.Annotations()
		if len(an) > 0 {
			w.Annotations(an...)
		}

		switch r.Type() {
		case NullType:
			err := w.WriteNull()
			if err != nil {
				t.Errorf("Something went wrong when writing Null value. " + err.Error())
			}

		case BoolType:
			val, err := r.BoolValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Boolean value. " + err.Error())
			}
			err = w.WriteBool(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Boolean value. " + err.Error())
			}

		case IntType:
			val, err := r.Int64Value()
			if err != nil {
				t.Errorf("Something went wrong when reading Int value. " + err.Error())
			}
			err = w.WriteInt(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Int value. " + err.Error())
			}

		case FloatType:
			val, err := r.FloatValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Float value. " + err.Error())
			}
			err = w.WriteFloat(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Float value. " + err.Error())
			}

		case DecimalType:
			val, err := r.DecimalValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Decimal value. " + err.Error())
			}
			err = w.WriteDecimal(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Decimal value. " + err.Error())
			}

		case TimestampType:
			val, err := r.TimeValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Timestamp value. " + err.Error())
			}
			err = w.WriteTimestamp(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Timestamp value. " + err.Error())
			}

		case SymbolType:
			val, err := r.StringValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Symbol value. " + err.Error())
			}
			err = w.WriteSymbol(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Symbol value. " + err.Error())
			}

		case StringType:
			val, err := r.StringValue()
			if err != nil {
				t.Errorf("Something went wrong when reading String value. " + err.Error())
			}
			err = w.WriteString(val)
			if err != nil {
				t.Errorf("Something went wrong when writing String value. " + err.Error())
			}

		case ClobType:
			val, err := r.ByteValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Clob value. " + err.Error())
			}
			err = w.WriteClob(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Clob value. " + err.Error())
			}

		case BlobType:
			val, err := r.ByteValue()
			if err != nil {
				t.Errorf("Something went wrong when reading Blob value. " + err.Error())
			}
			err = w.WriteBlob(val)
			if err != nil {
				t.Errorf("Something went wrong when writing Blob value. " + err.Error())
			}

		case SexpType:
			r.StepIn()
			w.BeginSexp()
			writeToWriterFromReader(t, r, w)
			r.StepOut()
			w.EndSexp()

		case ListType:
			r.StepIn()
			w.BeginList()
			writeToWriterFromReader(t, r, w)
			r.StepOut()
			w.EndList()

		case StructType:
			r.StepIn()
			w.BeginStruct()
			writeToWriterFromReader(t, r, w)
			r.StepOut()
			w.EndStruct()
		}
	}

	if r.Err() != nil {
		t.Errorf(r.Err().Error())
	}
}

func eqv(t *testing.T, r Reader) item {
	var i item

	an := r.Annotations()
	if len(an) > 0 {
		i.annotations = an
	}

	switch r.Type() {
	case NullType:
		i.value = append(i.value, textNulls[NoType])
		i.ionType = NullType

	case BoolType:
		val, err := r.BoolValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Boolean value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = BoolType

	case IntType:
		val, err := r.Int64Value()
		if err != nil {
			t.Errorf("Something went wrong when reading Int value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = IntType

	case FloatType:
		val, err := r.FloatValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Float value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = FloatType

	case DecimalType:
		val, err := r.DecimalValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Decimal value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = DecimalType

	case TimestampType:
		val, err := r.TimeValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Timestamp value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = TimestampType

	case SymbolType:
		val, err := r.StringValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Symbol value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = SymbolType

	case StringType:
		val, err := r.StringValue()
		if err != nil {
			t.Errorf("Something went wrong when reading String value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = StringType

	case ClobType:
		val, err := r.ByteValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Clob value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = ClobType

	case BlobType:
		val, err := r.ByteValue()
		if err != nil {
			t.Errorf("Something went wrong when reading Blob value. " + err.Error())
		}
		i.value = append(i.value, val)
		i.ionType = BlobType

	case SexpType:
		r.StepIn()
		for r.Next() {
			i.value = append(i.value, eqv(t, r))
		}
		i.ionType = SexpType
		r.StepOut()

	case ListType:
		r.StepIn()
		for r.Next() {
			i.value = append(i.value, eqv(t, r))
		}
		i.ionType = ListType
		r.StepOut()

	case StructType:
		r.StepIn()
		for r.Next() {
			i.value = append(i.value, eqv(t, r))
		}
		i.ionType = StructType
		r.StepOut()
	}
	return i
}
