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
	"testing"
)

// Just a few simple sanity tests for now.  Eventually we'll share end-to-end tests with the C++
// implementation but unsure if that should be done here or via some external framework.

type mainTest struct {
	name      string
	input     string
	golden    string
	errString string
}

var mainTests = []mainTest{
	{"numeric_literal", "100", "100", ""},
	{"boolean_literal", "true", "true", ""},
	{"simple_arith1", "3 + 3", "6", ""},
	{"simple_arith2", "3 + 3 + 3", "9", ""},
	{"simple_arith3", "(3 + 3) + (3 + 3)", "12", ""},
	{"simple_arith_string", "\"aaa\" + \"bbb\"", "\"aaabbb\"", ""},
	{"simple_arith_string2", "\"aaa\" + \"\"", "\"aaa\"", ""},
	{"simple_arith_string3", "\"\" + \"bbb\"", "\"bbb\"", ""},
	{"simple_arith_string_empty", "\"\" + \"\"", "\"\"", ""},
	{"empty_array", "[]", "[ ]", ""},
	{"array", "[1, 2, 1 + 2]", "[\n   1,\n   2,\n   3\n]", ""},
	{"empty_object", "{}", "{ }", ""},
	{"object", `{"x": 1+1}`, "{\n   \"x\": 2\n}", ""},

	{"use_object", `{a: 1}.a`, "1", ""},
	{"use_object_in_object", `{a: {a: 1}.a, b: {b: 1}.b}.a`, "1", ""},
	{"variable", `local x = 2; x`, "2", ""},
	{"variable_not_visible", "local x1 = local nested = 42; nested, x2 = nested; x2", "", "Unknown variable: nested"},
	{"array_index1", `[1][0]`, "1", ""},
	{"array_index2", `[1, 2, 3][0]`, "1", ""},
	{"array_index3", `[1, 2, 3][1]`, "2", ""},
	{"array_index4", `[1, 2, 3][2]`, "3", ""},
	{"function", `function() 42`, "", "Couldn't manifest function in JSON output."},
	{"function_call", `(function() 42)()`, "42", ""},
	{"function_with_argument", `(function(x) x)(42)`, "42", ""},
	{"function_capturing", `local y = 17; (function(x) y)(42)`, "17", ""},
	{"error", `error "42"`, "", "Error: 42"},
	{"filled_thunk", "local x = [1, 2, 3]; x[1] + x[1]", "4", ""},
	{"lazy", `local x = {'x': error "blah"}; x.x`, "", "Error: blah"},
	{"lazy", `local x = {'x': error "blah"}, f = function(x) 42, z = x.x; f(x.x)`, "42", ""},
	{"lazy_operator1", `false && error "shouldn't happen"`, "false", ""},
	{"lazy_operator2", `true && error "should happen"`, "", "Error: should happen"},

	// TODO(sbarzowski) - array comprehension
	// {"array_comp", `[x for x in [1, 2, 3]]`, "[1, 2, 3]", ""},
}

func TestMain(t *testing.T) {
	for _, test := range mainTests {
		vm := MakeVM()
		output, err := vm.EvaluateSnippet(test.name, test.input)
		var errString string
		if err != nil {
			errString = err.Error()
		}
		if errString != test.errString {
			t.Errorf("%s: error result does not match. got\n\t%+v\nexpected\n\t%+v",
				test.input, errString, test.errString)
		}
		if err == nil && output != test.golden {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%+v", test.name, output, test.golden)
		}
	}
}
