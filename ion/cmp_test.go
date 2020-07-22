package ion

import (
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

func (thisFloat ionFloat) eq(other ionEqual) bool {
	return cmp.Equal(thisFloat.float64, other.(ionFloat).float64, cmpopts.EquateNaNs())
}

func (thisDecimal ionDecimal) eq(other ionEqual) bool {
	if val, ok := other.(ionDecimal); ok {
		if thisDecimal.scale != val.scale {
			return false
		}
		return thisDecimal.Decimal.Equal(val.Decimal)
	}
	return false
}

func (thisTimestamp ionTimestamp) eq(other ionEqual) bool {
	if val, ok := other.(ionTimestamp); ok {
		return thisTimestamp.DateTime.Equal(val.DateTime) && thisTimestamp.precision == val.precision
	}
	return false
}

func cmpAnnotations(thisAnnotations, otherAnnotations []string) bool {
	return reflect.DeepEqual(thisAnnotations, otherAnnotations)
}

func cmpFloats(thisValue, otherValue interface{}) bool {
	if !haveSameTypes(thisValue, otherValue) {
		return false
	}

	switch val := thisValue.(type) {
	case string: // null.float
		return strNullTypeCmp(val, otherValue)
	case float64:
		thisFloat := ionFloat{val}
		return thisFloat.eq(ionFloat{otherValue.(float64)})
	default:
		return false
	}
}

func cmpDecimals(thisValue, otherValue interface{}) bool {
	if !haveSameTypes(thisValue, otherValue) {
		return false
	}

	switch val := thisValue.(type) {
	case string: // null.decimal
		return strNullTypeCmp(val, otherValue)
	case *Decimal:
		thisDecimal := ionDecimal{val}
		return thisDecimal.eq(ionDecimal{otherValue.(*Decimal)})
	default:
		return false
	}
}

func cmpTimestamps(thisValue, otherValue interface{}) bool {
	if !haveSameTypes(thisValue, otherValue) {
		return false
	}

	switch val := thisValue.(type) {
	case string: // null.timestamp
		return strNullTypeCmp(val, otherValue)
	case Timestamp:
		thisTimestamp := ionTimestamp{val}
		return thisTimestamp.eq(ionTimestamp{otherValue.(Timestamp)})
	default:
		return false
	}
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

func strNullTypeCmp(this, other interface{}) bool {
	thisStr, thisOk := this.(string)
	otherStr, otherOk := other.(string)
	if thisOk && otherOk {
		return cmp.Equal(thisStr, otherStr)
	}
	return false
}

func haveSameTypes(this, other interface{}) bool {
	return reflect.TypeOf(this) == reflect.TypeOf(other)
}

func getContainersType(in interface{}) interface{} {
	switch in.(type) {
	case string:
		return in.(string)
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
	case string: // null.list, null.sexp, null.struct
		if strNullTypeCmp(this, other) {
			return true
		}
	default:
		otherItem := other.(ionItem)
		thisItem := this.(ionItem)
		if thisItem.fieldName == otherItem.fieldName && thisItem.equal(otherItem) {
			return true
		}
	}
	return false
}
