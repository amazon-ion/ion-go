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

// A ImportLocation represents the import location of a SymbolToken.
type ImportLocation struct {
	ImportName *string
	SID        int64
}

// Equal figures out if two import locations are equal for each component.
func (il *ImportLocation) Equal(o *ImportLocation) bool {
	if il.ImportName == nil || o.ImportName == nil {
		if il.ImportName == nil && o.ImportName == nil && il.SID == o.SID {
			return true
		}
		return false
	}
	return *il.ImportName == *o.ImportName && il.SID == o.SID
}
