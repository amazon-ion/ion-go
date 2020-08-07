package ion

type ImportLocation struct {
	importName *string
	sid        int64
}

// Create a new ImportLocation struct.
func NewImportLocation(importName *string, sid int64) *ImportLocation {
	return &ImportLocation{importName, sid}
}

func (iL *ImportLocation) Equal(o *ImportLocation) bool {
	return *iL.importName == *o.importName && iL.sid == o.sid
}