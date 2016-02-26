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

type precedence int

const (
	applyPrecedence      precedence = 2  // Function calls and indexing.
	unaryPrecedence      precedence = 4  // Logical and bitwise negation, unary + -
	beforeElsePrecedence precedence = 15 // True branch of an if.
	maxPrecedence        precedence = 16 // Local, If, Import, Function, Error
)

// ---------------------------------------------------------------------------

func makeUnexpectedError(t token, while string) error {
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
			return identifiers{}, false, makeStaticError(fmt.Sprintf("Not an identifier: %v", n), *n.Loc())
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
			// got_comma can be true or false here.
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
			astNodeBase: astNodeBase{locFromTokenAST(begin, rest)},
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
			astNodeBase: astNodeBase{locFromTokenAST(begin, expr)},
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
			astNodeBase: astNodeBase{lr},
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
				astNodeBase:   astNodeBase{locFromTokenAST(begin, body)},
				parameters:    params,
				trailingComma: gotComma,
				body:          body,
			}, nil
		}

		return nil, makeStaticError(fmt.Sprintf("Expected ( but got %v", next), next.loc)
	}

	return nil, nil
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
