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


## Notes

* This package only supports text as UTF-8.  It does not support the UTF-16 or UTF-32 forms.
* Only a text and binary parsers, and a sketch of the types have been implemented so far.  
* The `Float` type is limited to the minimum and maximum values that Go is able to handle with float64.  These values
  are approximately `1.797693e+308` and `4.940656e-324`.  Values that are too large will round to infinity and
  values that are too small will round to zero.
* Textual representation of `Timestamp` currently stops at microsecond precision.

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

## Usage

```go
package main

// TODO
```
