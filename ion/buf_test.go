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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufnode(t *testing.T) {
	root := container{code: 0xE0}
	root.Append(atom([]byte{0x81, 0x83}))
	{
		symtab := &container{code: 0xD0}
		{
			symtab.Append(atom([]byte{0x86})) // varUint(6)
			{
				imps := &container{code: 0xB0}
				{
					imp0 := &container{code: 0xD0}
					{
						imp0.Append(atom([]byte{0x84})) // varUint(4)
						imp0.Append(atom([]byte{0x85, 'b', 'o', 'g', 'u', 's'}))
						imp0.Append(atom([]byte{0x85})) // varUint(5)
						imp0.Append(atom([]byte{0x21, 0x2A}))
						imp0.Append(atom([]byte{0x88})) // varUint(8)
						imp0.Append(atom([]byte{0x21, 0x64}))
					}
					imps.Append(imp0)
				}
				symtab.Append(imps)
			}

			symtab.Append(atom([]byte{0x87})) // varUint(7)
			{
				syms := &container{code: 0xB0}
				{
					syms.Append(atom([]byte{0x83, 'f', 'o', 'o'}))
					syms.Append(atom([]byte{0x83, 'b', 'a', 'r'}))
				}
				symtab.Append(syms)
			}
		}
		root.Append(symtab)
	}

	buf := bytes.Buffer{}

	require.NoError(t, root.EmitTo(&buf))

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

	assert.True(t, bytes.Equal(val, eval), "expected %v, got %v", fmtbytes(eval), fmtbytes(val))
}
