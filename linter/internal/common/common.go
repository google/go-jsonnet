package common

import (
	"github.com/google/go-jsonnet/ast"
)

// VariableKind allows distinguishing various kinds of variables.
type VariableKind int

const (
	// VarRegular is a "normal" variable with a definition in the code.
	VarRegular VariableKind = iota
	// VarParam is a function parameter.
	VarParam
	// VarStdlib is a special `std` variable.
	VarStdlib
)

// Variable is a representation of a variable somewhere in the code.
type Variable struct {
	Name         ast.Identifier
	BindNode     ast.Node
	Occurences   []ast.Node
	VariableKind VariableKind
	LocRange     ast.LocationRange
}

// VariableInfo holds information about a variables from one file
type VariableInfo struct {
	Variables []*Variable

	// Variable information at every use site.
	// More precisely it maps every *ast.Var to the variable.
	VarAt map[ast.Node]*Variable
}
