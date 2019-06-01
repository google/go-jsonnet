package ast

// StdAst is the AST for the standard library.
//
// Its inital value is the Jsonnet null value, such that the standard library is not available
// during evaluation. Set this variable to point at a non-null AST node to make that tree available
// as the standard library.
var StdAst Node = &LiteralNull{}
