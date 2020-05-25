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
	if IsScalar(NoType) {
		t.Errorf("Expected IsScalar() to return false for type NoType")
	}
	if !IsScalar(NullType) {
		t.Errorf("Expected IsScalar() to return true for type NullType")
	}
	if !IsScalar(BoolType) {
		t.Errorf("Expected IsScalar() to return true for type BoolType")
	}
	if !IsScalar(IntType) {
		t.Errorf("Expected IsScalar() to return true for type IntType")
	}
	if !IsScalar(FloatType) {
		t.Errorf("Expected IsScalar() to return true for type FloatType")
	}
	if !IsScalar(DecimalType) {
		t.Errorf("Expected IsScalar() to return true for type DecimalType")
	}
	if !IsScalar(TimestampType) {
		t.Errorf("Expected IsScalar() to return true for type TimestampType")
	}
	if !IsScalar(SymbolType) {
		t.Errorf("Expected IsScalar() to return true for type SymbolType")
	}
	if !IsScalar(StringType) {
		t.Errorf("Expected IsScalar() to return true for type StringType")
	}
	if !IsScalar(ClobType) {
		t.Errorf("Expected IsScalar() to return true for type ClobType")
	}
	if !IsScalar(BlobType) {
		t.Errorf("Expected IsScalar() to return true for type BlobType")
	}
	if IsScalar(ListType) {
		t.Errorf("Expected IsScalar() to return false for type ListType")
	}
	if IsScalar(SexpType) {
		t.Errorf("Expected IsScalar() to return false for type SexpType")
	}
	if IsScalar(StructType) {
		t.Errorf("Expected IsScalar() to return false for type StructType")
	}
}

func TestIsContainer(t *testing.T) {
	if IsContainer(NoType) {
		t.Errorf("Expected IsContainer() to return false for type NoType")
	}
	if IsContainer(NullType) {
		t.Errorf("Expected IsContainer() to return false for type NullType")
	}
	if IsContainer(BoolType) {
		t.Errorf("Expected IsContainer() to return false for type BoolType")
	}
	if IsContainer(IntType) {
		t.Errorf("Expected IsContainer() to return false for type IntType")
	}
	if IsContainer(FloatType) {
		t.Errorf("Expected IsContainer() to return false for type FloatType")
	}
	if IsContainer(DecimalType) {
		t.Errorf("Expected IsContainer() to return false for type DecimalType")
	}
	if IsContainer(TimestampType) {
		t.Errorf("Expected IsContainer() to return false for type TimestampType")
	}
	if IsContainer(SymbolType) {
		t.Errorf("Expected IsContainer() to return false for type SymbolType")
	}
	if IsContainer(StringType) {
		t.Errorf("Expected IsContainer() to return false for type StringType")
	}
	if IsContainer(ClobType) {
		t.Errorf("Expected IsContainer() to return false for type ClobType")
	}
	if IsContainer(BlobType) {
		t.Errorf("Expected IsContainer() to return false for type BlobType")
	}
	if !IsContainer(ListType) {
		t.Errorf("Expected IsContainer() to return true for type ListType")
	}
	if !IsContainer(SexpType) {
		t.Errorf("Expected IsContainer() to return true for type SexpType")
	}
	if !IsContainer(StructType) {
		t.Errorf("Expected IsContainer() to return true for type StructType")
	}
}
