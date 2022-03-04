package parser

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
	"github.com/google/go-jsonnet/internal/program"
)

func SnippetToRawAST(snippet string) (ast.Node, ast.Fodder, error) {
	return parser.SnippetToRawAST("", "", snippet)
}

func SnippetToAst(snippet string) (ast.Node, error) {
	return program.SnippetToAST("", "", snippet)
}

func PreprocessAst(node *ast.Node) error {
	return program.PreprocessAst(node)
}

func StringUnescape(loc *ast.LocationRange, s string) (string, error) {
	return parser.StringUnescape(loc, s)
}
