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

func (i *identifierSet) append(idents identifiers) {
	for _, ident := range idents {
		i.Add(ident)
	}
}

type analysisState struct {
	err      error
	freeVars identifierSet
}

func visitNext(a astNode, inObject bool, vars identifierSet, state *analysisState) {
	if state.err != nil {
		return
	}
	state.err = analyzeVisit(a, inObject, vars)
	state.freeVars.append(a.FreeVariables())
}

// TODO(dcunnin): Check for invalid use of self, super, and bound variables.
// TODO(dcunnin): Compute free variables at each AST.
func analyzeVisit(a astNode, inObject bool, vars identifierSet) error {
	s := &analysisState{freeVars: NewidentifierSet()}

	// TODO(sbarzowski) Test somehow that we're visiting all the nodes
	switch ast := a.(type) {
	case *astApply:
		visitNext(ast.target, inObject, vars, s)
	case *astArray:
		for _, elem := range ast.elements {
			visitNext(elem, inObject, vars, s)
		}
	case *astBinary:
		visitNext(ast.left, inObject, vars, s)
		visitNext(ast.right, inObject, vars, s)
	case *astBuiltin:
		// nothing to do here
	case *astConditional:
		visitNext(ast.cond, inObject, vars, s)
		visitNext(ast.branchTrue, inObject, vars, s)
		visitNext(ast.branchFalse, inObject, vars, s)
	case *astError:
		visitNext(ast.expr, inObject, vars, s)
	case *astFunction:
		// TODO(sbarzowski) check duplicate function parameters
		// or maybe somewhere else as it doesn't require any context
		visitNext(ast.body, inObject, vars, s)
		// TODO(sbarzowski) when we have default values of params check them
	case *astImport:
		//nothing to do here
	case *astImportStr:
		//nothing to do here
	case *astSuperIndex:
		if !inObject {
			return makeStaticError("Can't use super outside of an object.", ast.loc)
		}
		visitNext(ast.index, inObject, vars, s)
	case *astIndex:
		visitNext(ast.target, inObject, vars, s)
		visitNext(ast.index, inObject, vars, s)
	case *astLocal:
		newVars := vars.Clone()
		for _, bind := range ast.binds {
			newVars.Add(bind.variable)
		}
		// Binds in local can be mutually or even self recursive (TODO(sbarzowski) confirm that)
		for _, bind := range ast.binds {
			visitNext(bind.body, inObject, newVars, s)
		}
		visitNext(ast.body, inObject, newVars, s)

		// Any usage of newly created variables inside are considered free
		// but they are not here or outside
		for _, bind := range ast.binds {
			s.freeVars.Remove(bind.variable)
		}
	case *astLiteralBoolean:
		//nothing to do here
	case *astLiteralNull:
		//nothing to do here
	case *astLiteralNumber:
		//nothing to do here
	case *astLiteralString:
		//nothing to do here
	case *astDesugaredObject:
		for _, field := range ast.fields {
			// Field names are calculated *outside* of the object
			visitNext(field.name, inObject, vars, s)
			visitNext(field.body, true, vars, s)
		}
		for _, assert := range ast.asserts {
			visitNext(assert, true, vars, s)
		}
	case *astObjectComprehensionSimple:
		// TODO (sbarzowski) this
		panic("Comprehensions not supported yet")
	case *astSelf:
		if !inObject {
			return makeStaticError("Can't use self outside of an object.", ast.loc)
		}
	case *astUnary:
		visitNext(ast.expr, inObject, vars, s)
	case *astVar:
		if !vars.Contains(ast.id) {
			return makeStaticError(fmt.Sprintf("Unknown variable: %v", ast.id), ast.loc)
		}
		s.freeVars.Add(ast.id)
	default:
		panic(fmt.Sprintf("Unexpected node %#v", a))
	}
	a.setFreeVariables(s.freeVars.ToSlice())

	return nil
}

func analyze(ast astNode) error {
	return analyzeVisit(ast, false, NewidentifierSet())
}
