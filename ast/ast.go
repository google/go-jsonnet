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

package ast

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

type Context *string

type Node interface {
	Context() Context
	Loc() *LocationRange
	FreeVariables() Identifiers
	SetFreeVariables(Identifiers)
	SetContext(Context)
}
type Nodes []Node

// ---------------------------------------------------------------------------

type NodeBase struct {
	loc           LocationRange
	context       Context
	freeVariables Identifiers
}

func NewNodeBase(loc LocationRange, freeVariables Identifiers) NodeBase {
	return NodeBase{
		loc:           loc,
		freeVariables: freeVariables,
	}
}

func NewNodeBaseLoc(loc LocationRange) NodeBase {
	return NodeBase{
		loc:           loc,
		freeVariables: []Identifier{},
	}
}

func (n *NodeBase) Loc() *LocationRange {
	return &n.loc
}

func (n *NodeBase) FreeVariables() Identifiers {
	return n.freeVariables
}

func (n *NodeBase) SetFreeVariables(idents Identifiers) {
	n.freeVariables = idents
}

func (n *NodeBase) Context() Context {
	return n.context
}

func (n *NodeBase) SetContext(context Context) {
	n.context = context
}

// ---------------------------------------------------------------------------

type IfSpec struct {
	Expr Node
}

// Example:
// expr for x in arr1 for y in arr2 for z in arr3
// The order is the same as in python, i.e. the leftmost is the outermost.
//
// Our internal representation reflects how they are semantically nested:
// ForSpec(z, outer=ForSpec(y, outer=ForSpec(x, outer=nil)))
// Any ifspecs are attached to the relevant ForSpec.
//
// Ifs are attached to the one on the left, for example:
// expr for x in arr1 for y in arr2 if x % 2 == 0 for z in arr3
// The if is attached to the y forspec.
//
// It desugares to:
// flatMap(\x ->
//         flatMap(\y ->
//                 flatMap(\z -> [expr], arr3)
//                 arr2)
//         arr3)
type ForSpec struct {
	VarName    Identifier
	Expr       Node
	Conditions []IfSpec
	Outer      *ForSpec
}

// ---------------------------------------------------------------------------

// Apply represents a function call
type Apply struct {
	NodeBase
	Target        Node
	Arguments     Arguments
	TrailingComma bool
	TailStrict    bool
}

type NamedArgument struct {
	Name Identifier
	Arg  Node
}

type Arguments struct {
	Positional Nodes
	Named      []NamedArgument
}

// ---------------------------------------------------------------------------

// ApplyBrace represents e { }.  Desugared to e + { }.
type ApplyBrace struct {
	NodeBase
	Left  Node
	Right Node
}

// ---------------------------------------------------------------------------

// Array represents array constructors [1, 2, 3].
type Array struct {
	NodeBase
	Elements      Nodes
	TrailingComma bool
}

// ---------------------------------------------------------------------------

// ArrayComp represents array comprehensions (which are like Python list
// comprehensions)
type ArrayComp struct {
	NodeBase
	Body          Node
	TrailingComma bool
	Spec          ForSpec
}

// ---------------------------------------------------------------------------

// Assert represents an assert expression (not an object-level assert).
//
// After parsing, message can be nil indicating that no message was
// specified. This AST is elimiated by desugaring.
type Assert struct {
	NodeBase
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
	BopIn

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
	BopIn:        "in",

	BopManifestEqual:   "==",
	BopManifestUnequal: "!=",

	BopBitwiseAnd: "&",
	BopBitwiseXor: "^",
	BopBitwiseOr:  "|",

	BopAnd: "&&",
	BopOr:  "||",
}

var BopMap = map[string]BinaryOp{
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
	"in": BopIn,

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
	NodeBase
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
	NodeBase
	Cond        Node
	BranchTrue  Node
	BranchFalse Node
}

// ---------------------------------------------------------------------------

// Dollar represents the $ keyword
type Dollar struct{ NodeBase }

// ---------------------------------------------------------------------------

// Error represents the error e.
type Error struct {
	NodeBase
	Expr Node
}

// ---------------------------------------------------------------------------

// Function represents a function definition
type Function struct {
	NodeBase
	Parameters    Parameters
	TrailingComma bool
	Body          Node
}

type NamedParameter struct {
	Name       Identifier
	DefaultArg Node
}

type Parameters struct {
	Required Identifiers
	Optional []NamedParameter
}

// ---------------------------------------------------------------------------

// Import represents import "file".
type Import struct {
	NodeBase
	File *LiteralString
}

// ---------------------------------------------------------------------------

// ImportStr represents importstr "file".
type ImportStr struct {
	NodeBase
	File *LiteralString
}

// ---------------------------------------------------------------------------

// Index represents both e[e] and the syntax sugar e.f.
//
// One of index and id will be nil before desugaring.  After desugaring id
// will be nil.
type Index struct {
	NodeBase
	Target Node
	Index  Node
	Id     *Identifier
}

type Slice struct {
	NodeBase
	Target Node

	// Each of these can be nil
	BeginIndex Node
	EndIndex   Node
	Step       Node
}

// ---------------------------------------------------------------------------

// LocalBind is a helper struct for astLocal
type LocalBind struct {
	Variable Identifier
	Body     Node
	Fun      *Function
}
type LocalBinds []LocalBind

// Local represents local x = e; e.  After desugaring, functionSugar is false.
type Local struct {
	NodeBase
	Binds LocalBinds
	Body  Node
}

// ---------------------------------------------------------------------------

// LiteralBoolean represents true and false
type LiteralBoolean struct {
	NodeBase
	Value bool
}

// ---------------------------------------------------------------------------

// LiteralNull represents the null keyword
type LiteralNull struct{ NodeBase }

// ---------------------------------------------------------------------------

// LiteralNumber represents a JSON number
type LiteralNumber struct {
	NodeBase
	Value          float64
	OriginalString string
}

// ---------------------------------------------------------------------------

type LiteralStringKind int

const (
	StringSingle LiteralStringKind = iota
	StringDouble
	StringBlock
	VerbatimStringDouble
	VerbatimStringSingle
)

func (k LiteralStringKind) FullyEscaped() bool {
	switch k {
	case StringSingle, StringDouble:
		return true
	case StringBlock, VerbatimStringDouble, VerbatimStringSingle:
		return false
	}
	panic(fmt.Sprintf("Unknown string kind: %v", k))
}

// LiteralString represents a JSON string
type LiteralString struct {
	NodeBase
	Value       string
	Kind        LiteralStringKind
	BlockIndent string
}

// ---------------------------------------------------------------------------

type ObjectFieldKind int

const (
	ObjectAssert    ObjectFieldKind = iota // assert expr2 [: expr3]  where expr3 can be nil
	ObjectFieldID                          // id:[:[:]] expr2
	ObjectFieldExpr                        // '['expr1']':[:[:]] expr2
	ObjectFieldStr                         // expr1:[:[:]] expr2
	ObjectLocal                            // local id = expr2
	ObjectNullID                           // null id
	ObjectNullExpr                         // null '['expr1']'
	ObjectNullStr                          // null expr1
)

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
	Method        *Function
	Expr1         Node // Not in scope of the object
	Id            *Identifier
	Params        *Parameters // If methodSugar == true then holds the params.
	TrailingComma bool        // If methodSugar == true then remembers the trailing comma
	Expr2, Expr3  Node        // In scope of the object (can see self).
}

func ObjectFieldLocalNoMethod(id *Identifier, body Node) ObjectField {
	return ObjectField{ObjectLocal, ObjectFieldVisible, false, false, nil, nil, id, nil, false, body, nil}
}

type ObjectFields []ObjectField

// Object represents object constructors { f: e ... }.
//
// The trailing comma is only allowed if len(fields) > 0.  Converted to
// DesugaredObject during desugaring.
type Object struct {
	NodeBase
	Fields        ObjectFields
	TrailingComma bool
}

// ---------------------------------------------------------------------------

type DesugaredObjectField struct {
	Hide      ObjectFieldHide
	Name      Node
	Body      Node
	PlusSuper bool
}
type DesugaredObjectFields []DesugaredObjectField

// DesugaredObject represents object constructors { f: e ... } after
// desugaring.
//
// The assertions either return true or raise an error.
type DesugaredObject struct {
	NodeBase
	Asserts Nodes
	Fields  DesugaredObjectFields
}

// ---------------------------------------------------------------------------

// ObjectComp represents object comprehension
//   { [e]: e for x in e for.. if... }.
type ObjectComp struct {
	NodeBase
	Fields        ObjectFields
	TrailingComma bool
	Spec          ForSpec
}

// ---------------------------------------------------------------------------

// Self represents the self keyword.
type Self struct{ NodeBase }

// ---------------------------------------------------------------------------

// SuperIndex represents the super[e] and super.f constructs.
//
// Either index or identifier will be set before desugaring.  After desugaring, id will be
// nil.
type SuperIndex struct {
	NodeBase
	Index Node
	Id    *Identifier
}

// Represents the e in super construct.
type InSuper struct {
	NodeBase
	Index Node
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

var UopMap = map[string]UnaryOp{
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
	NodeBase
	Op   UnaryOp
	Expr Node
}

// ---------------------------------------------------------------------------

// Var represents variables.
type Var struct {
	NodeBase
	Id Identifier
}

// ---------------------------------------------------------------------------
