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

// Identifier represents a variable / parameter / field name.
//+gen set
type Identifier string
type Identifiers []Identifier

// TODO(jbeda) implement interning of identifiers if necessary.  The C++
// version does so.

// ---------------------------------------------------------------------------

type Node interface {
	Loc() *LocationRange
	FreeVariables() Identifiers
	setFreeVariables(Identifiers)
}
type Nodes []Node

// ---------------------------------------------------------------------------

type nodeBase struct {
	loc           LocationRange
	freeVariables Identifiers
}

func (n *nodeBase) Loc() *LocationRange {
	return &n.loc
}

func (n *nodeBase) FreeVariables() Identifiers {
	return n.freeVariables
}

func (n *nodeBase) setFreeVariables(idents Identifiers) {
	n.freeVariables = idents
}

// ---------------------------------------------------------------------------

// +gen stringer
type CompKind int

const (
	CompFor CompKind = iota
	CompIf
)

// TODO(sbarzowski) separate types for two kinds
// TODO(sbarzowski) bonus points for attaching ifs to the previous for
type CompSpec struct {
	Kind    CompKind
	VarName *Identifier // nil when kind != compSpecFor
	Expr    Node
}
type CompSpecs []CompSpec

// ---------------------------------------------------------------------------

// Apply represents a function call
type Apply struct {
	nodeBase
	Target        Node
	Arguments     Nodes
	TrailingComma bool
	TailStrict    bool
	// TODO(sbarzowski) support named arguments
}

// ---------------------------------------------------------------------------

// ApplyBrace represents e { }.  Desugared to e + { }.
type ApplyBrace struct {
	nodeBase
	Left  Node
	Right Node
}

// ---------------------------------------------------------------------------

// Array represents array constructors [1, 2, 3].
type Array struct {
	nodeBase
	Elements      Nodes
	TrailingComma bool
}

// ---------------------------------------------------------------------------

// ArrayComp represents array comprehensions (which are like Python list
// comprehensions)
type ArrayComp struct {
	nodeBase
	Body          Node
	TrailingComma bool
	Specs         CompSpecs
}

// ---------------------------------------------------------------------------

// Assert represents an assert expression (not an object-level assert).
//
// After parsing, message can be nil indicating that no message was
// specified. This AST is elimiated by desugaring.
type Assert struct {
	nodeBase
	Cond    Node
	Message Node
	Rest    Node
}

// ---------------------------------------------------------------------------

type BinaryOp int

const (
	BopMult BinaryOp = iota
	BopDiv
	BopPercent

	BopPlus
	BopMinus

	BopShiftL
	BopShiftR

	BopGreater
	BopGreaterEq
	BopLess
	BopLessEq

	BopManifestEqual
	BopManifestUnequal

	BopBitwiseAnd
	BopBitwiseXor
	BopBitwiseOr

	BopAnd
	BopOr
)

var bopStrings = []string{
	BopMult:    "*",
	BopDiv:     "/",
	BopPercent: "%",

	BopPlus:  "+",
	BopMinus: "-",

	BopShiftL: "<<",
	BopShiftR: ">>",

	BopGreater:   ">",
	BopGreaterEq: ">=",
	BopLess:      "<",
	BopLessEq:    "<=",

	BopManifestEqual:   "==",
	BopManifestUnequal: "!=",

	BopBitwiseAnd: "&",
	BopBitwiseXor: "^",
	BopBitwiseOr:  "|",

	BopAnd: "&&",
	BopOr:  "||",
}

var bopMap = map[string]BinaryOp{
	"*": BopMult,
	"/": BopDiv,
	"%": BopPercent,

	"+": BopPlus,
	"-": BopMinus,

	"<<": BopShiftL,
	">>": BopShiftR,

	">":  BopGreater,
	">=": BopGreaterEq,
	"<":  BopLess,
	"<=": BopLessEq,

	"==": BopManifestEqual,
	"!=": BopManifestUnequal,

	"&": BopBitwiseAnd,
	"^": BopBitwiseXor,
	"|": BopBitwiseOr,

	"&&": BopAnd,
	"||": BopOr,
}

func (b BinaryOp) String() string {
	if b < 0 || int(b) >= len(bopStrings) {
		panic(fmt.Sprintf("INTERNAL ERROR: Unrecognised binary operator: %d", b))
	}
	return bopStrings[b]
}

// Binary represents binary operators.
type Binary struct {
	nodeBase
	Left  Node
	Op    BinaryOp
	Right Node
}

// ---------------------------------------------------------------------------

// Conditional represents if/then/else.
//
// After parsing, branchFalse can be nil indicating that no else branch
// was specified.  The desugarer fills this in with a LiteralNull
type Conditional struct {
	nodeBase
	Cond        Node
	BranchTrue  Node
	BranchFalse Node
}

// ---------------------------------------------------------------------------

// Dollar represents the $ keyword
type Dollar struct{ nodeBase }

// ---------------------------------------------------------------------------

// Error represents the error e.
type Error struct {
	nodeBase
	Expr Node
}

// ---------------------------------------------------------------------------

// Function represents a function definition
type Function struct {
	nodeBase
	Parameters    Identifiers // TODO(sbarzowski) support default arguments
	TrailingComma bool
	Body          Node
}

// ---------------------------------------------------------------------------

// Import represents import "file".
type Import struct {
	nodeBase
	File string
}

// ---------------------------------------------------------------------------

// ImportStr represents importstr "file".
type ImportStr struct {
	nodeBase
	File string
}

// ---------------------------------------------------------------------------

// Index represents both e[e] and the syntax sugar e.f.
//
// One of index and id will be nil before desugaring.  After desugaring id
// will be nil.
type Index struct {
	nodeBase
	Target Node
	Index  Node
	Id     *Identifier
}

type Slice struct {
	nodeBase
	Target Node

	// Each of these can be nil
	BeginIndex Node
	EndIndex   Node
	Step       Node
}

// ---------------------------------------------------------------------------

// LocalBind is a helper struct for astLocal
type LocalBind struct {
	Variable      Identifier
	Body          Node
	FunctionSugar bool
	Params        Identifiers // if functionSugar is true
	TrailingComma bool
}
type LocalBinds []LocalBind

// Local represents local x = e; e.  After desugaring, functionSugar is false.
type Local struct {
	nodeBase
	Binds LocalBinds
	Body  Node
}

// ---------------------------------------------------------------------------

// LiteralBoolean represents true and false
type LiteralBoolean struct {
	nodeBase
	Value bool
}

// ---------------------------------------------------------------------------

// LiteralNull represents the null keyword
type LiteralNull struct{ nodeBase }

// ---------------------------------------------------------------------------

// LiteralNumber represents a JSON number
type LiteralNumber struct {
	nodeBase
	Value          float64
	OriginalString string
}

// ---------------------------------------------------------------------------

// +gen stringer
type LiteralStringKind int

const (
	StringSingle LiteralStringKind = iota
	StringDouble
	StringBlock
	VerbatimStringDouble
	VerbatimStringSingle
)

// LiteralString represents a JSON string
type LiteralString struct {
	nodeBase
	Value       string
	Kind        LiteralStringKind
	BlockIndent string
}

// ---------------------------------------------------------------------------

// +gen stringer
type ObjectFieldKind int

const (
	ObjectAssert    ObjectFieldKind = iota // assert expr2 [: expr3]  where expr3 can be nil
	ObjectFieldID                          // id:[:[:]] expr2
	ObjectFieldExpr                        // '['expr1']':[:[:]] expr2
	ObjectFieldStr                         // expr1:[:[:]] expr2
	ObjectLocal                            // local id = expr2
)

// +gen stringer
type ObjectFieldHide int

const (
	ObjectFieldHidden  ObjectFieldHide = iota // f:: e
	ObjectFieldInherit                        // f: e
	ObjectFieldVisible                        // f::: e
)

// TODO(sbarzowski) consider having separate types for various kinds
type ObjectField struct {
	Kind          ObjectFieldKind
	Hide          ObjectFieldHide // (ignore if kind != astObjectField*)
	SuperSugar    bool            // +:  (ignore if kind != astObjectField*)
	MethodSugar   bool            // f(x, y, z): ...  (ignore if kind  == astObjectAssert)
	Expr1         Node            // Not in scope of the object
	Id            *Identifier
	Ids           Identifiers // If methodSugar == true then holds the params.
	TrailingComma bool        // If methodSugar == true then remembers the trailing comma
	Expr2, Expr3  Node        // In scope of the object (can see self).
}

// TODO(jbeda): Add the remaining constructor helpers here

func ObjectFieldLocal(methodSugar bool, id *Identifier, ids Identifiers, trailingComma bool, body Node) ObjectField {
	return ObjectField{ObjectLocal, ObjectFieldVisible, false, methodSugar, nil, id, ids, trailingComma, body, nil}
}

func ObjectFieldLocalNoMethod(id *Identifier, body Node) ObjectField {
	return ObjectField{ObjectLocal, ObjectFieldVisible, false, false, nil, id, Identifiers{}, false, body, nil}
}

type ObjectFields []ObjectField

// Object represents object constructors { f: e ... }.
//
// The trailing comma is only allowed if len(fields) > 0.  Converted to
// DesugaredObject during desugaring.
type Object struct {
	nodeBase
	Fields        ObjectFields
	TrailingComma bool
}

// ---------------------------------------------------------------------------

type DesugaredObjectField struct {
	Hide ObjectFieldHide
	Name Node
	Body Node
}
type DesugaredObjectFields []DesugaredObjectField

// DesugaredObject represents object constructors { f: e ... } after
// desugaring.
//
// The assertions either return true or raise an error.
type DesugaredObject struct {
	nodeBase
	Asserts Nodes
	Fields  DesugaredObjectFields
}

// ---------------------------------------------------------------------------

// ObjectComp represents object comprehension
//   { [e]: e for x in e for.. if... }.
type ObjectComp struct {
	nodeBase
	Fields        ObjectFields
	TrailingComma bool
	Specs         CompSpecs
}

// ---------------------------------------------------------------------------

// ObjectComprehensionSimple represents post-desugaring object
// comprehension { [e]: e for x in e }.
type ObjectComprehensionSimple struct {
	nodeBase
	Field Node
	Value Node
	Id    Identifier
	Array Node
}

// ---------------------------------------------------------------------------

// Self represents the self keyword.
type Self struct{ nodeBase }

// ---------------------------------------------------------------------------

// SuperIndex represents the super[e] and super.f constructs.
//
// Either index or identifier will be set before desugaring.  After desugaring, id will be
// nil.
type SuperIndex struct {
	nodeBase
	Index Node
	Id    *Identifier
}

// ---------------------------------------------------------------------------

type UnaryOp int

const (
	UopNot UnaryOp = iota
	UopBitwiseNot
	UopPlus
	UopMinus
)

var uopStrings = []string{
	UopNot:        "!",
	UopBitwiseNot: "~",
	UopPlus:       "+",
	UopMinus:      "-",
}

var uopMap = map[string]UnaryOp{
	"!": UopNot,
	"~": UopBitwiseNot,
	"+": UopPlus,
	"-": UopMinus,
}

func (u UnaryOp) String() string {
	if u < 0 || int(u) >= len(uopStrings) {
		panic(fmt.Sprintf("INTERNAL ERROR: Unrecognised unary operator: %d", u))
	}
	return uopStrings[u]
}

// Unary represents unary operators.
type Unary struct {
	nodeBase
	Op   UnaryOp
	Expr Node
}

// ---------------------------------------------------------------------------

// Var represents variables.
type Var struct {
	nodeBase
	Id Identifier
}

// ---------------------------------------------------------------------------
