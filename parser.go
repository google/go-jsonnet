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
	"strconv"
)

type precedence int

const (
	applyPrecedence precedence = 2  // Function calls and indexing.
	unaryPrecedence precedence = 4  // Logical and bitwise negation, unary + -
	maxPrecedence   precedence = 16 // Local, If, Import, Function, Error
)

var bopPrecedence = map[binaryOp]precedence{
	bopMult:            5,
	bopDiv:             5,
	bopPercent:         5,
	bopPlus:            6,
	bopMinus:           6,
	bopShiftL:          7,
	bopShiftR:          7,
	bopGreater:         8,
	bopGreaterEq:       8,
	bopLess:            8,
	bopLessEq:          8,
	bopManifestEqual:   9,
	bopManifestUnequal: 9,
	bopBitwiseAnd:      10,
	bopBitwiseXor:      11,
	bopBitwiseOr:       12,
	bopAnd:             13,
	bopOr:              14,
}

// ---------------------------------------------------------------------------

func makeUnexpectedError(t *token, while string) error {
	return makeStaticError(
		fmt.Sprintf("Unexpected: %v while %v", t, while), t.loc)
}

func locFromTokens(begin, end *token) LocationRange {
	return makeLocationRange(begin.loc.FileName, begin.loc.Begin, end.loc.End)
}

func locFromTokenAST(begin *token, end astNode) LocationRange {
	return makeLocationRange(begin.loc.FileName, begin.loc.Begin, end.Loc().End)
}

// ---------------------------------------------------------------------------

type parser struct {
	t     tokens
	currT int
}

func makeParser(t tokens) *parser {
	return &parser{
		t: t,
	}
}

func (p *parser) pop() *token {
	t := &p.t[p.currT]
	p.currT++
	return t
}

func (p *parser) popExpect(tk tokenKind) (*token, error) {
	t := p.pop()
	if t.kind != tk {
		return nil, makeStaticError(
			fmt.Sprintf("Expected token %v but got %v", tk, t), t.loc)
	}
	return t, nil
}

func (p *parser) popExpectOp(op string) (*token, error) {
	t := p.pop()
	if t.kind != tokenOperator || t.data != op {
		return nil, makeStaticError(
			fmt.Sprintf("Expected operator %v but got %v", op, t), t.loc)
	}
	return t, nil
}

func (p *parser) peek() *token {
	return &p.t[p.currT]
}

func (p *parser) parseIdentifierList(elementKind string) (identifiers, bool, error) {
	_, exprs, gotComma, err := p.parseCommaList(tokenParenR, elementKind)
	if err != nil {
		return identifiers{}, false, err
	}
	var ids identifiers
	for _, n := range exprs {
		v, ok := n.(*astVar)
		if !ok {
			return identifiers{}, false, makeStaticError(fmt.Sprintf("Expected simple identifier but got a complex expression."), *n.Loc())
		}
		ids = append(ids, v.id)
	}
	return ids, gotComma, nil
}

func (p *parser) parseCommaList(end tokenKind, elementKind string) (*token, astNodes, bool, error) {
	var exprs astNodes
	gotComma := false
	first := true
	for {
		next := p.peek()
		if !first && !gotComma {
			if next.kind == tokenComma {
				p.pop()
				next = p.peek()
				gotComma = true
			}
		}
		if next.kind == end {
			// gotComma can be true or false here.
			return p.pop(), exprs, gotComma, nil
		}

		if !first && !gotComma {
			return nil, nil, false, makeStaticError(fmt.Sprintf("Expected a comma before next %s.", elementKind), next.loc)
		}

		expr, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, nil, false, err
		}
		exprs = append(exprs, expr)
		gotComma = false
		first = false
	}
}

func (p *parser) parseBind(binds *astLocalBinds) error {
	varID, err := p.popExpect(tokenIdentifier)
	if err != nil {
		return err
	}
	for _, b := range *binds {
		if b.variable == identifier(varID.data) {
			return makeStaticError(fmt.Sprintf("Duplicate local var: %v", varID.data), varID.loc)
		}
	}

	if p.peek().kind == tokenParenL {
		p.pop()
		params, gotComma, err := p.parseIdentifierList("function parameter")
		if err != nil {
			return err
		}
		_, err = p.popExpectOp("=")
		if err != nil {
			return err
		}
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return err
		}
		*binds = append(*binds, astLocalBind{
			variable:      identifier(varID.data),
			body:          body,
			functionSugar: true,
			params:        params,
			trailingComma: gotComma,
		})
	} else {
		_, err = p.popExpectOp("=")
		if err != nil {
			return err
		}
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return err
		}
		*binds = append(*binds, astLocalBind{
			variable: identifier(varID.data),
			body:     body,
		})
	}

	return nil
}

func (p *parser) parseObjectAssignmentOp() (plusSugar bool, hide astObjectFieldHide, err error) {
	op, err := p.popExpect(tokenOperator)
	if err != nil {
		return
	}
	opStr := op.data
	if opStr[0] == '+' {
		plusSugar = true
		opStr = opStr[1:]
	}

	numColons := 0
	for len(opStr) > 0 {
		if opStr[0] != ':' {
			err = makeStaticError(
				fmt.Sprintf("Expected one of :, ::, :::, +:, +::, +:::, got: %v", op.data), op.loc)
			return
		}
		opStr = opStr[1:]
		numColons++
	}

	switch numColons {
	case 1:
		hide = astObjectFieldInherit
	case 2:
		hide = astObjectFieldHidden
	case 3:
		hide = astObjectFieldVisible
	default:
		err = makeStaticError(
			fmt.Sprintf("Expected one of :, ::, :::, +:, +::, +:::, got: %v", op.data), op.loc)
		return
	}

	return
}

// +gen set
type literalField string

func (p *parser) parseObjectRemainder(tok *token) (astNode, *token, error) {
	var fields astObjectFields
	literalFields := make(literalFieldSet)
	binds := make(identifierSet)

	gotComma := false
	first := true

	for {
		next := p.pop()
		if !gotComma && !first {
			if next.kind == tokenComma {
				next = p.pop()
				gotComma = true
			}
		}

		if next.kind == tokenBraceR {
			return &astObject{
				astNodeBase:   astNodeBase{loc: locFromTokens(tok, next)},
				fields:        fields,
				trailingComma: gotComma,
			}, next, nil
		}

		if next.kind == tokenFor {
			// It's a comprehension
			numFields := 0
			numAsserts := 0
			var field astObjectField
			for _, field = range fields {
				if field.kind == astObjectLocal {
					continue
				}
				if field.kind == astObjectAssert {
					numAsserts++
					continue
				}
				numFields++
			}

			if numAsserts > 0 {
				return nil, nil, makeStaticError("Object comprehension cannot have asserts.", next.loc)
			}
			if numFields != 1 {
				return nil, nil, makeStaticError("Object comprehension can only have one field.", next.loc)
			}
			if field.hide != astObjectFieldInherit {
				return nil, nil, makeStaticError("Object comprehensions cannot have hidden fields.", next.loc)
			}
			if field.kind != astObjectFieldExpr {
				return nil, nil, makeStaticError("Object comprehensions can only have [e] fields.", next.loc)
			}
			specs, last, err := p.parseComprehensionSpecs(tokenBraceR)
			if err != nil {
				return nil, nil, err
			}
			return &astObjectComp{
				astNodeBase:   astNodeBase{loc: locFromTokens(tok, last)},
				fields:        fields,
				trailingComma: gotComma,
				specs:         *specs,
			}, last, nil
		}

		if !gotComma && !first {
			return nil, nil, makeStaticError("Expected a comma before next field.", next.loc)
		}
		first = false

		switch next.kind {
		case tokenBracketL, tokenIdentifier, tokenStringDouble, tokenStringSingle, tokenStringBlock:
			var kind astObjectFieldKind
			var expr1 astNode
			var id *identifier
			switch next.kind {
			case tokenIdentifier:
				kind = astObjectFieldID
				id = (*identifier)(&next.data)
			case tokenStringDouble:
				kind = astObjectFieldStr
				expr1 = &astLiteralString{
					astNodeBase: astNodeBase{loc: next.loc},
					value:       next.data,
					kind:        astStringDouble,
				}
			case tokenStringSingle:
				kind = astObjectFieldStr
				expr1 = &astLiteralString{
					astNodeBase: astNodeBase{loc: next.loc},
					value:       next.data,
					kind:        astStringSingle,
				}
			case tokenStringBlock:
				kind = astObjectFieldStr
				expr1 = &astLiteralString{
					astNodeBase: astNodeBase{loc: next.loc},
					value:       next.data,
					kind:        astStringBlock,
					blockIndent: next.stringBlockIndent,
				}
			// TODO(sbarzowski) are verbatim string literals allowed here?
			// if so, maybe it's time we extracted string literal creation somewhere...
			default:
				kind = astObjectFieldExpr
				var err error
				expr1, err = p.parse(maxPrecedence)
				if err != nil {
					return nil, nil, err
				}
				_, err = p.popExpect(tokenBracketR)
				if err != nil {
					return nil, nil, err
				}
			}

			isMethod := false
			methComma := false
			var params identifiers
			if p.peek().kind == tokenParenL {
				p.pop()
				var err error
				params, methComma, err = p.parseIdentifierList("method parameter")
				if err != nil {
					return nil, nil, err
				}
				isMethod = true
			}

			plusSugar, hide, err := p.parseObjectAssignmentOp()
			if err != nil {
				return nil, nil, err
			}

			if plusSugar && isMethod {
				return nil, nil, makeStaticError(
					fmt.Sprintf("Cannot use +: syntax sugar in a method: %v", next.data), next.loc)
			}

			if kind != astObjectFieldExpr {
				if !literalFields.Add(literalField(next.data)) {
					return nil, nil, makeStaticError(
						fmt.Sprintf("Duplicate field: %v", next.data), next.loc)
				}
			}

			body, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}

			fields = append(fields, astObjectField{
				kind:          kind,
				hide:          hide,
				superSugar:    plusSugar,
				methodSugar:   isMethod,
				expr1:         expr1,
				id:            id,
				ids:           params,
				trailingComma: methComma,
				expr2:         body,
			})

		case tokenLocal:
			varID, err := p.popExpect(tokenIdentifier)
			if err != nil {
				return nil, nil, err
			}

			id := identifier(varID.data)

			if binds.Contains(id) {
				return nil, nil, makeStaticError(fmt.Sprintf("Duplicate local var: %v", id), varID.loc)
			}

			isMethod := false
			funcComma := false
			var params identifiers
			if p.peek().kind == tokenParenL {
				p.pop()
				isMethod = true
				params, funcComma, err = p.parseIdentifierList("function parameter")
				if err != nil {
					return nil, nil, err
				}
			}
			_, err = p.popExpectOp("=")
			if err != nil {
				return nil, nil, err
			}

			body, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}

			binds.Add(id)

			fields = append(fields, astObjectField{
				kind:          astObjectLocal,
				hide:          astObjectFieldVisible,
				superSugar:    false,
				methodSugar:   isMethod,
				id:            &id,
				ids:           params,
				trailingComma: funcComma,
				expr2:         body,
			})

		case tokenAssert:
			cond, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}
			var msg astNode
			if p.peek().kind == tokenOperator && p.peek().data == ":" {
				p.pop()
				msg, err = p.parse(maxPrecedence)
				if err != nil {
					return nil, nil, err
				}
			}

			fields = append(fields, astObjectField{
				kind:  astObjectAssert,
				hide:  astObjectFieldVisible,
				expr2: cond,
				expr3: msg,
			})
		default:
			return nil, nil, makeUnexpectedError(next, "parsing field definition")
		}
		gotComma = false
	}
}

/* parses for x in expr for y in expr if expr for z in expr ... */
func (p *parser) parseComprehensionSpecs(end tokenKind) (*astCompSpecs, *token, error) {
	var specs astCompSpecs
	for {
		varID, err := p.popExpect(tokenIdentifier)
		if err != nil {
			return nil, nil, err
		}
		id := identifier(varID.data)
		_, err = p.popExpect(tokenIn)
		if err != nil {
			return nil, nil, err
		}
		arr, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, nil, err
		}
		specs = append(specs, astCompSpec{
			kind:    astCompFor,
			varName: &id,
			expr:    arr,
		})

		maybeIf := p.pop()
		for ; maybeIf.kind == tokenIf; maybeIf = p.pop() {
			cond, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}
			specs = append(specs, astCompSpec{
				kind:    astCompIf,
				varName: nil,
				expr:    cond,
			})
		}
		if maybeIf.kind == end {
			return &specs, maybeIf, nil
		}

		if maybeIf.kind != tokenFor {
			return nil, nil, makeStaticError(
				fmt.Sprintf("Expected for, if or %v after for clause, got: %v", end, maybeIf), maybeIf.loc)
		}

	}
}

// Assumes that the leading '[' has already been consumed and passed as tok.
// Should read up to and consume the trailing ']'
func (p *parser) parseArray(tok *token) (astNode, error) {
	next := p.peek()
	if next.kind == tokenBracketR {
		p.pop()
		return &astArray{
			astNodeBase: astNodeBase{loc: locFromTokens(tok, next)},
		}, nil
	}

	first, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, err
	}
	var gotComma bool
	next = p.peek()
	if next.kind == tokenComma {
		p.pop()
		next = p.peek()
		gotComma = true
	}

	if next.kind == tokenFor {
		// It's a comprehension
		p.pop()
		specs, last, err := p.parseComprehensionSpecs(tokenBracketR)
		if err != nil {
			return nil, err
		}
		return &astArrayComp{
			astNodeBase:   astNodeBase{loc: locFromTokens(tok, last)},
			body:          first,
			trailingComma: gotComma,
			specs:         *specs,
		}, nil
	}
	// Not a comprehension: It can have more elements.
	elements := astNodes{first}

	for {
		if next.kind == tokenBracketR {
			// TODO(dcunnin): SYNTAX SUGAR HERE (preserve comma)
			p.pop()
			break
		}
		if !gotComma {
			return nil, makeStaticError("Expected a comma before next array element.", next.loc)
		}
		nextElem, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		elements = append(elements, nextElem)
		next = p.peek()
		if next.kind == tokenComma {
			p.pop()
			next = p.peek()
			gotComma = true
		} else {
			gotComma = false
		}
	}

	return &astArray{
		astNodeBase:   astNodeBase{loc: locFromTokens(tok, next)},
		elements:      elements,
		trailingComma: gotComma,
	}, nil
}

func (p *parser) parseTerminal() (astNode, error) {
	tok := p.pop()
	switch tok.kind {
	case tokenAssert, tokenBraceR, tokenBracketR, tokenComma, tokenDot, tokenElse,
		tokenError, tokenFor, tokenFunction, tokenIf, tokenIn, tokenImport, tokenImportStr,
		tokenLocal, tokenOperator, tokenParenR, tokenSemicolon, tokenTailStrict, tokenThen:
		return nil, makeUnexpectedError(tok, "parsing terminal")

	case tokenEndOfFile:
		return nil, makeStaticError("Unexpected end of file.", tok.loc)

	case tokenBraceL:
		obj, _, err := p.parseObjectRemainder(tok)
		return obj, err

	case tokenBracketL:
		return p.parseArray(tok)

	case tokenParenL:
		inner, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		_, err = p.popExpect(tokenParenR)
		if err != nil {
			return nil, err
		}
		return inner, nil

	// Literals
	case tokenNumber:
		// This shouldn't fail as the lexer should make sure we have good input but
		// we handle the error regardless.
		num, err := strconv.ParseFloat(tok.data, 64)
		if err != nil {
			return nil, makeStaticError("Could not parse floating point number.", tok.loc)
		}
		return &astLiteralNumber{
			astNodeBase:    astNodeBase{loc: tok.loc},
			value:          num,
			originalString: tok.data,
		}, nil
	case tokenStringSingle:
		return &astLiteralString{
			astNodeBase: astNodeBase{loc: tok.loc},
			value:       tok.data,
			kind:        astStringSingle,
		}, nil
	case tokenStringDouble:
		return &astLiteralString{
			astNodeBase: astNodeBase{loc: tok.loc},
			value:       tok.data,
			kind:        astStringDouble,
		}, nil
	case tokenStringBlock:
		return &astLiteralString{
			astNodeBase: astNodeBase{loc: tok.loc},
			value:       tok.data,
			kind:        astStringDouble,
			blockIndent: tok.stringBlockIndent,
		}, nil
	case tokenVerbatimStringDouble:
		return &astLiteralString{
			astNodeBase: astNodeBase{loc: tok.loc},
			value:       tok.data,
			kind:        astVerbatimStringDouble,
		}, nil
	case tokenVerbatimStringSingle:
		return &astLiteralString{
			astNodeBase: astNodeBase{loc: tok.loc},
			value:       tok.data,
			kind:        astVerbatimStringSingle,
		}, nil
	case tokenFalse:
		return &astLiteralBoolean{
			astNodeBase: astNodeBase{loc: tok.loc},
			value:       false,
		}, nil
	case tokenTrue:
		return &astLiteralBoolean{
			astNodeBase: astNodeBase{loc: tok.loc},
			value:       true,
		}, nil
	case tokenNullLit:
		return &astLiteralNull{
			astNodeBase: astNodeBase{loc: tok.loc},
		}, nil

	// Variables
	case tokenDollar:
		return &astDollar{
			astNodeBase: astNodeBase{loc: tok.loc},
		}, nil
	case tokenIdentifier:
		return &astVar{
			astNodeBase: astNodeBase{loc: tok.loc},
			id:          identifier(tok.data),
		}, nil
	case tokenSelf:
		return &astSelf{
			astNodeBase: astNodeBase{loc: tok.loc},
		}, nil
	case tokenSuper:
		next := p.pop()
		var index astNode
		var id *identifier
		switch next.kind {
		case tokenDot:
			fieldID, err := p.popExpect(tokenIdentifier)
			if err != nil {
				return nil, err
			}
			id = (*identifier)(&fieldID.data)
		case tokenBracketL:
			var err error
			index, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			_, err = p.popExpect(tokenBracketR)
			if err != nil {
				return nil, err
			}
		default:
			return nil, makeStaticError("Expected . or [ after super.", tok.loc)
		}
		return &astSuperIndex{
			astNodeBase: astNodeBase{loc: tok.loc},
			index:       index,
			id:          id,
		}, nil
	}

	return nil, makeStaticError(fmt.Sprintf("INTERNAL ERROR: Unknown tok kind: %v", tok.kind), tok.loc)
}

func (p *parser) parse(prec precedence) (astNode, error) {
	begin := p.peek()

	switch begin.kind {
	// These cases have effectively maxPrecedence as the first
	// call to parse will parse them.
	case tokenAssert:
		p.pop()
		cond, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		var msg astNode
		if p.peek().kind == tokenOperator && p.peek().data == ":" {
			p.pop()
			msg, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
		}
		_, err = p.popExpect(tokenSemicolon)
		if err != nil {
			return nil, err
		}
		rest, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &astAssert{
			astNodeBase: astNodeBase{loc: locFromTokenAST(begin, rest)},
			cond:        cond,
			message:     msg,
			rest:        rest,
		}, nil

	case tokenError:
		p.pop()
		expr, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &astError{
			astNodeBase: astNodeBase{loc: locFromTokenAST(begin, expr)},
			expr:        expr,
		}, nil

	case tokenIf:
		p.pop()
		cond, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		_, err = p.popExpect(tokenThen)
		if err != nil {
			return nil, err
		}
		branchTrue, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		var branchFalse astNode
		lr := locFromTokenAST(begin, branchTrue)
		if p.peek().kind == tokenElse {
			p.pop()
			branchFalse, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			lr = locFromTokenAST(begin, branchFalse)
		}
		return &astConditional{
			astNodeBase: astNodeBase{loc: lr},
			cond:        cond,
			branchTrue:  branchTrue,
			branchFalse: branchFalse,
		}, nil

	case tokenFunction:
		p.pop()
		next := p.pop()
		if next.kind == tokenParenL {
			params, gotComma, err := p.parseIdentifierList("function parameter")
			if err != nil {
				return nil, err
			}
			body, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			return &astFunction{
				astNodeBase:   astNodeBase{loc: locFromTokenAST(begin, body)},
				parameters:    params,
				trailingComma: gotComma,
				body:          body,
			}, nil
		}
		return nil, makeStaticError(fmt.Sprintf("Expected ( but got %v", next), next.loc)

	case tokenImport:
		p.pop()
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		if lit, ok := body.(*astLiteralString); ok {
			return &astImport{
				astNodeBase: astNodeBase{loc: locFromTokenAST(begin, body)},
				file:        lit.value,
			}, nil
		}
		return nil, makeStaticError("Computed imports are not allowed", *body.Loc())

	case tokenImportStr:
		p.pop()
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		if lit, ok := body.(*astLiteralString); ok {
			return &astImportStr{
				astNodeBase: astNodeBase{loc: locFromTokenAST(begin, body)},
				file:        lit.value,
			}, nil
		}
		return nil, makeStaticError("Computed imports are not allowed", *body.Loc())

	case tokenLocal:
		p.pop()
		var binds astLocalBinds
		for {
			err := p.parseBind(&binds)
			if err != nil {
				return nil, err
			}
			delim := p.pop()
			if delim.kind != tokenSemicolon && delim.kind != tokenComma {
				return nil, makeStaticError(fmt.Sprintf("Expected , or ; but got %v", delim), delim.loc)
			}
			if delim.kind == tokenSemicolon {
				break
			}
		}
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &astLocal{
			astNodeBase: astNodeBase{loc: locFromTokenAST(begin, body)},
			binds:       binds,
			body:        body,
		}, nil

	default:
		// Unary operator
		if begin.kind == tokenOperator {
			uop, ok := uopMap[begin.data]
			if !ok {
				return nil, makeStaticError(fmt.Sprintf("Not a unary operator: %v", begin.data), begin.loc)
			}
			if prec == unaryPrecedence {
				op := p.pop()
				expr, err := p.parse(prec)
				if err != nil {
					return nil, err
				}
				return &astUnary{
					astNodeBase: astNodeBase{loc: locFromTokenAST(op, expr)},
					op:          uop,
					expr:        expr,
				}, nil
			}
		}

		// Base case
		if prec == 0 {
			return p.parseTerminal()
		}

		lhs, err := p.parse(prec - 1)
		if err != nil {
			return nil, err
		}

		for {
			// Then next token must be a binary operator.

			var bop binaryOp

			// Check precedence is correct for this level.  If we're parsing operators
			// with higher precedence, then return lhs and let lower levels deal with
			// the operator.
			switch p.peek().kind {
			case tokenOperator:
				_ = "breakpoint"
				if p.peek().data == ":" {
					// Special case for the colons in assert. Since COLON is no-longer a
					// special token, we have to make sure it does not trip the
					// op_is_binary test below.  It should terminate parsing of the
					// expression here, returning control to the parsing of the actual
					// assert AST.
					return lhs, nil
				}
				var ok bool
				bop, ok = bopMap[p.peek().data]
				if !ok {
					return nil, makeStaticError(fmt.Sprintf("Not a binary operator: %v", p.peek().data), p.peek().loc)
				}

				if bopPrecedence[bop] != prec {
					return lhs, nil
				}

			case tokenDot, tokenBracketL, tokenParenL, tokenBraceL:
				if applyPrecedence != prec {
					return lhs, nil
				}
			default:
				return lhs, nil
			}

			op := p.pop()
			switch op.kind {
			case tokenBracketL:
				index, err := p.parse(maxPrecedence)
				if err != nil {
					return nil, err
				}
				end, err := p.popExpect(tokenBracketR)
				if err != nil {
					return nil, err
				}
				lhs = &astIndex{
					astNodeBase: astNodeBase{loc: locFromTokens(begin, end)},
					target:      lhs,
					index:       index,
				}
			case tokenDot:
				fieldID, err := p.popExpect(tokenIdentifier)
				if err != nil {
					return nil, err
				}
				id := identifier(fieldID.data)
				lhs = &astIndex{
					astNodeBase: astNodeBase{loc: locFromTokens(begin, fieldID)},
					target:      lhs,
					id:          &id,
				}
			case tokenParenL:
				end, args, gotComma, err := p.parseCommaList(tokenParenR, "function argument")
				if err != nil {
					return nil, err
				}
				tailStrict := false
				if p.peek().kind == tokenTailStrict {
					p.pop()
					tailStrict = true
				}
				lhs = &astApply{
					astNodeBase:   astNodeBase{loc: locFromTokens(begin, end)},
					target:        lhs,
					arguments:     args,
					trailingComma: gotComma,
					tailStrict:    tailStrict,
				}
			case tokenBraceL:
				obj, end, err := p.parseObjectRemainder(op)
				if err != nil {
					return nil, err
				}
				lhs = &astApplyBrace{
					astNodeBase: astNodeBase{loc: locFromTokens(begin, end)},
					left:        lhs,
					right:       obj,
				}
			default:
				rhs, err := p.parse(prec - 1)
				if err != nil {
					return nil, err
				}
				lhs = &astBinary{
					astNodeBase: astNodeBase{loc: locFromTokenAST(begin, rhs)},
					left:        lhs,
					op:          bop,
					right:       rhs,
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------

func parse(t tokens) (astNode, error) {
	p := makeParser(t)
	expr, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, err
	}

	if p.peek().kind != tokenEndOfFile {
		return nil, makeStaticError(fmt.Sprintf("Did not expect: %v", p.peek()), p.peek().loc)
	}

	return expr, nil
}
