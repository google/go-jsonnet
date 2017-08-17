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
	"fmt"
)

func (i *IdentifierSet) Append(idents Identifiers) {
	for _, ident := range idents {
		i.Add(ident)
	}
}

type analysisState struct {
	err      error
	freeVars IdentifierSet
}

func visitNext(a Node, inObject bool, vars IdentifierSet, state *analysisState) {
	if state.err != nil {
		return
	}
	state.err = analyzeVisit(a, inObject, vars)
	state.freeVars.Append(a.FreeVariables())
}

func analyzeVisit(a Node, inObject bool, vars IdentifierSet) error {
	s := &analysisState{freeVars: NewIdentifierSet()}

	// TODO(sbarzowski) Test somehow that we're visiting all the nodes
	switch ast := a.(type) {
	case *Apply:
		visitNext(ast.Target, inObject, vars, s)
		for _, arg := range ast.Arguments {
			visitNext(arg, inObject, vars, s)
		}
	case *Array:
		for _, elem := range ast.Elements {
			visitNext(elem, inObject, vars, s)
		}
	case *Binary:
		visitNext(ast.Left, inObject, vars, s)
		visitNext(ast.Right, inObject, vars, s)
	case *Conditional:
		visitNext(ast.Cond, inObject, vars, s)
		visitNext(ast.BranchTrue, inObject, vars, s)
		visitNext(ast.BranchFalse, inObject, vars, s)
	case *Error:
		visitNext(ast.Expr, inObject, vars, s)
	case *Function:
		// TODO(sbarzowski) check duplicate function parameters
		// or maybe somewhere else as it doesn't require any context
		newVars := vars.Clone()
		for _, param := range ast.Parameters {
			newVars.Add(param)
		}
		visitNext(ast.Body, inObject, newVars, s)
		// Parameters are free inside the body, but not visible here or outside
		for _, param := range ast.Parameters {
			s.freeVars.Remove(param)
		}
		// TODO(sbarzowski) when we have default values of params check them
	case *Import:
		//nothing to do here
	case *ImportStr:
		//nothing to do here
	case *SuperIndex:
		if !inObject {
			return makeStaticError("Can't use super outside of an object.", ast.loc)
		}
		visitNext(ast.Index, inObject, vars, s)
	case *Index:
		visitNext(ast.Target, inObject, vars, s)
		visitNext(ast.Index, inObject, vars, s)
	case *Local:
		newVars := vars.Clone()
		for _, bind := range ast.Binds {
			newVars.Add(bind.Variable)
		}
		// Binds in local can be mutually or even self recursive
		for _, bind := range ast.Binds {
			visitNext(bind.Body, inObject, newVars, s)
		}
		visitNext(ast.Body, inObject, newVars, s)

		// Any usage of newly created variables inside are considered free
		// but they are not here or outside
		for _, bind := range ast.Binds {
			s.freeVars.Remove(bind.Variable)
		}
	case *LiteralBoolean:
		//nothing to do here
	case *LiteralNull:
		//nothing to do here
	case *LiteralNumber:
		//nothing to do here
	case *LiteralString:
		//nothing to do here
	case *DesugaredObject:
		for _, field := range ast.Fields {
			// Field names are calculated *outside* of the object
			visitNext(field.Name, inObject, vars, s)
			visitNext(field.Body, true, vars, s)
		}
		for _, assert := range ast.Asserts {
			visitNext(assert, true, vars, s)
		}
	case *ObjectComprehensionSimple:
		// TODO (sbarzowski) this
		panic("Comprehensions not supported yet")
	case *Self:
		if !inObject {
			return makeStaticError("Can't use self outside of an object.", ast.loc)
		}
	case *Unary:
		visitNext(ast.Expr, inObject, vars, s)
	case *Var:
		if !vars.Contains(ast.Id) {
			return makeStaticError(fmt.Sprintf("Unknown variable: %v", ast.Id), ast.loc)
		}
		s.freeVars.Add(ast.Id)
	default:
		panic(fmt.Sprintf("Unexpected node %#v", a))
	}
	a.setFreeVariables(s.freeVars.ToSlice())
	return s.err
}

func analyze(ast Node) error {
	return analyzeVisit(ast, false, NewIdentifierSet("std"))
}
