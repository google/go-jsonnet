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

// identifier represents a variable / parameter / field name.
//+gen set
type identifier string
type identifiers []identifier

// TODO(jbeda) implement interning of identifiers if necessary.  The C++
// version does so.

// ---------------------------------------------------------------------------

type astNode interface {
	Loc() *LocationRange
	FreeVariables() identifiers
}
type astNodes []astNode

// ---------------------------------------------------------------------------

type astNodeBase struct {
	loc           LocationRange
	freeVariables identifiers
}

func (n *astNodeBase) Loc() *LocationRange {
	return &n.loc
}

func (n *astNodeBase) FreeVariables() identifiers {
	return n.freeVariables
}

// ---------------------------------------------------------------------------

// +gen stringer
type astCompKind int

const (
	astCompFor astCompKind = iota
	astCompIf
)

type astCompSpec struct {
	kind    astCompKind
	varName *identifier // nil when kind != compSpecFor
	expr    astNode
}
type astCompSpecs []astCompSpec

// ---------------------------------------------------------------------------

// astApply represents a function call
type astApply struct {
	astNodeBase
	target        astNode
	arguments     astNodes
	trailingComma bool
	tailStrict    bool
}

// ---------------------------------------------------------------------------

// astApplyBrace represents e { }.  Desugared to e + { }.
type astApplyBrace struct {
	astNodeBase
	left  astNode
	right astNode
}

// ---------------------------------------------------------------------------

// astArray represents array constructors [1, 2, 3].
type astArray struct {
	astNodeBase
	elements      astNodes
	trailingComma bool
}

// ---------------------------------------------------------------------------

// astArrayComp represents array comprehensions (which are like Python list
// comprehensions)
type astArrayComp struct {
	astNodeBase
	body          astNode
	trailingComma bool
	specs         astCompSpecs
}

// ---------------------------------------------------------------------------

// astAssert represents an assert expression (not an object-level assert).
//
// After parsing, message can be nil indicating that no message was
// specified. This AST is elimiated by desugaring.
type astAssert struct {
	astNodeBase
	cond    astNode
	message astNode
	rest    astNode
}

// ---------------------------------------------------------------------------

type binaryOp int

const (
	bopMult binaryOp = iota
	bopDiv
	bopPercent

	bopPlus
	bopMinus

	bopShiftL
	bopShiftR

	bopGreater
	bopGreaterEq
	bopLess
	bopLessEq

	bopManifestEqual
	bopManifestUnequal

	bopBitwiseAnd
	bopBitwiseXor
	bopBitwiseOr

	bopAnd
	bopOr
)

var bopStrings = []string{
	bopMult:    "*",
	bopDiv:     "/",
	bopPercent: "%",

	bopPlus:  "+",
	bopMinus: "-",

	bopShiftL: "<<",
	bopShiftR: ">>",

	bopGreater:   ">",
	bopGreaterEq: ">=",
	bopLess:      "<",
	bopLessEq:    "<=",

	bopManifestEqual:   "==",
	bopManifestUnequal: "!=",

	bopBitwiseAnd: "&",
	bopBitwiseXor: "^",
	bopBitwiseOr:  "|",

	bopAnd: "&&",
	bopOr:  "||",
}

var bopMap = map[string]binaryOp{
	"*": bopMult,
	"/": bopDiv,
	"%": bopPercent,

	"+": bopPlus,
	"-": bopMinus,

	"<<": bopShiftL,
	">>": bopShiftR,

	">":  bopGreater,
	">=": bopGreaterEq,
	"<":  bopLess,
	"<=": bopLessEq,

	"==": bopManifestEqual,
	"!=": bopManifestUnequal,

	"&": bopBitwiseAnd,
	"^": bopBitwiseXor,
	"|": bopBitwiseOr,

	"&&": bopAnd,
	"||": bopOr,
}

func (b binaryOp) String() string {
	if b < 0 || int(b) >= len(bopStrings) {
		panic(fmt.Sprintf("INTERNAL ERROR: Unrecognised binary operator: %d", b))
	}
	return bopStrings[b]
}

// astBinary represents binary operators.
type astBinary struct {
	astNodeBase
	left  astNode
	op    binaryOp
	right astNode
}

// ---------------------------------------------------------------------------

// astBuiltin represents built-in functions.
//
// There is no parse rule to build this AST.  Instead, it is used to build the
// std object in the interpreter.
type astBuiltin struct {
	astNodeBase
	id     int
	params identifiers
}

// ---------------------------------------------------------------------------

// astConditional represents if/then/else.
//
// After parsing, branchFalse can be nil indicating that no else branch
// was specified.  The desugarer fills this in with a LiteralNull
type astConditional struct {
	astNodeBase
	cond        astNode
	branchTrue  astNode
	branchFalse astNode
}

// ---------------------------------------------------------------------------

// astDollar represents the $ keyword
type astDollar struct{ astNodeBase }

// ---------------------------------------------------------------------------

// astError represents the error e.
type astError struct {
	astNodeBase
	expr astNode
}

// ---------------------------------------------------------------------------

// astFunction represents a function call. (jbeda: or is it function defn?)
type astFunction struct {
	astNodeBase
	parameters    identifiers
	trailingComma bool
	body          astNode
}

// ---------------------------------------------------------------------------

// astImport represents import "file".
type astImport struct {
	astNodeBase
	file string
}

// ---------------------------------------------------------------------------

// astImportStr represents importstr "file".
type astImportStr struct {
	astNodeBase
	file string
}

// ---------------------------------------------------------------------------

// astIndex represents both e[e] and the syntax sugar e.f.
//
// One of index and id will be nil before desugaring.  After desugaring id
// will be nil.
type astIndex struct {
	astNodeBase
	target astNode
	index  astNode
	id     *identifier
}

type astSlice struct {
	astNodeBase
	target astNode

	// Each of these can be nil
	beginIndex astNode
	endIndex   astNode
	step       astNode
}

// ---------------------------------------------------------------------------

// astLocalBind is a helper struct for astLocal
type astLocalBind struct {
	variable      identifier
	body          astNode
	functionSugar bool
	params        identifiers // if functionSugar is true
	trailingComma bool
}
type astLocalBinds []astLocalBind

// astLocal represents local x = e; e.  After desugaring, functionSugar is false.
type astLocal struct {
	astNodeBase
	binds astLocalBinds
	body  astNode
}

// ---------------------------------------------------------------------------

// astLiteralBoolean represents true and false
type astLiteralBoolean struct {
	astNodeBase
	value bool
}

// ---------------------------------------------------------------------------

// astLiteralNull represents the null keyword
type astLiteralNull struct{ astNodeBase }

// ---------------------------------------------------------------------------

// astLiteralNumber represents a JSON number
type astLiteralNumber struct {
	astNodeBase
	value          float64
	originalString string
}

// ---------------------------------------------------------------------------

// +gen stringer
type astLiteralStringKind int

const (
	astStringSingle astLiteralStringKind = iota
	astStringDouble
	astStringBlock
	astVerbatimStringDouble
	astVerbatimStringSingle
)

// astLiteralString represents a JSON string
type astLiteralString struct {
	astNodeBase
	value       string
	kind        astLiteralStringKind
	blockIndent string
}

// ---------------------------------------------------------------------------

// +gen stringer
type astObjectFieldKind int

const (
	astObjectAssert    astObjectFieldKind = iota // assert expr2 [: expr3]  where expr3 can be nil
	astObjectFieldID                             // id:[:[:]] expr2
	astObjectFieldExpr                           // '['expr1']':[:[:]] expr2
	astObjectFieldStr                            // expr1:[:[:]] expr2
	astObjectLocal                               // local id = expr2
)

// +gen stringer
type astObjectFieldHide int

const (
	astObjectFieldHidden  astObjectFieldHide = iota // f:: e
	astObjectFieldInherit                           // f: e
	astObjectFieldVisible                           // f::: e
)

// TODO(sbarzowski) consider having separate types for various kinds
type astObjectField struct {
	kind          astObjectFieldKind
	hide          astObjectFieldHide // (ignore if kind != astObjectField*)
	superSugar    bool               // +:  (ignore if kind != astObjectField*)
	methodSugar   bool               // f(x, y, z): ...  (ignore if kind  == astObjectAssert)
	expr1         astNode            // Not in scope of the object
	id            *identifier
	ids           identifiers // If methodSugar == true then holds the params.
	trailingComma bool        // If methodSugar == true then remembers the trailing comma
	expr2, expr3  astNode     // In scope of the object (can see self).
}

// TODO(jbeda): Add the remaining constructor helpers here

func astObjectFieldLocal(methodSugar bool, id *identifier, ids identifiers, trailingComma bool, body astNode) astObjectField {
	return astObjectField{astObjectLocal, astObjectFieldVisible, false, methodSugar, nil, id, ids, trailingComma, body, nil}
}

func astObjectFieldLocalNoMethod(id *identifier, body astNode) astObjectField {
	return astObjectField{astObjectLocal, astObjectFieldVisible, false, false, nil, id, identifiers{}, false, body, nil}
}

type astObjectFields []astObjectField

// astObject represents object constructors { f: e ... }.
//
// The trailing comma is only allowed if len(fields) > 0.  Converted to
// DesugaredObject during desugaring.
type astObject struct {
	astNodeBase
	fields        astObjectFields
	trailingComma bool
}

// ---------------------------------------------------------------------------

type astDesugaredObjectField struct {
	hide astObjectFieldHide
	name astNode
	body astNode
}
type astDesugaredObjectFields []astDesugaredObjectField

// astDesugaredObject represents object constructors { f: e ... } after
// desugaring.
//
// The assertions either return true or raise an error.
type astDesugaredObject struct {
	astNodeBase
	asserts astNodes
	fields  astDesugaredObjectFields
}

// ---------------------------------------------------------------------------

// astObjectComp represents object comprehension
//   { [e]: e for x in e for.. if... }.
type astObjectComp struct {
	astNodeBase
	fields        astObjectFields
	trailingComma bool
	specs         astCompSpecs
}

// ---------------------------------------------------------------------------

// astObjectComprehensionSimple represents post-desugaring object
// comprehension { [e]: e for x in e }.
type astObjectComprehensionSimple struct {
	astNodeBase
	field astNode
	value astNode
	id    identifier
	array astNode
}

// ---------------------------------------------------------------------------

// astSelf represents the self keyword.
type astSelf struct{ astNodeBase }

// ---------------------------------------------------------------------------

// astSuperIndex represents the super[e] and super.f constructs.
//
// Either index or identifier will be set before desugaring.  After desugaring, id will be
// nil.
type astSuperIndex struct {
	astNodeBase
	index astNode
	id    *identifier
}

// ---------------------------------------------------------------------------

type unaryOp int

const (
	uopNot unaryOp = iota
	uopBitwiseNot
	uopPlus
	uopMinus
)

var uopStrings = []string{
	uopNot:        "!",
	uopBitwiseNot: "~",
	uopPlus:       "+",
	uopMinus:      "-",
}

var uopMap = map[string]unaryOp{
	"!": uopNot,
	"~": uopBitwiseNot,
	"+": uopPlus,
	"-": uopMinus,
}

func (u unaryOp) String() string {
	if u < 0 || int(u) >= len(uopStrings) {
		panic(fmt.Sprintf("INTERNAL ERROR: Unrecognised unary operator: %d", u))
	}
	return uopStrings[u]
}

// astUnary represents unary operators.
type astUnary struct {
	astNodeBase
	op   unaryOp
	expr astNode
}

// ---------------------------------------------------------------------------

// astVar represents variables.
type astVar struct {
	astNodeBase
	id identifier
}

// ---------------------------------------------------------------------------
