/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/amzn/ion-go/ion"
)

// process reads the specified input file(s) and re-writes the contents in the
// specified format.
func process(args []string) error {
	p, err := newProcessor(args)
	if err != nil {
		return err
	}
	return p.run()
}

type processor struct {
	infs []string
	outf string
	errf string

	format string

	out ion.Writer
	err *ErrorReport
	loc string
	idx int
}

func newProcessor(args []string) (*processor, error) {
	ret := &processor{}

	i := 0
	for ; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			break
		}
		if arg == "-" || arg == "--" {
			i++
			break
		}

		switch arg {
		case "-o", "--output":
			i++
			if i >= len(args) {
				return nil, errors.New("no output file specified")
			}
			ret.outf = args[i]

		case "-f", "--output-format":
			i++
			if i >= len(args) {
				return nil, errors.New("no output format specified")
			}
			ret.format = args[i]

		case "-e", "--error-report":
			i++
			if i >= len(args) {
				return nil, errors.New("no error report file specified")
			}
			ret.errf = args[i]

		// https://github.com/amzn/ion-go/issues/121

		default:
			return nil, errors.New("unrecognized option \"" + arg + "\"")
		}
	}

	// Any remaining args are input files.
	for ; i < len(args); i++ {
		ret.infs = append(ret.infs, args[i])
	}

	return ret, nil
}

func (p *processor) run() (deferredErr error) {
	outf, err := OpenOutput(p.outf)
	if err != nil {
		return err
	}
	defer func() {
		closeError := outf.Close()
		if err == nil {
			deferredErr = closeError
		} else {
			deferredErr = err
		}
	}()

	switch p.format {
	case "", "pretty":
		p.out = ion.NewTextWriterOpts(outf, ion.TextWriterPretty)
	case "text":
		p.out = ion.NewTextWriter(outf)
	case "binary":
		p.out = ion.NewBinaryWriter(outf)
	case "events":
		p.out = NewEventWriter(outf)
	case "none":
		p.out = NewNopWriter()
	default:
		err = errors.New("unrecognized output format \"" + p.format + "\"")
		return err
	}

	errf, err := OpenError(p.errf)
	if err != nil {
		return err
	}
	defer func() {
		finishError := errf.Close()
		if err == nil {
			deferredErr = finishError
		} else {
			deferredErr = err
		}
	}()

	p.err = NewErrorReport(errf)
	defer func() {
		finishError := p.err.Finish()
		if err == nil {
			deferredErr = finishError
		} else {
			deferredErr = err
		}
	}()

	if len(p.infs) == 0 {
		p.processStdin()
		return nil
	}

	err = p.processFiles()
	if err != nil {
		return err
	}

	return err
}

func (p *processor) processStdin() {
	p.loc = "stdin"
	p.processReader(stdin{})
	p.loc = ""

	if err := p.out.Finish(); err != nil {
		p.error(write, err)
	}
}

func (p *processor) processFiles() error {
	for _, inf := range p.infs {
		if err := p.processFile(inf); err != nil {
			return err
		}
	}

	if err := p.out.Finish(); err != nil {
		p.error(write, err)
	}

	return nil
}

func (p *processor) processFile(in string) (err error) {
	f, err := OpenInput(in)
	if err != nil {
		return err
	}

	defer func() {
		err = f.Close()
	}()

	p.loc = in
	p.processReader(f)
	p.loc = ""

	return nil
}

func (p *processor) processReader(in io.Reader) {
	// We intentionally ignore the returned error; it's been written
	// to p.err, and only gets returned to short-circuit further execution.
	p.process(ion.NewReader(in))
}

func (p *processor) process(in ion.Reader) error {
	var err error

	for in.Next() {
		p.idx++
		name, e := in.FieldName()
		if e != nil {
			return p.error(read, err)
		}
		if name != nil {
			if err = p.out.FieldName(*name); err != nil {
				return p.error(write, err)
			}
		}

		annos, err := in.Annotations()
		if err != nil {
			return p.error(read, err)
		}
		if len(annos) > 0 {
			if err = p.out.Annotations(annos...); err != nil {
				return p.error(write, err)
			}
		}

		switch in.Type() {
		case ion.NullType:
			err = p.out.WriteNull()

		case ion.BoolType:
			val, err := in.BoolValue()
			if err != nil {
				return p.error(read, err)
			}
			err = p.out.WriteBool(*val)

		case ion.IntType:
			size, err := in.IntSize()
			if err != nil {
				return p.error(read, err)
			}

			switch size {
			case ion.Int32:
				val, err := in.IntValue()
				if err != nil {
					return p.error(read, err)
				}
				err = p.out.WriteInt(int64(*val))

			case ion.Int64:
				val, err := in.Int64Value()
				if err != nil {
					return p.error(read, err)
				}
				err = p.out.WriteInt(*val)

			case ion.BigInt:
				val, err := in.BigIntValue()
				if err != nil {
					return p.error(read, err)
				}
				err = p.out.WriteBigInt(val)

			default:
				panic(fmt.Sprintf("bad int size: %v", size))
			}

		case ion.FloatType:
			val, err := in.FloatValue()
			if err != nil {
				return p.error(read, err)
			}
			err = p.out.WriteFloat(*val)

		case ion.DecimalType:
			val, err := in.DecimalValue()
			if err != nil {
				return p.error(read, err)
			}
			err = p.out.WriteDecimal(val)

		case ion.TimestampType:
			val, err := in.TimestampValue()
			if err != nil {
				return p.error(read, err)
			}
			err = p.out.WriteTimestamp(*val)

		case ion.SymbolType:
			val, err := in.SymbolValue()
			if err != nil {
				return p.error(read, err)
			}
			if val != nil {
				err = p.out.WriteSymbol(*val)
			}

		case ion.StringType:
			val, err := in.StringValue()
			if err != nil {
				return p.error(read, err)
			}
			if val != nil {
				err = p.out.WriteString(*val)
			}

		case ion.ClobType:
			val, err := in.ByteValue()
			if err != nil {
				return p.error(read, err)
			}
			err = p.out.WriteClob(val)

		case ion.BlobType:
			val, err := in.ByteValue()
			if err != nil {
				return p.error(read, err)
			}
			err = p.out.WriteBlob(val)

		case ion.ListType:
			if err := in.StepIn(); err != nil {
				return p.error(read, err)
			}
			if err := p.out.BeginList(); err != nil {
				return p.error(write, err)
			}
			if err := p.process(in); err != nil {
				return err
			}
			p.idx++
			if err := in.StepOut(); err != nil {
				return p.error(read, err)
			}
			err = p.out.EndList()

		case ion.SexpType:
			if err := in.StepIn(); err != nil {
				return p.error(read, err)
			}
			if err := p.out.BeginSexp(); err != nil {
				return p.error(write, err)
			}
			if err := p.process(in); err != nil {
				return err
			}
			p.idx++
			if err := in.StepOut(); err != nil {
				return p.error(read, err)
			}
			err = p.out.EndSexp()

		case ion.StructType:
			if err := in.StepIn(); err != nil {
				return p.error(read, err)
			}
			if err := p.out.BeginStruct(); err != nil {
				return p.error(write, err)
			}
			if err := p.process(in); err != nil {
				return err
			}
			p.idx++
			if err := in.StepOut(); err != nil {
				return p.error(read, err)
			}
			err = p.out.EndStruct()

		default:
			panic(fmt.Sprintf("bad ion type: %v", in.Type()))
		}

		if err != nil {
			return p.error(write, err)
		}
	}

	if err := in.Err(); err != nil {
		return p.error(read, err)
	}
	return nil
}

func (p *processor) error(typ errortype, err error) error {
	p.err.Append(typ, err.Error(), p.loc, p.idx)
	return err
}
