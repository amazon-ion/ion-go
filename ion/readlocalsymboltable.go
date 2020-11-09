package ion

import "fmt"

// ReadLocalSymbolTable reads and installs a new local symbol table.
func readLocalSymbolTable(r Reader, cat Catalog) (SymbolTable, error) {
	if err := r.StepIn(); err != nil {
		return nil, err
	}

	var imps []SharedSymbolTable
	var syms []string

	foundImport := false
	foundLocals := false

	for r.Next() {
		var err error
		fieldName, err := r.FieldName()
		if err != nil {
			return nil, err
		}
		if fieldName == nil || fieldName.Text == nil {
			return nil, fmt.Errorf("ion: field name is nil")
		}

		switch *fieldName.Text {
		case "symbols":
			if foundLocals {
				return nil, fmt.Errorf("ion: multiple symbol fields found within a single local symbol table")
			}
			foundLocals = true
			syms, err = readSymbols(r)
		case "imports":
			if foundImport {
				return nil, fmt.Errorf("ion: multiple imports fields found within a single local symbol table")
			}
			foundImport = true
			imps, err = readImports(r, cat)
		}
		if err != nil {
			return nil, err
		}
	}

	if err := r.StepOut(); err != nil {
		return nil, err
	}

	return NewLocalSymbolTable(imps, syms), nil
}

// ReadImports reads the imports field of a local symbol table.
func readImports(r Reader, cat Catalog) ([]SharedSymbolTable, error) {
	if r.Type() == SymbolType {
		val, err := r.SymbolValue()
		if err != nil {
			return nil, err
		}

		if val.LocalSID == 3 {
			// Special case that imports the current local symbol table.
			if r.SymbolTable() == nil || r.SymbolTable() == V1SystemSymbolTable {
				return nil, nil
			}

			imps := r.SymbolTable().Imports()
			lsst := NewSharedSymbolTable("", 0, r.SymbolTable().Symbols())
			return append(imps, lsst), nil
		}
	}

	if r.Type() != ListType || r.IsNull() {
		return nil, nil
	}
	if err := r.StepIn(); err != nil {
		return nil, err
	}

	var imps []SharedSymbolTable
	for r.Next() {
		imp, err := readImport(r, cat)
		if err != nil {
			return nil, err
		}
		if imp != nil {
			imps = append(imps, imp)
		}
	}

	err := r.StepOut()
	return imps, err
}

// ReadImport reads an import definition.
func readImport(r Reader, cat Catalog) (SharedSymbolTable, error) {
	if r.Type() != StructType || r.IsNull() {
		return nil, nil
	}
	if err := r.StepIn(); err != nil {
		return nil, err
	}

	name := ""
	version := -1
	maxID := int64(-1)

	for r.Next() {
		fieldName, err := r.FieldName()
		if err != nil {
			return nil, err
		}
		if fieldName == nil || fieldName.Text == nil {
			return nil, fmt.Errorf("ion: field name is nil")
		}

		switch *fieldName.Text {
		case "name":
			if r.Type() == StringType {
				val, err := r.StringValue()
				if err != nil {
					return nil, err
				}
				name = *val
			}
		case "version":
			if r.Type() == IntType {
				val, err := r.IntValue()
				if err != nil {
					return nil, err
				}
				version = *val
			}
		case "max_id":
			if r.Type() == IntType {
				if r.IsNull() {
					return nil, fmt.Errorf("ion: max id is null")
				}
				i, err := r.Int64Value()
				if err != nil {
					return nil, err
				}
				maxID = *i
			}
		}
	}

	if err := r.StepOut(); err != nil {
		return nil, err
	}

	if name == "" || name == "$ion" {
		return nil, nil
	}
	if version < 1 {
		version = 1
	}

	var imp SharedSymbolTable
	if cat != nil {
		imp = cat.FindExact(name, version)
		if imp == nil {
			imp = cat.FindLatest(name)
		}
	}

	if maxID < 0 {
		if imp == nil || version != imp.Version() {
			return nil, fmt.Errorf("ion: import of shared table %v/%v lacks a valid max_id, but an exact "+
				"match was not found in the catalog", name, version)
		}
		maxID = int64(imp.MaxID())
	}

	if imp == nil {
		imp = &bogusSST{
			name:    name,
			version: version,
			maxID:   uint64(maxID),
		}
	} else {
		imp = imp.Adjust(uint64(maxID))
	}

	return imp, nil
}

// ReadSymbols reads the symbols from a symbol table.
func readSymbols(r Reader) ([]string, error) {
	if r.Type() != ListType {
		return nil, nil
	}
	if err := r.StepIn(); err != nil {
		return nil, err
	}

	var syms []string
	for r.Next() {
		if r.Type() == StringType {
			sym, err := r.StringValue()

			if err != nil {
				return nil, err
			}
			if sym != nil {
				syms = append(syms, *sym)
			} else {
				syms = append(syms, "")
			}
		} else {
			syms = append(syms, "")
		}
	}

	err := r.StepOut()
	return syms, err
}
