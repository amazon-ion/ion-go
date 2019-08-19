package ion

import (
	"errors"
	"math"
	"math/big"
	"time"
)

// A reader holds common implementation stuff to both the text and binary readers.
type reader struct {
	ctx ctxstack
	eof bool
	err error

	fieldName   string
	annotations []string
	valueType   Type
	value       interface{}
}

// Err returns the current error.
func (r *reader) Err() error {
	return r.err
}

// Type returns the current value's type.
func (r *reader) Type() Type {
	return r.valueType
}

// IsNull returns true if the current value is null.
func (r *reader) IsNull() bool {
	return r.valueType != NoType && r.value == nil
}

// FieldName returns the current value's field name.
func (r *reader) FieldName() string {
	return r.fieldName
}

// Annotations returns the current value's annotations.
func (r *reader) Annotations() []string {
	return r.annotations
}

// BoolValue returns the current value as a bool.
func (r *reader) BoolValue() (bool, error) {
	if r.valueType == BoolType {
		if r.value == nil {
			return false, nil
		}
		return r.value.(bool), nil
	}
	return false, errors.New("ion: value is not a bool")
}

// IntSize returns the size of the current int value.
func (r *reader) IntSize() (IntSize, error) {
	if r.valueType != IntType {
		return NullInt, errors.New("ion: value is not an int")
	}
	if r.value == nil {
		return NullInt, nil
	}

	if i, ok := r.value.(int64); ok {
		if i > math.MaxInt32 || i < math.MinInt32 {
			return Int64, nil
		}
		return Int32, nil
	}

	return BigInt, nil
}

// IntValue returns the current value as an int.
func (r *reader) IntValue() (int, error) {
	i, err := r.Int64Value()
	if err != nil {
		return 0, err
	}
	if i > math.MaxInt32 || i < math.MinInt32 {
		return 0, errors.New("ion: int value out of bounds")
	}
	return int(i), nil
}

// Int64Value returns the current value as an int64.
func (r *reader) Int64Value() (int64, error) {
	if r.valueType == IntType {
		if r.value == nil {
			return 0, nil
		}

		if i, ok := r.value.(int64); ok {
			return i, nil
		}

		bi := r.value.(*big.Int)
		if bi.IsInt64() {
			return bi.Int64(), nil
		}

		return 0, errors.New("ion: int value out of bounds")
	}
	return 0, errors.New("ion: value is not an int")
}

// BigIntValue returns the current value as a big int.
func (r *reader) BigIntValue() (*big.Int, error) {
	if r.valueType == IntType {
		if r.value == nil {
			return nil, nil
		}
		if i, ok := r.value.(int64); ok {
			return big.NewInt(i), nil
		}
		return r.value.(*big.Int), nil
	}
	return nil, errors.New("ion: value is not an int")
}

// FloatValue returns the current value as a float.
func (r *reader) FloatValue() (float64, error) {
	if r.valueType == FloatType {
		if r.value == nil {
			return 0.0, nil
		}
		return r.value.(float64), nil
	}
	return 0.0, errors.New("ion: value is not a float")
}

// DecimalValue returns the current value as a Decimal.
func (r *reader) DecimalValue() (*Decimal, error) {
	if r.valueType == DecimalType {
		if r.value == nil {
			return nil, nil
		}
		return r.value.(*Decimal), nil
	}
	return nil, errors.New("ion: value is not a decimal")
}

// TimeValue returns the current value as a time.
func (r *reader) TimeValue() (time.Time, error) {
	if r.valueType == TimestampType {
		if r.value == nil {
			return time.Time{}, nil
		}
		return r.value.(time.Time), nil
	}
	return time.Time{}, errors.New("ion: value is not a timestamp")
}

// StringValue returns the current value as a string.
func (r *reader) StringValue() (string, error) {
	if r.valueType == StringType || r.valueType == SymbolType {
		if r.value == nil {
			return "", nil
		}
		return r.value.(string), nil
	}
	return "", errors.New("ion: value is not a string")
}

// ByteValue returns the current value as a byte slice.
func (r *reader) ByteValue() ([]byte, error) {
	if r.valueType == BlobType || r.valueType == ClobType {
		if r.value == nil {
			return nil, nil
		}
		return r.value.([]byte), nil
	}
	return nil, errors.New("ion: value is not a byte array")
}

func (r *reader) clear() {
	r.fieldName = ""
	r.annotations = nil
	r.valueType = NoType
	r.value = nil
}
