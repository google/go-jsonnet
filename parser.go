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

var bopPrecedence = map[BinaryOp]precedence{
	BopMult:            5,
	BopDiv:             5,
	BopPercent:         5,
	BopPlus:            6,
	BopMinus:           6,
	BopShiftL:          7,
	BopShiftR:          7,
	BopGreater:         8,
	BopGreaterEq:       8,
	BopLess:            8,
	BopLessEq:          8,
	BopManifestEqual:   9,
	BopManifestUnequal: 9,
	BopBitwiseAnd:      10,
	BopBitwiseXor:      11,
	BopBitwiseOr:       12,
	BopAnd:             13,
	BopOr:              14,
}

// ---------------------------------------------------------------------------

func makeUnexpectedError(t *token, while string) error {
	return makeStaticError(
		fmt.Sprintf("Unexpected: %v while %v", t, while), t.loc)
}

func locFromTokens(begin, end *token) LocationRange {
	return makeLocationRange(begin.loc.FileName, begin.loc.Begin, end.loc.End)
}

func locFromTokenAST(begin *token, end Node) LocationRange {
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

func (p *parser) unexpectedTokenError(tk tokenKind, t *token) error {
	if tk == t.kind {
		panic("Unexpectedly expected token kind.")
	}
	return makeStaticError(fmt.Sprintf("Expected token %v but got %v", tk, t), t.loc)
}

func (p *parser) popExpect(tk tokenKind) (*token, error) {
	t := p.pop()
	if t.kind != tk {
		return nil, p.unexpectedTokenError(tk, t)
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

func (p *parser) parseIdentifierList(elementKind string) (Identifiers, bool, error) {
	_, exprs, gotComma, err := p.parseCommaList(tokenParenR, elementKind)
	if err != nil {
		return Identifiers{}, false, err
	}
	var ids Identifiers
	for _, n := range exprs {
		v, ok := n.(*Var)
		if !ok {
			return Identifiers{}, false, makeStaticError(fmt.Sprintf("Expected simple identifier but got a complex expression."), *n.Loc())
		}
		ids = append(ids, v.Id)
	}
	return ids, gotComma, nil
}

func (p *parser) parseCommaList(end tokenKind, elementKind string) (*token, Nodes, bool, error) {
	var exprs Nodes
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

func (p *parser) parseBind(binds *LocalBinds) error {
	varID, err := p.popExpect(tokenIdentifier)
	if err != nil {
		return err
	}
	for _, b := range *binds {
		if b.Variable == Identifier(varID.data) {
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
		*binds = append(*binds, LocalBind{
			Variable:      Identifier(varID.data),
			Body:          body,
			FunctionSugar: true,
			Params:        params,
			TrailingComma: gotComma,
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
		*binds = append(*binds, LocalBind{
			Variable: Identifier(varID.data),
			Body:     body,
		})
	}

	return nil
}

func (p *parser) parseObjectAssignmentOp() (plusSugar bool, hide ObjectFieldHide, err error) {
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
		hide = ObjectFieldInherit
	case 2:
		hide = ObjectFieldHidden
	case 3:
		hide = ObjectFieldVisible
	default:
		err = makeStaticError(
			fmt.Sprintf("Expected one of :, ::, :::, +:, +::, +:::, got: %v", op.data), op.loc)
		return
	}

	return
}

// +gen set
type literalField string

func (p *parser) parseObjectRemainder(tok *token) (Node, *token, error) {
	var fields ObjectFields
	literalFields := make(literalFieldSet)
	binds := make(IdentifierSet)

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
			return &Object{
				nodeBase:      nodeBase{loc: locFromTokens(tok, next)},
				Fields:        fields,
				TrailingComma: gotComma,
			}, next, nil
		}

		if next.kind == tokenFor {
			// It's a comprehension
			numFields := 0
			numAsserts := 0
			var field ObjectField
			for _, f := range fields {
				if f.Kind == ObjectLocal {
					continue
				}
				if f.Kind == ObjectAssert {
					numAsserts++
					continue
				}
				numFields++
				field = f
			}

			if numAsserts > 0 {
				return nil, nil, makeStaticError("Object comprehension cannot have asserts.", next.loc)
			}
			if numFields != 1 {
				return nil, nil, makeStaticError("Object comprehension can only have one field.", next.loc)
			}
			if field.Hide != ObjectFieldInherit {
				return nil, nil, makeStaticError("Object comprehensions cannot have hidden fields.", next.loc)
			}
			if field.Kind != ObjectFieldExpr {
				return nil, nil, makeStaticError("Object comprehensions can only have [e] fields.", next.loc)
			}
			specs, last, err := p.parseComprehensionSpecs(tokenBraceR)
			if err != nil {
				return nil, nil, err
			}
			return &ObjectComp{
				nodeBase:      nodeBase{loc: locFromTokens(tok, last)},
				Fields:        fields,
				TrailingComma: gotComma,
				Specs:         *specs,
			}, last, nil
		}

		if !gotComma && !first {
			return nil, nil, makeStaticError("Expected a comma before next field.", next.loc)
		}
		first = false

		switch next.kind {
		case tokenBracketL, tokenIdentifier, tokenStringDouble, tokenStringSingle, tokenStringBlock:
			var kind ObjectFieldKind
			var expr1 Node
			var id *Identifier
			switch next.kind {
			case tokenIdentifier:
				kind = ObjectFieldID
				id = (*Identifier)(&next.data)
			case tokenStringDouble:
				kind = ObjectFieldStr
				expr1 = &LiteralString{
					nodeBase: nodeBase{loc: next.loc},
					Value:    next.data,
					Kind:     StringDouble,
				}
			case tokenStringSingle:
				kind = ObjectFieldStr
				expr1 = &LiteralString{
					nodeBase: nodeBase{loc: next.loc},
					Value:    next.data,
					Kind:     StringSingle,
				}
			case tokenStringBlock:
				kind = ObjectFieldStr
				expr1 = &LiteralString{
					nodeBase:    nodeBase{loc: next.loc},
					Value:       next.data,
					Kind:        StringBlock,
					BlockIndent: next.stringBlockIndent,
				}
			// TODO(sbarzowski) are verbatim string literals allowed here?
			// if so, maybe it's time we extracted string literal creation somewhere...
			default:
				kind = ObjectFieldExpr
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
			var params Identifiers
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

			if kind != ObjectFieldExpr {
				if !literalFields.Add(literalField(next.data)) {
					return nil, nil, makeStaticError(
						fmt.Sprintf("Duplicate field: %v", next.data), next.loc)
				}
			}

			body, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}

			fields = append(fields, ObjectField{
				Kind:          kind,
				Hide:          hide,
				SuperSugar:    plusSugar,
				MethodSugar:   isMethod,
				Expr1:         expr1,
				Id:            id,
				Ids:           params,
				TrailingComma: methComma,
				Expr2:         body,
			})

		case tokenLocal:
			varID, err := p.popExpect(tokenIdentifier)
			if err != nil {
				return nil, nil, err
			}

			id := Identifier(varID.data)

			if binds.Contains(id) {
				return nil, nil, makeStaticError(fmt.Sprintf("Duplicate local var: %v", id), varID.loc)
			}

			isMethod := false
			funcComma := false
			var params Identifiers
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

			fields = append(fields, ObjectField{
				Kind:          ObjectLocal,
				Hide:          ObjectFieldVisible,
				SuperSugar:    false,
				MethodSugar:   isMethod,
				Id:            &id,
				Ids:           params,
				TrailingComma: funcComma,
				Expr2:         body,
			})

		case tokenAssert:
			cond, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}
			var msg Node
			if p.peek().kind == tokenOperator && p.peek().data == ":" {
				p.pop()
				msg, err = p.parse(maxPrecedence)
				if err != nil {
					return nil, nil, err
				}
			}

			fields = append(fields, ObjectField{
				Kind:  ObjectAssert,
				Hide:  ObjectFieldVisible,
				Expr2: cond,
				Expr3: msg,
			})
		default:
			return nil, nil, makeUnexpectedError(next, "parsing field definition")
		}
		gotComma = false
	}
}

/* parses for x in expr for y in expr if expr for z in expr ... */
func (p *parser) parseComprehensionSpecs(end tokenKind) (*CompSpecs, *token, error) {
	var specs CompSpecs
	for {
		varID, err := p.popExpect(tokenIdentifier)
		if err != nil {
			return nil, nil, err
		}
		id := Identifier(varID.data)
		_, err = p.popExpect(tokenIn)
		if err != nil {
			return nil, nil, err
		}
		arr, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, nil, err
		}
		specs = append(specs, CompSpec{
			Kind:    CompFor,
			VarName: &id,
			Expr:    arr,
		})

		maybeIf := p.pop()
		for ; maybeIf.kind == tokenIf; maybeIf = p.pop() {
			cond, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}
			specs = append(specs, CompSpec{
				Kind:    CompIf,
				VarName: nil,
				Expr:    cond,
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
func (p *parser) parseArray(tok *token) (Node, error) {
	next := p.peek()
	if next.kind == tokenBracketR {
		p.pop()
		return &Array{
			nodeBase: nodeBase{loc: locFromTokens(tok, next)},
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
		return &ArrayComp{
			nodeBase:      nodeBase{loc: locFromTokens(tok, last)},
			Body:          first,
			TrailingComma: gotComma,
			Specs:         *specs,
		}, nil
	}
	// Not a comprehension: It can have more elements.
	elements := Nodes{first}

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

	return &Array{
		nodeBase:      nodeBase{loc: locFromTokens(tok, next)},
		Elements:      elements,
		TrailingComma: gotComma,
	}, nil
}

func (p *parser) parseTerminal() (Node, error) {
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
		return &LiteralNumber{
			nodeBase:       nodeBase{loc: tok.loc},
			Value:          num,
			OriginalString: tok.data,
		}, nil
	case tokenStringSingle:
		return &LiteralString{
			nodeBase: nodeBase{loc: tok.loc},
			Value:    tok.data,
			Kind:     StringSingle,
		}, nil
	case tokenStringDouble:
		return &LiteralString{
			nodeBase: nodeBase{loc: tok.loc},
			Value:    tok.data,
			Kind:     StringDouble,
		}, nil
	case tokenStringBlock:
		return &LiteralString{
			nodeBase:    nodeBase{loc: tok.loc},
			Value:       tok.data,
			Kind:        StringDouble,
			BlockIndent: tok.stringBlockIndent,
		}, nil
	case tokenVerbatimStringDouble:
		return &LiteralString{
			nodeBase: nodeBase{loc: tok.loc},
			Value:    tok.data,
			Kind:     VerbatimStringDouble,
		}, nil
	case tokenVerbatimStringSingle:
		return &LiteralString{
			nodeBase: nodeBase{loc: tok.loc},
			Value:    tok.data,
			Kind:     VerbatimStringSingle,
		}, nil
	case tokenFalse:
		return &LiteralBoolean{
			nodeBase: nodeBase{loc: tok.loc},
			Value:    false,
		}, nil
	case tokenTrue:
		return &LiteralBoolean{
			nodeBase: nodeBase{loc: tok.loc},
			Value:    true,
		}, nil
	case tokenNullLit:
		return &LiteralNull{
			nodeBase: nodeBase{loc: tok.loc},
		}, nil

	// Variables
	case tokenDollar:
		return &Dollar{
			nodeBase: nodeBase{loc: tok.loc},
		}, nil
	case tokenIdentifier:
		return &Var{
			nodeBase: nodeBase{loc: tok.loc},
			Id:       Identifier(tok.data),
		}, nil
	case tokenSelf:
		return &Self{
			nodeBase: nodeBase{loc: tok.loc},
		}, nil
	case tokenSuper:
		next := p.pop()
		var index Node
		var id *Identifier
		switch next.kind {
		case tokenDot:
			fieldID, err := p.popExpect(tokenIdentifier)
			if err != nil {
				return nil, err
			}
			id = (*Identifier)(&fieldID.data)
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
		return &SuperIndex{
			nodeBase: nodeBase{loc: tok.loc},
			Index:    index,
			Id:       id,
		}, nil
	}

	return nil, makeStaticError(fmt.Sprintf("INTERNAL ERROR: Unknown tok kind: %v", tok.kind), tok.loc)
}

func (p *parser) parsingFailure(msg string, tok *token) (Node, error) {
	return nil, makeStaticError(msg, tok.loc)
}

func (p *parser) parse(prec precedence) (Node, error) {
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
		var msg Node
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
		return &Assert{
			nodeBase: nodeBase{loc: locFromTokenAST(begin, rest)},
			Cond:     cond,
			Message:  msg,
			Rest:     rest,
		}, nil

	case tokenError:
		p.pop()
		expr, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &Error{
			nodeBase: nodeBase{loc: locFromTokenAST(begin, expr)},
			Expr:     expr,
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
		var branchFalse Node
		lr := locFromTokenAST(begin, branchTrue)
		if p.peek().kind == tokenElse {
			p.pop()
			branchFalse, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			lr = locFromTokenAST(begin, branchFalse)
		}
		return &Conditional{
			nodeBase:    nodeBase{loc: lr},
			Cond:        cond,
			BranchTrue:  branchTrue,
			BranchFalse: branchFalse,
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
			return &Function{
				nodeBase:      nodeBase{loc: locFromTokenAST(begin, body)},
				Parameters:    params,
				TrailingComma: gotComma,
				Body:          body,
			}, nil
		}
		return nil, makeStaticError(fmt.Sprintf("Expected ( but got %v", next), next.loc)

	case tokenImport:
		p.pop()
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		if lit, ok := body.(*LiteralString); ok {
			return &Import{
				nodeBase: nodeBase{loc: locFromTokenAST(begin, body)},
				File:     lit.Value,
			}, nil
		}
		return nil, makeStaticError("Computed imports are not allowed", *body.Loc())

	case tokenImportStr:
		p.pop()
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		if lit, ok := body.(*LiteralString); ok {
			return &ImportStr{
				nodeBase: nodeBase{loc: locFromTokenAST(begin, body)},
				File:     lit.Value,
			}, nil
		}
		return nil, makeStaticError("Computed imports are not allowed", *body.Loc())

	case tokenLocal:
		p.pop()
		var binds LocalBinds
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
		return &Local{
			nodeBase: nodeBase{loc: locFromTokenAST(begin, body)},
			Binds:    binds,
			Body:     body,
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
				return &Unary{
					nodeBase: nodeBase{loc: locFromTokenAST(op, expr)},
					Op:       uop,
					Expr:     expr,
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

			var bop BinaryOp

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
				if p.peek().data == "::" {
					// Special case for [e::]
					// We need to stop parsing e when we see the :: and
					// avoid tripping the op_is_binary test below.
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
				// handle slice
				var indexes [3]Node
				colonsConsumed := 0

				var end *token
				readyForNextIndex := true
				for colonsConsumed < 3 {
					if p.peek().kind == tokenBracketR {
						end = p.pop()
						break
					} else if p.peek().data == ":" {
						colonsConsumed++
						end = p.pop()
						readyForNextIndex = true
					} else if p.peek().data == "::" {
						colonsConsumed += 2
						end = p.pop()
						readyForNextIndex = true
					} else if readyForNextIndex {
						indexes[colonsConsumed], err = p.parse(maxPrecedence)
						if err != nil {
							return nil, err
						}
						readyForNextIndex = false
					} else {
						return nil, p.unexpectedTokenError(tokenBracketR, p.peek())
					}
				}
				if colonsConsumed > 2 {
					// example: target[42:42:42:42]
					return p.parsingFailure("Invalid slice: too many colons", end)
				}
				if colonsConsumed == 0 && readyForNextIndex {
					// example: target[]
					return p.parsingFailure("Index requires an expression", end)
				}
				isSlice := colonsConsumed > 0

				if isSlice {
					lhs = &Slice{
						nodeBase:   nodeBase{loc: locFromTokens(begin, end)},
						Target:     lhs,
						BeginIndex: indexes[0],
						EndIndex:   indexes[1],
						Step:       indexes[2],
					}
				} else {
					lhs = &Index{
						nodeBase: nodeBase{loc: locFromTokens(begin, end)},
						Target:   lhs,
						Index:    indexes[0],
					}
				}
			case tokenDot:
				fieldID, err := p.popExpect(tokenIdentifier)
				if err != nil {
					return nil, err
				}
				id := Identifier(fieldID.data)
				lhs = &Index{
					nodeBase: nodeBase{loc: locFromTokens(begin, fieldID)},
					Target:   lhs,
					Id:       &id,
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
				lhs = &Apply{
					nodeBase:      nodeBase{loc: locFromTokens(begin, end)},
					Target:        lhs,
					Arguments:     args,
					TrailingComma: gotComma,
					TailStrict:    tailStrict,
				}
			case tokenBraceL:
				obj, end, err := p.parseObjectRemainder(op)
				if err != nil {
					return nil, err
				}
				lhs = &ApplyBrace{
					nodeBase: nodeBase{loc: locFromTokens(begin, end)},
					Left:     lhs,
					Right:    obj,
				}
			default:
				rhs, err := p.parse(prec - 1)
				if err != nil {
					return nil, err
				}
				lhs = &Binary{
					nodeBase: nodeBase{loc: locFromTokenAST(begin, rhs)},
					Left:     lhs,
					Op:       bop,
					Right:    rhs,
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------

func parse(t tokens) (Node, error) {
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
