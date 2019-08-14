package ion

import "fmt"

type binaryReader struct {
	bits bitstream
	ctx  ctxstack
	eof  bool
	err  error

	lst         SymbolTable
	fieldName   string
	annotations []string
	valueType   Type
	value       interface{}
}

func (r *binaryReader) SymbolTable() SymbolTable {
	return r.lst
}

func (r *binaryReader) Next() bool {
	if r.eof || r.err != nil {
		return false
	}

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

	if r.bits.Code() == bitcodeFieldID {
		if err := r.readFieldName(); err != nil {
			return false, err
		}
	}

	if r.bits.Code() == bitcodeAnnotation {
		if err := r.readAnnotations(); err != nil {
			return false, err
		}
	}

	switch r.bits.Code() {
	case bitcodeEOF:
		r.eof = true
		return true, nil

	case bitcodeBVM:
		if err := r.readBVM(); err != nil {
			return false, err
		}
		return false, nil

	}
	panic(fmt.Sprintf("unsupported bitcode %v", r.bits.Code()))
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

	return r.bits.Next()
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

	return r.bits.Next()
}

func (r *binaryReader) resolve(id uint64) string {
	s, ok := r.lst.FindByID(int(id))
	if !ok {
		return fmt.Sprintf("$%v", id)
	}
	return s
}
