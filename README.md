# Amazon Ion Go

[![Build Status](https://github.com/amzn/ion-go/workflows/Go%20Build/badge.svg)](https://github.com/amzn/ion-go/actions?query=workflow%3A%22Go+Build%22)

This package is for parsing and writing text and binary based Ion data.

***This package is considered experimental, under active early development,  and the API is subject to change.***

http://amzn.github.io/ion-docs/docs/spec.html

It aims to be efficient by only evaluating values when they are accessed, and retaining the original
text or binary representation where possible for efficient re-serialization of the same form.

## Git Setup


This repository contains a [git submodule](https://git-scm.com/docs/git-submodule)
called `ion-tests`, which holds test data used by `ion-go`'s unit tests.

The easiest way to clone the `ion-go` repository and initialize its `ion-tests`
submodule is to run the following command.

```
$ git clone --recursive https://github.com/amzn/ion-go.git ion-go
```

Alternatively, the submodule may be initialized independently from the clone
by running the following commands.

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
thing. Both respect json name tags, and `Marshal` honors `omitempty`.

```Go
type T struct {
  A string
  B struct {
    RenamedC int   `json:"C"`
    D        []int `json:",omitempty"`
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

```Go
func copy(in ion.Reader, out ion.Writer) {
  for in.Next() {
    name := in.FieldName()
    if name != "" {
      out.FieldName(name)
    }

    annos := in.Annotations()
    if len(annos) > 0 {
      out.Annotations(annos...)
    }

    switch in.Type() {
    case ion.BoolType:
      val, err := in.BoolValue()
      if err != nil {
        panic(err)
      }
      out.WriteBool(val)

    case ion.IntType:
      val, err := in.Int64Value()
      if err != nil {
        panic(err)
      }
      out.WriteInt(val)

    case ion.StringType:
      val, err := in.StringValue()
      if err != nil {
        panic(err)
      }
      out.WriteString(val)

    case ion.ListType:
      in.StepIn()
      out.BeginList()
      copy(in, out)
      in.StepOut()
      out.EndList()

    case ion.StructType:
      in.StepIn()
      out.BeginStruct()
      copy(in, out)
      in.StepOut()
      out.EndStruct()
    }
  }

  if in.Err() != nil {
    panic(in.Err())
  }
}

func main() {
  in := ion.NewReader(os.Stdin)
  out := ion.NewBinaryWriter(os.Stdout)

  copy(in, out)

  if err := out.Finish(); err != nil {
    panic(err)
  }
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
  ID          string    `json:"id"`
  Name        string    `json:"name"`
  Description string    `json:"description"`
}

var ItemSharedSymbols = ion.NewSharedSymbolTable("item", 1, []string{
  "item",
  "id",
  "name",
  "description",
})

type SpicyItem struct {
  Item
  Spiciness   int       `json:"spiciness"`
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
## Notes

* This package only supports text as UTF-8.  It does not support the UTF-16 or UTF-32 forms.
* The `Float` type is limited to the minimum and maximum values that Go is able to handle with float64.  These values
  are approximately `1.797693e+308` and `4.940656e-324`.  Values that are too large will round to infinity and
  values that are too small will round to zero.

## TODO

* Symbol table construction and verification.
* Define the external interfaces for marshalling and unmarshalling.
* Serializing Values to the textual and binary forms.
* Unmarshal into a struct.
* Profiling.
* Make the `Decimal` type keep track of precision, so trailing zeros won't be lost when translating from
  the textual to binary form.
* Make the `Timestamp` type handle the case of unknown local offsets.
* Make the `Float` and `Decimal` types recognize negative zero.


