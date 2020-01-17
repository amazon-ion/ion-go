/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
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
