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
	if NoType.IsScalar() {
		t.Errorf("Expected IsScalar() to return false for type NoType")
	}
	for i := NullType; i <= BlobType; i++ {
		if !i.IsScalar() {
			t.Errorf("Expected IsScalar() to return true for type %s", i.String())
		}
	}
	for i := ListType; i <= StructType; i++ {
		if i.IsScalar() {
			t.Errorf("Expected IsScalar() to return false for type %s", i.String())
		}
	}
}

func TestIsContainer(t *testing.T) {
	for i := NoType; i <= BlobType; i++ {
		if i.IsContainer() {
			t.Errorf("Expected IsContainer() to return false for type %s", i.String())
		}
	}
	for i := ListType; i <= StructType; i++ {
		if !i.IsContainer() {
			t.Errorf("Expected IsContainer() to return true for type %s", i.String())
		}
	}
}
