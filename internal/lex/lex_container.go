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

package lex

// This file contains the state functions for container types:
// List, Struct, and S-Expression.

// lexList emits the start of a list.
func lexList(x *Lexer) stateFn {
	x.emit(IonListStart)
	x.containers = append(x.containers, '[')
	return lexValue
}

// lexListEnd emits the end of a list.
func lexListEnd(x *Lexer) stateFn {
	return containerEnd(x, IonListEnd)
}

// lexSExp emits the start of an s-expression.
func lexSExp(x *Lexer) stateFn {
	x.emit(IonSExpStart)
	x.containers = append(x.containers, '(')
	return lexValue
}

// lexSExpEnd emits the end of an s-expression.
func lexSExpEnd(x *Lexer) stateFn {
	return containerEnd(x, IonSExpEnd)
}

// lexStruct emits the start of a structure.
func lexStruct(x *Lexer) stateFn {
	x.emit(IonStructStart)
	x.containers = append(x.containers, '{')
	return lexValue
}

// lexStructEnd ensures that ending the struct corresponds to a struct start and
// returns lexValue since we don't know what will come next.  Inappropriate ending
// of the struct will be handled by the parser.
func lexStructEnd(x *Lexer) stateFn {
	return containerEnd(x, IonStructEnd)
}

// containerEnd makes sure that the container being ended matches the last one
// opened.  It emits the given itemType if everything is fine.
func containerEnd(x *Lexer, it itemType) stateFn {
	if len(x.containers) == 0 {
		return x.error("unexpected closing of container")
	}

	switch ch := x.containers[len(x.containers)-1]; {
	case ch == '(' && it != IonSExpEnd:
		return x.errorf("expected closing of s-expression but found %s", it)
	case ch == '{' && it != IonStructEnd:
		return x.errorf("expected closing of struct but found %s", it)
	case ch == '[' && it != IonListEnd:
		return x.errorf("expected closing of list but found %s", it)
	}

	x.containers = x.containers[:len(x.containers)-1]
	x.emit(it)

	return lexValue
}
