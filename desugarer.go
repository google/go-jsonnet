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

func makeStr(s string) *astLiteralString {
	return &astLiteralString{astNodeBase{loc: LocationRange{}}, s, astStringDouble, ""}
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

func desugarFields(location LocationRange, fields *astObjectFields, objLevel int) error {

	// Desugar children
	for i := range *fields {
		field := &((*fields)[i])
		if field.expr1 != nil {
			err := desugar(&field.expr1, objLevel)
			if err != nil {
				return err
			}
		}
		err := desugar(&field.expr2, objLevel+1)
		if err != nil {
			return err
		}
		if field.expr3 != nil {
			err := desugar(&field.expr3, objLevel+1)
			if err != nil {
				return err
			}
		}
	}

	// Simplify asserts
	// TODO(dcunnin): this
	for _, field := range *fields {
		if field.kind != astObjectAssert {
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
		if !field.methodSugar {
			continue
		}
		origBody := field.expr2
		function := &astFunction{
			// TODO(sbarzowski) better location
			astNodeBase: astNodeBase{loc: *origBody.Loc()},
			parameters:  field.ids,
			body:        origBody,
		}
		field.methodSugar = false
		field.ids = nil
		field.expr2 = function
	}

	// Remove object-level locals
	newFields := []astObjectField{}
	var binds astLocalBinds
	for _, local := range *fields {
		if local.kind != astObjectLocal {
			continue
		}
		binds = append(binds, astLocalBind{variable: *local.id, body: local.expr2})
	}
	for _, field := range *fields {
		if field.kind == astObjectLocal {
			continue
		}
		if len(binds) > 0 {
			field.expr2 = &astLocal{astNodeBase{loc: *field.expr2.Loc()}, binds, field.expr2}
		}
		newFields = append(newFields, field)
	}
	*fields = newFields

	// Change all to FIELD_EXPR
	for i := range *fields {
		field := &(*fields)[i]
		switch field.kind {
		case astObjectAssert:
		// Nothing to do.

		case astObjectFieldID:
			field.expr1 = makeStr(string(*field.id))
			field.kind = astObjectFieldExpr

		case astObjectFieldExpr:
		// Nothing to do.

		case astObjectFieldStr:
			// Just set the flag.
			field.kind = astObjectFieldExpr

		case astObjectLocal:
			return fmt.Errorf("INTERNAL ERROR: Locals should be removed by now")
		}
	}

	// Remove +:
	// TODO(dcunnin): this
	for _, field := range *fields {
		if !field.superSugar {
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

func desugarArrayComp(astComp *astArrayComp, objLevel int) (astNode, error) {
	return &astLiteralNull{}, nil
	// TODO(sbarzowski) this
	switch astComp.specs[0].kind {
	case astCompFor:
		panic("TODO")
	case astCompIf:
		panic("TODO")
	default:
		panic("TODO")
	}
}

func desugarObjectComp(astComp *astObjectComp, objLevel int) (astNode, error) {
	return &astLiteralNull{}, nil
	// TODO(sbarzowski) this
}

func buildSimpleIndex(obj astNode, member identifier) astNode {
	return &astIndex{
		target: obj,
		id:     &member,
	}
}

func buildStdCall(builtinName identifier, args ...astNode) astNode {
	std := &astVar{id: "std"}
	builtin := buildSimpleIndex(std, builtinName)
	return &astApply{
		target:    builtin,
		arguments: args,
	}
}

// Desugar Jsonnet expressions to reduce the number of constructs the rest of the implementation
// needs to understand.

// Desugaring should happen immediately after parsing, i.e. before static analysis and execution.
// Temporary variables introduced here should be prefixed with $ to ensure they do not clash with
// variables used in user code.
// TODO(sbarzowski) Actually we may want to do some static analysis before desugaring, e.g.
// warning user about dangerous use of constructs that we desugar.
func desugar(astPtr *astNode, objLevel int) (err error) {
	ast := *astPtr

	if ast == nil {
		return
	}

	switch ast := ast.(type) {
	case *astApply:
		desugar(&ast.target, objLevel)
		for i := range ast.arguments {
			err = desugar(&ast.arguments[i], objLevel)
			if err != nil {
				return
			}
		}

	case *astApplyBrace:
		err = desugar(&ast.left, objLevel)
		if err != nil {
			return
		}
		err = desugar(&ast.right, objLevel)
		if err != nil {
			return
		}
		*astPtr = &astBinary{
			astNodeBase: ast.astNodeBase,
			left:        ast.left,
			op:          bopPlus,
			right:       ast.right,
		}

	case *astArray:
		for i := range ast.elements {
			err = desugar(&ast.elements[i], objLevel)
			if err != nil {
				return
			}
		}

	case *astArrayComp:
		comp, err := desugarArrayComp(ast, objLevel)
		if err != nil {
			return err
		}
		*astPtr = comp

	case *astAssert:
		// TODO(sbarzowski) this
		*astPtr = &astLiteralNull{}

	case *astBinary:
		// some operators get replaced by stdlib functions
		if funcname, replaced := desugaredBop[ast.op]; replaced {
			if funcname == "notEquals" {
				// TODO(sbarzowski) maybe we can handle it in more regular way
				// but let's be consistent with the spec
				*astPtr = &astUnary{
					op:   uopNot,
					expr: buildStdCall(desugaredBop[bopManifestEqual], ast.left, ast.right),
				}
			} else {
				*astPtr = buildStdCall(funcname, ast.left, ast.right)
			}
			return desugar(astPtr, objLevel)
		}

		err = desugar(&ast.left, objLevel)
		if err != nil {
			return
		}
		err = desugar(&ast.right, objLevel)
		if err != nil {
			return
		}
		// TODO(dcunnin): Need to handle bopPercent, bopManifestUnequal, bopManifestEqual

	case *astConditional:
		err = desugar(&ast.cond, objLevel)
		if err != nil {
			return
		}
		err = desugar(&ast.branchTrue, objLevel)
		if err != nil {
			return
		}
		if ast.branchFalse == nil {
			ast.branchFalse = &astLiteralNull{}
		}
		err = desugar(&ast.branchFalse, objLevel)
		if err != nil {
			return
		}

	case *astDollar:
		if objLevel == 0 {
			return makeStaticError("No top-level object found.", *ast.Loc())
		}
		*astPtr = &astVar{astNodeBase: ast.astNodeBase, id: identifier("$")}

	case *astError:
		err = desugar(&ast.expr, objLevel)
		if err != nil {
			return
		}

	case *astFunction:
		err = desugar(&ast.body, objLevel)
		if err != nil {
			return
		}

	case *astImport:
		// Nothing to do.

	case *astImportStr:
		// Nothing to do.

	case *astIndex:
		err = desugar(&ast.target, objLevel)
		if err != nil {
			return
		}
		if ast.id != nil {
			if ast.index != nil {
				panic("TODO")
			}
			ast.index = makeStr(string(*ast.id))
			ast.id = nil
		}
		err = desugar(&ast.index, objLevel)
		if err != nil {
			return
		}

	case *astSlice:
		if ast.beginIndex == nil {
			ast.beginIndex = &astLiteralNull{}
		}
		if ast.endIndex == nil {
			ast.endIndex = &astLiteralNull{}
		}
		if ast.step == nil {
			ast.step = &astLiteralNull{}
		}
		*astPtr = buildStdCall("std.slice", ast.target, ast.beginIndex, ast.endIndex, ast.step)
		desugar(astPtr, objLevel)

	case *astLocal:
		for i := range ast.binds {
			if ast.binds[i].functionSugar {
				origBody := ast.binds[i].body
				function := &astFunction{
					// TODO(sbarzowski) better location
					astNodeBase: astNodeBase{loc: *origBody.Loc()},
					parameters:  ast.binds[i].params,
					body:        origBody,
				}
				ast.binds[i] = astLocalBind{
					variable:      ast.binds[i].variable,
					body:          function,
					functionSugar: false,
					params:        nil,
				}
			}
			err = desugar(&ast.binds[i].body, objLevel)
			if err != nil {
				return
			}
		}
		err = desugar(&ast.body, objLevel)
		if err != nil {
			return
		}

	case *astLiteralBoolean:
		// Nothing to do.

	case *astLiteralNull:
		// Nothing to do.

	case *astLiteralNumber:
		// Nothing to do.

	case *astLiteralString:
		if ast.kind != astVerbatimStringDouble && ast.kind != astVerbatimStringSingle {
			unescaped, err := stringUnescape(ast.Loc(), ast.value)
			if err != nil {
				return err
			}
			ast.value = unescaped
			ast.kind = astStringDouble
			ast.blockIndent = ""
		}
	case *astObject:
		// Hidden variable to allow $ binding.
		if objLevel == 0 {
			dollar := identifier("$")
			ast.fields = append(ast.fields, astObjectFieldLocalNoMethod(&dollar, &astSelf{}))
		}

		err = desugarFields(*ast.Loc(), &ast.fields, objLevel)
		if err != nil {
			return
		}

		var newFields astDesugaredObjectFields
		var newAsserts astNodes

		for _, field := range ast.fields {
			if field.kind == astObjectAssert {
				newAsserts = append(newAsserts, field.expr2)
			} else if field.kind == astObjectFieldExpr {
				newFields = append(newFields, astDesugaredObjectField{field.hide, field.expr1, field.expr2})
			} else {
				panic(fmt.Sprintf("INTERNAL ERROR: field should have been desugared: %s", field.kind))
			}
		}

		*astPtr = &astDesugaredObject{ast.astNodeBase, newAsserts, newFields}

	case *astDesugaredObject:
		panic("Desugaring desugared object")

	case *astObjectComp:
		comp, err := desugarObjectComp(ast, objLevel)
		if err != nil {
			return err
		}
		*astPtr = comp

	case *astObjectComprehensionSimple:
		panic("Desugaring desugared object comprehension")

	case *astSelf:
		// Nothing to do.

	case *astSuperIndex:
		if ast.id != nil {
			ast.index = &astLiteralString{value: string(*ast.id)}
			ast.id = nil
		}

	case *astUnary:
		err = desugar(&ast.expr, objLevel)
		if err != nil {
			return
		}

	case *astVar:
		// Nothing to do.

	default:
		panic(fmt.Sprintf("Desugarer does not recognize ast: %s", reflect.TypeOf(ast)))
	}

	return nil
}

func desugarFile(ast *astNode) error {
	err := desugar(ast, 0)
	if err != nil {
		return err
	}
	// TODO(dcunnin): wrap in std local
	return nil
}
