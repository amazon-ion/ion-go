package ion

import "testing"

func TestTypeToString(t *testing.T) {
	for i := NoType; i <= StructType+1; i++ {
		str := i.String()
		if str == "" {
			t.Errorf("expected a non-empty string for type %v", uint8(i))
		}
	}
}

func TestIntSizeToString(t *testing.T) {
	for i := NullInt; i <= BigInt+1; i++ {
		str := i.String()
		if str == "" {
			t.Errorf("expected a non-empty string for size %v", uint8(i))
		}
	}
}

func TestIsScalar(t *testing.T) {
	scalarTypes := []Type{NullType, BoolType, IntType, FloatType, DecimalType,
		TimestampType, SymbolType, StringType, ClobType, BlobType}

	for _, ionType := range scalarTypes {
		if !IsScalar(ionType) {
			t.Errorf("Expected IsScalar() to return true for type %s", ionType.String())
		}
	}

	nonScalarTypes := []Type{NoType, ListType, SexpType, StructType}

	for _, ionType := range nonScalarTypes {
		if IsScalar(ionType) {
			t.Errorf("Expected IsScalar() to return false for type %s", ionType.String())
		}
	}
}

func TestIsContainer(t *testing.T) {
	containerTypes := []Type{ListType, SexpType, StructType}

	for _, ionType := range containerTypes {
		if !IsContainer(ionType) {
			t.Errorf("Expected IsContainer() to return true for type %s", ionType.String())
		}
	}

	nonContainerTypes := []Type{NoType, NullType, BoolType, IntType, FloatType, DecimalType,
		TimestampType, SymbolType, StringType, ClobType, BlobType}

	for _, ionType := range nonContainerTypes {
		if IsContainer(ionType) {
			t.Errorf("Expected IsContainer() to return false for type %s", ionType.String())
		}
	}
}
