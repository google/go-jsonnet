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

import "fmt"

type evaluator struct {
	i         *interpreter
	fromWhere *LocationRange

	// It accumulates the error, which allows evaluating multiple things
	// without checking for error each time
	// TODO(sbarzowski) is it really a good idea? So far it only causes problems
	err error
}

func (e *evaluator) evaluate(ph potentialValue) value {
	if e.err != nil {
		return nil
	}
	val, err := ph.getValue(e.i, e.fromWhere)
	if err != nil {
		e.err = err
		return nil
	}
	return val
}

func (e *evaluator) typeErrorSpecific(bad value, good value) error {
	return makeRuntimeError(
		e.fromWhere,
		fmt.Sprintf("Unexpected type %v, expected %v", bad.typename(), good.typename()),
	)
}

func (e *evaluator) typeErrorGeneral(bad value) error {
	return makeRuntimeError(
		e.fromWhere,
		fmt.Sprintf("Unexpected type %v", bad.typename()),
	)
}

func (e *evaluator) getNumber(val value) *valueNumber {
	switch v := val.(type) {
	case *valueNumber:
		return v
	default:
		e.err = e.typeErrorSpecific(val, &valueNumber{})
		return nil
	}
}

func (e *evaluator) evaluateNumber(pv potentialValue) *valueNumber {
	v := e.evaluate(pv)
	if e.err != nil {
		return nil
	}
	return e.getNumber(v)
}

func (e *evaluator) getString(val value) *valueString {
	switch v := val.(type) {
	case *valueString:
		return v
	default:
		e.err = e.typeErrorSpecific(val, &valueString{})
		return nil
	}
}

func (e *evaluator) evaluateString(pv potentialValue) *valueString {
	v := e.evaluate(pv)
	if e.err != nil {
		return nil
	}
	return e.getString(v)
}

func (e *evaluator) getBoolean(val value) *valueBoolean {
	switch v := val.(type) {
	case *valueBoolean:
		return v
	default:
		e.err = e.typeErrorSpecific(val, &valueBoolean{})
		return nil
	}
}

func (e *evaluator) evaluateBoolean(pv potentialValue) *valueBoolean {
	v := e.evaluate(pv)
	if e.err != nil {
		return nil
	}
	return e.getBoolean(v)
}

func (e *evaluator) lookUpVar(ident identifier) potentialValue {
	th := e.i.stack.lookUpVar(ident)
	if th == nil {
		// fmt.Println(dumpCallStack(&i.stack))
		// This, should be caught during static check, right?
		e.err = makeRuntimeError(e.fromWhere, fmt.Sprintf("Unknown variable: %v", ident))
	}
	return th
}

type builtinPlus struct {
}

func (b *builtinPlus) Binary(xp, yp potentialValue, e *evaluator) (value, error) {
	x := e.evaluate(xp)
	if e.err != nil {
		return nil, e.err
	}
	switch left := x.(type) {
	case *valueNumber:
		right := e.evaluateNumber(yp)
		if e.err != nil {
			return nil, e.err
		}
		return makeValueNumber(left.value + right.value), nil
	case *valueString:
		right := e.evaluateString(yp)
		if e.err != nil {
			return nil, e.err
		}
		return makeValueString(left.value + right.value), nil
	default:
		return nil, e.typeErrorGeneral(x)
	}
}

type builtinMinus struct {
}

func (b *builtinMinus) Binary(xp, yp potentialValue, e *evaluator) (value, error) {
	x := e.evaluateNumber(xp)
	y := e.evaluateNumber(yp)
	if e.err != nil {
		return nil, e.err
	}
	return makeValueNumber(x.value - y.value), nil
}

type builtinAnd struct {
}

func (b *builtinAnd) Binary(xp, yp potentialValue, e *evaluator) (value, error) {
	x := e.evaluateBoolean(xp)
	if e.err != nil {
		return nil, e.err
	}
	if !x.value {
		return x, nil
	}
	y := e.evaluateBoolean(yp)
	if e.err != nil {
		return nil, e.err
	}
	return y, nil
}

type builtinLength struct {
}

func (b *builtinLength) Unary(x value, e *evaluator) (value, error) {
	var num int
	switch x := x.(type) {
	case *valueSimpleObject:
		panic("TODO getting all the fields")
	case *valueArray:
		num = len(x.elements)
	case *valueString:
		num = len(x.value)
	case *valueClosure:
		num = len(x.function.parameters)
	default:
		return nil, e.typeErrorGeneral(x)
	}
	return makeValueNumber(float64(num)), nil
}

func (b *builtinLength) Parameters() identifiers {
	return identifiers{"x", "y"}
}

type binaryBuiltin interface {
	Binary(x, y potentialValue, e *evaluator) (value, error)
}

var bopBuiltins = []binaryBuiltin{
	// bopMult:    "*",
	// bopDiv:     "/",
	// bopPercent: "%",

	bopPlus:  &builtinPlus{},
	bopMinus: &builtinMinus{},

	// bopShiftL: "<<",
	// bopShiftR: ">>",

	// bopGreater:   ">",
	// bopGreaterEq: ">=",
	// bopLess:      "<",
	// bopLessEq:    "<=",

	// bopManifestEqual:   "==",
	// bopManifestUnequal: "!=",

	// bopBitwiseAnd: "&",
	// bopBitwiseXor: "^",
	// bopBitwiseOr:  "|",

	bopAnd: &builtinAnd{},
	// bopOr:  "||",
}
