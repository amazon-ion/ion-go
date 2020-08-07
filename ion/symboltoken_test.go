package ion

import "testing"

func TestNewSymbolTokenSidAndTextUnknown(t *testing.T) {
	st := NewSymbolToken(nil, UnknownSid, nil)
	if st.text != nil {
		t.Errorf("expected %v, got %v", nil, st.text)
	}
	if st.sid != UnknownSid {
		t.Errorf("expected %v, got %v", UnknownSid, st.sid)
	}
	if st.importLocation != nil {
		t.Errorf("expected %v, got %v", nil, st.importLocation)
	}
}

var boolEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 123},
	{"text2", 456, "text2", 456},
}

func TestBoolEqualsOperator(t *testing.T) {
	for i, _ := range boolEqualsTestData {
		st1 := NewSymbolToken(&boolEqualsTestData[i].text1, boolEqualsTestData[i].sid1, nil)
		st2 := NewSymbolToken(&boolEqualsTestData[i].text2, boolEqualsTestData[i].sid2, nil)

		if !st1.Equal(st2) {
			t.Errorf("expected %v, got %v", true, false)
		}
	}
}

var boolNotEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 456},
	{"text2", 456, "text3", 456},
}

func TestBoolNotEqualsOperator(t *testing.T) {
	for i, _ := range boolNotEqualsTestData {
		st1 := NewSymbolToken(&boolNotEqualsTestData[i].text1, boolNotEqualsTestData[i].sid1, nil)
		st2 := NewSymbolToken(&boolNotEqualsTestData[i].text2, boolNotEqualsTestData[i].sid2, nil)

		if st1.Equal(st2) {
			t.Errorf("expected %v, got %v", false, true)
		}
	}
}