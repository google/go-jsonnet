package types

import (
	"fmt"
	"io"
	"time"

	"github.com/google/go-jsonnet/ast"
)

type typeGraph struct {
	_placeholders   []typePlaceholder
	exprPlaceholder map[ast.Node]placeholderID

	topoOrder []placeholderID
	sccOf     []stronglyConnectedComponentID

	elementType []*elementDesc

	upperBound []TypeDesc

	// Additional information about the program
	// varAt map[ast.Node]*common.Variable

	// TODO(sbarzowski) what was this for?
	importFunc ImportFunc

	// For performance tuning
	timeStats timeStats

	counters counters
}

// Subject to change at any point. For fine-tuning only.
type timeStats struct {
	simplifyRef       time.Duration
	separateElemTypes time.Duration
	topoOrder         time.Duration
	findTypes         time.Duration
}

type counters struct {
	sccCount                   int
	containWidenCount          int
	builtinWidenConcreteCount  int
	builtinWidenContainedCount int
	preNormalizationSumSize    int64
	postNormalizationSumSize   int64
}

func (g *typeGraph) debugStats(w io.Writer) {
	fmt.Fprintf(w, "placeholders no: %d\n", len(g._placeholders))
}

func (g *typeGraph) placeholder(id placeholderID) *typePlaceholder {
	return &g._placeholders[id]
}

func (g *typeGraph) newPlaceholder() placeholderID {
	g._placeholders = append(g._placeholders, typePlaceholder{})
	g.elementType = append(g.elementType, nil)

	return placeholderID(len(g._placeholders) - 1)
}

// exprTypes is a map containing a type of each expression.
type exprTypes map[ast.Node]TypeDesc

func (g *typeGraph) newSimpleFuncType(returnType placeholderID, argNames ...ast.Identifier) placeholderID {
	p := g.newPlaceholder()
	params := []ast.Parameter{}
	for _, argName := range argNames {
		params = append(params, ast.Parameter{Name: argName})
	}
	g._placeholders[p] = concreteTP(TypeDesc{
		FunctionDesc: &functionDesc{
			resultContains: []placeholderID{returnType},
			params:         params,
			minArity:       len(argNames),
			maxArity:       len(argNames),
		},
	})
	return p
}

func (g *typeGraph) newFuncType(returnType placeholderID, params []ast.Parameter) placeholderID {
	p := g.newPlaceholder()
	g._placeholders[p] = concreteTP(TypeDesc{
		FunctionDesc: &functionDesc{
			resultContains: []placeholderID{returnType},
			params:         params,
			minArity:       countRequiredParameters(params),
			maxArity:       len(params),
		},
	})
	return p
}

// NewTypeGraph creates a new type graph, with the basic types and stdlib ready.
// It does not contain any representation based on user-provided code yet.
//
// It requires importFunc for importing the code from other files.
func newTypeGraph(importFunc ImportFunc) *typeGraph {
	g := typeGraph{
		exprPlaceholder: make(map[ast.Node]placeholderID),
		importFunc:      importFunc,
	}

	anyObjectDesc := &objectDesc{
		allFieldsKnown: false,
		fieldContains:  make(map[string][]placeholderID),
		unknownContain: []placeholderID{anyType},
	}

	anyFunctionDesc := &functionDesc{
		minArity:       0,
		maxArity:       maxPossibleArity,
		resultContains: []placeholderID{anyType},
	}

	anyArrayDesc := &arrayDesc{
		furtherContain: []placeholderID{anyType},
	}

	// Create the "no-type" sentinel placeholder
	g.newPlaceholder()

	// any type
	g.newPlaceholder()
	g._placeholders[anyType] = concreteTP(TypeDesc{
		Bool:         true,
		Number:       true,
		String:       true,
		Null:         true,
		FunctionDesc: anyFunctionDesc,
		ObjectDesc:   anyObjectDesc,
		ArrayDesc:    anyArrayDesc,
	})

	g.newPlaceholder()
	g._placeholders[boolType] = concreteTP(TypeDesc{
		Bool: true,
	})

	g.newPlaceholder()
	g._placeholders[numberType] = concreteTP(TypeDesc{
		Number: true,
	})

	g.newPlaceholder()
	g._placeholders[stringType] = concreteTP(TypeDesc{
		String: true,
	})

	g.newPlaceholder()
	g._placeholders[nullType] = concreteTP(TypeDesc{
		Null: true,
	})

	g.newPlaceholder()
	g._placeholders[anyArrayType] = concreteTP(TypeDesc{
		ArrayDesc: anyArrayDesc,
	})

	g.newPlaceholder()
	g._placeholders[numberArrayType] = concreteTP(TypeDesc{
		ArrayDesc: &arrayDesc{
			furtherContain: []placeholderID{numberType},
		},
	})

	g.newPlaceholder()
	g._placeholders[boolArrayType] = concreteTP(TypeDesc{
		ArrayDesc: &arrayDesc{
			furtherContain: []placeholderID{boolType},
		},
	})

	g.newPlaceholder()
	g._placeholders[anyObjectType] = concreteTP(TypeDesc{
		ObjectDesc: anyObjectDesc,
	})

	g.newPlaceholder()
	g._placeholders[anyFunctionType] = concreteTP(TypeDesc{
		FunctionDesc: anyFunctionDesc,
	})

	prepareStdlib(&g)

	return &g
}

// prepareTypes produces a final type for each expression in the graph.
// No further operations on the graph are valid after this is called.
func (g *typeGraph) prepareTypes(node ast.Node, typeOf exprTypes) {
	tStart := time.Now()
	g.simplifyReferences()
	tSimplify := time.Now()
	g.separateElementTypes()
	tSeparate := time.Now()
	g.makeTopoOrder()
	tTopo := time.Now()
	g.findTypes()
	tTypes := time.Now()
	g.timeStats = timeStats{
		simplifyRef:       tSimplify.Sub(tStart),
		separateElemTypes: tSeparate.Sub(tSimplify),
		topoOrder:         tTopo.Sub(tSeparate),
		findTypes:         tTypes.Sub(tTopo),
	}
	for e, p := range g.exprPlaceholder {
		typeOf[e] = g.upperBound[p]
	}
}
