package ion

import "testing"

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
		if b.Null() != null {
			t.Errorf("expected null=%v, got %v", null, b.Null())
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
	ids, err := b.ReadAnnotations()
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
