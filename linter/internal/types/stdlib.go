package types

import "github.com/google/go-jsonnet/ast"

func prepareStdlib(g *typeGraph) {
	g.newPlaceholder()

	arrayOfString := anyArrayType
	arrayOfNumber := anyArrayType
	stringOrArray := anyType
	stringOrNumber := anyType
	jsonType := anyType // It actually cannot functions anywhere

	required := func(name string) ast.Parameter {
		return ast.Parameter{Name: ast.Identifier(name)}
	}

	dummyDefaultArg := &ast.LiteralNull{}
	optional := func(name string) ast.Parameter {
		return ast.Parameter{Name: ast.Identifier(name), DefaultArg: dummyDefaultArg}
	}

	fields := map[string]placeholderID{

		// External variables
		"extVar": g.newSimpleFuncType(anyType, "x"),

		// Types and reflection
		"thisFile":        stringType,
		"type":            g.newSimpleFuncType(stringType, "x"),
		"length":          g.newSimpleFuncType(numberType, "x"),
		"objectHas":       g.newSimpleFuncType(boolType, "o", "f"),
		"objectFields":    g.newSimpleFuncType(arrayOfString, "o"),
		"objectHasAll":    g.newSimpleFuncType(boolType, "o", "f"),
		"objectFieldsAll": g.newSimpleFuncType(arrayOfString, "o"),
		"prune":           g.newSimpleFuncType(anyObjectType, "a"),
		"mapWithKey":      g.newSimpleFuncType(anyObjectType, "func", "obj"),

		// Mathematical utilities
		"abs":      g.newSimpleFuncType(numberType, "n"),
		"sign":     g.newSimpleFuncType(numberType, "n"),
		"max":      g.newSimpleFuncType(numberType, "a", "b"),
		"min":      g.newSimpleFuncType(numberType, "a", "b"),
		"pow":      g.newSimpleFuncType(numberType, "x", "n"),
		"exp":      g.newSimpleFuncType(numberType, "x"),
		"log":      g.newSimpleFuncType(numberType, "x"),
		"exponent": g.newSimpleFuncType(numberType, "x"),
		"mantissa": g.newSimpleFuncType(numberType, "x"),
		"floor":    g.newSimpleFuncType(numberType, "x"),
		"ceil":     g.newSimpleFuncType(numberType, "x"),
		"sqrt":     g.newSimpleFuncType(numberType, "x"),
		"sin":      g.newSimpleFuncType(numberType, "x"),
		"cos":      g.newSimpleFuncType(numberType, "x"),
		"tan":      g.newSimpleFuncType(numberType, "x"),
		"asin":     g.newSimpleFuncType(numberType, "x"),
		"acos":     g.newSimpleFuncType(numberType, "x"),
		"atan":     g.newSimpleFuncType(numberType, "x"),

		// Assertions and debugging
		"assertEqual": g.newSimpleFuncType(boolType, "a", "b"),

		// String Manipulation

		"toString":    g.newSimpleFuncType(stringType, "a"),
		"codepoint":   g.newSimpleFuncType(numberType, "str"),
		"char":        g.newSimpleFuncType(stringType, "n"),
		"substr":      g.newSimpleFuncType(stringType, "str", "from", "len"),
		"findSubstr":  g.newSimpleFuncType(arrayOfNumber, "pat", "str"),
		"startsWith":  g.newSimpleFuncType(boolType, "a", "b"),
		"endsWith":    g.newSimpleFuncType(boolType, "a", "b"),
		"split":       g.newSimpleFuncType(arrayOfString, "str", "c"),
		"splitLimit":  g.newSimpleFuncType(arrayOfString, "str", "c", "maxsplits"),
		"strReplace":  g.newSimpleFuncType(stringType, "str", "from", "to"),
		"asciiUpper":  g.newSimpleFuncType(stringType, "str"),
		"asciiLower":  g.newSimpleFuncType(stringType, "str"),
		"stringChars": g.newSimpleFuncType(stringType, "str"),
		"format":      g.newSimpleFuncType(stringType, "str", "vals"),
		// TODO(sbarzowski) Fix when they match the documentation
		"escapeStringBash":    g.newSimpleFuncType(stringType, "str_"),
		"escapeStringDollars": g.newSimpleFuncType(stringType, "str_"),
		"escapeStringJson":    g.newSimpleFuncType(stringType, "str_"),
		"escapeStringPython":  g.newSimpleFuncType(stringType, "str"),

		// Parsing

		"parseInt":   g.newSimpleFuncType(numberType, "str"),
		"parseOctal": g.newSimpleFuncType(numberType, "str"),
		"parseHex":   g.newSimpleFuncType(numberType, "str"),
		"parseJson":  g.newSimpleFuncType(jsonType, "str"),
		"encodeUTF8": g.newSimpleFuncType(arrayOfNumber, "str"),
		"decodeUTF8": g.newSimpleFuncType(stringType, "arr"),

		// Manifestation

		"manifestIni":        g.newSimpleFuncType(stringType, "ini"),
		"manifestPython":     g.newSimpleFuncType(stringType, "v"),
		"manifestPythonVars": g.newSimpleFuncType(stringType, "conf"),
		"manifestJsonEx":     g.newSimpleFuncType(stringType, "value", "indent"),
		"manifestYamlDoc":    g.newSimpleFuncType(stringType, "value"),
		"manifestYamlStream": g.newSimpleFuncType(stringType, "value"),
		"manifestXmlJsonml":  g.newSimpleFuncType(stringType, "value"),

		// Arrays

		"makeArray":     g.newSimpleFuncType(anyArrayType, "sz", "func"),
		"count":         g.newSimpleFuncType(numberType, "arr", "x"),
		"find":          g.newSimpleFuncType(arrayOfNumber, "value", "arr"),
		"map":           g.newSimpleFuncType(anyArrayType, "func", "arr"),
		"mapWithIndex":  g.newSimpleFuncType(anyArrayType, "func", "arr"),
		"filterMap":     g.newSimpleFuncType(anyArrayType, "filter_func", "map_func", "arr"),
		"filter":        g.newSimpleFuncType(anyArrayType, "func", "arr"),
		"foldl":         g.newSimpleFuncType(anyType, "func", "arr", "init"),
		"foldr":         g.newSimpleFuncType(anyType, "func", "arr", "init"),
		"range":         g.newSimpleFuncType(arrayOfNumber, "from", "to"),
		"join":          g.newSimpleFuncType(stringOrArray, "sep", "arr"),
		"lines":         g.newSimpleFuncType(arrayOfString, "arr"),
		"flattenArrays": g.newSimpleFuncType(anyArrayType, "arrs"),
		"sort":          g.newFuncType(anyArrayType, []ast.Parameter{required("arr"), optional("keyF")}),
		"uniq":          g.newFuncType(anyArrayType, []ast.Parameter{required("arr"), optional("keyF")}),

		// Sets

		"set":       g.newFuncType(anyArrayType, []ast.Parameter{required("arr"), optional("keyF")}),
		"setInter":  g.newFuncType(anyArrayType, []ast.Parameter{required("a"), required("b"), optional("keyF")}),
		"setUnion":  g.newFuncType(anyArrayType, []ast.Parameter{required("a"), required("b"), optional("keyF")}),
		"setDiff":   g.newFuncType(anyArrayType, []ast.Parameter{required("a"), required("b"), optional("keyF")}),
		"setMember": g.newFuncType(anyArrayType, []ast.Parameter{required("x"), required("arr"), optional("keyF")}),

		// Encoding

		"base64":            g.newSimpleFuncType(stringType, "input"),
		"base64DecodeBytes": g.newSimpleFuncType(numberType, "str"),
		"base64Decode":      g.newSimpleFuncType(stringType, "str"),
		"md5":               g.newSimpleFuncType(stringType, "s"),

		// JSON Merge Patch

		"mergePatch": g.newSimpleFuncType(anyType, "target", "patch"),

		// Debugging

		"trace": g.newSimpleFuncType(anyType, "str", "rest"),

		// Undocumented
		"manifestJson":     g.newSimpleFuncType(stringType, "value"),
		"objectHasEx":      g.newSimpleFuncType(boolType, "obj", "fname", "hidden"),
		"objectFieldsEx":   g.newSimpleFuncType(arrayOfString, "obj", "hidden"),
		"flatMap":          g.newSimpleFuncType(anyArrayType, "func", "arr"),
		"modulo":           g.newSimpleFuncType(numberType, "x", "y"),
		"slice":            g.newSimpleFuncType(arrayOfString, "indexable", "index", "end", "step"),
		"primitiveEquals":  g.newSimpleFuncType(boolType, "x", "y"),
		"mod":              g.newSimpleFuncType(stringOrNumber, "a", "b"),
		"native":           g.newSimpleFuncType(anyFunctionType, "x"),
		"$objectFlatMerge": g.newSimpleFuncType(anyObjectType, "x"),
	}

	fieldContains := map[string][]placeholderID{}
	for name, t := range fields {
		fieldContains[name] = []placeholderID{t}
	}

	g._placeholders[stdlibType] = concreteTP(TypeDesc{
		ObjectDesc: &objectDesc{
			allFieldsKnown: true,
			unknownContain: nil,
			fieldContains:  fieldContains,
		},
	})
}
