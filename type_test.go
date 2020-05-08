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
