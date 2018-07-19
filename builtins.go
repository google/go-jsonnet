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
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/google/go-jsonnet/ast"
)

func builtinPlus(i *interpreter, trace TraceElement, x, y value) (value, error) {
	// TODO(sbarzowski) more types, mixing types
	// TODO(sbarzowski) perhaps a more elegant way to dispatch
	switch right := y.(type) {
	case *valueString:
		left, err := builtinToString(i, trace, x)
		if err != nil {
			return nil, err
		}
		return concatStrings(left.(*valueString), right), nil

	}
	switch left := x.(type) {
	case *valueNumber:
		right, err := i.getNumber(y, trace)
		if err != nil {
			return nil, err
		}
		return makeValueNumber(left.value + right.value), nil
	case *valueString:
		right, err := builtinToString(i, trace, y)
		if err != nil {
			return nil, err
		}
		return concatStrings(left, right.(*valueString)), nil
	case valueObject:
		switch right := y.(type) {
		case valueObject:
			return makeValueExtendedObject(left, right), nil
		default:
			return nil, i.typeErrorSpecific(y, &valueSimpleObject{}, trace)
		}

	case *valueArray:
		right, err := i.getArray(y, trace)
		if err != nil {
			return nil, err
		}
		return concatArrays(left, right), nil
	default:
		return nil, i.typeErrorGeneral(x, trace)
	}
}

func builtinMinus(i *interpreter, trace TraceElement, xv, yv value) (value, error) {
	x, err := i.getNumber(xv, trace)
	if err != nil {
		return nil, err
	}
	y, err := i.getNumber(yv, trace)
	if err != nil {
		return nil, err
	}
	return makeValueNumber(x.value - y.value), nil
}

func builtinMult(i *interpreter, trace TraceElement, xv, yv value) (value, error) {
	x, err := i.getNumber(xv, trace)
	if err != nil {
		return nil, err
	}
	y, err := i.getNumber(yv, trace)
	if err != nil {
		return nil, err
	}
	return makeValueNumber(x.value * y.value), nil
}

func builtinDiv(i *interpreter, trace TraceElement, xv, yv value) (value, error) {
	x, err := i.getNumber(xv, trace)
	if err != nil {
		return nil, err
	}
	y, err := i.getNumber(yv, trace)
	if err != nil {
		return nil, err
	}
	if y.value == 0 {
		return nil, i.Error("Division by zero.", trace)
	}
	return makeDoubleCheck(i, trace, x.value/y.value)
}

func builtinModulo(i *interpreter, trace TraceElement, xv, yv value) (value, error) {
	x, err := i.getNumber(xv, trace)
	if err != nil {
		return nil, err
	}
	y, err := i.getNumber(yv, trace)
	if err != nil {
		return nil, err
	}
	if y.value == 0 {
		return nil, i.Error("Division by zero.", trace)
	}
	return makeDoubleCheck(i, trace, math.Mod(x.value, y.value))
}

func builtinLess(i *interpreter, trace TraceElement, x, yv value) (value, error) {
	switch left := x.(type) {
	case *valueNumber:
		right, err := i.getNumber(yv, trace)
		if err != nil {
			return nil, err
		}
		return makeValueBoolean(left.value < right.value), nil
	case *valueString:
		right, err := i.getString(yv, trace)
		if err != nil {
			return nil, err
		}
		return makeValueBoolean(stringLessThan(left, right)), nil
	default:
		return nil, i.typeErrorGeneral(x, trace)
	}
}

func builtinGreater(i *interpreter, trace TraceElement, x, y value) (value, error) {
	return builtinLess(i, trace, y, x)
}

func builtinGreaterEq(i *interpreter, trace TraceElement, x, y value) (value, error) {
	res, err := builtinLess(i, trace, x, y)
	if err != nil {
		return nil, err
	}
	return res.(*valueBoolean).not(), nil
}

func builtinLessEq(i *interpreter, trace TraceElement, x, y value) (value, error) {
	res, err := builtinGreater(i, trace, x, y)
	if err != nil {
		return nil, err
	}
	return res.(*valueBoolean).not(), nil
}

func builtinLength(i *interpreter, trace TraceElement, x value) (value, error) {
	var num int
	switch x := x.(type) {
	case valueObject:
		num = len(objectFields(x, withoutHidden))
	case *valueArray:
		num = len(x.elements)
	case *valueString:
		num = x.length()
	case *valueFunction:
		num = len(x.parameters().required)
	default:
		return nil, i.typeErrorGeneral(x, trace)
	}
	return makeValueNumber(float64(num)), nil
}

func builtinToString(i *interpreter, trace TraceElement, x value) (value, error) {
	switch x := x.(type) {
	case *valueString:
		return x, nil
	}
	var buf bytes.Buffer
	err := i.manifestAndSerializeJSON(&buf, trace, x, false, "")
	if err != nil {
		return nil, err
	}
	return makeValueString(buf.String()), nil
}

func builtinTrace(i *interpreter, trace TraceElement, x value, y value) (value, error) {
	xStr, err := i.getString(x, trace)
	if err != nil {
		return nil, err
	}
	filename := trace.loc.FileName
	line := trace.loc.Begin.Line
	fmt.Fprintf(
		os.Stderr, "TRACE: %s:%d %s\n", filename, line, xStr.getString())
	return y, nil
}

// astMakeArrayElement wraps the function argument of std.makeArray so that
// it can be embedded in cachedThunk without needing to execute it ahead of
// time.  It is equivalent to `local i = 42; func(i)`.  It therefore has no
// free variables and needs only an empty environment to execute.
type astMakeArrayElement struct {
	ast.NodeBase
	function *valueFunction
	index    int
}

func builtinMakeArray(i *interpreter, trace TraceElement, szv, funcv value) (value, error) {
	sz, err := i.getInt(szv, trace)
	if err != nil {
		return nil, err
	}
	fun, err := i.getFunction(funcv, trace)
	if err != nil {
		return nil, err
	}
	var elems []*cachedThunk
	for i := 0; i < sz; i++ {
		elem := &cachedThunk{
			env: &environment{},
			body: &astMakeArrayElement{
				NodeBase: ast.NodeBase{},
				function: fun,
				index:    i,
			},
		}
		elems = append(elems, elem)
	}
	return makeValueArray(elems), nil
}

func builtinFlatMap(i *interpreter, trace TraceElement, funcv, arrv value) (value, error) {
	arr, err := i.getArray(arrv, trace)
	if err != nil {
		return nil, err
	}
	fun, err := i.getFunction(funcv, trace)
	if err != nil {
		return nil, err
	}
	num := arr.length()
	// Start with capacity of the original array.
	// This may spare us a few reallocations.
	// TODO(sbarzowski) verify that it actually helps
	elems := make([]*cachedThunk, 0, num)
	for counter := 0; counter < num; counter++ {
		returnedValue, err := fun.call(i, trace, args(arr.elements[counter]))
		if err != nil {
			return nil, err
		}
		returned, err := i.getArray(returnedValue, trace)
		if err != nil {
			return nil, err
		}
		for _, elem := range returned.elements {
			elems = append(elems, elem)
		}
	}
	return makeValueArray(elems), nil
}

func joinArrays(i *interpreter, trace TraceElement, sep *valueArray, arr *valueArray) (value, error) {
	result := make([]*cachedThunk, 0, arr.length())
	first := true
	for _, elem := range arr.elements {
		elemValue, err := i.evaluatePV(elem, trace)
		if err != nil {
			return nil, err
		}
		switch v := elemValue.(type) {
		case *valueNull:
			continue
		case *valueArray:
			if !first {
				for _, subElem := range sep.elements {
					result = append(result, subElem)
				}
			}
			for _, subElem := range v.elements {
				result = append(result, subElem)
			}
		default:
			return nil, i.typeErrorSpecific(elemValue, &valueArray{}, trace)
		}
		first = false

	}
	return makeValueArray(result), nil
}

func joinStrings(i *interpreter, trace TraceElement, sep *valueString, arr *valueArray) (value, error) {
	result := make([]rune, 0, arr.length())
	first := true
	for _, elem := range arr.elements {
		elemValue, err := i.evaluatePV(elem, trace)
		if err != nil {
			return nil, err
		}
		switch v := elemValue.(type) {
		case *valueNull:
			continue
		case *valueString:
			if !first {
				result = append(result, sep.value...)
			}
			result = append(result, v.value...)
		default:
			return nil, i.typeErrorSpecific(elemValue, &valueString{}, trace)
		}
		first = false
	}
	return &valueString{value: result}, nil
}

func builtinJoin(i *interpreter, trace TraceElement, sep, arrv value) (value, error) {
	arr, err := i.getArray(arrv, trace)
	if err != nil {
		return nil, err
	}
	switch sep := sep.(type) {
	case *valueString:
		return joinStrings(i, trace, sep, arr)
	case *valueArray:
		return joinArrays(i, trace, sep, arr)
	default:
		return nil, i.Error("join first parameter should be string or array, got "+sep.getType().name, trace)
	}
}

func builtinFilter(i *interpreter, trace TraceElement, funcv, arrv value) (value, error) {
	arr, err := i.getArray(arrv, trace)
	if err != nil {
		return nil, err
	}
	fun, err := i.getFunction(funcv, trace)
	if err != nil {
		return nil, err
	}
	num := arr.length()
	// Start with capacity of the original array.
	// This may spare us a few reallocations.
	// TODO(sbarzowski) verify that it actually helps
	elems := make([]*cachedThunk, 0, num)
	for counter := 0; counter < num; counter++ {
		includedValue, err := fun.call(i, trace, args(arr.elements[counter]))
		if err != nil {
			return nil, err
		}
		included, err := i.getBoolean(includedValue, trace)
		if err != nil {
			return nil, err
		}
		if included.value {
			elems = append(elems, arr.elements[counter])
		}
	}
	return makeValueArray(elems), nil
}

func builtinRange(i *interpreter, trace TraceElement, fromv, tov value) (value, error) {
	from, err := i.getInt(fromv, trace)
	if err != nil {
		return nil, err
	}
	to, err := i.getInt(tov, trace)
	if err != nil {
		return nil, err
	}
	elems := make([]*cachedThunk, to-from+1)
	for i := from; i <= to; i++ {
		elems[i-from] = readyThunk(intToValue(i))
	}
	return makeValueArray(elems), nil
}

func builtinNegation(i *interpreter, trace TraceElement, x value) (value, error) {
	b, err := i.getBoolean(x, trace)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(!b.value), nil
}

func builtinBitNeg(i *interpreter, trace TraceElement, x value) (value, error) {
	n, err := i.getNumber(x, trace)
	if err != nil {
		return nil, err
	}
	intValue := int64(n.value)
	return int64ToValue(^intValue), nil
}

func builtinIdentity(i *interpreter, trace TraceElement, x value) (value, error) {
	return x, nil
}

func builtinUnaryMinus(i *interpreter, trace TraceElement, x value) (value, error) {
	n, err := i.getNumber(x, trace)
	if err != nil {
		return nil, err
	}
	return makeValueNumber(-n.value), nil
}

// TODO(sbarzowski) since we have a builtin implementation of equals it's no longer really
// needed and we should deprecate it eventually
func primitiveEquals(i *interpreter, trace TraceElement, x, y value) (value, error) {
	if x.getType() != y.getType() {
		return makeValueBoolean(false), nil
	}
	switch left := x.(type) {
	case *valueBoolean:
		right, err := i.getBoolean(y, trace)
		if err != nil {
			return nil, err
		}
		return makeValueBoolean(left.value == right.value), nil
	case *valueNumber:
		right, err := i.getNumber(y, trace)
		if err != nil {
			return nil, err
		}
		return makeValueBoolean(left.value == right.value), nil
	case *valueString:
		right, err := i.getString(y, trace)
		if err != nil {
			return nil, err
		}
		return makeValueBoolean(stringEqual(left, right)), nil
	case *valueNull:
		return makeValueBoolean(true), nil
	case *valueFunction:
		return nil, i.Error("Cannot test equality of functions", trace)
	default:
		return nil, i.Error(
			"primitiveEquals operates on primitive types, got "+x.getType().name,
			trace,
		)
	}
}

func rawEquals(i *interpreter, trace TraceElement, x, y value) (bool, error) {
	if x.getType() != y.getType() {
		return false, nil
	}
	switch left := x.(type) {
	case *valueBoolean:
		right, err := i.getBoolean(y, trace)
		if err != nil {
			return false, err
		}
		return left.value == right.value, nil
	case *valueNumber:
		right, err := i.getNumber(y, trace)
		if err != nil {
			return false, err
		}
		return left.value == right.value, nil
	case *valueString:
		right, err := i.getString(y, trace)
		if err != nil {
			return false, err
		}
		return stringEqual(left, right), nil
	case *valueNull:
		return true, nil
	case *valueArray:
		right, err := i.getArray(y, trace)
		if err != nil {
			return false, err
		}
		if left.length() != right.length() {
			return false, nil
		}
		for j := range left.elements {
			leftElem, err := i.evaluatePV(left.elements[j], trace)
			if err != nil {
				return false, err
			}
			rightElem, err := i.evaluatePV(right.elements[j], trace)
			if err != nil {
				return false, err
			}
			eq, err := rawEquals(i, trace, leftElem, rightElem)
			if err != nil {
				return false, err
			}
			if !eq {
				return false, nil
			}
		}
		return true, nil
	case valueObject:
		right, err := i.getObject(y, trace)
		if err != nil {
			return false, err
		}
		leftFields := objectFields(left, withoutHidden)
		rightFields := objectFields(right, withoutHidden)
		sort.Strings(leftFields)
		sort.Strings(rightFields)
		if len(leftFields) != len(rightFields) {
			return false, nil
		}
		for i := range leftFields {
			if leftFields[i] != rightFields[i] {
				return false, nil
			}
		}
		for j := range leftFields {
			fieldName := leftFields[j]
			leftField, err := left.index(i, trace, fieldName)
			if err != nil {
				return false, err
			}
			rightField, err := right.index(i, trace, fieldName)
			if err != nil {
				return false, err
			}
			eq, err := rawEquals(i, trace, leftField, rightField)
			if err != nil {
				return false, err
			}
			if !eq {
				return false, nil
			}
		}
		return true, nil
	case *valueFunction:
		return false, i.Error("Cannot test equality of functions", trace)
	}
	panic(fmt.Sprintf("Unhandled case in equals %#+v %#+v", x, y))
}

func builtinEquals(i *interpreter, trace TraceElement, x, y value) (value, error) {
	eq, err := rawEquals(i, trace, x, y)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(eq), nil
}

func builtinNotEquals(i *interpreter, trace TraceElement, x, y value) (value, error) {
	eq, err := rawEquals(i, trace, x, y)
	if err != nil {
		return nil, err
	}
	return makeValueBoolean(!eq), nil
}

func builtinType(i *interpreter, trace TraceElement, x value) (value, error) {
	return makeValueString(x.getType().name), nil
}

func builtinMd5(i *interpreter, trace TraceElement, x value) (value, error) {
	str, err := i.getString(x, trace)
	if err != nil {
		return nil, err
	}
	hash := md5.Sum([]byte(string(str.value)))
	return makeValueString(hex.EncodeToString(hash[:])), nil
}

// Maximum allowed unicode codepoint
// https://en.wikipedia.org/wiki/Unicode#Architecture_and_terminology
const codepointMax = 0x10FFFF

func builtinChar(i *interpreter, trace TraceElement, x value) (value, error) {
	n, err := i.getNumber(x, trace)
	if err != nil {
		return nil, err
	}
	if n.value > codepointMax {
		return nil, i.Error(fmt.Sprintf("Invalid unicode codepoint, got %v", n.value), trace)
	} else if n.value < 0 {
		return nil, i.Error(fmt.Sprintf("Codepoints must be >= 0, got %v", n.value), trace)
	}
	return makeValueString(string(rune(n.value))), nil
}

func builtinCodepoint(i *interpreter, trace TraceElement, x value) (value, error) {
	str, err := i.getString(x, trace)
	if err != nil {
		return nil, err
	}
	if str.length() != 1 {
		return nil, i.Error(fmt.Sprintf("codepoint takes a string of length 1, got length %v", str.length()), trace)
	}
	return makeValueNumber(float64(str.value[0])), nil
}

func makeDoubleCheck(i *interpreter, trace TraceElement, x float64) (value, error) {
	if math.IsNaN(x) {
		return nil, i.Error("Not a number", trace)
	}
	if math.IsInf(x, 0) {
		return nil, i.Error("Overflow", trace)
	}
	return makeValueNumber(x), nil
}

func liftNumeric(f func(float64) float64) func(*interpreter, TraceElement, value) (value, error) {
	return func(i *interpreter, trace TraceElement, x value) (value, error) {
		n, err := i.getNumber(x, trace)
		if err != nil {
			return nil, err
		}
		return makeDoubleCheck(i, trace, f(n.value))
	}
}

var builtinSqrt = liftNumeric(math.Sqrt)
var builtinCeil = liftNumeric(math.Ceil)
var builtinFloor = liftNumeric(math.Floor)
var builtinSin = liftNumeric(math.Sin)
var builtinCos = liftNumeric(math.Cos)
var builtinTan = liftNumeric(math.Tan)
var builtinAsin = liftNumeric(math.Asin)
var builtinAcos = liftNumeric(math.Acos)
var builtinAtan = liftNumeric(math.Atan)
var builtinLog = liftNumeric(math.Log)
var builtinExp = liftNumeric(func(f float64) float64 {
	res := math.Exp(f)
	if res == 0 && f > 0 {
		return math.Inf(1)
	}
	return res
})
var builtinMantissa = liftNumeric(func(f float64) float64 {
	mantissa, _ := math.Frexp(f)
	return mantissa
})
var builtinExponent = liftNumeric(func(f float64) float64 {
	_, exponent := math.Frexp(f)
	return float64(exponent)
})

func liftBitwise(f func(int64, int64) int64) func(*interpreter, TraceElement, value, value) (value, error) {
	return func(i *interpreter, trace TraceElement, xv, yv value) (value, error) {
		x, err := i.getNumber(xv, trace)
		if err != nil {
			return nil, err
		}
		y, err := i.getNumber(yv, trace)
		if err != nil {
			return nil, err
		}
		return makeDoubleCheck(i, trace, float64(f(int64(x.value), int64(y.value))))
	}
}

// TODO(sbarzowski) negative shifts
var builtinShiftL = liftBitwise(func(x, y int64) int64 { return x << uint(y) })
var builtinShiftR = liftBitwise(func(x, y int64) int64 { return x >> uint(y) })
var builtinBitwiseAnd = liftBitwise(func(x, y int64) int64 { return x & y })
var builtinBitwiseOr = liftBitwise(func(x, y int64) int64 { return x | y })
var builtinBitwiseXor = liftBitwise(func(x, y int64) int64 { return x ^ y })

func builtinObjectFieldsEx(i *interpreter, trace TraceElement, objv, includeHiddenV value) (value, error) {
	obj, err := i.getObject(objv, trace)
	if err != nil {
		return nil, err
	}
	includeHidden, err := i.getBoolean(includeHiddenV, trace)
	if err != nil {
		return nil, err
	}
	fields := objectFields(obj, withHiddenFromBool(includeHidden.value))
	sort.Strings(fields)
	elems := []*cachedThunk{}
	for _, fieldname := range fields {
		elems = append(elems, readyThunk(makeValueString(fieldname)))
	}
	return makeValueArray(elems), nil
}

func builtinObjectHasEx(i *interpreter, trace TraceElement, objv value, fnamev value, includeHiddenV value) (value, error) {
	obj, err := i.getObject(objv, trace)
	if err != nil {
		return nil, err
	}
	fname, err := i.getString(fnamev, trace)
	if err != nil {
		return nil, err
	}
	includeHidden, err := i.getBoolean(includeHiddenV, trace)
	if err != nil {
		return nil, err
	}
	h := withHiddenFromBool(includeHidden.value)
	hasField := objectHasField(objectBinding(obj), string(fname.value), h)
	return makeValueBoolean(hasField), nil
}

func builtinPow(i *interpreter, trace TraceElement, basev value, expv value) (value, error) {
	base, err := i.getNumber(basev, trace)
	if err != nil {
		return nil, err
	}
	exp, err := i.getNumber(expv, trace)
	if err != nil {
		return nil, err
	}
	return makeDoubleCheck(i, trace, math.Pow(base.value, exp.value))
}

func builtinStrReplace(i *interpreter, trace TraceElement, strv, fromv, tov value) (value, error) {
	str, err := i.getString(strv, trace)
	if err != nil {
		return nil, err
	}
	from, err := i.getString(fromv, trace)
	if err != nil {
		return nil, err
	}
	to, err := i.getString(tov, trace)
	if err != nil {
		return nil, err
	}
	sStr := str.getString()
	sFrom := from.getString()
	sTo := to.getString()
	if len(sFrom) == 0 {
		return nil, i.Error("'from' string must not be zero length.", trace)
	}
	return makeValueString(strings.Replace(sStr, sFrom, sTo, -1)), nil
}

func builtinUglyObjectFlatMerge(i *interpreter, trace TraceElement, x value) (value, error) {
	objarr, err := i.getArray(x, trace)
	if err != nil {
		return nil, err
	}
	if len(objarr.elements) == 0 {
		return &valueSimpleObject{}, nil
	}
	newFields := make(simpleObjectFieldMap)
	for _, elem := range objarr.elements {
		obj, err := i.evaluateObject(elem, trace)
		if err != nil {
			return nil, err
		}
		// starts getting ugly - we mess with object internals
		simpleObj := obj.(*valueSimpleObject)
		for fieldName, fieldVal := range simpleObj.fields {
			if _, alreadyExists := newFields[fieldName]; alreadyExists {
				return nil, i.Error(duplicateFieldNameErrMsg(fieldName), trace)
			}
			newFields[fieldName] = simpleObjectField{
				hide: fieldVal.hide,
				field: &bindingsUnboundField{
					inner:    fieldVal.field,
					bindings: simpleObj.upValues,
				},
			}
		}
	}
	return makeValueSimpleObject(
		nil, // no binding frame
		newFields,
		[]unboundField{}, // No asserts allowed
	), nil
}

func builtinExtVar(i *interpreter, trace TraceElement, name value) (value, error) {
	str, err := i.getString(name, trace)
	if err != nil {
		return nil, err
	}
	index := str.getString()
	if pv, ok := i.extVars[index]; ok {
		return i.evaluatePV(pv, trace)
	}
	return nil, i.Error("Undefined external variable: "+string(index), trace)
}

func builtinNative(i *interpreter, trace TraceElement, name value) (value, error) {
	str, err := i.getString(name, trace)
	if err != nil {
		return nil, err
	}
	index := str.getString()
	if f, exists := i.nativeFuncs[index]; exists {
		return &valueFunction{ec: f}, nil
	}
	return &valueNull{}, nil
}

type unaryBuiltinFunc func(*interpreter, TraceElement, value) (value, error)
type binaryBuiltinFunc func(*interpreter, TraceElement, value, value) (value, error)
type ternaryBuiltinFunc func(*interpreter, TraceElement, value, value, value) (value, error)

type unaryBuiltin struct {
	name       ast.Identifier
	function   unaryBuiltinFunc
	parameters ast.Identifiers
}

func getBuiltinTrace(trace TraceElement, name ast.Identifier) TraceElement {
	context := "builtin function <" + string(name) + ">"
	return TraceElement{loc: trace.loc, context: &context}
}

func (b *unaryBuiltin) EvalCall(args callArguments, i *interpreter, trace TraceElement) (value, error) {
	flatArgs := flattenArgs(args, b.Parameters())
	builtinTrace := getBuiltinTrace(trace, b.name)
	x, err := flatArgs[0].getValue(i, trace)
	if err != nil {
		return nil, err
	}
	return b.function(i, builtinTrace, x)
}

func (b *unaryBuiltin) Parameters() Parameters {
	return Parameters{required: b.parameters}
}

func (b *unaryBuiltin) Name() ast.Identifier {
	return b.name
}

type binaryBuiltin struct {
	name       ast.Identifier
	function   binaryBuiltinFunc
	parameters ast.Identifiers
}

// flattenArgs transforms all arguments to a simple array of positional arguments.
// It's needed, because it's possible to use named arguments for required parameters.
// For example both `toString("x")` and `toString(a="x")` are allowed.
// It assumes that we have already checked for duplicates.
func flattenArgs(args callArguments, params Parameters) []*cachedThunk {
	if len(args.named) == 0 {
		return args.positional
	}
	if len(params.optional) != 0 {
		panic("Can't normalize arguments if optional parameters are present")
	}
	needed := make(map[ast.Identifier]int)

	for i := len(args.positional); i < len(params.required); i++ {
		needed[params.required[i]] = i
	}

	flatArgs := make([]*cachedThunk, len(params.required))
	copy(flatArgs, args.positional)
	for _, arg := range args.named {
		flatArgs[needed[arg.name]] = arg.pv
	}
	return flatArgs
}

func (b *binaryBuiltin) EvalCall(args callArguments, i *interpreter, trace TraceElement) (value, error) {
	flatArgs := flattenArgs(args, b.Parameters())
	builtinTrace := getBuiltinTrace(trace, b.name)
	x, err := flatArgs[0].getValue(i, trace)
	if err != nil {
		return nil, err
	}
	y, err := flatArgs[1].getValue(i, trace)
	if err != nil {
		return nil, err
	}
	return b.function(i, builtinTrace, x, y)
}

func (b *binaryBuiltin) Parameters() Parameters {
	return Parameters{required: b.parameters}
}

func (b *binaryBuiltin) Name() ast.Identifier {
	return b.name
}

type ternaryBuiltin struct {
	name       ast.Identifier
	function   ternaryBuiltinFunc
	parameters ast.Identifiers
}

func (b *ternaryBuiltin) EvalCall(args callArguments, i *interpreter, trace TraceElement) (value, error) {
	flatArgs := flattenArgs(args, b.Parameters())
	builtinTrace := getBuiltinTrace(trace, b.name)
	x, err := flatArgs[0].getValue(i, trace)
	if err != nil {
		return nil, err
	}
	y, err := flatArgs[1].getValue(i, trace)
	if err != nil {
		return nil, err
	}
	z, err := flatArgs[2].getValue(i, trace)
	if err != nil {
		return nil, err
	}
	return b.function(i, builtinTrace, x, y, z)
}

func (b *ternaryBuiltin) Parameters() Parameters {
	return Parameters{required: b.parameters}
}

func (b *ternaryBuiltin) Name() ast.Identifier {
	return b.name
}

var desugaredBop = map[ast.BinaryOp]ast.Identifier{
	ast.BopPercent: "mod",
	ast.BopIn:      "objectHasAll",
}

var bopBuiltins = []*binaryBuiltin{
	ast.BopMult: &binaryBuiltin{name: "operator*", function: builtinMult, parameters: ast.Identifiers{"x", "y"}},
	ast.BopDiv:  &binaryBuiltin{name: "operator/", function: builtinDiv, parameters: ast.Identifiers{"x", "y"}},
	// ast.BopPercent:  <desugared>,

	ast.BopPlus:  &binaryBuiltin{name: "operator+", function: builtinPlus, parameters: ast.Identifiers{"x", "y"}},
	ast.BopMinus: &binaryBuiltin{name: "operator-", function: builtinMinus, parameters: ast.Identifiers{"x", "y"}},

	ast.BopShiftL: &binaryBuiltin{name: "operator<<", function: builtinShiftL, parameters: ast.Identifiers{"x", "y"}},
	ast.BopShiftR: &binaryBuiltin{name: "operator>>", function: builtinShiftR, parameters: ast.Identifiers{"x", "y"}},

	ast.BopGreater:   &binaryBuiltin{name: "operator>", function: builtinGreater, parameters: ast.Identifiers{"x", "y"}},
	ast.BopGreaterEq: &binaryBuiltin{name: "operator>=", function: builtinGreaterEq, parameters: ast.Identifiers{"x", "y"}},
	ast.BopLess:      &binaryBuiltin{name: "operator<,", function: builtinLess, parameters: ast.Identifiers{"x", "y"}},
	ast.BopLessEq:    &binaryBuiltin{name: "operator<=", function: builtinLessEq, parameters: ast.Identifiers{"x", "y"}},

	ast.BopManifestEqual:   &binaryBuiltin{name: "operator==", function: builtinEquals, parameters: ast.Identifiers{"x", "y"}},
	ast.BopManifestUnequal: &binaryBuiltin{name: "operator!=", function: builtinNotEquals, parameters: ast.Identifiers{"x", "y"}}, // Special case

	ast.BopBitwiseAnd: &binaryBuiltin{name: "operator&", function: builtinBitwiseAnd, parameters: ast.Identifiers{"x", "y"}},
	ast.BopBitwiseXor: &binaryBuiltin{name: "operator^", function: builtinBitwiseXor, parameters: ast.Identifiers{"x", "y"}},
	ast.BopBitwiseOr:  &binaryBuiltin{name: "operator|", function: builtinBitwiseOr, parameters: ast.Identifiers{"x", "y"}},
}

var uopBuiltins = []*unaryBuiltin{
	ast.UopNot:        &unaryBuiltin{name: "operator!", function: builtinNegation, parameters: ast.Identifiers{"x"}},
	ast.UopBitwiseNot: &unaryBuiltin{name: "operator~", function: builtinBitNeg, parameters: ast.Identifiers{"x"}},
	ast.UopPlus:       &unaryBuiltin{name: "operator+ (unary)", function: builtinIdentity, parameters: ast.Identifiers{"x"}},
	ast.UopMinus:      &unaryBuiltin{name: "operator- (unary)", function: builtinUnaryMinus, parameters: ast.Identifiers{"x"}},
}

type builtin interface {
	evalCallable
	Name() ast.Identifier
}

func buildBuiltinMap(builtins []builtin) map[string]evalCallable {
	result := make(map[string]evalCallable)
	for _, b := range builtins {
		result[string(b.Name())] = b
	}
	return result
}

var funcBuiltins = buildBuiltinMap([]builtin{
	&unaryBuiltin{name: "extVar", function: builtinExtVar, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "length", function: builtinLength, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "toString", function: builtinToString, parameters: ast.Identifiers{"a"}},
	&binaryBuiltin{name: "trace", function: builtinTrace, parameters: ast.Identifiers{"str", "rest"}},
	&binaryBuiltin{name: "makeArray", function: builtinMakeArray, parameters: ast.Identifiers{"sz", "func"}},
	&binaryBuiltin{name: "flatMap", function: builtinFlatMap, parameters: ast.Identifiers{"func", "arr"}},
	&binaryBuiltin{name: "join", function: builtinJoin, parameters: ast.Identifiers{"sep", "arr"}},
	&binaryBuiltin{name: "filter", function: builtinFilter, parameters: ast.Identifiers{"func", "arr"}},
	&binaryBuiltin{name: "range", function: builtinRange, parameters: ast.Identifiers{"from", "to"}},
	&binaryBuiltin{name: "primitiveEquals", function: primitiveEquals, parameters: ast.Identifiers{"x", "y"}},
	&binaryBuiltin{name: "equals", function: builtinEquals, parameters: ast.Identifiers{"x", "y"}},
	&binaryBuiltin{name: "objectFieldsEx", function: builtinObjectFieldsEx, parameters: ast.Identifiers{"obj", "hidden"}},
	&ternaryBuiltin{name: "objectHasEx", function: builtinObjectHasEx, parameters: ast.Identifiers{"obj", "fname", "hidden"}},
	&unaryBuiltin{name: "type", function: builtinType, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "char", function: builtinChar, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "codepoint", function: builtinCodepoint, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "ceil", function: builtinCeil, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "floor", function: builtinFloor, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "sqrt", function: builtinSqrt, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "sin", function: builtinSin, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "cos", function: builtinCos, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "tan", function: builtinTan, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "asin", function: builtinAsin, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "acos", function: builtinAcos, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "atan", function: builtinAtan, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "log", function: builtinLog, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "exp", function: builtinExp, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "mantissa", function: builtinMantissa, parameters: ast.Identifiers{"x"}},
	&unaryBuiltin{name: "exponent", function: builtinExponent, parameters: ast.Identifiers{"x"}},
	&binaryBuiltin{name: "pow", function: builtinPow, parameters: ast.Identifiers{"base", "exp"}},
	&binaryBuiltin{name: "modulo", function: builtinModulo, parameters: ast.Identifiers{"x", "y"}},
	&unaryBuiltin{name: "md5", function: builtinMd5, parameters: ast.Identifiers{"x"}},
	&ternaryBuiltin{name: "strReplace", function: builtinStrReplace, parameters: ast.Identifiers{"str", "from", "to"}},
	&unaryBuiltin{name: "native", function: builtinNative, parameters: ast.Identifiers{"x"}},

	// internal
	&unaryBuiltin{name: "$objectFlatMerge", function: builtinUglyObjectFlatMerge, parameters: ast.Identifiers{"x"}},
})
