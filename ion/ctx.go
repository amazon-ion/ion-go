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

package ion

import "fmt"

// ctx is the current reader or writer context.
type ctx uint8

const (
	ctxAtTopLevel ctx = iota
	ctxInStruct
	ctxInList
	ctxInSexp
)

func ctxToContainerType(c ctx) Type {
	switch c {
	case ctxInList:
		return ListType
	case ctxInSexp:
		return SexpType
	case ctxInStruct:
		return StructType
	default:
		return NoType
	}
}

func containerTypeToCtx(t Type) ctx {
	switch t {
	case ListType:
		return ctxInList
	case SexpType:
		return ctxInSexp
	case StructType:
		return ctxInStruct
	default:
		panic(fmt.Sprintf("type %v is not a container type", t))
	}
}

// ctxstack is a context stack.
type ctxstack struct {
	arr []ctx
}

// peek returns the current context.
func (c *ctxstack) peek() ctx {
	if len(c.arr) == 0 {
		return ctxAtTopLevel
	}
	return c.arr[len(c.arr)-1]
}

// push pushes a new context onto the stack.
func (c *ctxstack) push(ctx ctx) {
	c.arr = append(c.arr, ctx)
}

// pop pops the top context off the stack.
func (c *ctxstack) pop() {
	if len(c.arr) == 0 {
		panic("pop called at top level")
	}
	c.arr = c.arr[:len(c.arr)-1]
}
