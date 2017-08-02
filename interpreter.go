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

// TODO what is it?
// External variable (or code) provided before execution
// TODO how do they work?
type vmExt struct {
	value  string // what is it?
	isCode bool   // what is it?
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

type value interface {
	aValue()
}

type valueBase struct{}

func (v *valueBase) aValue() {}

// Something that may get evaluated to a value
// It may or may not result in computation.
// Getting the value a second time may or may not result in additional evaluation.
//
// TODO(sbarzowski) Maybe it should always be cached, also for objects etc.?
type potentialValue interface {
	getValue(i *interpreter, loc *LocationRange) (value, error)
}

type bindingFrame map[identifier]potentialValue

type valueString struct {
	valueBase
	value string
}

func makeValueString(v string) *valueString {
	return &valueString{value: v}
}

type valueBoolean struct {
	valueBase
	value bool
}

func makeValueBoolean(v bool) *valueBoolean {
	return &valueBoolean{value: v}
}

type valueNumber struct {
	valueBase
	value float64
}

func makeValueNumber(v float64) *valueNumber {
	return &valueNumber{value: v}
}

// TODO(dcunnin): Maybe intern values null, true, and false?
type valueNull struct {
	valueBase
}

func makeValueNull() *valueNull {
	return &valueNull{}
}

type valueArray struct {
	valueBase
	elements []*thunk
}

func makeValueArray(elements []*thunk) *valueArray {
	return &valueArray{
		elements: elements,
	}
}

type valueClosure struct {
	valueBase
	upValues bindingFrame
	self     value
	offset   int
	function *astFunction
}

func (v *valueClosure) aValue() {}

func makeValueClosure(upValues bindingFrame, function *astFunction) *valueClosure {
	return &valueClosure{
		upValues: upValues,
		function: function,
	}
}

type valueSimpleObjectField struct {
	hide astObjectFieldHide
	body astNode
}

type valueSimpleObjectFieldMap map[string]valueSimpleObjectField

type valueSimpleObject struct {
	valueBase
	upValues bindingFrame
	fields   valueSimpleObjectFieldMap
	asserts  []astNode
}

// findObject returns an object in which there is a given field.
// It is used for field lookups, potentially involving super.
func findObject(f string, curr value, startFrom int, counter *int) value {
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

func objectIndex(loc *LocationRange, obj value, f string, offset int) (potentialValue, error) {
	var foundAt int
	found := findObject(f, obj, offset, &foundAt)
	if found == nil {
		return nil, makeRuntimeError(loc, fmt.Sprintf("Field does not exist: %s", f))
	}
	switch found := found.(type) {
	case *valueSimpleObject:
		self := found
		offset := foundAt
		upValues := bindingFrame{}
		field := found.fields[f]

		return makeThunk("???", makeEnvironment(upValues, self, offset), field.body), nil
	// case *valueComprehensionObject:
	/*
		// TODO(dcunnin): this
	*/
	default:
		return nil, fmt.Errorf("Internal error: findObject returned unrecognized type: %s", reflect.TypeOf(found))
	}
}

func makeValueSimpleObject(b bindingFrame, fields valueSimpleObjectFieldMap, asserts astNodes) *valueSimpleObject {
	return &valueSimpleObject{
		upValues: b,
		fields:   fields,
		asserts:  asserts,
	}
}

func (v *valueSimpleObject) aValue() {}

// TODO(sbarzowski) what is extendedObject supposed to be?
// TODO(dcunnin): extendedObject
// TODO(dcunnin): comprehensionObject
// TODO(dcunnin): closure

// TODO(sbarzowski) use it as a pointer in most places b/c it can sometimes be shared
// for example it can be shared between array elements and function arguments
type environment struct {
	// The lexically nearest object we are in, or nil.  Note
	// that this is not the same as context, because we could be inside a function,
	// inside an object and then context would be the function, but self would still point
	// to the object.
	self value

	// The "super" level of self.  Sometimes, we look upwards in the
	// inheritance tree, e.g. via an explicit use of super, or because a given field
	// has been inherited.  When evaluating a field from one of these super objects,
	// we need to bind self to the concrete object (so self must point
	// there) but uses of super should be resolved relative to the object whose
	// field we are evaluating.  Thus, we keep a second field for that.  This is
	// usually 0, unless we are evaluating a super object's field.
	// TODO(sbarzowski) provide some examples
	// TODO(sbarzowski) provide somewhere a complete explanation of the object model
	// How deep in super we are. If we're executing
	// TODO(sbarzowski) I was confused when I saw this name? Is it a standard term?
	// If not maybe something along the lines of superDepth would be more obvious.
	offset int

	// Bindings introduced in this frame. The way previous bindings are treated
	// depends on the type of a frame.
	// If isCall == true then previous bindings are ignored (it's a clean
	// environment with just the variables we have here).
	// If isCall == false then if this frame doesn't contain a binding
	// previous bindings will be used.
	upValues bindingFrame
}

func makeEnvironment(upValues bindingFrame, self value, offset int) environment {
	return environment{
		upValues: upValues,
		self:     self,
		offset:   offset,
	}
}

type thunk struct {
	content value // nil if not filled
	name    identifier
	env     environment
	body    astNode
}

func makeThunk(name identifier, env environment, body astNode) *thunk {
	return &thunk{
		name: name,
		env:  env,
		body: body,
	}
}

func (t *thunk) fill(v value) {
	t.content = v
	// no point in keeping the environment - we already have the value
	t.env = environment{}
}

func (t *thunk) filled() bool {
	return t.content != nil
}

func (t *thunk) getValue(i *interpreter, loc *LocationRange) (value, error) {
	if t.filled() {
		// TODO(sbarzowski) what for? Stack trace?
		// We're getting out of here anyway this seems useless
		i.stack.newCall(loc, t, t.env)
		return t.content, nil
	}
	return i.EvalInCleanEnv(loc, &t.env, t.body)
}

// The stack

// TraceFrame is tracing information about a single frame of the call stack.
type TraceFrame struct {
	Loc  LocationRange
	Name string
}

type callFrame struct {
	// True if it switches to a clean environment (function call or thunk)
	// False otherwise, e.g. for local
	// This makes callFrame a misnomer as it is technically not always a call...
	isCall bool

	// The code we were executing before.
	// TODO(sbarzowski) what is it? Before? I guess it's the place from where it's
	// called etc.
	// TODO(sbarzowski) how could it be nil?
	ast astNode

	// The location of the code we were executing before.
	// location == ast->location when ast != nil
	// TODO(sbarzowski) what if ast == nil?
	location LocationRange // TODO what is it?

	/** Reuse this stack frame for the purpose of tail call optimization. */
	tailCall bool // TODO what is it?

	/** Used for a variety of purposes. - > ???*/
	// TODO what is it? It's only used in dead code...
	thunks []*thunk

	/** The context is used in error messages to attempt to find a reasonable name for the
	 * object, function, or thunk value being executed.  If it is a thunk, it is filled
	 * with the value when the frame terminates.
	 */
	// TODO(sbarzowski) Why ast/location is not enough for that?
	context interface{}

	env environment
}

func dumpCallFrame(c *callFrame) string {
	return fmt.Sprintf("<callFrame isCall = %t location = %v tailCall = %t>",
		c.isCall,
		c.location,
		c.tailCall,
	)
}

type callStack struct {
	calls int
	limit int
	stack []*callFrame
}

func dumpCallStack(c *callStack) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "<callStack calls = %d limit = %d stack:\n", c.calls, c.limit)
	for _, callFrame := range c.stack {
		fmt.Fprintf(&buf, "  %v\n", dumpCallFrame(callFrame))
	}
	buf.WriteString("\n>")
	return buf.String()
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

func (s *callStack) newCall(loc *LocationRange, context interface{}, env environment) error {
	s.tailCallTrimStack()
	if s.calls >= s.limit {
		return makeRuntimeError(loc, "Max stack frames exceeded.")
	}
	s.stack = append(s.stack, &callFrame{
		isCall:   true,
		location: *loc,
		context:  context,
		env:      env,
		tailCall: false,
	})
	s.calls++
	return nil
}

func (s *callStack) newLocal(vars bindingFrame) {
	s.stack = append(s.stack, &callFrame{
		env: makeEnvironment(vars, nil, 0),
	})
}

// getSelfBinding resolves the self construct
func (s *callStack) getSelfBinding() (value, int) {
	for i := len(s.stack) - 1; i >= 0; i-- {
		if s.stack[i].isCall {
			return s.stack[i].env.self, s.stack[i].env.offset
		}
	}
	panic(fmt.Sprintf("INTERNAL ERROR: malformed stack %v", dumpCallStack(s)))
}

// lookUpVar finds for the closest variable in scope that matches the given name.
func (s *callStack) lookUpVar(id identifier) potentialValue {
	for i := len(s.stack) - 1; i >= 0; i-- {
		bind := s.stack[i].env.upValues[id]
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

// Keeps current execution context and evaluates things
type interpreter struct {
	stack          callStack  // TODO what is it?
	idArrayElement identifier // TODO what is it?
	idInvariant    identifier // TODO what is it?
	externalVars   vmExtMap   // TODO what is it?
}

// Build a binding frame (closure environment) containing specified variables
func (i *interpreter) capture(freeVars identifiers) bindingFrame {
	env := make(bindingFrame)
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

func addBindings(a, b bindingFrame) bindingFrame {
	result := make(bindingFrame)

	for k, v := range a {
		result[k] = v
	}

	for k, v := range b {
		result[k] = v
	}

	return result
}

// TODO(sbarzowski) what happens to the stack?
func (i *interpreter) evaluate(a astNode) (value, error) {
	// TODO(dcunnin): All the other cases...
	switch ast := a.(type) {
	case *astArray:
		self, offset := i.stack.getSelfBinding()
		var elements []*thunk
		for _, el := range ast.elements {
			env := makeEnvironment(i.capture(el.FreeVariables()), self, offset)
			elThunk := makeThunk(i.idArrayElement, env, el)
			elements = append(elements, elThunk)
		}
		return makeValueArray(elements), nil

	case *astBinary:
		leftVal, err := i.evaluate(ast.left)
		if err != nil {
			return nil, err
		}
		rightVal, err := i.evaluate(ast.right)
		if err != nil {
			return nil, err
		}

		builtin := bopBuiltins[ast.op]

		result, err := builtin.Binary(leftVal, rightVal, i, a.Loc())
		if err != nil {
			return nil, err
		}
		return result, nil

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
		return makeValueSimpleObject(upValues, fields, ast.asserts), nil

	case *astError:
		msgVal, err := i.evaluate(ast.expr)
		if err != nil {
			// error when evaluating error message
			// TODO(sbarzowski) maybe add it to stack trace, to avoid confusion
			return nil, err
		}
		// TODO(sbarzowski) handle more gracefully
		msg := msgVal.(*valueString)
		return nil, makeRuntimeError(&ast.loc, fmt.Sprintf("Error: %v", msg.value))

	case *astIndex:
		targetValue, err := i.evaluate(ast.target)
		if err != nil {
			return nil, err
		}
		index, err := i.evaluate(ast.index)
		if err != nil {
			return nil, err
		}
		switch target := targetValue.(type) {
		case *valueSimpleObject:
			indexString := index.(*valueString).value
			v, err := objectIndex(&ast.loc, target, indexString, 0) // why offset = 0?
			if err != nil {
				return nil, err
			}
			return v.getValue(i, &ast.loc)
		case *valueArray:
			indexInt := int(index.(*valueNumber).value)
			return target.elements[indexInt].getValue(i, &ast.loc)
		}

		return nil, makeRuntimeError(ast.Loc(), fmt.Sprintf("Value non indexable: %v", reflect.TypeOf(targetValue)))

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
			upValues := i.capture(bind.body.FreeVariables())
			env := makeEnvironment(upValues, self, offset)
			th := makeThunk(bind.variable, env, bind.body)
			vars[bind.variable] = th
		}
		i.stack.newLocal(vars)
		// Add new stack frame, with new thunk for this variable
		// execute body WRT stack frame.
		return i.evaluate(ast.body)

	case *astVar:
		th := i.stack.lookUpVar(ast.id)
		if th == nil {
			//fmt.Println(dumpCallStack(&i.stack))
			// This, should be caught during static check, right?
			return nil, makeRuntimeError(ast.Loc(), fmt.Sprintf("Unknown variable: %v", ast.id))
		}
		return th.getValue(i, &ast.loc)

	case *astFunction:
		bf := i.capture(ast.FreeVariables())
		return makeValueClosure(bf, ast), nil

	case *astApply:
		// Eval target
		target, err := i.evaluate(ast.target)
		if err != nil {
			return nil, err
		}
		closure := target.(*valueClosure) // TODO(sbarzowski) check gracefully

		// Prepare argument thunks

		self, offset := i.stack.getSelfBinding()

		// environment in which we can evaluate arguments
		argEnv := makeEnvironment(
			i.capture(ast.FreeVariables()),
			self,
			offset,
		)

		argThunks := make(bindingFrame)
		for index, arg := range ast.arguments {
			paramName := closure.function.parameters[index]
			argThunks[paramName] = makeThunk("???", argEnv, arg)
		}

		// Variables visible inside the called function
		// There will be:
		// * Variables captured by the closure
		// *
		calledEnvironment := makeEnvironment(
			addBindings(closure.upValues, argThunks),
			closure.self,
			closure.offset,
		)

		return i.EvalInCleanEnv(&ast.loc, &calledEnvironment, closure.function.body)
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

// TODO(sbarzowski) What is this loc? It's apparently used for error reporting, but what's the idea?
// It looks like it tries to get the location of the "origin" of the value.
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
				elVal, err := th.getValue(i, tloc) // TODO(sbarzowski) perhaps manifestJSON should just take potentialValue
				if err != nil {
					return err
				}
				buf.WriteString(prefix)
				buf.WriteString(indent2)
				err = i.manifestJSON(tloc, elVal, multiline, indent2, buf)
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

				fieldPotentialVal, err := objectIndex(loc, v, fieldName, 0)
				if err != nil {
					return err
				}

				fieldVal, err := fieldPotentialVal.getValue(i, loc)
				if err != nil {
					return err
				}

				buf.WriteString(prefix)
				buf.WriteString(indent2)

				buf.WriteString("\"")
				buf.WriteString(fieldName)
				buf.WriteString("\"")
				buf.WriteString(": ")

				// TODO body.Loc()
				err = i.manifestJSON(loc, fieldVal, multiline, indent2, buf)
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

func (i *interpreter) EvalInCleanEnv(fromWhere *LocationRange, env *environment, ast astNode) (value, error) {
	// TODO(sbarzowski) Figure out if this context arg (nil here) is needed for anything
	i.stack.newCall(fromWhere, nil, *env)
	val, err := i.evaluate(ast)
	i.stack.pop()
	return val, err
}

func evaluate(ast astNode, ext vmExtMap, maxStack int) (string, error) {
	i := interpreter{
		stack:          makeCallStack(maxStack),
		idArrayElement: identifier("array_element"),
		idInvariant:    identifier("object_assert"),
		externalVars:   ext,
	}
	// TODO(sbarzowski) include extVars in this newCall
	initialEnv := makeEnvironment(
		bindingFrame{},
		nil,
		123456789, // poison value
	)
	evalLoc := makeLocationRangeMessage("During evaluation")
	result, err := i.EvalInCleanEnv(&evalLoc, &initialEnv, ast)
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
