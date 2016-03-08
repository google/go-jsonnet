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

// TODO(dcunnin): Check for invalid use of self, super, and bound variables.
// TODO(dcunnin): Compute free variables at each AST.
func analyzeVisit(ast astNode, inObject bool, vars identifierSet) (identifierSet, error) {
	var r identifierSet
	return r, nil
}

func analyze(ast astNode) error {
	_, err := analyzeVisit(ast, false, NewidentifierSet())
	return err
}
