package ion

import (
	"bytes"
	"testing"
)

func TestBufnode(t *testing.T) {
	root := container{code: 0xE0}
	root.Add(atom([]byte{0x81, 0x83}))
	{
		symtab := &container{code: 0xD0}
		{
			symtab.Add(fieldname(6))
			{
				imps := &container{code: 0xB0}
				{
					imp0 := &container{code: 0xD0}
					{
						imp0.Add(fieldname(4))
						imp0.Add(atom([]byte{0x85, 'b', 'o', 'g', 'u', 's'}))
						imp0.Add(fieldname(5))
						imp0.Add(atom([]byte{0x21, 0x2A}))
						imp0.Add(fieldname(8))
						imp0.Add(atom([]byte{0x21, 0x64}))
					}
					imps.Add(imp0)
				}
				symtab.Add(imps)
			}

			symtab.Add(fieldname(7))
			{
				syms := &container{code: 0xB0}
				{
					syms.Add(atom([]byte{0x83, 'f', 'o', 'o'}))
					syms.Add(atom([]byte{0x83, 'b', 'a', 'r'}))
				}
				symtab.Add(syms)
			}
		}
		root.Add(symtab)
	}

	buf := bytes.Buffer{}
	if err := root.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	val := buf.Bytes()
	eval := []byte{
		// $ion_symbol_table::{
		0xEE, 0x9F, 0x81, 0x83, 0xDE, 0x9B,
		//   imports:[
		0x86, 0xBE, 0x8E,
		//     {
		0xDD,
		//       name: "bogus"
		0x84, 0x85, 'b', 'o', 'g', 'u', 's',
		//       version: 42
		0x85, 0x21, 0x2A,
		//       max_id: 100
		0x88, 0x21, 0x64,
		//     }
		//   ],
		//   symbols:[
		0x87, 0xB8,
		//     "foo",
		0x83, 'f', 'o', 'o',
		//     "bar"
		0x83, 'b', 'a', 'r',
		//   ]
		// }
	}

	if !bytes.Equal(val, eval) {
		t.Errorf("expected %v, got %v", fmtbytes(eval), fmtbytes(val))
	}
}
