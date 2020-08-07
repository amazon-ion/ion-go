package ion

var UnknownSid int64 = -1

// A SymbolToken providing both the symbol text and the assigned symbol ID.
// Symbol tokens may be interned into a SymbolTable.
// A text=nil or sid=-1 value might indicate that such field is unknown in the contextual symbol table.
type SymbolToken struct {
	text           *string
	sid            int64
	importLocation *ImportLocation
}

// Gets the ID of this symbol token.
func (st *SymbolToken) Sid() int64 {
	return st.sid
}

// Gets the text of this symbol token.
func (st *SymbolToken) Text() *string {
	return st.text
}

func (st *SymbolToken) Equal(o *SymbolToken) bool {
	return *st.text == *o.text && st.sid == o.sid
}

// Create a new SymbolToken struct.
func NewSymbolToken(text *string, sid int64, importLocation *ImportLocation) *SymbolToken {
	return &SymbolToken{text: text, sid: sid, importLocation: importLocation}
}
