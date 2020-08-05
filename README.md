# Amazon Ion Go

[![Build Status](https://github.com/amzn/ion-go/workflows/Go%20Build/badge.svg)](https://github.com/amzn/ion-go/actions?query=workflow%3A%22Go+Build%22)

Amazon Ion ( http://amzn.github.io/ion-docs/ ) library for Go

***This package is consider beta. While the API is relatively stable it is still subject to change***

This package is based on work from David Murray ([fernomac](https://github.com/fernomac/)) on https://github.com/fernomac/ion-go.
The Ion team greatly appreciates David's contributions to the Ion community.


## Users

Here are some projects that use the Ion Go library

* [Restish](https://rest.sh/): "...a CLI for interacting with REST-ish HTTP APIs with some nice features built-in"


We'll be happy to add you to our list, send us a pull request.


## Git Setup


This repository contains a [git submodule](https://git-scm.com/docs/git-submodule)
called `ion-tests`, which holds test data used by `ion-go`'s unit tests.

The easiest way to clone the `ion-go` repository and initialize its `ion-tests`
submodule is to run the following command.

```
$ git clone --recursive https://github.com/amzn/ion-go.git ion-go
```

Alternatively, the submodule may be initialized independent of the clone
by running the following commands:

```
$ git submodule init
$ git submodule update
```

## Development

This package uses [Go Modules](https://github.com/golang/go/wiki/Modules) to model
its dependencies.

Assuming the `go` command is in your path, building the module can be done as:

```
$ go build -v ./...
```

Running all the tests can be executed with:

```
$ go test -v ./...
```

We use [`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports?tab=doc) to format
our imports and files in general.  Running this before commit is advised:

```
$ goimports -w .
```

It is recommended that you hook this in your favorite IDE (`Tools` > `File Watchers` in Goland, for example).

## Usage

Import `github.com/amzn/ion-go/ion` and you're off to the races.

### Marshaling and Unmarshaling

Similar to GoLang's built-in [json](https://golang.org/pkg/encoding/json/) package,
you can marshal and unmarshal Go types to Ion. Marshaling requires you to specify
whether you'd like text or binary Ion. Unmarshaling is smart enough to do the right
thing. Both follow the style of json name tags, and `Marshal` honors `omitempty`.

```Go
type T struct {
  A string
  B struct {
    RenamedC int   `ion:"C"`
    D        []int `ion:",omitempty"`
  }
}

func main() {
  t := T{}

  err := ion.Unmarshal([]byte(`{A:"Ion!",B:{C:2,D:[3,4]}}`), &t)
  if err != nil {
    panic(err)
  }
  fmt.Printf("--- t:\n%v\n\n", t)

  text, err := ion.MarshalText(&t)
  if err != nil {
    panic(err)
  }
  fmt.Printf("--- text:\n%s\n\n", string(text))

  binary, err := ion.MarshalBinary(&t)
  if err != nil {
    panic(err)
  }
  fmt.Printf("--- binary:\n%X\n\n", binary)
}
```

In order to Marshal/Unmarshal Ion values with annotation, we use a Go struct with two fields,

1. one field of type `[]string` and tagged  with `ion:",annotation"`.
2. the other field with appropriate type and optional tag to hold our Ion value. For instance,
to Marshal `age::20`, it must be in a struct as below:
```GO
  type foo struct {
    Value   interface{}
    AnyName []string `ion:",annotations"`
  }
  data := foo{20, []string{"age"}}
  val, err := ion.MarshalText(data)
  if err != nil {
     panic(err)
  }
  fmt.Println("Ion text: ", string(val)) // Ion text: age::20
```

And to Unmarshal the same data, we can do as shown below:
```Go
  type foo struct {
    Value   interface{}
    AnyName []string `ion:",annotations"`
  }
  var val foo
  err := ion.UnmarshalString("age::20", &val)
  if err != nil {
    panic(err)
  }
  fmt.Printf("Val = %+v\n", val) // Val = {Value:20 AnyName:[age]}
```


### Encoding and Decoding

To read or write multiple values at once, use an `Encoder` or `Decoder`:

```Go
func main() {
  dec := ion.NewTextDecoder(os.Stdin)
  enc := ion.NewBinaryEncoder(os.Stdout)

  for {
    // Decode one Ion whole value from stdin.
    val, err := dec.Decode()
    if err == ion.ErrNoInput {
      break
    } else if err != nil {
      panic(err)
    }

    // Encode it to stdout.
    if err := enc.Encode(val); err != nil {
      panic(err)
    }
  }

  if err := enc.Finish(); err != nil {
    panic(err)
  }
}
```

### Reading and Writing

For low-level streaming read and write access, use a `Reader` or `Writer`.
The following example shows how to create a reader, read values from that reader,
and write those values out using a writer:

```Go
func writeFromReaderToWriter(reader Reader, writer Writer) {
	for reader.Next() {
		name := reader.FieldName()
		if name != nil {
			err := writer.FieldName(*name)
			if err != nil {
				panic(err)
			}
		}

		an := reader.Annotations()
		if len(an) > 0 {
			err := writer.Annotations(an...)
			if err != nil {
				panic(err)
			}
		}

		currentType := reader.Type()
		if reader.IsNull() {
			err := writer.WriteNullType(currentType)
			if err != nil {
				panic(err)
			}
			continue
		}

		switch currentType {
		case BoolType:
			val, err := reader.BoolValue()
			if err != nil {
				panic("Something went wrong while reading a Boolean value: " + err.Error())
			}
			err = writer.WriteBool(val)
			if err != nil {
				panic("Something went wrong while writing a Boolean value: " + err.Error())
			}

		case StringType:
			val, err := reader.StringValue()
			if err != nil {
				panic("Something went wrong while reading a String value: " + err.Error())
			}
			err = writer.WriteString(val)
			if err != nil {
				panic("Something went wrong while writing a String value: " + err.Error())
			}

		case StructType:
			err := reader.StepIn()
			if err != nil {
				panic(err)
			}
			err = writer.BeginStruct()
			if err != nil {
				panic(err)
			}
			writeFromReaderToWriter(reader, writer)
			err = reader.StepOut()
			if err != nil {
				panic(err)
			}
			err = writer.EndStruct()
			if err != nil {
				panic(err)
			}
        default:
            panic("This is an example, only taking in Bool, String and Struct")
		}
	}

	if reader.Err() != nil {
		panic(reader.Err().Error())
	}
}

func main() {
	reader := NewReaderString("foo::{name:\"bar\", complete:false}")
	str := strings.Builder{}
	writer := NewTextWriter(&str)

	writeFromReaderToWriter(reader, writer)
	err := writer.Finish()
	if err != nil {
		panic(err)
	}
	fmt.Println(str.String())
}
```

### Symbol Tables

By default, when writing binary Ion, a local symbol table is built as you write
values (which are buffered in memory until you call `Finish` so the symbol table
can be written out first). You can optionally provide one or more
`SharedSymbolTable`s to the writer, which it will reference as needed rather
than directly including those symbols in the local symbol table.

```Go
type Item struct {
  ID          string    `ion:"id"`
  Name        string    `ion:"name"`
  Description string    `ion:"description"`
}

var ItemSharedSymbols = ion.NewSharedSymbolTable("item", 1, []string{
  "item",
  "id",
  "name",
  "description",
})

type SpicyItem struct {
  Item
  Spiciness   int       `ion:"spiciness"`
}

func WriteSpicyItemsTo(out io.Writer, items []SpicyItem) error {
  writer := ion.NewBinaryWriter(out, ItemSharedSymbols)

  for _, item := range items {
    writer.Annotation("item")
    if err := ion.EncodeTo(writer, item); err != nil {
      return err
    }
  }

  return writer.Finish()
}
```

You can alternatively provide the writer with a complete, pre-built local symbol table.
This allows values to be written without buffering, however any attempt to write a
symbol that is not included in the symbol table will result in an error:

```Go
func WriteItemsToLST(out io.Writer, items []SpicyItem) error {
  lst := ion.NewLocalSymbolTable([]SharedSymbolTable{ItemSharedSymbols}, []string{
    "spiciness",
  })

  writer := ion.NewBinaryWriterLST(out, lst)

  for _, item := range items {
    writer.Annotation("item")
    if err := ion.EncodeTo(writer, item); err != nil {
      return err
    }
  }

  return writer.Finish()
}
```

When reading binary Ion, shared symbol tables are provided by a `Catalog`. A basic
catalog can be constructed by calling `NewCatalog`; a smarter implementation may
load shared symbol tables from a database on demand.

```Go

func ReadItemsFrom(in io.Reader) ([]Item, error) {
  item := Item{}
  items := []Item{}

  cat := ion.NewCatalog(ItemSharedSymbols)
  dec := ion.NewDecoder(ion.NewReaderCat(in, cat))

  for {
    err := dec.DecodeTo(&item)
    if err == ion.ErrNoInput {
      return items, nil
    }
    if err != nil {
      return nil, err
    }

    items = append(items, item)
  }
}
```
### License

This library is licensed under the Apache 2.0 License.
