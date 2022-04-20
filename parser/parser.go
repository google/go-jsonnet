package parser

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
	"github.com/google/go-jsonnet/internal/program"
)

func SnippetToRawAST(snippet, fileName string) (ast.Node, ast.Fodder, error) {
	return parser.SnippetToRawAST(ast.DiagnosticFileName(fileName), fileName, snippet)
}

func SnippetToAst(snippet, fileName string, globalVars ...ast.Identifier) (ast.Node, error) {
	return program.SnippetToAST(ast.DiagnosticFileName(fileName), fileName, snippet, globalVars...)
}

func PreprocessAst(node *ast.Node, globalVars ...ast.Identifier) error {
	return program.PreprocessAst(node, globalVars...)
}

func StringUnescape(loc *ast.LocationRange, s string) (string, error) {
	return parser.StringUnescape(loc, s)
}
