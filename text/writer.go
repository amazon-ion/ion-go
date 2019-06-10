package text

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/fernomac/ion-go"
)

type contextType uint8

const (
	topLevelCtx contextType = iota
	inStructCtx
	inListCtx
	inSexpCtx
)

type context struct {
	value  contextType
	parent *context
}

var topLevel = &context{value: topLevelCtx, parent: nil}

type writer struct {
	out io.Writer
	ctx *context
	err error

	fieldName       string
	typeAnnotations []string
	needsSeparator  bool
}

// NewWriter returns a new text writer.
func NewWriter(out io.Writer) ion.Writer {
	return &writer{
		out: out,
		ctx: topLevel,
	}
}

func (w *writer) push(t contextType) {
	ctx := &context{
		value:  t,
		parent: w.ctx,
	}
	w.ctx = ctx
}

func (w *writer) pop() {
	if w.ctx.parent == nil {
		panic("pop called at the top level")
	}
	w.ctx = w.ctx.parent
}

func (w *writer) InStruct() bool {
	return (w.ctx.value == inStructCtx)
}

func (w *writer) Err() error {
	return w.err
}

func (w *writer) FieldName(val string) {
	if w.err != nil {
		return
	}
	if !w.InStruct() {
		w.err = errors.New("field name called while not in a struct")
		return
	}
	w.fieldName = val
}

func (w *writer) TypeAnnotation(val string) {
	if w.err != nil {
		return
	}
	w.typeAnnotations = append(w.typeAnnotations, val)
}

func (w *writer) TypeAnnotations(val ...string) {
	if w.err != nil {
		return
	}
	w.typeAnnotations = append(w.typeAnnotations, val...)
}

func (w *writer) beginValue() error {
	if w.needsSeparator {
		var sep byte
		switch w.ctx.value {
		case inStructCtx, inListCtx:
			sep = ','
		case inSexpCtx:
			sep = ' '
		default:
			sep = '\n'
		}

		if err := writeChar(sep, w.out); err != nil {
			return err
		}
	}

	if w.InStruct() {
		if w.fieldName == "" {
			return errors.New("field name not set")
		}
		name := w.fieldName
		w.fieldName = ""

		if err := writeSymbol(name, w.out); err != nil {
			return err
		}
		if err := writeChar(':', w.out); err != nil {
			return err
		}
	}

	if len(w.typeAnnotations) > 0 {
		as := w.typeAnnotations
		w.typeAnnotations = nil

		for _, a := range as {
			if err := writeSymbol(a, w.out); err != nil {
				return err
			}
			if err := writeString("::", w.out); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *writer) endValue() {
	w.needsSeparator = true
}

func (w *writer) begin(t contextType, c byte) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	w.push(t)
	w.needsSeparator = false

	return writeChar(c, w.out)
}

func (w *writer) end(t contextType, c byte) error {
	if w.ctx.value != t {
		return errors.New("not in an appropriate container")
	}

	if err := writeChar(c, w.out); err != nil {
		return err
	}

	w.fieldName = ""
	w.typeAnnotations = nil
	w.pop()
	w.endValue()

	return nil
}

func (w *writer) BeginStruct() {
	if w.err != nil {
		return
	}
	w.err = w.begin(inStructCtx, '{')
}

func (w *writer) EndStruct() {
	if w.err != nil {
		return
	}
	w.err = w.end(inStructCtx, '}')
}

func (w *writer) BeginList() {
	if w.err != nil {
		return
	}
	w.err = w.begin(inListCtx, '[')
}

func (w *writer) EndList() {
	if w.err != nil {
		return
	}
	w.err = w.end(inListCtx, ']')
}

func (w *writer) BeginSexp() {
	if w.err != nil {
		return
	}
	w.err = w.begin(inSexpCtx, '(')
}

func (w *writer) EndSexp() {
	if w.err != nil {
		return
	}
	w.err = w.end(inSexpCtx, ')')
}

func (w *writer) writeValue(f func() string) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	sym := f()
	if err := writeString(sym, w.out); err != nil {
		return err
	}

	w.endValue()
	return nil
}

func (w *writer) writeValueStreaming(f func() error) error {
	if err := w.beginValue(); err != nil {
		return err
	}

	if err := f(); err != nil {
		return err
	}

	w.endValue()
	return nil
}

func (w *writer) WriteNull() {
	w.WriteNullWithType(ion.NullType)
}

func (w *writer) WriteNullWithType(t ion.Type) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		switch t {
		case ion.NullType:
			return "null"
		case ion.BoolType:
			return "null.bool"
		case ion.IntType:
			return "null.int"
		case ion.FloatType:
			return "null.float"
		case ion.DecimalType:
			return "null.decimal"
		case ion.TimestampType:
			return "null.timestamp"
		case ion.StringType:
			return "null.string"
		case ion.SymbolType:
			return "null.symbol"
		case ion.BlobType:
			return "null.blob"
		case ion.ClobType:
			return "null.clob"
		case ion.StructType:
			return "null.struct"
		case ion.ListType:
			return "null.list"
		case ion.SexpType:
			return "null.sexp"
		default:
			panic("invalid type")
		}
	})
}

func symbolForBool(val bool) string {
	if val {
		return "true"
	}
	return "false"
}

func (w *writer) WriteBool(val bool) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		if val {
			return "true"
		}
		return "false"
	})
}

func (w *writer) WriteInt(val int64) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		return fmt.Sprintf("%d", val)
	})
}

func (w *writer) WriteBigInt(val *big.Int) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		return val.String()
	})
}

func (w *writer) WriteFloat(val float64) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		// Built-in go formatting isn't up to the task. :(
		str := strconv.FormatFloat(val, 'e', -1, 64)

		switch str {
		case "NaN": return "nan"
		case "+Inf": return "+inf"
		case "-Inf": return "-inf"
		default: break
		}

		idx := strings.Index(str, "e")
		if idx < 0 {
			str += "e0"
		} else if idx+2 < len(str) && str[idx+2] == '0' {
			str = str[:idx+2] + str[idx+3:]
		}

		return str
	})
}

func (w *writer) WriteDecimal(val *ion.Decimal) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		return val.String()
	})
}

func (w *writer) WriteTimestamp(val time.Time) {
	if w.err != nil {
		return
	}
	w.err = w.writeValue(func() string {
		return val.Format(time.RFC3339Nano)
	})
}

func (w *writer) WriteSymbol(val string) {
	if w.err != nil {
		return
	}
	w.err = w.writeValueStreaming(func() error {
		return writeSymbol(val, w.out)
	})
}

func (w *writer) WriteString(val string) {
	if w.err != nil {
		return
	}
	w.err = w.writeValueStreaming(func() error {
		if err := writeChar('"', w.out); err != nil {
			return err
		}
		if err := writeEscapedString(val, w.out); err != nil {
			return err
		}
		return writeChar('"', w.out)
	})
}

func (w *writer) WriteBlob(val []byte) {
	if w.err != nil {
		return
	}
	w.err = w.writeValueStreaming(func() error {
		if err := writeString("{{", w.out); err != nil {
			return err
		}

		enc := base64.NewEncoder(base64.StdEncoding, w.out)
		enc.Write(val)
		if err := enc.Close(); err != nil {
			return err
		}

		return writeString("}}", w.out)
	})
}

func (w *writer) WriteClob(val []byte) {
	if w.err != nil {
		return
	}
	w.err = w.writeValueStreaming(func() error {
		if err := writeString("{{\"", w.out); err != nil {
			return err
		}

		for _, c := range val {
			if c < 32 || c == '\\' || c == '"' || c > 0x7F {
				if err := writeEscapedChar(c, w.out); err != nil {
					return err
				}
			} else {
				if err := writeChar(c, w.out); err != nil {
					return err
				}
			}
		}

		return writeString("\"}}", w.out)
	})
}

func (w *writer) Finish() error {
	if w.err != nil {
		return w.err
	}
	if w.ctx.value != topLevelCtx {
		w.err = errors.New("not at top level")
		return w.err
	}

	if w.err = writeChar('\n', w.out); w.err != nil {
		return w.err
	}

	w.fieldName = ""
	w.typeAnnotations = nil
	w.needsSeparator = false
	return nil
}
