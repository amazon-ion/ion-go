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
	"math"
	"reflect"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type ionEqual interface {
	eq(other ionEqual) bool
}

type ionFloat struct{ float64 }
type ionDecimal struct{ *Decimal }
type ionTimestamp struct{ Timestamp }
type ionSymbol struct{ *SymbolToken }

func (thisFloat ionFloat) eq(other ionEqual) bool {
	float1 := thisFloat.float64
	float2 := other.(ionFloat).float64

	return math.Signbit(float1) == math.Signbit(float2) &&
		cmp.Equal(float1, float2, cmpopts.EquateNaNs())
}

func (thisDecimal ionDecimal) eq(other ionEqual) bool {
	if val, ok := other.(ionDecimal); ok {
		if thisDecimal.scale != val.scale || thisDecimal.isNegZero != val.isNegZero {
			return false
		}
		return thisDecimal.Decimal.Equal(val.Decimal)
	}
	return false
}

func (thisTimestamp ionTimestamp) eq(other ionEqual) bool {
	if val, ok := other.(ionTimestamp); ok {
		return thisTimestamp.Equal(val.Timestamp)
	}
	return false
}

func (thisSymbol ionSymbol) eq(other ionEqual) bool {
	if val, ok := other.(ionSymbol); ok {
		return thisSymbol.SymbolToken.Equal(val.SymbolToken)
	}
	return false
}

func cmpAnnotations(thisAnnotations, otherAnnotations []SymbolToken) bool {
	if len(thisAnnotations) == 0 && len(otherAnnotations) == 0 {
		return true
	}

	if len(thisAnnotations) != len(otherAnnotations) {
		return false
	}

	res := false
	for idx, this := range thisAnnotations {
		other := otherAnnotations[idx]
		res = this.Equal(&other)

		if !res {
			return false
		}
	}
	return res
}

func cmpFloats(thisValue, otherValue interface{}) bool {
	if !haveSameTypes(thisValue, otherValue) {
		return false
	}

	switch val := thisValue.(type) {
	case float64:
		thisFloat := ionFloat{val}
		return thisFloat.eq(ionFloat{otherValue.(float64)})
	case nil:
		return otherValue == nil
	default:
		return false
	}
}

func cmpDecimals(thisValue, otherValue interface{}) bool {
	if !haveSameTypes(thisValue, otherValue) {
		return false
	}

	switch val := thisValue.(type) {
	case *Decimal:
		thisDecimal := ionDecimal{val}
		return thisDecimal.eq(ionDecimal{otherValue.(*Decimal)})
	case nil:
		return otherValue == nil
	default:
		return false
	}
}

func cmpTimestamps(thisValue, otherValue interface{}) bool {
	if !haveSameTypes(thisValue, otherValue) {
		return false
	}

	switch val := thisValue.(type) {
	case Timestamp:
		thisTimestamp := ionTimestamp{val}
		return thisTimestamp.eq(ionTimestamp{otherValue.(Timestamp)})
	case nil:
		return otherValue == nil
	default:
		return false
	}
}

func cmpSymbols(thisValue, otherValue interface{}) bool {
	if thisValue == nil || otherValue == nil {
		return thisValue == nil && otherValue == nil
	}

	if val1, ok := thisValue.(SymbolToken); ok {
		if val2, ok := otherValue.(SymbolToken); ok {
			return val1.Equal(&val2)
		}
	} else if val1, ok := thisValue.(*SymbolToken); ok {
		if val2, ok := otherValue.(*SymbolToken); ok {
			return val1.Equal(val2)
		}
	}

	return false
}

func cmpValueSlices(thisValues, otherValues []interface{}) bool {
	if len(thisValues) == 0 && len(otherValues) == 0 {
		return true
	}

	if len(thisValues) != len(otherValues) {
		return false
	}

	res := false
	for idx, this := range thisValues {
		other := otherValues[idx]

		if !haveSameTypes(this, other) {
			return false
		}

		thisItem := getContainersType(this)
		otherItem := getContainersType(other)
		res = containersEquality(thisItem, otherItem)

		if !res {
			return false
		}
	}
	return res
}

func cmpStruct(thisValues, otherValues []interface{}) bool {
	if len(thisValues) == 0 && len(otherValues) == 0 {
		return true
	}

	if len(thisValues) != len(otherValues) {
		return false
	}

	var res bool
	var checked []int
	for _, this := range thisValues {
		res = false
		var thisItem = getContainersType(this)
		for i := 0; i < len(otherValues); i++ {
			if contains(checked, i) {
				continue
			}
			if !haveSameTypes(this, otherValues[i]) {
				continue
			} else {
				otherItem := getContainersType(otherValues[i])
				res = containersEquality(thisItem, otherItem)
				if res {
					if !contains(checked, i) {
						checked = append(checked, i)
					}
					break
				}
			}
		}
	}
	if len(otherValues) != len(checked) {
		return false
	}

	return res
}

func haveSameTypes(this, other interface{}) bool {
	return reflect.TypeOf(this) == reflect.TypeOf(other)
}

func getContainersType(in interface{}) interface{} {
	switch in.(type) {
	case *string:
		return in.(*string)
	case nil:
		return nil
	default:
		return in.(ionItem)
	}
}

func contains(list []int, idx int) bool {
	for _, num := range list {
		if num == idx {
			return true
		}
	}
	return false
}

// non-null containers have ionItems inside them
func containersEquality(this, other interface{}) bool {
	switch this.(type) {
	case nil:
		return other == nil
	default:
		otherItem := other.(ionItem)
		thisItem := this.(ionItem)
		if thisItem.fieldName.Equal(&otherItem.fieldName) && thisItem.equal(otherItem) {
			return true
		}
	}
	return false
}
