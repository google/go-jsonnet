// Package program provides API for AST pre-processing (desugaring, static analysis).
package program

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
)

// SnippetToAST converts a Jsonnet code snippet to a desugared and analyzed AST.
func SnippetToAST(diagnosticFilename ast.DiagnosticFileName, importedFilename, snippet string, globalVars ...ast.Identifier) (ast.Node, error) {
	node, _, err := parser.SnippetToRawAST(diagnosticFilename, importedFilename, snippet)
	if err != nil {
		return nil, err
	}
	if err := PreprocessAst(&node, globalVars...); err != nil {
		return nil, err
	}
	return node, nil
}

func PreprocessAst(node *ast.Node, globalVars ...ast.Identifier) error {
	err := desugarAST(node)
	if err != nil {
		return err
	}
	err = analyze(*node, globalVars...)
	if err != nil {
		return err
	}
	return nil
}
