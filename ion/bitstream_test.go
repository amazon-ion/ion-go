package ion

import (
	"testing"
)

func TestBitstream(t *testing.T) {
	ion := []byte{
		0xE0, 0x01, 0x00, 0xEA, // $ion_1_0
		0xEE, 0x9F, 0x81, 0x83, 0xDE, 0x9B, // $ion_symbol_table::{
		0x86, 0xBE, 0x8E, // imports:[
		0xDD,                                // {
		0x84, 0x85, 'b', 'o', 'g', 'u', 's', // name: "bogus"
		0x85, 0x21, 0x2A, // version: 42
		0x88, 0x21, 0x64, // max_id: 100
		// }]
		0x87, 0xB8, // symbols: [
		0x83, 'f', 'o', 'o', // "foo"
		0x83, 'b', 'a', 'r', // "bar"
		// ]
		// }
		0xD0,                   // {}
		0xEA, 0x81, 0xEE, 0xD7, // foo::{
		0x84, 0xE3, 0x81, 0xEF, 0x0F, // name:bar::null,
		0x88, 0x20, // max_id:0
		// }
	}

	b := bitstream{}
	b.InitBytes(ion)

	next := func(code bitcode, null bool, len uint64) {
		if err := b.Next(); err != nil {
			t.Fatal(err)
		}
		if b.Code() != code {
			t.Errorf("expected code=%v, got %v", code, b.Code())
		}
		if b.IsNull() != null {
			t.Errorf("expected null=%v, got %v", null, b.IsNull())
		}
		if b.Len() != len {
			t.Errorf("expected len=%v, got %v", len, b.Len())
		}
	}

	fieldid := func(eid uint64) {
		id, err := b.ReadFieldID()
		if err != nil {
			t.Fatal(err)
		}
		if id != eid {
			t.Errorf("expected %v, got %v", eid, id)
		}
	}

	next(bitcodeBVM, false, 3)
	maj, min, err := b.ReadBVM()
	if err != nil {
		t.Fatal(err)
	}
	if maj != 1 && min != 0 {
		t.Errorf("expected $ion_1.0, got $ion_%v.%v", maj, min)
	}

	next(bitcodeAnnotation, false, 31)
	ids, err := b.ReadAnnotationIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 3 { // $ion_symbol_table
		t.Errorf("expected [3], got %v", ids)
	}

	next(bitcodeStruct, false, 27)
	b.StepIn()
	{
		next(bitcodeFieldID, false, 0)
		fieldid(6) // imports

		next(bitcodeList, false, 14)
		b.StepIn()
		{
			next(bitcodeStruct, false, 13)
		}
		if err := b.StepOut(); err != nil {
			t.Fatal(err)
		}

		next(bitcodeFieldID, false, 0)
		// fieldid(7) // symbols

		next(bitcodeList, false, 8)
		next(bitcodeEOF, false, 0)
	}
	if err := b.StepOut(); err != nil {
		t.Fatal(err)
	}

	next(bitcodeStruct, false, 0)
	next(bitcodeAnnotation, false, 10)
	next(bitcodeEOF, false, 0)
	next(bitcodeEOF, false, 0)
}

func TestBitcodeString(t *testing.T) {
	for i := bitcodeNone; i <= bitcodeAnnotation+1; i++ {
		str := i.String()
		if str == "" {
			t.Errorf("expected non-empty string for bitcode %v", uint8(i))
		}
	}
}

func TestBinaryReadTimestamp(t *testing.T) {
	test := func(ion []byte, expectedValue string, expectedPrecision TimestampPrecision, expectedKind TimezoneKind) {
		t.Run(expectedValue, func(t *testing.T) {
			b := bitstream{}
			b.InitBytes(ion)
			b.Next()

			val, err := b.ReadTimestamp()
			if err != nil {
				t.Fatal(err)
			}

			expectedTimestamp, err := NewTimestampFromStr(expectedValue, expectedPrecision, expectedKind)
			if err != nil {
				t.Fatal(err)
			}

			if !val.Equal(expectedTimestamp) {
				t.Errorf("expected %v, got %v", expectedTimestamp, val)
			}
		})
	}

	test([]byte{
		0x63,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
	}, "2000T", Year, Unspecified)

	test([]byte{
		0x64,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x85, // month: 5
	}, "2000-05T", Month, Unspecified)

	test([]byte{
		0x65,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x85, // month: 5
		0x86, // day: 6
	}, "2000-05-06T", Day, Unspecified)

	test([]byte{
		0x67,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x85, // month: 5
		0x86, // day: 6
		0x87, // hour: 7
		0x88, // minute: 8
	}, "2000-05-06T07:08Z", Minute, UTC)

	test([]byte{
		0x68,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x85, // month: 5
		0x86, // day: 6
		0x87, // hour: 7
		0x88, // minute: 8
		0x89, // second: 9
	}, "2000-05-06T07:08:09Z", Second, UTC)

	test([]byte{
		0x6A,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x81, // month: 1
		0x81, // day: 1
		0x80, // hour: 0
		0x80, // minute: 0
		0x80, // second: 0
		0x80, // 0 precision units
		0x00, // 0
	}, "2000-01-01T00:00:00Z", Second, UTC)

	test([]byte{
		0x69,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x81, // month: 1
		0x81, // day: 1
		0x80, // hour: 0
		0x80, // minute: 0
		0x80, // second: 0
		0xC2, // 2 precision units
	}, "2000-01-01T00:00:00.00Z", Nanosecond, UTC)

	test([]byte{
		0x6A,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x85, // month: 5
		0x86, // day: 6
		0x87, // hour: 7
		0x88, // minute: 8
		0x89, // second: 9
		0xC3, // 3 precision units
		0x64, // 100
	}, "2000-05-06T07:08:09.100Z", Nanosecond, UTC)

	test([]byte{
		0x6C,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x85,             // month: 5
		0x86,             // day: 6
		0x87,             // hour: 7
		0x88,             // minute: 8
		0x89,             // second: 9
		0xC6,             // 6 precision units
		0x01, 0x87, 0x04, // 100100
	}, "2000-05-06T07:08:09.100100Z", Nanosecond, UTC)

	test([]byte{
		0x6C,
		0x88,       // offset +8
		0x0F, 0xD0, // year: 2000
		0x85,             // month: 5
		0x86,             // day: 6
		0x87,             // hour: 7
		0x88,             // minute: 8 utc (16 local)
		0x89,             // second: 9
		0xC6,             // 6 precision units
		0x01, 0x87, 0x04, // 100100
	}, "2000-05-06T07:16:09.100100+00:08", Nanosecond, Local)

	// Test >9 fractional seconds.
	test([]byte{
		0x6A,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x81, // month: 1
		0x81, // day: 1
		0x80, // hour: 0
		0x80, // minute: 0
		0x80, // second: 0
		0xCA, // 10 precision units
		0x2C, // 44
	}, "2000-01-01T00:00:00.000000004Z", Nanosecond, UTC)

	test([]byte{
		0x6A,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x81, // month: 1
		0x81, // day: 1
		0x80, // hour: 0
		0x80, // minute: 0
		0x80, // second: 0
		0xCA, // 10 precision units
		0x2D, // 45
	}, "2000-01-01T00:00:00.000000005Z", Nanosecond, UTC)

	test([]byte{
		0x6A,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x81, // month: 1
		0x81, // day: 1
		0x80, // hour: 0
		0x80, // minute: 0
		0x80, // second: 0
		0xCA, // 10 precision units
		0x2E, // 46
	}, "2000-01-01T00:00:00.000000005Z", Nanosecond, UTC)

	test([]byte{
		0x6E,
		0x8E,
		0x80,       // offset 0
		0x0F, 0xD0, // year: 2000
		0x8C,                         // month: 12
		0x9F,                         // day: 31
		0x97,                         // hour: 23
		0xBB,                         // minute: 59
		0xBB,                         // second: 59
		0xCA,                         // 10 precision units
		0x02, 0x54, 0x0B, 0xE3, 0xFF, // 9999999999
	}, "2001-01-01T00:00:00.000000000Z", Nanosecond, UTC)
}
