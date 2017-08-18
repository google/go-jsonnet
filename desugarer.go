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
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/google/go-jsonnet/ast"
)

func makeStr(s string) *ast.LiteralString {
	return &ast.LiteralString{ast.NodeBase{}, s, ast.StringDouble, ""}
}

func stringUnescape(loc *ast.LocationRange, s string) (string, error) {
	var buf bytes.Buffer
	// read one rune at a time
	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])
		i += w
		switch r {
		case '\\':
			if i >= len(s) {
				return "", makeStaticError("Truncated escape sequence in string literal.", *loc)
			}
			r2, w := utf8.DecodeRuneInString(s[i:])
			i += w
			switch r2 {
			case '"':
				buf.WriteRune('"')
			case '\'':
				buf.WriteRune('\'')
			case '\\':
				buf.WriteRune('\\')
			case '/':
				buf.WriteRune('/') // This one is odd, maybe a mistake.
			case 'b':
				buf.WriteRune('\b')
			case 'f':
				buf.WriteRune('\f')
			case 'n':
				buf.WriteRune('\n')
			case 'r':
				buf.WriteRune('\r')
			case 't':
				buf.WriteRune('\t')
			case 'u':
				if i+4 > len(s) {
					return "", makeStaticError("Truncated unicode escape sequence in string literal.", *loc)
				}
				codeBytes, err := hex.DecodeString(s[i : i+4])
				if err != nil {
					return "", makeStaticError(fmt.Sprintf("Unicode escape sequence was malformed: %s", s[0:4]), *loc)
				}
				code := int(codeBytes[0])*256 + int(codeBytes[1])
				buf.WriteRune(rune(code))
				i += 4
			default:
				return "", makeStaticError(fmt.Sprintf("Unknown escape sequence in string literal: \\%c", r2), *loc)
			}

		default:
			buf.WriteRune(r)
		}
	}
	return buf.String(), nil
}

func desugarFields(location ast.LocationRange, fields *ast.ObjectFields, objLevel int) error {

	// Desugar children
	for i := range *fields {
		field := &((*fields)[i])
		if field.Expr1 != nil {
			err := desugar(&field.Expr1, objLevel)
			if err != nil {
				return err
			}
		}
		err := desugar(&field.Expr2, objLevel+1)
		if err != nil {
			return err
		}
		if field.Expr3 != nil {
			err := desugar(&field.Expr3, objLevel+1)
			if err != nil {
				return err
			}
		}
	}

	// Simplify asserts
	// TODO(dcunnin): this
	for _, field := range *fields {
		if field.Kind != ast.ObjectAssert {
			continue
		}
		/*
			AST *msg = field.expr3
			field.expr3 = nil
			if (msg == nil) {
				auto msg_str = U"Object assertion failed."
				msg = alloc->make<ast.LiteralString>(field.expr2->location, msg_str,
												 ast.LiteralString::DOUBLE, "")
			}

			// if expr2 then true else error msg
			field.expr2 = alloc->make<ast.Conditional>(
				ast->location,
				field.expr2,
				alloc->make<ast.LiteralBoolean>(E, true),
				alloc->make<Error>(msg->location, msg))
		*/
	}

	// Remove methods
	// TODO(dcunnin): this
	for i := range *fields {
		field := &((*fields)[i])
		if !field.MethodSugar {
			continue
		}
		origBody := field.Expr2
		function := &ast.Function{
			// TODO(sbarzowski) better location
			NodeBase:   ast.NewNodeBaseLoc(*origBody.Loc()),
			Parameters: field.Ids,
			Body:       origBody,
		}
		field.MethodSugar = false
		field.Ids = nil
		field.Expr2 = function
	}

	// Remove object-level locals
	newFields := []ast.ObjectField{}
	var binds ast.LocalBinds
	for _, local := range *fields {
		if local.Kind != ast.ObjectLocal {
			continue
		}
		binds = append(binds, ast.LocalBind{Variable: *local.Id, Body: local.Expr2})
	}
	for _, field := range *fields {
		if field.Kind == ast.ObjectLocal {
			continue
		}
		if len(binds) > 0 {
			field.Expr2 = &ast.Local{ast.NewNodeBaseLoc(*field.Expr2.Loc()), binds, field.Expr2}
		}
		newFields = append(newFields, field)
	}
	*fields = newFields

	// Change all to FIELD_EXPR
	for i := range *fields {
		field := &(*fields)[i]
		switch field.Kind {
		case ast.ObjectAssert:
		// Nothing to do.

		case ast.ObjectFieldID:
			field.Expr1 = makeStr(string(*field.Id))
			field.Kind = ast.ObjectFieldExpr

		case ast.ObjectFieldExpr:
		// Nothing to do.

		case ast.ObjectFieldStr:
			// Just set the flag.
			field.Kind = ast.ObjectFieldExpr

		case ast.ObjectLocal:
			return fmt.Errorf("INTERNAL ERROR: Locals should be removed by now")
		}
	}

	// Remove +:
	// TODO(dcunnin): this
	for _, field := range *fields {
		if !field.SuperSugar {
			continue
		}
		/*
			AST *super_f = alloc->make<SuperIndex>(field.expr1->location, field.expr1, nil)
			field.expr2 = alloc->make<ast.Binary>(ast->location, super_f, BOP_PLUS, field.expr2)
			field.superSugar = false
		*/
	}

	return nil
}

func desugarArrayComp(astComp *ast.ArrayComp, objLevel int) (ast.Node, error) {
	return &ast.LiteralNull{}, nil
	// TODO(sbarzowski) this
	switch astComp.Specs[0].Kind {
	case ast.CompFor:
		panic("TODO")
	case ast.CompIf:
		panic("TODO")
	default:
		panic("TODO")
	}
}

func desugarObjectComp(astComp *ast.ObjectComp, objLevel int) (ast.Node, error) {
	return &ast.LiteralNull{}, nil
	// TODO(sbarzowski) this
}

func buildSimpleIndex(obj ast.Node, member ast.Identifier) ast.Node {
	return &ast.Index{
		Target: obj,
		Id:     &member,
	}
}

func buildStdCall(builtinName ast.Identifier, args ...ast.Node) ast.Node {
	std := &ast.Var{Id: "std"}
	builtin := buildSimpleIndex(std, builtinName)
	return &ast.Apply{
		Target:    builtin,
		Arguments: args,
	}
}

// Desugar Jsonnet expressions to reduce the number of constructs the rest of the implementation
// needs to understand.

// Desugaring should happen immediately after parsing, i.e. before static analysis and execution.
// Temporary variables introduced here should be prefixed with $ to ensure they do not clash with
// variables used in user code.
// TODO(sbarzowski) Actually we may want to do some static analysis before desugaring, e.g.
// warning user about dangerous use of constructs that we desugar.
func desugar(astPtr *ast.Node, objLevel int) (err error) {
	node := *astPtr

	if node == nil {
		return
	}

	switch node := node.(type) {
	case *ast.Apply:
		desugar(&node.Target, objLevel)
		for i := range node.Arguments {
			err = desugar(&node.Arguments[i], objLevel)
			if err != nil {
				return
			}
		}

	case *ast.ApplyBrace:
		err = desugar(&node.Left, objLevel)
		if err != nil {
			return
		}
		err = desugar(&node.Right, objLevel)
		if err != nil {
			return
		}
		*astPtr = &ast.Binary{
			NodeBase: node.NodeBase,
			Left:     node.Left,
			Op:       ast.BopPlus,
			Right:    node.Right,
		}

	case *ast.Array:
		for i := range node.Elements {
			err = desugar(&node.Elements[i], objLevel)
			if err != nil {
				return
			}
		}

	case *ast.ArrayComp:
		comp, err := desugarArrayComp(node, objLevel)
		if err != nil {
			return err
		}
		*astPtr = comp

	case *ast.Assert:
		// TODO(sbarzowski) this
		*astPtr = &ast.LiteralNull{}

	case *ast.Binary:
		// some operators get replaced by stdlib functions
		if funcname, replaced := desugaredBop[node.Op]; replaced {
			if funcname == "notEquals" {
				// TODO(sbarzowski) maybe we can handle it in more regular way
				// but let's be consistent with the spec
				*astPtr = &ast.Unary{
					Op:   ast.UopNot,
					Expr: buildStdCall(desugaredBop[ast.BopManifestEqual], node.Left, node.Right),
				}
			} else {
				*astPtr = buildStdCall(funcname, node.Left, node.Right)
			}
			return desugar(astPtr, objLevel)
		}

		err = desugar(&node.Left, objLevel)
		if err != nil {
			return
		}
		err = desugar(&node.Right, objLevel)
		if err != nil {
			return
		}
		// TODO(dcunnin): Need to handle bopPercent, bopManifestUnequal, bopManifestEqual

	case *ast.Conditional:
		err = desugar(&node.Cond, objLevel)
		if err != nil {
			return
		}
		err = desugar(&node.BranchTrue, objLevel)
		if err != nil {
			return
		}
		if node.BranchFalse == nil {
			node.BranchFalse = &ast.LiteralNull{}
		}
		err = desugar(&node.BranchFalse, objLevel)
		if err != nil {
			return
		}

	case *ast.Dollar:
		if objLevel == 0 {
			return makeStaticError("No top-level object found.", *node.Loc())
		}
		*astPtr = &ast.Var{NodeBase: node.NodeBase, Id: ast.Identifier("$")}

	case *ast.Error:
		err = desugar(&node.Expr, objLevel)
		if err != nil {
			return
		}

	case *ast.Function:
		err = desugar(&node.Body, objLevel)
		if err != nil {
			return
		}

	case *ast.Import:
		// Nothing to do.

	case *ast.ImportStr:
		// Nothing to do.

	case *ast.Index:
		err = desugar(&node.Target, objLevel)
		if err != nil {
			return
		}
		if node.Id != nil {
			if node.Index != nil {
				panic("TODO")
			}
			node.Index = makeStr(string(*node.Id))
			node.Id = nil
		}
		err = desugar(&node.Index, objLevel)
		if err != nil {
			return
		}

	case *ast.Slice:
		if node.BeginIndex == nil {
			node.BeginIndex = &ast.LiteralNull{}
		}
		if node.EndIndex == nil {
			node.EndIndex = &ast.LiteralNull{}
		}
		if node.Step == nil {
			node.Step = &ast.LiteralNull{}
		}
		*astPtr = buildStdCall("std.slice", node.Target, node.BeginIndex, node.EndIndex, node.Step)
		desugar(astPtr, objLevel)

	case *ast.Local:
		for i := range node.Binds {
			if node.Binds[i].FunctionSugar {
				origBody := node.Binds[i].Body
				function := &ast.Function{
					// TODO(sbarzowski) better location
					NodeBase:   ast.NewNodeBaseLoc(*origBody.Loc()),
					Parameters: node.Binds[i].Params,
					Body:       origBody,
				}
				node.Binds[i] = ast.LocalBind{
					Variable:      node.Binds[i].Variable,
					Body:          function,
					FunctionSugar: false,
					Params:        nil,
				}
			}
			err = desugar(&node.Binds[i].Body, objLevel)
			if err != nil {
				return
			}
		}
		err = desugar(&node.Body, objLevel)
		if err != nil {
			return
		}

	case *ast.LiteralBoolean:
		// Nothing to do.

	case *ast.LiteralNull:
		// Nothing to do.

	case *ast.LiteralNumber:
		// Nothing to do.

	case *ast.LiteralString:
		if node.Kind != ast.VerbatimStringDouble && node.Kind != ast.VerbatimStringSingle {
			unescaped, err := stringUnescape(node.Loc(), node.Value)
			if err != nil {
				return err
			}
			node.Value = unescaped
			node.Kind = ast.StringDouble
			node.BlockIndent = ""
		}
	case *ast.Object:
		// Hidden variable to allow $ binding.
		if objLevel == 0 {
			dollar := ast.Identifier("$")
			node.Fields = append(node.Fields, ast.ObjectFieldLocalNoMethod(&dollar, &ast.Self{}))
		}

		err = desugarFields(*node.Loc(), &node.Fields, objLevel)
		if err != nil {
			return
		}

		var newFields ast.DesugaredObjectFields
		var newAsserts ast.Nodes

		for _, field := range node.Fields {
			if field.Kind == ast.ObjectAssert {
				newAsserts = append(newAsserts, field.Expr2)
			} else if field.Kind == ast.ObjectFieldExpr {
				newFields = append(newFields, ast.DesugaredObjectField{field.Hide, field.Expr1, field.Expr2})
			} else {
				panic(fmt.Sprintf("INTERNAL ERROR: field should have been desugared: %s", field.Kind))
			}
		}

		*astPtr = &ast.DesugaredObject{node.NodeBase, newAsserts, newFields}

	case *ast.DesugaredObject:
		panic("Desugaring desugared object")

	case *ast.ObjectComp:
		comp, err := desugarObjectComp(node, objLevel)
		if err != nil {
			return err
		}
		*astPtr = comp

	case *ast.ObjectComprehensionSimple:
		panic("Desugaring desugared object comprehension")

	case *ast.Self:
		// Nothing to do.

	case *ast.SuperIndex:
		if node.Id != nil {
			node.Index = &ast.LiteralString{Value: string(*node.Id)}
			node.Id = nil
		}

	case *ast.Unary:
		err = desugar(&node.Expr, objLevel)
		if err != nil {
			return
		}

	case *ast.Var:
		// Nothing to do.

	default:
		panic(fmt.Sprintf("Desugarer does not recognize ast: %s", reflect.TypeOf(node)))
	}

	return nil
}

func desugarFile(ast *ast.Node) error {
	err := desugar(ast, 0)
	if err != nil {
		return err
	}
	// TODO(dcunnin): wrap in std local
	return nil
}
