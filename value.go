/*
Copyright 2017 Google Inc. All rights reserved.

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
	"errors"
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

// value represents a concrete jsonnet value of a specific type.
// Various operations on values are allowed, depending on their type.
// All values are of course immutable.
type value interface {
	aValue()

	// TODO(sbarzowski) consider representing each type as golang object
	typename() string
}

// potentialValue is something that may be evaluated to a concrete value.
// The result of the evaluation may *NOT* depend on the current state
// of the interpreter. The evaluation may fail.
//
// It can be used to represent lazy values (e.g. variables values in jsonnet
// are not calculated before they are used). It is also a useful abstraction
// in other cases like error handling.
//
// It may or may not require computation.
//
// Getting the value a second time may or may not result in additional evaluation.
//
// TODO(sbarzowski) perhaps call it just "Thunk"?
type potentialValue interface {
	// fromWhere keeps the information from where the evaluation was requested.
	getValue(i *interpreter, fromWhere *TraceElement) (value, error)
}

// A set of variables with associated potentialValues.
type bindingFrame map[ast.Identifier]potentialValue

type valueBase struct{}

func (v *valueBase) aValue() {}

// Primitive values
// -------------------------------------

type valueString struct {
	valueBase
	// We use rune slices instead of strings for quick indexing
	value []rune
}

func (s *valueString) index(e *evaluator, index int) (value, error) {
	if 0 <= index && index < s.length() {
		return makeValueString(string(s.value[index])), nil
	}
	return nil, e.Error(fmt.Sprintf("Index %d out of bounds, not within [0, %v)", index, s.length()))
}

func concatStrings(a, b *valueString) *valueString {
	result := make([]rune, 0, len(a.value)+len(b.value))
	for _, r := range a.value {
		result = append(result, r)
	}
	for _, r := range b.value {
		result = append(result, r)
	}
	return &valueString{value: result}
}

func stringLessThan(a, b *valueString) bool {
	var length int
	if len(a.value) < len(b.value) {
		length = len(a.value)
	} else {
		length = len(b.value)
	}
	for i := 0; i < length; i++ {
		if a.value[i] != b.value[i] {
			return a.value[i] < b.value[i]
		}
	}
	return len(a.value) < len(b.value)
}

func stringEqual(a, b *valueString) bool {
	if len(a.value) != len(b.value) {
		return false
	}
	for i := 0; i < len(a.value); i++ {
		if a.value[i] != b.value[i] {
			return false
		}
	}
	return true
}

func (s *valueString) length() int {
	return len(s.value)
}

func (s *valueString) getString() string {
	return string(s.value)
}

func makeValueString(v string) *valueString {
	return &valueString{value: []rune(v)}
}

func (*valueString) typename() string {
	return "string"
}

type valueBoolean struct {
	valueBase
	value bool
}

func (*valueBoolean) typename() string {
	return "boolean"
}

func makeValueBoolean(v bool) *valueBoolean {
	return &valueBoolean{value: v}
}

func (b *valueBoolean) not() *valueBoolean {
	return makeValueBoolean(!b.value)
}

type valueNumber struct {
	valueBase
	value float64
}

func (*valueNumber) typename() string {
	return "number"
}

func makeValueNumber(v float64) *valueNumber {
	return &valueNumber{value: v}
}

func intToValue(i int) *valueNumber {
	return makeValueNumber(float64(i))
}

func int64ToValue(i int64) *valueNumber {
	return makeValueNumber(float64(i))
}

// TODO(dcunnin): Maybe intern values null, true, and false?
type valueNull struct {
	valueBase
}

func makeValueNull() *valueNull {
	return &valueNull{}
}

func (*valueNull) typename() string {
	return "null"
}

// ast.Array
// -------------------------------------

type valueArray struct {
	valueBase
	elements []potentialValue
}

func (arr *valueArray) length() int {
	return len(arr.elements)
}

func makeValueArray(elements []potentialValue) *valueArray {
	// We don't want to keep a bigger array than necessary
	// so we create a new one with minimal capacity
	var arrayElems []potentialValue
	if len(elements) == cap(elements) {
		arrayElems = elements
	} else {
		arrayElems = make([]potentialValue, len(elements))
		for i := range elements {
			arrayElems[i] = elements[i]
		}
	}
	return &valueArray{
		elements: arrayElems,
	}
}

func concatArrays(a, b *valueArray) *valueArray {
	result := make([]potentialValue, 0, len(a.elements)+len(b.elements))
	for _, r := range a.elements {
		result = append(result, r)
	}
	for _, r := range b.elements {
		result = append(result, r)
	}
	return &valueArray{elements: result}
}

func (*valueArray) typename() string {
	return "array"
}

// ast.Function
// -------------------------------------

type valueFunction struct {
	valueBase
	ec evalCallable
}

// TODO(sbarzowski) better name?
type evalCallable interface {
	EvalCall(args callArguments, e *evaluator) (value, error)
	Parameters() ast.Identifiers
}

func (f *valueFunction) call(args callArguments) potentialValue {
	return makeCallThunk(f.ec, args)
}

func (f *valueFunction) parameters() ast.Identifiers {
	return f.ec.Parameters()
}

func (f *valueFunction) typename() string {
	return "function"
}

type callArguments struct {
	positional []potentialValue
	// TODO named arguments
}

func args(xs ...potentialValue) callArguments {
	return callArguments{positional: xs}
}

// Objects
// -------------------------------------

// Object is a value that allows indexing (taking a value of a field)
// and combining through mixin inheritence (operator +).
//
// Accessing a field multiple times results in multiple evaluations.
// TODO(sbarzowski) This can be very easily avoided and currently innocent looking
// 					code may be in fact exponential.
type valueObject interface {
	value
	inheritanceSize() int
	index(e *evaluator, field string) (value, error)
	assertionsChecked() bool
	setAssertionsCheckResult(err error)
	getAssertionsCheckResult() error
}

type selfBinding struct {
	// self is the lexically nearest object we are in, or nil.  Note
	// that this is not the same as context, because we could be inside a function,
	// inside an object and then context would be the function, but self would still point
	// to the object.
	self valueObject

	// superDepth is the "super" level of self.  Sometimes, we look upwards in the
	// inheritance tree, e.g. via an explicit use of super, or because a given field
	// has been inherited.  When evaluating a field from one of these super objects,
	// we need to bind self to the concrete object (so self must point
	// there) but uses of super should be resolved relative to the object whose
	// field we are evaluating.  Thus, we keep a second field for that.  This is
	// usually 0, unless we are evaluating a super object's field.
	// TODO(sbarzowski) provide some examples
	// TODO(sbarzowski) provide somewhere a complete explanation of the object model
	superDepth int
}

func makeUnboundSelfBinding() selfBinding {
	return selfBinding{
		nil,
		123456789, // poison value
	}
}

func objectBinding(obj valueObject) selfBinding {
	return selfBinding{self: obj, superDepth: 0}
}

func (sb selfBinding) super() selfBinding {
	return selfBinding{self: sb.self, superDepth: sb.superDepth + 1}
}

type Hidden int

const (
	withHidden Hidden = iota
	withoutHidden
)

func withHiddenFromBool(with bool) Hidden {
	if with {
		return withHidden
	} else {
		return withoutHidden
	}
}

// Hack - we need to distinguish not-checked-yet and no error situations
// so we have a special value for no error and nil means that we don't know yet.
var errNoErrorInObjectInvariants = errors.New("No error - assertions passed")

type valueObjectBase struct {
	valueBase
	assertionError error
}

func (*valueObjectBase) typename() string {
	return "object"
}

func (obj *valueObjectBase) assertionsChecked() bool {
	// nil - not checked yet
	// errNoErrorInObjectInvariants - we checked and there is no error (or checking in progress)
	return obj.assertionError != nil
}

func (obj *valueObjectBase) setAssertionsCheckResult(err error) {
	if err != nil {
		obj.assertionError = err
	} else {
		obj.assertionError = errNoErrorInObjectInvariants
	}
}

func (obj *valueObjectBase) getAssertionsCheckResult() error {
	if obj.assertionError == nil {
		panic("Assertions not checked yet")
	}
	if obj.assertionError == errNoErrorInObjectInvariants {
		return nil
	}
	return obj.assertionError
}

// valueSimpleObject represents a flat object (no inheritance).
// Note that it can be used as part of extended objects
// in inheritance using operator +.
//
// Fields are late bound (to object), so they are not values or potentialValues.
// This is important for inheritance, for example:
// Let a = {x: 42} and b = {y: self.x}. Evaluating b.y is an error,
// but (a+b).y evaluates to 42.
type valueSimpleObject struct {
	valueObjectBase
	upValues bindingFrame
	fields   valueSimpleObjectFieldMap
	asserts  []unboundField
}

func checkAssertionsHelper(e *evaluator, obj valueObject, curr valueObject, superDepth int) error {
	switch curr := curr.(type) {
	case *valueExtendedObject:
		err := checkAssertionsHelper(e, obj, curr.right, superDepth)
		if err != nil {
			return err
		}
		err = checkAssertionsHelper(e, obj, curr.left, superDepth+curr.right.inheritanceSize())
		if err != nil {
			return err
		}
		return nil
	case *valueSimpleObject:
		for _, assert := range curr.asserts {
			_, err := e.evaluate(assert.bindToObject(selfBinding{self: obj, superDepth: superDepth}, curr.upValues, ""))
			if err != nil {
				return err
			}
		}
		return nil
	default:
		panic(fmt.Sprintf("Unknown object type %#v", obj))
	}
}

func checkAssertions(e *evaluator, obj valueObject) error {
	if !obj.assertionsChecked() {
		// Assertions may refer to the object that will normally
		// trigger checking of assertions, resulting in an endless recursion.
		// To avoid that, while we check them, we treat them as already passed.
		obj.setAssertionsCheckResult(errNoErrorInObjectInvariants)
		obj.setAssertionsCheckResult(checkAssertionsHelper(e, obj, obj, 0))
	}
	return obj.getAssertionsCheckResult()
}

func (o *valueSimpleObject) index(e *evaluator, field string) (value, error) {
	return objectIndex(e, objectBinding(o), field)
}

func (*valueSimpleObject) inheritanceSize() int {
	return 1
}

func makeValueSimpleObject(b bindingFrame, fields valueSimpleObjectFieldMap, asserts []unboundField) *valueSimpleObject {
	return &valueSimpleObject{
		upValues: b,
		fields:   fields,
		asserts:  asserts,
	}
}

type valueSimpleObjectFieldMap map[string]valueSimpleObjectField

// TODO(sbarzowski) this is not a value and the name suggests it is...
// TODO(sbarzowski) better name? This is basically just a (hide, field) pair.
type valueSimpleObjectField struct {
	hide  ast.ObjectFieldHide
	field unboundField
}

// unboundField is a field that doesn't know yet in which object it is.
type unboundField interface {
	bindToObject(sb selfBinding, origBinding bindingFrame, fieldName string) potentialValue
}

// valueExtendedObject represents an object created through inheritence (left + right).
// We represent it as the pair of objects. This results in a tree-like structure.
// Example:
// (A + B) + C
//
//        +
//       / \
//      +   C
//     / \
//    A   B
//
// It is possible to create an arbitrary binary tree.
// Note however, that because + is associative the only thing that matters
// is the order of leafs.
//
// This represenation allows us to implement "+" in O(1),
// but requires going through the tree and trying subsequent leafs for field access.
//
// TODO(sbarzowski) consider other representations (this representation was chosen to stay close to C++ version)
type valueExtendedObject struct {
	valueObjectBase
	left, right          valueObject
	totalInheritanceSize int
}

func (o *valueExtendedObject) index(e *evaluator, field string) (value, error) {
	return objectIndex(e, objectBinding(o), field)
}

func (o *valueExtendedObject) inheritanceSize() int {
	return o.totalInheritanceSize
}

func makeValueExtendedObject(left, right valueObject) *valueExtendedObject {
	return &valueExtendedObject{
		left:                 left,
		right:                right,
		totalInheritanceSize: left.inheritanceSize() + right.inheritanceSize(),
	}
}

// findField returns a field in object curr, with superDepth at least minSuperDepth
// It also returns an associated bindingFrame and actual superDepth that the field
// was found at.
func findField(curr value, minSuperDepth int, f string) (*valueSimpleObjectField, bindingFrame, int) {
	switch curr := curr.(type) {
	case *valueExtendedObject:
		if curr.right.inheritanceSize() > minSuperDepth {
			field, frame, counter := findField(curr.right, minSuperDepth, f)
			if field != nil {
				return field, frame, counter
			}
		}
		field, frame, counter := findField(curr.left, minSuperDepth-curr.right.inheritanceSize(), f)
		return field, frame, counter + curr.right.inheritanceSize()

	case *valueSimpleObject:
		if minSuperDepth <= 0 {
			if field, ok := curr.fields[f]; ok {
				return &field, curr.upValues, 0
			}
		}
		// TODO(sbarzowski) add handling of "Attempt to use super when there is no super class."
		return nil, nil, 0
	default:
		panic(fmt.Sprintf("Unknown object type %#v", curr))
	}
}

func objectIndex(e *evaluator, sb selfBinding, fieldName string) (value, error) {
	err := checkAssertions(e, sb.self)
	if err != nil {
		return nil, err
	}
	objp := tryObjectIndex(sb, fieldName, withHidden)
	if objp == nil {
		return nil, e.Error(fmt.Sprintf("Field does not exist: %s", fieldName))
	}
	return e.evaluate(objp)
}

func tryObjectIndex(sb selfBinding, fieldName string, h Hidden) potentialValue {
	field, upValues, foundAt := findField(sb.self, sb.superDepth, fieldName)
	if field == nil || (h == withoutHidden && field.hide == ast.ObjectFieldHidden) {
		return nil
	}
	fieldSelfBinding := selfBinding{self: sb.self, superDepth: foundAt}

	return field.field.bindToObject(fieldSelfBinding, upValues, fieldName)
}

type fieldHideMap map[string]ast.ObjectFieldHide

func objectFieldsVisibility(obj valueObject) fieldHideMap {
	r := make(fieldHideMap)
	switch obj := obj.(type) {
	case *valueExtendedObject:
		r = objectFieldsVisibility(obj.left)
		rightMap := objectFieldsVisibility(obj.right)
		for k, v := range rightMap {
			if v == ast.ObjectFieldInherit {
				if _, alreadyExists := r[k]; !alreadyExists {
					r[k] = v
				}
			} else {
				r[k] = v
			}
		}
		return r

	case *valueSimpleObject:
		for fieldName, field := range obj.fields {
			r[fieldName] = field.hide
		}
	}
	return r
}

func objectFields(obj valueObject, h Hidden) []string {
	var r []string
	for fieldName, hide := range objectFieldsVisibility(obj) {
		if h == withHidden || hide != ast.ObjectFieldHidden {
			r = append(r, fieldName)
		}
	}
	return r
}

func duplicateFieldNameErrMsg(fieldName string) string {
	return fmt.Sprintf("Duplicate field name: %s", unparseString(fieldName))
}
