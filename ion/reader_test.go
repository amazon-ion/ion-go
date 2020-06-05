package ion

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var blacklist = map[string]bool{
	"../ion-tests/iontestdata/good/emptyAnnotatedInt.10n":    true,
	"../ion-tests/iontestdata/good/subfieldVarUInt32bit.ion": true,
	"../ion-tests/iontestdata/good/utf16.ion":                true,
	"../ion-tests/iontestdata/good/utf32.ion":                true,
	"../ion-tests/iontestdata/good/whitespace.ion":           true,
	"../ion-tests/iontestdata/good/item1.10n":                true,
	"../ion-tests/iontestdata/good/typecodes/T7-large.10n":   true,
}

type drainfunc func(t *testing.T, r Reader, f string)

func TestDecodeFiles(t *testing.T) {
	testReadDir(t, "../ion-tests/iontestdata/good", func(t *testing.T, r Reader, f string) {
		// fmt.Println(f)
		d := NewDecoder(r)
		for {
			v, err := d.Decode()
			if err == ErrNoInput {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			// fmt.Println(v)
			_ = v
		}
	})
}

func testReadDir(t *testing.T, path string, d drainfunc) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		fp := filepath.Join(path, file.Name())
		if file.IsDir() {
			testReadDir(t, fp, d)
		} else {
			t.Run(fp, func(t *testing.T) {
				testReadFile(t, fp, d)
			})
		}
	}
}

func testReadFile(t *testing.T, path string, d drainfunc) {
	if _, ok := blacklist[path]; ok {
		return
	}
	if strings.HasSuffix(path, "md") {
		return
	}

	//fmt.Printf("**** PATH = %s\n", path)

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	r := NewReader(file)

	d(t, r, path)
}
