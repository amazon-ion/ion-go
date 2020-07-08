package ion

import (
	"reflect"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type ionEqual interface {
	eq(other ionEqual) bool
}

type ionFloat struct{ float64 }
type ionDecimal struct{ *Decimal }
type ionTimestamp struct{ time.Time }

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
		return thisTimestamp.Time.Equal(val.Time)
	}
	return false
}

func cmpAnnotations(thisAnnotations, otherAnnotations []string) bool {
	return reflect.DeepEqual(thisAnnotations, otherAnnotations)
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

		switch this.(type) {
		case string: // null.sexp, null.list, null.struct
			res = strNullTypeCmp(this, other)
		default:
			thisItem := this.(ionItem)
			otherItem := other.(ionItem)
			res = thisItem.equal(otherItem)
		}
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
	case time.Time:
		thisTimestamp := ionTimestamp{val}
		return thisTimestamp.eq(ionTimestamp{otherValue.(time.Time)})
	default:
		return false
	}
}

func strNullTypeCmp(this, other interface{}) bool {
	thisStr := this.(string)
	otherStr := other.(string)
	return cmp.Equal(thisStr, otherStr)
}

func haveSameTypes(this, other interface{}) bool {
	return reflect.TypeOf(this) == reflect.TypeOf(other)
}
