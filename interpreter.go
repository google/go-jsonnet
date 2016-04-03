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
	"reflect"
	"sort"
)

// Misc top-level stuff

type vmExt struct {
	value  string
	isCode bool
}

type vmExtMap map[string]vmExt

// RuntimeError is an error discovered during evaluation of the program
type RuntimeError struct {
	StackTrace []TraceFrame
	Msg        string
}

func makeRuntimeError(loc *LocationRange, msg string) RuntimeError {
	// TODO(dcunnin): Build proper stacktrace.
	return RuntimeError{
		StackTrace: []TraceFrame{
			{
				Loc:  *loc,
				Name: "name",
			},
		},
		Msg: msg,
	}
}

func (err RuntimeError) Error() string {
	// TODO(dcunnin): Include stacktrace.
	return err.Msg
}

// Values and state

type bindingFrame map[identifier]*thunk

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
	content  value // nil if not filled
	name     identifier
	upValues bindingFrame
	self     value
	offset   int
	body     astNode
}

func makeThunk(name identifier, self value, offset int, body astNode) *thunk {
	return &thunk{
		name:   name,
		self:   self,
		offset: offset,
		body:   body,
	}
}

func (t *thunk) fill(v value) {
	t.content = v
	t.self = nil
	t.upValues = make(bindingFrame) // clear the map
}

func (t *thunk) filled() bool {
	return t.content != nil
}

type valueArray struct {
	elements []*thunk
}

func makeValueArray(elements []*thunk) *valueArray {
	return &valueArray{
		elements: elements,
	}
}

type valueClosure struct {
	upValues bindingFrame
}

type valueSimpleObjectField struct {
	hide astObjectFieldHide
	body astNode
}

type valueSimpleObjectFieldMap map[string]valueSimpleObjectField

type valueSimpleObject struct {
	upValues bindingFrame
	fields   valueSimpleObjectFieldMap
	asserts  []astNode
}

// TODO(dcunnin): extendedObject
// TODO(dcunnin): comprehensionObject
// TODO(dcunnin): closure

// The stack

// TraceFrame is a single frame of the call stack.
type TraceFrame struct {
	Loc  LocationRange
	Name string
}

type callFrame struct {
	isCall   bool
	ast      astNode
	location LocationRange
	tailCall bool
	thunks   []*thunk
	context  value
	self     value
	offset   int
	bindings bindingFrame
}

type callStack struct {
	calls int
	limit int
	stack []*callFrame
}

func (s *callStack) top() *callFrame {
	return s.stack[len(s.stack)-1]

}

func (s *callStack) pop() {
	if s.top().isCall {
		s.calls--
	}
	s.stack = s.stack[:len(s.stack)-1]

}

/** If there is a tailstrict annotated frame followed by some locals, pop them all. */
func (s *callStack) tailCallTrimStack() {
	for i := len(s.stack) - 1; i >= 0; i-- {
		if s.stack[i].isCall {
			// If thunks > 0 that means we are still executing args (tailstrict).
			if !s.stack[i].tailCall || len(s.stack[i].thunks) > 0 {
				return
			}
			// Remove all stack frames including this one.
			s.stack = s.stack[:i]
			s.calls--
			return
		}
	}
}

func (s *callStack) newCall(loc *LocationRange, context value, self value, offset int, upValues bindingFrame) error {
	s.tailCallTrimStack()
	if s.calls >= s.limit {
		return makeRuntimeError(loc, "Max stack frames exceeded.")
	}
	s.stack = append(s.stack, &callFrame{
		isCall:   true,
		location: *loc,
		context:  context,
		self:     self,
		offset:   offset,
		bindings: upValues,
		tailCall: false,
	})
	s.calls++
	return nil
}

func (s *callStack) newLocal(vars bindingFrame) {
	s.stack = append(s.stack, &callFrame{
		bindings: vars,
	})
}

// getSelfBinding resolves the self construct
func (s *callStack) getSelfBinding() (value, int) {
	for i := len(s.stack) - 1; i >= 0; i-- {
		if s.stack[i].isCall {
			return s.stack[i].self, s.stack[i].offset
		}
	}
	// Should never get here if the stack is well-formed.
	return nil, 0
}

// lookUpVar finds for the closest variable in scope that matches the given name.
func (s *callStack) lookUpVar(id identifier) *thunk {
	for i := len(s.stack) - 1; i >= 0; i-- {
		bind := s.stack[i].bindings[id]
		if bind != nil {
			return bind
		}
		if s.stack[i].isCall {
			// Nothing beyond the captured environment of the thunk / closure.
			break
		}
	}
	return nil
}

func makeCallStack(limit int) callStack {
	return callStack{
		calls: 0,
		limit: limit,
	}
}

// TODO(dcunnin): Add import callbacks.
// TODO(dcunnin): Add string output.
// TODO(dcunnin): Add multi output.

type interpreter struct {
	stack          callStack
	idArrayElement identifier
	idInvariant    identifier
	externalVars   vmExtMap
}

func (i *interpreter) capture(freeVars identifiers) bindingFrame {
	var env bindingFrame
	for _, fv := range freeVars {
		env[fv] = i.stack.lookUpVar(fv)
	}
	return env
}

type fieldHideMap map[string]astObjectFieldHide

func (i *interpreter) objectFieldsAux(obj value) fieldHideMap {
	r := make(fieldHideMap)
	switch obj := obj.(type) {
	// case *valueExtendedObject:
	// TODO(dcunnin): this

	case *valueSimpleObject:
		for fieldName, field := range obj.fields {
			r[fieldName] = field.hide
		}

		// case *valueComprehensionObject:
		// TODO(dcunnin): this
	}
	return r
}

func (i *interpreter) objectFields(obj value, manifesting bool) []string {
	var r []string
	for fieldName, hide := range i.objectFieldsAux(obj) {
		if !manifesting || hide != astObjectFieldHidden {
			r = append(r, fieldName)
		}
	}
	return r
}

func (i *interpreter) findObject(f string, curr value, startFrom int, counter *int) value {
	switch curr := curr.(type) {
	// case *valueExtendedObject:
	// TODO(dcunnin): this

	case *valueSimpleObject:
		if *counter >= startFrom {
			if _, ok := curr.fields[f]; ok {
				return curr
			}
		}
		*counter++

		// case *valueComprehensionObject:
		/*
			if *counter >= startFrom {
				// TODO(dcunnin): this
			}
			*counter++
		*/
	}
	return nil
}

func (i *interpreter) objectIndex(loc *LocationRange, obj value, f string, offset int) (astNode, error) {
	var foundAt int
	self := obj
	found := i.findObject(f, obj, offset, &foundAt)
	if found == nil {
		return nil, makeRuntimeError(loc, fmt.Sprintf("Field does not exist: %s", f))
	}
	switch found := found.(type) {
	case *valueSimpleObject:
		field := found.fields[f]
		i.stack.newCall(loc, found, self, foundAt, found.upValues)
		return field.body, nil
	// case *valueComprehensionObject:
	/*
		// TODO(dcunnin): this
	*/
	default:
		return nil, fmt.Errorf("Internal error: findObject returned unrecognized type: %s", reflect.TypeOf(found))
	}
}

func (i *interpreter) evaluate(a astNode) (value, error) {
	// TODO(dcunnin): All the other cases...
	switch ast := a.(type) {
	case *astArray:
		self, offset := i.stack.getSelfBinding()
		var elements []*thunk
		for _, el := range ast.elements {
			elThunk := makeThunk(i.idArrayElement, self, offset, el)
			elThunk.upValues = i.capture(el.FreeVariables())
			elements = append(elements, elThunk)
		}
		return &valueArray{elements}, nil

	case *astBinary:
		// TODO(dcunnin): Assume it's + on numbers for now
		leftVal, err := i.evaluate(ast.left)
		if err != nil {
			return nil, err
		}
		leftNum := leftVal.(*valueNumber).value
		rightVal, err := i.evaluate(ast.right)
		if err != nil {
			return nil, err
		}
		rightNum := rightVal.(*valueNumber).value
		return makeValueNumber(leftNum + rightNum), nil

	case *astDesugaredObject:
		// Evaluate all the field names.  Check for null, dups, etc.
		fields := make(valueSimpleObjectFieldMap)
		for _, field := range ast.fields {
			fieldNameValue, err := i.evaluate(field.name)
			if err != nil {
				return nil, err
			}
			var fieldName string
			switch fieldNameValue := fieldNameValue.(type) {
			case *valueString:
				fieldName = fieldNameValue.value
			case *valueNull:
				// Omitted field.
				continue
			default:
				return nil, makeRuntimeError(ast.Loc(), "Field name was not a string.")
			}

			if err != nil {
				return nil, err
			}
			if _, ok := fields[fieldName]; ok {
				return nil, makeRuntimeError(ast.Loc(), fmt.Sprintf("Duplicate field name: \"%s\"", fieldName))
			}
			fields[fieldName] = valueSimpleObjectField{field.hide, field.body}
		}
		upValues := i.capture(ast.FreeVariables())
		return &valueSimpleObject{upValues, fields, ast.asserts}, nil

	case *astLiteralBoolean:
		return makeValueBoolean(ast.value), nil

	case *astLiteralNull:
		return makeValueNull(), nil

	case *astLiteralNumber:
		return makeValueNumber(ast.value), nil

	case *astLiteralString:
		return makeValueString(ast.value), nil

	case *astLocal:
		vars := make(bindingFrame)
		self, offset := i.stack.getSelfBinding()
		for _, bind := range ast.binds {
			th := makeThunk(bind.variable, self, offset, bind.body)
			vars[bind.variable] = th
		}
		for _, bind := range ast.binds {
			th := vars[bind.variable]
			th.upValues = i.capture(bind.body.FreeVariables())
		}
		i.stack.newLocal(vars)
		// Add new stack frame, with new thunk for this variable
		// execute body WRT stack frame.
		return i.evaluate(ast.body)

	default:
		return nil, makeRuntimeError(ast.Loc(), fmt.Sprintf("Executing this AST type not implemented yet: %v", reflect.TypeOf(a)))
	}
}

// unparseString Wraps in "" and escapes stuff to make the string JSON-compliant and human-readable.
func unparseString(v string) string {
	var buf bytes.Buffer
	buf.WriteString("\"")
	for _, c := range v {
		switch c {
		case '"':
			buf.WriteString("\n")
		case '\\':
			buf.WriteString("\\\\")
		case '\b':
			buf.WriteString("\\b")
		case '\f':
			buf.WriteString("\\f")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		case 0:
			buf.WriteString("\\u0000")
		default:
			if c < 0x20 || (c >= 0x7f && c <= 0x9f) {
				buf.WriteString(fmt.Sprintf("\\u%04x", int(c)))
			} else {
				buf.WriteRune(c)
			}
		}
	}
	buf.WriteString("\"")
	return buf.String()
}

func unparseNumber(v float64) string {
	if v == math.Floor(v) {
		return fmt.Sprintf("%.0f", v)
	}

	// See "What Every Computer Scientist Should Know About Floating-Point Arithmetic"
	// Theorem 15
	// http://docs.oracle.com/cd/E19957-01/806-3568/ncg_goldberg.html
	return fmt.Sprintf("%.17g", v)
}

func (i *interpreter) manifestJSON(loc *LocationRange, v value, multiline bool, indent string, buf *bytes.Buffer) error {
	// TODO(dcunnin): All the other types...
	switch v := v.(type) {
	case *valueArray:
		if len(v.elements) == 0 {
			buf.WriteString("[ ]")
		} else {
			var prefix string
			var indent2 string
			if multiline {
				prefix = "[\n"
				indent2 = indent + "   "
			} else {
				prefix = "["
				indent2 = indent
			}
			for _, th := range v.elements {
				tloc := loc
				if th.body != nil {
					tloc = th.body.Loc()
				}
				var elVal value
				if th.filled() {
					i.stack.newCall(loc, th, nil, 0, make(bindingFrame))
					elVal = th.content
				} else {
					i.stack.newCall(loc, th, th.self, th.offset, th.upValues)
					var err error
					elVal, err = i.evaluate(th.body)
					if err != nil {
						return err
					}
				}
				buf.WriteString(prefix)
				buf.WriteString(indent2)
				err := i.manifestJSON(tloc, elVal, multiline, indent2, buf)
				if err != nil {
					return err
				}
				i.stack.pop()
				if multiline {
					prefix = ",\n"
				} else {
					prefix = ", "
				}
			}
			if multiline {
				buf.WriteString("\n")
			}
			buf.WriteString(indent)
			buf.WriteString("]")
		}

	case *valueBoolean:
		if v.value {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}

	case *valueClosure:
		return makeRuntimeError(loc, "Couldn't manifest function in JSON output.")

	case *valueNumber:
		buf.WriteString(unparseNumber(v.value))

	case *valueNull:
		buf.WriteString("null")

	// TODO(dcunnin): Other types representing objects will be handled by the same code here.
	case *valueSimpleObject:
		// TODO(dcunnin): Run invariants (object-level assertions).

		fieldNames := i.objectFields(v, true)
		sort.Strings(fieldNames)

		if len(fieldNames) == 0 {
			buf.WriteString("{ }")
		} else {
			var prefix string
			var indent2 string
			if multiline {
				prefix = "{\n"
				indent2 = indent + "   "
			} else {
				prefix = "{"
				indent2 = indent
			}
			for _, fieldName := range fieldNames {

				body, err := i.objectIndex(loc, v, fieldName, 0)
				if err != nil {
					return err
				}

				fieldVal, err := i.evaluate(body)
				if err != nil {
					return err
				}

				buf.WriteString(prefix)
				buf.WriteString(indent2)

				buf.WriteString("\"")
				buf.WriteString(fieldName)
				buf.WriteString("\"")
				buf.WriteString(": ")

				err = i.manifestJSON(body.Loc(), fieldVal, multiline, indent2, buf)
				if err != nil {
					return err
				}

				if multiline {
					prefix = ",\n"
				} else {
					prefix = ", "
				}
			}

			if multiline {
				buf.WriteString("\n")
			}
			buf.WriteString(indent)
			buf.WriteString("}")
		}

	case *valueString:
		buf.WriteString(unparseString(v.value))

	default:
		return makeRuntimeError(loc, fmt.Sprintf("Manifesting this value not implemented yet: %s", reflect.TypeOf(v)))

	}
	return nil
}

func evaluate(ast astNode, ext vmExtMap, maxStack int) (string, error) {
	i := interpreter{
		stack:          makeCallStack(maxStack),
		idArrayElement: identifier("array_element"),
		idInvariant:    identifier("object_assert"),
		externalVars:   ext,
	}
	result, err := i.evaluate(ast)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	loc := makeLocationRangeMessage("During manifestation")
	err = i.manifestJSON(&loc, result, true, "", &buffer)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
