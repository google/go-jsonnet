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

import "testing"

// func dummyNodeBase() astNodeBase {
// 	return astNode
// }

func TestSimpleNull(t *testing.T) {
	ast := &astLiteralNull{}
	err := analyze(ast)
	if err != nil {
		t.Errorf("Unexpected error: %+v", err)
	}
	if ast.FreeVariables() != nil {
		t.Errorf("Unexpected free variabled %+v", ast.FreeVariables())
	}
}

func hasTheseFreeVars(returned identifiers, expected identifiers) bool {
	if len(returned) != len(expected) {
		return false
	}
	for i := range expected {
		if returned[i] != expected[i] {
			return false
		}
	}
	return true
}

func TestSimpleLocal(t *testing.T) {
	ast := &astLocal{
		binds: astLocalBinds{
			astLocalBind{
				variable: "x",
				body:     &astLiteralNull{},
			},
		},
		body: &astVar{id: "x"},
	}

	err := analyze(ast)
	if err != nil {
		t.Errorf("Unexpected error: %+v", err)
	}
	if ast.FreeVariables() != nil {
		t.Errorf("Unexpected free variables %+v in root local. Expected none.", ast.FreeVariables())
	}
	returned := ast.body.FreeVariables()
	expectedVars := identifiers{"x"}
	if !hasTheseFreeVars(returned, expectedVars) {
		t.Errorf("Unexpected free variables %+v in local body. Expected %+v.", returned, expectedVars)
	}
}
