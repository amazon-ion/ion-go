/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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

type eventText struct {
    event
}

// Returns the current value's annotations.
func (e eventText) TypeAnnotations() []string {
    return nil
}

// Returns the current value's annotations.
func (e eventText) TypeAnnotationSymbols() []SymbolToken {
    return nil
}

// Returns true if the current value has the specified annotation.
func (e eventText) HasAnnotation(annotation string) bool {
    return false
}

// Returns the current value's field name.
func (e eventText) FieldName() string {
    return ""
}

// Returns the current value's field name symbol.
func (e eventText) FieldNameSymbol() SymbolToken {
    return symbolTokenUndefined
}
