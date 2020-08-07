package ion

import "testing"

func TestNewImportLocationSidAndTextUnknown(t *testing.T) {
	st := NewImportLocation(nil, UnknownSid)
	if st.importName != nil {
		t.Errorf("expected %v, got %v", nil, st.importName)
	}
	if st.sid != UnknownSid {
		t.Errorf("expected %v, got %v", UnknownSid, st.sid)
	}
}

var importLocationBoolEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 123},
	{"text2", 456, "text2", 456},
}

func TestImportLocationBoolEqualsOperator(t *testing.T) {
	for i, _ := range importLocationBoolEqualsTestData {
		st1 := NewImportLocation(&importLocationBoolEqualsTestData[i].text1, importLocationBoolEqualsTestData[i].sid1)
		st2 := NewImportLocation(&importLocationBoolEqualsTestData[i].text2, importLocationBoolEqualsTestData[i].sid2)

		if !st1.Equal(st2) {
			t.Errorf("expected %v, got %v", true, false)
		}
	}
}

var importLocationBoolNotEqualsTestData = []struct {
	text1 string
	sid1  int64
	text2 string
	sid2  int64
}{
	{"text1", 123, "text1", 456},
	{"text2", 456, "text3", 456},
}

func TestImportLocationBoolNotEqualsOperator(t *testing.T) {
	for i, _ := range importLocationBoolNotEqualsTestData {
		st1 := NewImportLocation(&importLocationBoolNotEqualsTestData[i].text1, importLocationBoolNotEqualsTestData[i].sid1)
		st2 := NewImportLocation(&importLocationBoolNotEqualsTestData[i].text2, importLocationBoolNotEqualsTestData[i].sid2)

		if st1.Equal(st2) {
			t.Errorf("expected %v, got %v", false, true)
		}
	}
}
