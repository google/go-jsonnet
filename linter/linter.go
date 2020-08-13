// Package linter analyses Jsonnet code for code "smells".
package linter

import (
	"io"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/errors"
	"github.com/google/go-jsonnet/internal/parser"

	"github.com/google/go-jsonnet/linter/internal/common"
	"github.com/google/go-jsonnet/linter/internal/traversal"
	"github.com/google/go-jsonnet/linter/internal/types"
	"github.com/google/go-jsonnet/linter/internal/utils"
	"github.com/google/go-jsonnet/linter/internal/variables"
)

// ErrorWriter encapsulates a writer and an error state indicating when at least
// one error has been written to the writer.
type ErrorWriter struct {
	ErrorsFound bool
	Writer      io.Writer
}

func (e *ErrorWriter) writeError(vm *jsonnet.VM, err errors.StaticError) {
	e.ErrorsFound = true
	_, writeErr := e.Writer.Write([]byte(vm.ErrorFormatter.Format(err) + "\n"))
	if writeErr != nil {
		panic(writeErr)
	}
}

// nodeWithLocation represents a Jsonnet program with its location
// for the importer.
type nodeWithLocation struct {
	node ast.Node
	path string
}

// Lint analyses a node and reports any issues it encounters to an error writer.
func lint(vm *jsonnet.VM, node nodeWithLocation, errWriter *ErrorWriter) {
	roots := make(map[string]ast.Node)
	roots[node.path] = node.node
	getImports(vm, node, roots, errWriter)

	variablesInFile := make(map[string]common.VariableInfo)

	std := common.Variable{
		Name:         "std",
		Occurences:   nil,
		VariableKind: common.VarStdlib,
	}

	findVariables := func(node nodeWithLocation) *common.VariableInfo {
		return variables.FindVariables(node.node, variables.Environment{"std": &std})
	}

	variableInfo := findVariables(node)
	for importedPath, rootNode := range roots {
		variablesInFile[importedPath] = *findVariables(nodeWithLocation{rootNode, importedPath})
	}

	for _, v := range variableInfo.Variables {
		if len(v.Occurences) == 0 && v.VariableKind == common.VarRegular && v.Name != "$" {
			errWriter.writeError(vm, errors.MakeStaticError("Unused variable: "+string(v.Name), v.LocRange))
		}
	}
	ec := utils.ErrCollector{}

	vars := make(map[string]map[ast.Node]*common.Variable)
	for importedPath, info := range variablesInFile {
		vars[importedPath] = info.VarAt
	}

	types.Check(node.node, roots, vars, func(currentPath, importedPath string) ast.Node {
		node, _, err := vm.ImportAST(currentPath, importedPath)
		if err != nil {
			return nil
		}
		return node
	}, &ec)

	traversal.Traverse(node.node, &ec)

	for _, err := range ec.Errs {
		errWriter.writeError(vm, err)
	}
}

func getImports(vm *jsonnet.VM, node nodeWithLocation, roots map[string]ast.Node, errWriter *ErrorWriter) {
	// TODO(sbarzowski) consider providing some way to disable warnings about nonexistent imports
	// At least for 3rd party code.
	// Perhaps there may be some valid use cases for conditional imports where one of the imported
	// files doesn't exist.
	currentPath := node.path
	switch node := node.node.(type) {
	case *ast.Import:
		p := node.File.Value
		contents, foundAt, err := vm.ImportAST(currentPath, p)
		if err != nil {
			errWriter.writeError(vm, errors.MakeStaticError(err.Error(), *node.Loc()))
		} else {
			if _, visited := roots[foundAt]; !visited {
				roots[foundAt] = contents
				getImports(vm, nodeWithLocation{contents, foundAt}, roots, errWriter)
			}
		}
	case *ast.ImportStr:
		p := node.File.Value
		_, err := vm.ResolveImport(currentPath, p)
		if err != nil {
			errWriter.writeError(vm, errors.MakeStaticError(err.Error(), *node.Loc()))
		}
	default:
		for _, c := range parser.Children(node) {
			getImports(vm, nodeWithLocation{c, currentPath}, roots, errWriter)
		}
	}
}

// LintSnippet checks for problems in a single code snippet.
func LintSnippet(vm *jsonnet.VM, output io.Writer, filename, code string) bool {
	errWriter := ErrorWriter{
		Writer:      output,
		ErrorsFound: false,
	}

	node, err := jsonnet.SnippetToAST(filename, code)
	if err != nil {
		errWriter.writeError(vm, err.(errors.StaticError)) // ugly but true
		return true
	}
	lint(vm, nodeWithLocation{node, filename}, &errWriter)
	return errWriter.ErrorsFound
}
