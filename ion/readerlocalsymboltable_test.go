package ion

import (
	"testing"
)

func TestLocalSymbolTableAppend(t *testing.T) {
	text := "$ion_symbol_table::" +
		"{" +
		"  symbols:[ \"s1\", \"s2\" ]" +
		"}\n" +
		"$ion_symbol_table::" +
		"{" +
		"  imports: $ion_symbol_table," +
		"  symbols:[ \"s3\", \"s4\", \"s5\" ]" +
		"}\n" +
		"null"

	r := NewReaderString(text)
	r.Next()
	st := r.SymbolTable()

	imports := st.Imports()
	systemTable := imports[0]
	systemMaxID := systemTable.MaxID()

	checkSymbol(t, "s1", systemMaxID+1, st)
	checkSymbol(t, "s2", systemMaxID+2, st)
	checkSymbol(t, "s3", systemMaxID+3, st)
	checkSymbol(t, "s4", systemMaxID+4, st)
	checkSymbol(t, "s5", systemMaxID+5, st)
	checkUnknownSymbolText(t, "unknown", st)
	checkUnknownSymbolID(t, 33, st)
}

func TestLocalSymbolTableMultiAppend(t *testing.T) {
	text := "$ion_symbol_table::" +
		"{" +
		"  symbols:[ \"s1\", \"s2\" ]" +
		"}\n" +
		"$ion_symbol_table::" +
		"{" +
		"  imports: $ion_symbol_table," +
		"  symbols:[ \"s3\" ]" +
		"}\n" +
		"$ion_symbol_table::" +
		"{" +
		"  imports: $ion_symbol_table," +
		"  symbols:[ \"s4\", \"s5\" ]" +
		"}\n" +
		"$ion_symbol_table::" +
		"{" +
		"  imports: $ion_symbol_table," +
		"  symbols:[ \"s6\" ]" +
		"}\n" +
		"null"

	r := NewReaderString(text)
	r.Next()
	st := r.SymbolTable()

	imports := st.Imports()
	systemTable := imports[0]
	systemMaxID := systemTable.MaxID()

	checkSymbol(t, "s1", systemMaxID+1, st)
	checkSymbol(t, "s2", systemMaxID+2, st)
	checkSymbol(t, "s3", systemMaxID+3, st)
	checkSymbol(t, "s4", systemMaxID+4, st)
	checkSymbol(t, "s5", systemMaxID+5, st)
	checkSymbol(t, "s6", systemMaxID+6, st)

	checkUnknownSymbolText(t, "unknown", st)
	checkUnknownSymbolID(t, 33, st)
}

func TestLocalSymbolTableAppendEmptyList(t *testing.T) {
	original := "$ion_symbol_table::" +
		"{" +
		"  symbols:[ \"s1\" ]" +
		"}\n"

	appended := original +
		"$ion_symbol_table::" +
		"{" +
		"  imports: $ion_symbol_table," +
		"  symbols:[]" +
		"}\n" +
		"null"

	r := NewReaderString(original + "null")
	r.Next()
	ost := r.SymbolTable()

	originalSymbol := ost.Find("s1")

	r = NewReaderString(appended)
	r.Next()
	ast := r.SymbolTable()
	appendedSymbol := ast.Find("s1")

	if originalSymbol.LocalSID != appendedSymbol.LocalSID {
		t.Errorf("Original symbol SID: %v,did not match Appended symbol SID: %v", originalSymbol.LocalSID, appendedSymbol.LocalSID)
	}
}

func TestLocalSymbolTableAppendNonUnique(t *testing.T) {
	text := "$ion_symbol_table::" +
		"{" +
		"  symbols:[ \"foo\" ]" +
		"}" +
		"$10\n" +
		"$ion_symbol_table::" +
		"{" +
		"  imports: $ion_symbol_table," +
		"  symbols:[ \"foo\", \"bar\" ]" +
		"}" +
		"$11\n" +
		"$12\n"

	r := NewReaderString(text)
	r.Next()
	r.Next()
	st := r.SymbolTable()
	systemMaxID := getSystemMaxId(st)

	checkSymbol(t, "foo", systemMaxID+1, st)
	checkSymbol(t, "foo", systemMaxID+2, st)
	checkSymbol(t, "bar", systemMaxID+3, st)
}

func TestLocalSymbolTableAppendOutOfBounds(t *testing.T) {
	text := "$ion_symbol_table::" +
		"{" +
		"  symbols:[ \"foo\" ]" +
		"}" +
		"$10\n" +
		"$ion_symbol_table::" +
		"{" +
		"  imports: $ion_symbol_table," +
		"  symbols:[ \"foo\" ]" +
		"}" +
		"$11\n" +
		"$12\n"

	r := NewReaderString(text)
	r.Next()
	r.Next()
	st := r.SymbolTable()
	systemMaxID := getSystemMaxId(st)

	checkSymbol(t, "foo", systemMaxID+1, st)
	checkSymbol(t, "foo", systemMaxID+2, st)
	checkUnknownSymbolID(t, systemMaxID+3, st)
}

func getSystemMaxId(st SymbolTable) uint64 {
	imports := st.Imports()
	systemTable := imports[0]
	return systemTable.MaxID()
}

func checkSymbol(t *testing.T, eval string, SID uint64, st SymbolTable) {
	val, ok := st.FindByID(SID)
	if !ok {
		t.Errorf("Failed on checking symbol. Symbol table could not find symbol given the ID: %v", SID)
	}
	if val != eval {
		t.Errorf("Failed on checking symbol. Symbol table returned the symbol name: %v, did not match the expected name: %v", val, eval)
	}
}

func checkUnknownSymbolText(t *testing.T, val string, st SymbolTable) {
	symbolToken := st.Find(val)
	if symbolToken != nil {
		t.Errorf("Failed on checking unknown symbol. Symbol table found symbol given the name: %v", val)
	}
	_, ok := st.FindByName(val)
	if ok {
		t.Errorf("Failed on checking unknown symbol. Symbol table found symbol given the name: %v", val)
	}
}

func checkUnknownSymbolID(t *testing.T, val uint64, st SymbolTable) {
	_, ok := st.FindByID(val)
	if ok {
		t.Errorf("Failed on checking unknown symbol. Symbol table found symbol given the SID: %v", val)
	}
}
