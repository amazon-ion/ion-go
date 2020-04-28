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

package ion

import "bufio"

// A readerBinary reads binary Ion.
type readerBinary struct {
	catalog Catalog
	event eventBinary
}

func newBinaryReader(in *bufio.Reader, catalog Catalog) Reader {
	r := &readerBinary{
		catalog: catalog,
	}

	return r
}

func (r *readerBinary) Next() Event {
	return r.event
}

func (r *readerBinary) StepIn() error {
	return nil
}

func (r *readerBinary) StepOut() error {
	return nil
}
