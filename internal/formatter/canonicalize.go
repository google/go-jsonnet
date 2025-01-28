package formatter

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/pass"
	"sort"
)

// Canonicalize is a formatter that will update objects and arrays
// to their canonical form, i.e. sorting object fields and array elements.
type Canonicalize struct {
	pass.Base
}

type SortableField struct {
	Kind  int
	Key   string
	Field ast.ObjectField
}

func (c *Canonicalize) sortFields(fields ast.ObjectFields) {

	var sortableFields = make([]SortableField, len(fields))

	// We first construct a sortable representation of the fields using
	// the kind as precedence indicator:
	//   - asserts
	//   - locals
	//   - any object field
	for index, field := range fields {
		var kind = 0

		switch field.Kind {
		case ast.ObjectAssert:
			kind = 1
		case ast.ObjectFieldID:
			kind = 2
		case ast.ObjectFieldExpr:
			kind = 2
		case ast.ObjectFieldStr:
			kind = 2
		case ast.ObjectLocal:
			kind = 0
		}

		// generate a string representation of each field
		u := &unparser{options: Options{StripEverything: true}}
		var singleField = make([]ast.ObjectField, 1)
		singleField[0] = field
		u.unparseFields(singleField, false)

		sortableFields[index] = SortableField{Kind: kind, Key: u.string(), Field: field}
	}

	// sort the fields using a stable sort to ensure that order
	// of some fields (local and assert expressions) is retained as
	// contained in the original ast.
	sort.SliceStable(sortableFields, func(i, j int) bool {
		if sortableFields[i].Kind != sortableFields[j].Kind {
			return sortableFields[i].Kind < sortableFields[j].Kind
		}

		// retain original order local and assert expressions,
		if sortableFields[i].Kind < 2 {
			return false
		} else {
			return sortableFields[i].Key < sortableFields[j].Key
		}
	})

	for index, field := range sortableFields {
		fields[index] = field.Field
	}
}

type SortableElement struct {
	Key     string
	Element ast.CommaSeparatedExpr
}

func (c *Canonicalize) sortArrayElements(elements []ast.CommaSeparatedExpr) {
	var sortableElements = make([]SortableElement, len(elements))

	for index, element := range elements {
		u := &unparser{options: Options{StripEverything: true}}
		u.unparse(element.Expr, false)
		sortableElements[index] = SortableElement{Key: u.string(), Element: element}
	}

	sort.SliceStable(sortableElements, func(i, j int) bool {
		return sortableElements[i].Key < sortableElements[j].Key
	})

	for index, element := range sortableElements {
		elements[index] = element.Element
	}
}

// Array handles that type of node
func (c *Canonicalize) Array(p pass.ASTPass, node *ast.Array, ctx pass.Context) {
	if len(node.Elements) == 0 {
		// No comma present and none can be added.
		return
	}
	c.sortArrayElements(node.Elements)
	c.Base.Array(p, node, ctx)
}

// Object handles that type of node
func (c *Canonicalize) Object(p pass.ASTPass, node *ast.Object, ctx pass.Context) {
	if len(node.Fields) == 0 {
		// No fields present.
		return
	}
	c.sortFields(node.Fields)
	c.Base.Object(p, node, ctx)
}
