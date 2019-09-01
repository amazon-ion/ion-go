package ion

import (
	"bufio"
	"errors"
	"fmt"
)

// A binaryReader reads binary Ion.
type binaryReader struct {
	reader

	bits bitstream
	cat  *Catalog
	lst  SymbolTable
}

func newBinaryReaderBuf(in *bufio.Reader, cat *Catalog) Reader {
	r := &binaryReader{
		cat: cat,
	}
	r.bits.Init(in)
	return r
}

func (r *binaryReader) SymbolTable() SymbolTable {
	return r.lst
}

func (r *binaryReader) Next() bool {
	if r.eof || r.err != nil {
		return false
	}

	r.clear()

	done := false
	for !done {
		done, r.err = r.next()
		if r.err != nil {
			return false
		}
	}

	return !r.eof
}

func (r *binaryReader) next() (bool, error) {
	if err := r.bits.Next(); err != nil {
		return false, err
	}

	switch r.bits.Code() {
	case bitcodeEOF:
		r.eof = true
		return true, nil

	case bitcodeBVM:
		err := r.readBVM()
		return false, err

	case bitcodeFieldID:
		err := r.readFieldName()
		return false, err

	case bitcodeAnnotation:
		err := r.readAnnotations()
		return false, err

	case bitcodeNull:
		if !r.bits.Null() {
			// NOP padding; skip it and keep going.
			err := r.bits.SkipValue()
			return false, err
		}
		r.valueType = NullType
		return true, nil

	case bitcodeFalse, bitcodeTrue:
		r.valueType = BoolType
		if !r.bits.Null() {
			r.value = (r.bits.Code() == bitcodeTrue)
		}
		return true, nil

	case bitcodeInt, bitcodeNegInt:
		r.valueType = IntType
		if !r.bits.Null() {
			val, err := r.bits.ReadInt()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeFloat:
		r.valueType = FloatType
		if !r.bits.Null() {
			val, err := r.bits.ReadFloat()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeDecimal:
		r.valueType = DecimalType
		if !r.bits.Null() {
			val, err := r.bits.ReadDecimal()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeTimestamp:
		r.valueType = TimestampType
		if !r.bits.Null() {
			val, err := r.bits.ReadTimestamp()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeSymbol:
		r.valueType = SymbolType
		if !r.bits.Null() {
			id, err := r.bits.ReadSymbol()
			if err != nil {
				return false, err
			}
			r.value = r.resolve(id)
		}
		return true, nil

	case bitcodeString:
		r.valueType = StringType
		if !r.bits.Null() {
			val, err := r.bits.ReadString()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeClob:
		r.valueType = ClobType
		if !r.bits.Null() {
			val, err := r.bits.ReadBytes()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeBlob:
		r.valueType = BlobType
		if !r.bits.Null() {
			val, err := r.bits.ReadBytes()
			if err != nil {
				return false, err
			}
			r.value = val
		}
		return true, nil

	case bitcodeList:
		r.valueType = ListType
		if !r.bits.Null() {
			r.value = ListType
		}
		return true, nil

	case bitcodeSexp:
		r.valueType = SexpType
		if !r.bits.Null() {
			r.value = SexpType
		}
		return true, nil

	case bitcodeStruct:
		r.valueType = StructType
		if !r.bits.Null() {
			r.value = StructType
		}

		if len(r.annotations) > 0 && r.annotations[0] == "$ion_symbol_table" {
			err := r.readLocalSymbolTable()
			return false, err
		}

		return true, nil

	default:
		panic(fmt.Sprintf("unsupported bitcode %v", r.bits.Code()))
	}
}

func (r *binaryReader) readBVM() error {
	major, minor, err := r.bits.ReadBVM()
	if err != nil {
		return err
	}

	if major != 1 && minor != 0 {
		return fmt.Errorf("ion: unsupported version %v.%v", major, minor)
	}

	r.lst = V1SystemSymbolTable
	return nil
}

func (r *binaryReader) readFieldName() error {
	id, err := r.bits.ReadFieldID()
	if err != nil {
		return err
	}

	r.fieldName = r.resolve(id)
	return nil
}

func (r *binaryReader) readAnnotations() error {
	ids, err := r.bits.ReadAnnotations()
	if err != nil {
		return err
	}

	as := make([]string, len(ids))
	for i, id := range ids {
		as[i] = r.resolve(id)
	}

	r.annotations = as
	return nil
}

func (r *binaryReader) resolve(id uint64) string {
	s, ok := r.lst.FindByID(int(id))
	if !ok {
		return fmt.Sprintf("$%v", id)
	}
	return s
}

func (r *binaryReader) readLocalSymbolTable() error {
	if err := r.StepIn(); err != nil {
		return err
	}

	imps := []SharedSymbolTable{}
	syms := []string{}

	for r.Next() {
		var err error
		switch r.FieldName() {
		case "imports":
			imps, err = r.readImports()
		case "symbols":
			syms, err = r.readSymbols()
		}
		if err != nil {
			return err
		}
	}

	if err := r.StepOut(); err != nil {
		return err
	}

	r.lst = NewLocalSymbolTable(imps, syms)
	return nil
}

func (r *binaryReader) readImports() ([]SharedSymbolTable, error) {
	if r.Type() != ListType {
		return nil, nil
	}
	if err := r.StepIn(); err != nil {
		return nil, err
	}

	imps := []SharedSymbolTable{}
	for r.Next() {
		imp, err := r.readImport()
		if err != nil {
			return nil, err
		}
		imps = append(imps, imp)
	}

	err := r.StepOut()
	return imps, err
}

func (r *binaryReader) readImport() (SharedSymbolTable, error) {
	if r.Type() != StructType {
		return nil, nil
	}
	if err := r.StepIn(); err != nil {
		return nil, err
	}

	name := ""
	version := 0
	maxID := 0

	for r.Next() {
		var err error
		switch r.FieldName() {
		case "name":
			name, err = r.StringValue()
		case "version":
			version, err = r.IntValue()
		case "max_id":
			maxID, err = r.IntValue()
		}
		if err != nil {
			return nil, err
		}
	}

	if err := r.StepOut(); err != nil {
		return nil, err
	}

	if name == "" || version == 0 || maxID == 0 {
		return nil, errors.New("ion: invalid import in local symbol table")
	}

	var imp SharedSymbolTable
	if r.cat != nil {
		imp = r.cat.Find(name, version)
		if imp != nil && imp.MaxID() != maxID {
			// TODO: Better error.
			return nil, errors.New("ion: maxID mismatch in imported symbol table")
		}
	}

	if imp == nil {
		imp = &bogusSST{
			name:    name,
			version: version,
			maxID:   maxID,
		}
	}

	return imp, nil
}

func (r *binaryReader) readSymbols() ([]string, error) {
	if r.Type() != ListType {
		return nil, nil
	}
	if err := r.StepIn(); err != nil {
		return nil, err
	}

	syms := []string{}
	for r.Next() {
		if r.Type() == StringType {
			sym, err := r.StringValue()
			if err != nil {
				return nil, err
			}
			syms = append(syms, sym)
		}
	}

	err := r.StepOut()

	return syms, err
}

func (r *binaryReader) StepIn() error {
	if r.err != nil {
		return r.err
	}
	switch r.valueType {
	case ListType, SexpType, StructType:
	default:
		return errors.New("ion: StepIn called when not on a container")
	}

	ctx := containerTypeToCtx(r.valueType)
	r.ctx.push(ctx)

	r.clear()
	r.bits.StepIn()

	return nil
}

func (r *binaryReader) StepOut() error {
	if err := r.bits.StepOut(); err != nil {
		return err
	}

	r.clear()
	r.eof = false

	return nil
}
