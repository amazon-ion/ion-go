package ion

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var blacklist = map[string]bool{
	"ion-tests/iontestdata/good/emptyAnnotatedInt.10n":    true,
	"ion-tests/iontestdata/good/subfieldVarUInt32bit.ion": true,
	"ion-tests/iontestdata/good/utf16.ion":                true,
	"ion-tests/iontestdata/good/utf32.ion":                true,
	"ion-tests/iontestdata/good/whitespace.ion":           true,
	"ion-tests/iontestdata/good/item1.10n":                true,
}

func print(level int, obj interface{}) {
	fmt.Print(" > ")
	for i := 0; i < level; i++ {
		fmt.Print("  ")
	}
	fmt.Println(obj)
}

func drain(t *testing.T, r Reader, level int) {
	for r.Next() {
		// print(level, r.Type())

		if !r.IsNull() {
			switch r.Type() {
			case StructType, ListType, SexpType:
				if err := r.StepIn(); err != nil {
					t.Fatal(err)
				}

				drain(t, r, level+1)

				if err := r.StepOut(); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	if r.Err() != nil {
		t.Fatal(r.Err())
	}
}

func testReaderFile(t *testing.T, path string) {
	if _, ok := blacklist[path]; ok {
		return
	}

	// fmt.Println(path)

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var r Reader

	if strings.HasSuffix(path, ".ion") {
		r = NewTextReader(file)
		// r.(*textReader).debug = true
	} else if strings.HasSuffix(path, ".10n") {
		// Binary ion not yet supported.
		return
	} else {
		t.Fatal("unexpected suffix on file", path)
	}

	drain(t, r, 0)
}

func testReaderDir(t *testing.T, path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		fp := filepath.Join(path, file.Name())
		if file.IsDir() {
			testReaderDir(t, fp)
		} else {
			t.Run(fp, func(t *testing.T) {
				testReaderFile(t, fp)
			})
		}
	}
}

func TestReader(t *testing.T) {
	testReaderDir(t, "ion-tests/iontestdata/good")
}

// func TestAllNulls(t *testing.T) {
// 	testReaderFile(t, "ion-tests/iontestdata/good/allNulls.ion")
// }

// func TestStructWhitespace(t *testing.T) {
// 	testReaderFile(t, "ion-tests/iontestdata/good/equivs/structWhitespace.ion")
// }
