# Ion Go
A Golang implementation of Amazon's [Ion data notation](https://amzn.github.io/ion-docs/).

| ❗ | You should probably use [amzn/ion-go](https://github.com/amzn/ion-go) now instead of this 😊 |
|---|----|

## Using the Library
Import `github.com/fernomac/ion-go` and you're off to the races.

### Marshaling and Unmarshaling
Similar to Golang's built-in [json](https://golang.org/pkg/encoding/json/) package,
you can marshal and unmarshal go types to Ion. Marshaling requires you to specify
whether you'd like text or binary Ion. Unmarshaling is smart enough to do the right
thing. Both respect json name tags, and `Marshal` honors omitempty.
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

  err := ion.Unmarshal([]byte("{A:\"Ion!\",B:{C:2,D:[3,4]}}"), &t)
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
