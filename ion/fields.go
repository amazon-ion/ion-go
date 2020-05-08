package ion

import (
	"fmt"
	"reflect"
	"strings"
)

// A field is a reflectively-accessed field of a struct type.
type field struct {
	name      string
	typ       reflect.Type
	path      []int
	omitEmpty bool
}

// A fielder maps out the fields of a type.
type fielder struct {
	fields []field
	index  map[string]bool
}

// FieldsFor returns the fields of the given struct type.
// TODO: cache me.
func fieldsFor(t reflect.Type) []field {
	fldr := fielder{index: map[string]bool{}}
	fldr.inspect(t, nil)
	return fldr.fields
}

// Inspect recursively inspects a type to determine all of its fields.
func (f *fielder) inspect(t reflect.Type, path []int) {
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !visible(&sf) {
			// Skip non-visible fields.
			continue
		}

		tag := sf.Tag.Get("json")
		if tag == "-" {
			// Skip fields that are explicitly hidden by tag.
			continue
		}
		name, opts := parseJSONTag(tag)

		newpath := make([]int, len(path)+1)
		copy(newpath, path)
		newpath[len(path)] = i

		ft := sf.Type
		if ft.Name() == "" && ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		if name == "" && sf.Anonymous && ft.Kind() == reflect.Struct {
			// Dig in to the embedded struct.
			f.inspect(ft, newpath)
		} else {
			// Add this named field.
			if name == "" {
				name = sf.Name
			}

			if f.index[name] {
				panic(fmt.Sprintf("too many fields named %v", name))
			}
			f.index[name] = true

			f.fields = append(f.fields, field{
				name:      name,
				typ:       ft,
				path:      newpath,
				omitEmpty: omitEmpty(opts),
			})
		}
	}
}

// Visible returns true if the given StructField should show up in the output.
func visible(sf *reflect.StructField) bool {
	exported := sf.PkgPath == ""
	if sf.Anonymous {
		t := sf.Type
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Struct {
			// Fields of embedded structs are visible even if the struct type itself is not.
			return true
		}
	}
	return exported
}

// ParseJSONTag parses a `json:"..."` field tag, returning the name and opts.
func parseJSONTag(tag string) (string, string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		// Ignore additional JSON options, at least for now.
		return tag[:idx], tag[idx+1:]
	}
	return tag, ""
}

// OmitEmpty returns true if opts includes "omitempty".
func omitEmpty(opts string) bool {
	for opts != "" {
		var o string

		i := strings.Index(opts, ",")
		if i >= 0 {
			o, opts = opts[:i], opts[i+1:]
		} else {
			o, opts = opts, ""
		}

		if o == "omitempty" {
			return true
		}
	}
	return false
}
