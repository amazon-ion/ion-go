### Amazon Ion Go

This package is for parsing and writing text and binary based Ion data.

http://amzn.github.io/ion-docs/docs/spec.html

It aims to be efficient by only evaluating values when they are accessed, and retaining the original
text or binary representation where possible for efficient re-serialization of the same form.

### Notes

* This package only supports text as UTF-8.  It does not support the UTF-16 or UTF-32 forms.
* Only a text and binary parsers, and a sketch of the types have been implemented so far.  
* The text parser is able to parse all of the text files (`.ion`) from the `iontestdata/good` directory of the 
  `Ion-tests` package.  It returns an error for the majority of text files from the `iontestdata/bad` directory.
* The text parser recognizes the majority of the lists expressed in the `iontestdata/good/equivs` directory as 
  having equivalent values.  It also recognizes the majority of the lists expressed in the 
  `iontestdata/good/non-equivs` directory as having non-equivalent values.
* The binary parser is able to parse all of the binary files (`.10n`) from the `iontestdata/good` directory of the 
  `Ion-tests` package.  It returns an error for the majority of binary files from the `iontestdata/bad` directory.
* The `Float` type is limited to the minimum and maximum values that Go is able to handle with float64.  These values
  are approximately `1.797693e+308` and `4.940656e-324`.  Values that are too large will round to infinity and
  values that are too small will round to zero.
* Textual representation of `Timestamp` currently stops at microsecond precision.

### TODO
* Symbol table construction and verification.
* Define the external interfaces for marshalling and unmarshalling.
* Serializing Values to the textual and binary forms.
* Unmarshal into a struct.
* Profiling.
* Make the `Decimal` type keep track of precision, so trailing zeros won't be lost when translating from
  the textual to binary form.
* Make the `Timestamp` type handle the case of unknown local offsets.
* Make the `Float` and `Decimal` types recognize negative zero.

### Usage

```go
package main

// TODO
```
