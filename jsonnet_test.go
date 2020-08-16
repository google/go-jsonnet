package jsonnet

import (
	"bytes"
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/google/go-jsonnet/ast"
)

type errorFormattingTest struct {
	name      string
	input     string
	errString string
}

func genericTestErrorMessage(t *testing.T, tests []errorFormattingTest, format func(RuntimeError) string) {
	for _, test := range tests {
		vm := MakeVM()
		rawOutput, err := vm.evaluateSnippet(ast.DiagnosticFileName(test.name), "", test.input, evalKindRegular)
		var errString string
		if err != nil {
			switch typedErr := err.(type) {
			case RuntimeError:
				errString = format(typedErr)
			default:
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}
		}
		output := rawOutput.(string)
		if errString != test.errString {
			t.Errorf("%s: error result does not match. got\n\t%+#v\nexpected\n\t%+#v",
				test.name, errString, test.errString)
		}
		if err == nil {
			t.Errorf("%s, Expected error, but execution succeded and the here's the result:\n %v\n", test.name, output)
		}
	}
}

// TODO(sbarzowski) Perhaps we should have just one set of tests with all the variants?
// TODO(sbarzowski) Perhaps this should be handled in external tests?
var oneLineTests = []errorFormattingTest{
	{"error", `error "x"`, "RUNTIME ERROR: x"},
}

func TestOneLineError(t *testing.T) {
	genericTestErrorMessage(t, oneLineTests, func(r RuntimeError) string {
		return r.Error()
	})
}

// TODO(sbarzowski) checking if the whitespace is right is quite unpleasant, what can we do about it?
var minimalErrorTests = []errorFormattingTest{
	{"error", `error "x"`, "RUNTIME ERROR: x\n" +
		"	error:1:1-10	$\n" + // TODO(sbarzowski) if seems we have off-by-one in location
		"	During evaluation	\n" +
		""},
	{"error_in_func", `local x(n) = if n == 0 then error "x" else x(n - 1); x(3)`, "RUNTIME ERROR: x\n" +
		"	error_in_func:1:29-38	function <x>\n" +
		"	error_in_func:1:44-52	function <x>\n" +
		"	error_in_func:1:44-52	function <x>\n" +
		"	error_in_func:1:44-52	function <x>\n" +
		"	error_in_func:1:54-58	$\n" +
		"	During evaluation	\n" +
		""},
	{"error_in_error", `error (error "x")`, "RUNTIME ERROR: x\n" +
		"	error_in_error:1:8-17	$\n" +
		"	During evaluation	\n" +
		""},
}

func TestMinimalError(t *testing.T) {
	formatter := termErrorFormatter{maxStackTraceSize: 20}
	genericTestErrorMessage(t, minimalErrorTests, func(r RuntimeError) string {
		return formatter.Format(r)
	})
}

// TODO(sbarzowski) test pretty errors once they are stable-ish
// probably "golden" pattern is the right one for that

func removeExcessiveWhitespace(s string) string {
	var buf bytes.Buffer
	needsSeparation := false
	for i, w := 0, 0; i < len(s); i += w {
		runeValue, width := utf8.DecodeRuneInString(s[i:])
		if runeValue == '\n' || runeValue == ' ' {
			needsSeparation = true
		} else {
			if needsSeparation {
				buf.WriteString(" ")
				needsSeparation = false
			}
			buf.WriteRune(runeValue)
		}
		w = width
	}
	return buf.String()
}

func TestCustomImporter(t *testing.T) {
	vm := MakeVM()
	vm.Importer(&MemoryImporter{
		map[string]Contents{
			"a.jsonnet": MakeContents("2 + 2"),
			"b.jsonnet": MakeContents("3 + 3"),
		},
	})
	input := `[import "a.jsonnet", importstr "b.jsonnet"]`
	expected := `[ 4, "3 + 3" ]`
	actual, err := vm.EvaluateSnippet("custom_import.jsonnet", input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	actual = removeExcessiveWhitespace(actual)
	if actual != expected {
		t.Errorf("Expected %q, but got %q", expected, actual)
	}
}

type importHistoryEntry struct {
	importedFrom string
	importedPath string
}

type importerWithHistory struct {
	i       MemoryImporter
	history []importHistoryEntry
}

func (importer *importerWithHistory) Import(importedFrom, importedPath string) (contents Contents, foundAt string, err error) {
	importer.history = append(importer.history, importHistoryEntry{importedFrom, importedPath})
	return importer.i.Import(importedFrom, importedPath)
}

func TestExtVarImportedFrom(t *testing.T) {
	vm := MakeVM()
	vm.ExtCode("aaa", "import 'a.jsonnet'")
	importer := importerWithHistory{
		i: MemoryImporter{
			map[string]Contents{
				"a.jsonnet": MakeContents("2 + 2"),
			},
		},
	}
	vm.Importer(&importer)
	input := `std.extVar('aaa')`
	expected := `4`
	actual, err := vm.EvaluateSnippet("blah.jsonnet", input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	actual = removeExcessiveWhitespace(actual)
	if actual != expected {
		t.Errorf("Expected %q, but got %q", expected, actual)
	}
	expectedImportHistory := []importHistoryEntry{importHistoryEntry{"", "a.jsonnet"}}
	if !reflect.DeepEqual(importer.history, expectedImportHistory) {
		t.Errorf("Expected %q, but got %q", expectedImportHistory, importer.history)
	}
}

func TestTLAImportedFrom(t *testing.T) {
	vm := MakeVM()
	vm.TLACode("aaa", "import 'a.jsonnet'")
	importer := importerWithHistory{
		i: MemoryImporter{
			map[string]Contents{
				"a.jsonnet": MakeContents("2 + 2"),
			},
		},
	}
	vm.Importer(&importer)
	input := `function(aaa) aaa`
	expected := `4`
	actual, err := vm.EvaluateSnippet("blah.jsonnet", input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	actual = removeExcessiveWhitespace(actual)
	if actual != expected {
		t.Errorf("Expected %q, but got %q", expected, actual)
	}
	expectedImportHistory := []importHistoryEntry{importHistoryEntry{"", "a.jsonnet"}}
	if !reflect.DeepEqual(importer.history, expectedImportHistory) {
		t.Errorf("Expected %q, but got %q", expectedImportHistory, importer.history)
	}
}

func TestAnonymousImportedFrom(t *testing.T) {
	vm := MakeVM()
	importer := importerWithHistory{
		i: MemoryImporter{
			map[string]Contents{
				"a.jsonnet": MakeContents("2 + 2"),
			},
		},
	}
	vm.Importer(&importer)
	input := `import "a.jsonnet"`
	expected := `4`
	actual, err := vm.EvaluateAnonymousSnippet("blah.jsonnet", input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	actual = removeExcessiveWhitespace(actual)
	if actual != expected {
		t.Errorf("Expected %q, but got %q", expected, actual)
	}
	expectedImportHistory := []importHistoryEntry{importHistoryEntry{"", "a.jsonnet"}}
	if !reflect.DeepEqual(importer.history, expectedImportHistory) {
		t.Errorf("Expected %q, but got %q", expectedImportHistory, importer.history)
	}
}

func TestContents(t *testing.T) {
	a := "aaa"
	c1 := MakeContents(a)
	a = "bbb"
	if c1.String() != "aaa" {
		t.Errorf("Contents should be immutable")
	}
	c2 := MakeContents(a)
	c3 := MakeContents(a)
	if c2 == c3 {
		t.Errorf("Contents should distinguish between different instances even if they have the same data inside")
	}
}
