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
	"fmt"
	"github.com/google/go-cmp/cmp"
	"testing"
)

// Make sure SymbolToken conforms to Stringer
var _ fmt.Stringer = SymbolToken{}

func TestSymbolToken_String(t *testing.T) {
	cases := []struct {
		desc     string
		token    SymbolToken
		expected string
	}{
		{
			desc: "Text and SID",
			token: SymbolToken{
				Text:     newString("hello"),
				localSID: 10,
				Source:   nil,
			},
			expected: `{"hello" 10 nil}`,
		},
		{
			desc: "nil Text",
			token: SymbolToken{
				Text:     nil,
				localSID: 11,
				Source:   nil,
			},
			expected: `{nil 11 nil}`,
		},
		{
			desc: "Text and SID with Import",
			token: SymbolToken{
				Text:     newString("world"),
				localSID: 12,
				Source:   newSource("foobar", 3),
			},
			expected: `{"world" 12 {"foobar" 3}}`,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if diff := cmp.Diff(c.expected, c.token.String()); diff != "" {
				t.Errorf("Token String() differs (-expected, +actual):\n%s", diff)
			}
		})
	}
}
