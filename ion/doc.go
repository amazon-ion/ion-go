/* Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved. */

// ion is a data format that is comprised of three parts:
// * A set of data types
// * A textual notation for values of those types
// * A binary notation for values of those types
//
// There are many considerations that go into an Ion implementation
// that expand past those basic representations.  This includes but
// is not limited to a customizable Symbol Catalog to aid in efficient
// binary decoding and a System Symbol Catalog for symbols defined in
// the specification.
//
// More information can be found from these links
// * http://amzn.github.io/ion-docs/docs/spec.html
package ion
