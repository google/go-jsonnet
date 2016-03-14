/*
Copyright 2016 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonnet

import (
	"bytes"
	"fmt"
	"math"
)

// Misc top-level stuff

type vmExt struct {
	value  string
	isCode bool
}

type vmExtMap map[string]vmExt

type RuntimeError struct {
	StackTrace []traceFrame
	Msg        string
}

func makeRuntimeError(msg string) RuntimeError {
	return RuntimeError{
		Msg: msg,
	}
}

func (err RuntimeError) Error() string {
	// TODO(dcunnin): Include stacktrace.
	return err.Msg
}

// Values and state

type bindingFrame map[*identifier]thunk

type value interface {
}

type valueString struct {
	value string
}

func makeValueString(v string) *valueString {
	return &valueString{value: v}
}

type valueBoolean struct {
	value bool
}

func makeValueBoolean(v bool) *valueBoolean {
	return &valueBoolean{value: v}
}

type valueNumber struct {
	value float64
}

func makeValueNumber(v float64) *valueNumber {
	return &valueNumber{value: v}
}

// TODO(dcunnin): Maybe intern values null, true, and false?
type valueNull struct {
}

func makeValueNull() *valueNull {
	return &valueNull{}
}

type thunk struct {
	Content  value // nil if not filled
	Name     *identifier
	UpValues bindingFrame
	Self     value
	Offset   int
	Body     astNode
}

func makeThunk(name *identifier, self *value, offset int, body astNode) *thunk {
	return &thunk{
		Name:   name,
		Self:   self,
		Offset: offset,
		Body:   body,
	}
}

func (t *thunk) fill(v value) {
	t.Content = v
	t.Self = nil
	t.UpValues = make(bindingFrame) // clear the map
}

type valueArray struct {
	Elements []thunk
}

func makeValueArray(elements []thunk) *valueArray {
	return &valueArray{
		Elements: elements,
	}
}

// TODO(dcunnin): SimpleObject
// TODO(dcunnin): ExtendedObject
// TODO(dcunnin): ComprehensionObject
// TODO(dcunnin): Closure

// The stack

type traceFrame struct {
	Loc  LocationRange
	Name string
}

type callFrame struct {
	bindings bindingFrame
}

type callStack struct {
	Calls int
	Limit int
	Stack []callFrame
}

func makeCallStack(limit int) callStack {
	return callStack{
		Calls: 0,
		Limit: limit,
	}
}

// TODO(dcunnin): Add import callbacks.
// TODO(dcunnin): Add string output.
// TODO(dcunnin): Add multi output.

type interpreter struct {
	Stack        callStack
	ExternalVars vmExtMap
}

func (this interpreter) execute(ast_ astNode) (value, error) {
	// TODO(dcunnin): All the other cases...
	switch ast := ast_.(type) {
	case *astBinary:
		// TODO(dcunnin): Assume it's + on numbers for now
		leftVal, err := this.execute(ast.left)
		if err != nil {
			return nil, err
		}
		leftNum := leftVal.(*valueNumber).value
		rightVal, err := this.execute(ast.right)
		if err != nil {
			return nil, err
		}
		rightNum := rightVal.(*valueNumber).value
		return makeValueNumber(leftNum + rightNum), nil
	case *astLiteralNull:
		return makeValueNull(), nil
	case *astLiteralBoolean:
		return makeValueBoolean(ast.value), nil
	case *astLiteralNumber:
		return makeValueNumber(ast.value), nil
	default:
		return nil, makeRuntimeError("Executing this AST type not implemented yet.")
	}
}

func unparseNumber(v float64) string {
	if v == math.Floor(v) {
		return fmt.Sprintf("%.0f", v)
	} else {
		// See "What Every Computer Scientist Should Know About Floating-Point Arithmetic"
		// Theorem 15
		// http://docs.oracle.com/cd/E19957-01/806-3568/ncg_goldberg.html
		return fmt.Sprintf("%.17g", v)
	}
}

func (this interpreter) manifestJson(
	v_ value, multiline bool, indent string, buf *bytes.Buffer) error {
	// TODO(dcunnin): All the other types...
	switch v := v_.(type) {
	case *valueBoolean:
		if v.value {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case *valueNull:
		buf.WriteString("null")
	case *valueNumber:
		buf.WriteString(unparseNumber(v.value))
	default:
		return makeRuntimeError("Manifesting this value not implemented yet.")
	}
	return nil
}

func execute(ast astNode, ext vmExtMap, maxStack int) (string, error) {
	theInterpreter := interpreter{
		Stack:        makeCallStack(maxStack),
		ExternalVars: ext,
	}
	result, err := theInterpreter.execute(ast)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	err = theInterpreter.manifestJson(result, true, "", &buffer)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
