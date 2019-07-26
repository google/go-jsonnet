package transformations

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/parser"
)

func snippetToRawAST(filename string, snippet string) (ast.Node, error) {
	tokens, err := parser.Lex(filename, snippet)
	if err != nil {
		return nil, err
	}
	return parser.Parse(tokens)
}

// SnippetToAST converts Jsonnet code snippet to desugared and analyzed AST
func SnippetToAST(filename string, snippet string) (ast.Node, error) {
	node, err := snippetToRawAST(filename, snippet)
	if err != nil {
		return nil, err
	}
	err = Desugar(&node)
	if err != nil {
		return nil, err
	}
	err = Analyze(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}
