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

type builtinPlus struct {
}

func (b *builtinPlus) Binary(x, y value, i *interpreter, loc *LocationRange) (value, error) {
	// TODO(sbarzowski) More types and more graceful error handling
	switch leftVal := x.(type) {
	case *valueNumber:
		left := leftVal.value
		right := y.(*valueNumber).value
		return makeValueNumber(left + right), nil
	case *valueString:
		left := leftVal.value
		right := y.(*valueString).value
		return makeValueString(left + right), nil
	default:
		panic(fmt.Sprintf("INTERNAL ERROR: Unrecognised value type: %T", leftVal))
	}
}

type binaryBuiltin interface {
	Binary(x, y value, i *interpreter, loc *LocationRange) (value, error)
}

var bopBuiltins = []binaryBuiltin{
	// bopMult:    "*",
	// bopDiv:     "/",
	// bopPercent: "%",

	bopPlus: &builtinPlus{},
	// bopMinus: "-",

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

	// bopAnd: "&&",
	// bopOr:  "||",
}
