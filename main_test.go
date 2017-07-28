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
	{"empty_array", "[]", "[ ]", ""},
	{"array", "[1, 2, 1 + 2]", "[\n   1,\n   2,\n   3\n]", ""},
	{"empty_object", "{}", "{ }", ""},
	{"object", `{"x": 1+1}`, "{\n   \"x\": 2\n}", ""},
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
