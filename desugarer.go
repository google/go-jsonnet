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
)

func makeStr(s string) *LiteralString {
	return &LiteralString{nodeBase{loc: LocationRange{}}, s, StringDouble, ""}
}

func stringUnescape(loc *LocationRange, s string) (string, error) {
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

func desugarFields(location LocationRange, fields *ObjectFields, objLevel int) error {

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
		if field.Kind != ObjectAssert {
			continue
		}
		/*
			AST *msg = field.expr3
			field.expr3 = nil
			if (msg == nil) {
				auto msg_str = U"Object assertion failed."
				msg = alloc->make<LiteralString>(field.expr2->location, msg_str,
												 LiteralString::DOUBLE, "")
			}

			// if expr2 then true else error msg
			field.expr2 = alloc->make<Conditional>(
				ast->location,
				field.expr2,
				alloc->make<LiteralBoolean>(E, true),
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
		function := &Function{
			// TODO(sbarzowski) better location
			nodeBase:   nodeBase{loc: *origBody.Loc()},
			Parameters: field.Ids,
			Body:       origBody,
		}
		field.MethodSugar = false
		field.Ids = nil
		field.Expr2 = function
	}

	// Remove object-level locals
	newFields := []ObjectField{}
	var binds LocalBinds
	for _, local := range *fields {
		if local.Kind != ObjectLocal {
			continue
		}
		binds = append(binds, LocalBind{Variable: *local.Id, Body: local.Expr2})
	}
	for _, field := range *fields {
		if field.Kind == ObjectLocal {
			continue
		}
		if len(binds) > 0 {
			field.Expr2 = &Local{nodeBase{loc: *field.Expr2.Loc()}, binds, field.Expr2}
		}
		newFields = append(newFields, field)
	}
	*fields = newFields

	// Change all to FIELD_EXPR
	for i := range *fields {
		field := &(*fields)[i]
		switch field.Kind {
		case ObjectAssert:
		// Nothing to do.

		case ObjectFieldID:
			field.Expr1 = makeStr(string(*field.Id))
			field.Kind = ObjectFieldExpr

		case ObjectFieldExpr:
		// Nothing to do.

		case ObjectFieldStr:
			// Just set the flag.
			field.Kind = ObjectFieldExpr

		case ObjectLocal:
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
			field.expr2 = alloc->make<Binary>(ast->location, super_f, BOP_PLUS, field.expr2)
			field.superSugar = false
		*/
	}

	return nil
}

func desugarArrayComp(astComp *ArrayComp, objLevel int) (Node, error) {
	return &LiteralNull{}, nil
	// TODO(sbarzowski) this
	switch astComp.Specs[0].Kind {
	case CompFor:
		panic("TODO")
	case CompIf:
		panic("TODO")
	default:
		panic("TODO")
	}
}

func desugarObjectComp(astComp *ObjectComp, objLevel int) (Node, error) {
	return &LiteralNull{}, nil
	// TODO(sbarzowski) this
}

func buildSimpleIndex(obj Node, member Identifier) Node {
	return &Index{
		Target: obj,
		Id:     &member,
	}
}

func buildStdCall(builtinName Identifier, args ...Node) Node {
	std := &Var{Id: "std"}
	builtin := buildSimpleIndex(std, builtinName)
	return &Apply{
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
func desugar(astPtr *Node, objLevel int) (err error) {
	ast := *astPtr

	if ast == nil {
		return
	}

	switch ast := ast.(type) {
	case *Apply:
		desugar(&ast.Target, objLevel)
		for i := range ast.Arguments {
			err = desugar(&ast.Arguments[i], objLevel)
			if err != nil {
				return
			}
		}

	case *ApplyBrace:
		err = desugar(&ast.Left, objLevel)
		if err != nil {
			return
		}
		err = desugar(&ast.Right, objLevel)
		if err != nil {
			return
		}
		*astPtr = &Binary{
			nodeBase: ast.nodeBase,
			Left:     ast.Left,
			Op:       BopPlus,
			Right:    ast.Right,
		}

	case *Array:
		for i := range ast.Elements {
			err = desugar(&ast.Elements[i], objLevel)
			if err != nil {
				return
			}
		}

	case *ArrayComp:
		comp, err := desugarArrayComp(ast, objLevel)
		if err != nil {
			return err
		}
		*astPtr = comp

	case *Assert:
		// TODO(sbarzowski) this
		*astPtr = &LiteralNull{}

	case *Binary:
		// some operators get replaced by stdlib functions
		if funcname, replaced := desugaredBop[ast.Op]; replaced {
			if funcname == "notEquals" {
				// TODO(sbarzowski) maybe we can handle it in more regular way
				// but let's be consistent with the spec
				*astPtr = &Unary{
					Op:   UopNot,
					Expr: buildStdCall(desugaredBop[BopManifestEqual], ast.Left, ast.Right),
				}
			} else {
				*astPtr = buildStdCall(funcname, ast.Left, ast.Right)
			}
			return desugar(astPtr, objLevel)
		}

		err = desugar(&ast.Left, objLevel)
		if err != nil {
			return
		}
		err = desugar(&ast.Right, objLevel)
		if err != nil {
			return
		}
		// TODO(dcunnin): Need to handle bopPercent, bopManifestUnequal, bopManifestEqual

	case *Conditional:
		err = desugar(&ast.Cond, objLevel)
		if err != nil {
			return
		}
		err = desugar(&ast.BranchTrue, objLevel)
		if err != nil {
			return
		}
		if ast.BranchFalse == nil {
			ast.BranchFalse = &LiteralNull{}
		}
		err = desugar(&ast.BranchFalse, objLevel)
		if err != nil {
			return
		}

	case *Dollar:
		if objLevel == 0 {
			return makeStaticError("No top-level object found.", *ast.Loc())
		}
		*astPtr = &Var{nodeBase: ast.nodeBase, Id: Identifier("$")}

	case *Error:
		err = desugar(&ast.Expr, objLevel)
		if err != nil {
			return
		}

	case *Function:
		err = desugar(&ast.Body, objLevel)
		if err != nil {
			return
		}

	case *Import:
		// Nothing to do.

	case *ImportStr:
		// Nothing to do.

	case *Index:
		err = desugar(&ast.Target, objLevel)
		if err != nil {
			return
		}
		if ast.Id != nil {
			if ast.Index != nil {
				panic("TODO")
			}
			ast.Index = makeStr(string(*ast.Id))
			ast.Id = nil
		}
		err = desugar(&ast.Index, objLevel)
		if err != nil {
			return
		}

	case *Slice:
		if ast.BeginIndex == nil {
			ast.BeginIndex = &LiteralNull{}
		}
		if ast.EndIndex == nil {
			ast.EndIndex = &LiteralNull{}
		}
		if ast.Step == nil {
			ast.Step = &LiteralNull{}
		}
		*astPtr = buildStdCall("std.slice", ast.Target, ast.BeginIndex, ast.EndIndex, ast.Step)
		desugar(astPtr, objLevel)

	case *Local:
		for i := range ast.Binds {
			if ast.Binds[i].FunctionSugar {
				origBody := ast.Binds[i].Body
				function := &Function{
					// TODO(sbarzowski) better location
					nodeBase:   nodeBase{loc: *origBody.Loc()},
					Parameters: ast.Binds[i].Params,
					Body:       origBody,
				}
				ast.Binds[i] = LocalBind{
					Variable:      ast.Binds[i].Variable,
					Body:          function,
					FunctionSugar: false,
					Params:        nil,
				}
			}
			err = desugar(&ast.Binds[i].Body, objLevel)
			if err != nil {
				return
			}
		}
		err = desugar(&ast.Body, objLevel)
		if err != nil {
			return
		}

	case *LiteralBoolean:
		// Nothing to do.

	case *LiteralNull:
		// Nothing to do.

	case *LiteralNumber:
		// Nothing to do.

	case *LiteralString:
		if ast.Kind != VerbatimStringDouble && ast.Kind != VerbatimStringSingle {
			unescaped, err := stringUnescape(ast.Loc(), ast.Value)
			if err != nil {
				return err
			}
			ast.Value = unescaped
			ast.Kind = StringDouble
			ast.BlockIndent = ""
		}
	case *Object:
		// Hidden variable to allow $ binding.
		if objLevel == 0 {
			dollar := Identifier("$")
			ast.Fields = append(ast.Fields, ObjectFieldLocalNoMethod(&dollar, &Self{}))
		}

		err = desugarFields(*ast.Loc(), &ast.Fields, objLevel)
		if err != nil {
			return
		}

		var newFields DesugaredObjectFields
		var newAsserts Nodes

		for _, field := range ast.Fields {
			if field.Kind == ObjectAssert {
				newAsserts = append(newAsserts, field.Expr2)
			} else if field.Kind == ObjectFieldExpr {
				newFields = append(newFields, DesugaredObjectField{field.Hide, field.Expr1, field.Expr2})
			} else {
				panic(fmt.Sprintf("INTERNAL ERROR: field should have been desugared: %s", field.Kind))
			}
		}

		*astPtr = &DesugaredObject{ast.nodeBase, newAsserts, newFields}

	case *DesugaredObject:
		panic("Desugaring desugared object")

	case *ObjectComp:
		comp, err := desugarObjectComp(ast, objLevel)
		if err != nil {
			return err
		}
		*astPtr = comp

	case *ObjectComprehensionSimple:
		panic("Desugaring desugared object comprehension")

	case *Self:
		// Nothing to do.

	case *SuperIndex:
		if ast.Id != nil {
			ast.Index = &LiteralString{Value: string(*ast.Id)}
			ast.Id = nil
		}

	case *Unary:
		err = desugar(&ast.Expr, objLevel)
		if err != nil {
			return
		}

	case *Var:
		// Nothing to do.

	default:
		panic(fmt.Sprintf("Desugarer does not recognize ast: %s", reflect.TypeOf(ast)))
	}

	return nil
}

func desugarFile(ast *Node) error {
	err := desugar(ast, 0)
	if err != nil {
		return err
	}
	// TODO(dcunnin): wrap in std local
	return nil
}
