/*
Copyright 2016 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonnet

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/go-jsonnet/ast"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var update = flag.Bool("update", false, "update .golden files")

// TODO(sbarzowski) figure out how to measure coverage on the external tests

type testMetadata struct {
	extVars map[string]string
	extCode map[string]string
}

var standardExtVars = map[string]string{
	"stringVar": "2 + 2",
}

var standardExtCode = map[string]string{
	"codeVar":               "3 + 3",
	"errorVar":              "error 'xxx'",
	"staticErrorVar":        ")",
	"UndeclaredX":           "x",
	"selfRecursiveVar":      `[42, std.extVar("selfRecursiveVar")[0] + 1]`,
	"mutuallyRecursiveVar1": `[42, std.extVar("mutuallyRecursiveVar2")[0] + 1]`,
	"mutuallyRecursiveVar2": `[42, std.extVar("mutuallyRecursiveVar1")[0] + 1]`,
}

var metadataForTests = map[string]testMetadata{
	"testdata/extvar_code":               testMetadata{extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_error":              testMetadata{extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_hermetic":           testMetadata{extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_mutually_recursive": testMetadata{extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_self_recursive":     testMetadata{extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_static_error":       testMetadata{extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_string":             testMetadata{extVars: standardExtVars, extCode: standardExtCode},
}

type mainTest struct {
	name   string
	input  string
	golden string
	meta   *testMetadata
}

func TestMain(t *testing.T) {
	flag.Parse()
	var mainTests []mainTest
	match, err := filepath.Glob("testdata/*.jsonnet")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range match {
		golden := input
		name := input
		if strings.HasSuffix(input, ".jsonnet") {
			name = input[:len(input)-len(".jsonnet")]
			golden = name + ".golden"
		}
		var meta testMetadata
		if val, exists := metadataForTests[name]; exists {
			meta = val
		}
		mainTests = append(mainTests, mainTest{name: name, input: input, golden: golden, meta: &meta})
	}
	errFormatter := ErrorFormatter{pretty: true, MaxStackTraceSize: 9}
	for _, test := range mainTests {
		t.Run(test.name, func(t *testing.T) {
			vm := MakeVM()

			for name, value := range test.meta.extVars {
				vm.ExtVar(name, value)
			}
			for name, value := range test.meta.extCode {
				vm.ExtCode(name, value)
			}

			vm.NativeFunction(&NativeFunction{
				Name:   "jsonToString",
				Params: ast.Identifiers{"x"},
				Func: func(x []interface{}) (interface{}, error) {
					bytes, err := json.Marshal(x[0])
					if err != nil {
						return nil, err
					}
					return string(bytes), nil
				},
			})
			read := func(file string) []byte {
				bytz, err := ioutil.ReadFile(file)
				if err != nil {
					t.Fatalf("reading file: %s: %v", file, err)
				}
				return bytz
			}

			input := read(test.input)
			output, err := vm.evaluateSnippet(test.name, string(input))
			if err != nil {
				// TODO(sbarzowski) perhaps somehow mark that we are processing
				// an error. But for now we can treat them the same.
				output = errFormatter.format(err)
			}
			output += "\n"
			if *update {
				err := ioutil.WriteFile(test.golden, []byte(output), 0666)
				if err != nil {
					t.Errorf("error updating golden files: %v", err)
				}
				return
			}
			golden := read(test.golden)
			if bytes.Compare(golden, []byte(output)) != 0 {
				// TODO(sbarzowski) better reporting of differences in whitespace
				// missing newline issues can be very subtle now
				t.Fail()
				t.Errorf("Mismatch when running %s.jsonnet. Golden: %s\n", test.name, test.golden)
				data := diff(output, string(golden))
				if err != nil {
					t.Errorf("computing diff: %s", err)
				}
				t.Errorf("diff %s jsonnet %s.jsonnet\n", test.golden, test.name)
				t.Errorf(string(data))

			}
		})
	}
}

func diff(a, b string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a, b, false)
	return dmp.DiffPrettyText(diffs)
}

type errorFormattingTest struct {
	name      string
	input     string
	errString string
}

func genericTestErrorMessage(t *testing.T, tests []errorFormattingTest, format func(RuntimeError) string) {
	for _, test := range tests {
		vm := MakeVM()
		output, err := vm.evaluateSnippet(test.name, test.input)
		var errString string
		if err != nil {
			switch typedErr := err.(type) {
			case RuntimeError:
				errString = format(typedErr)
			default:
				t.Errorf("%s: unexpected error: %v", test.name, err)
			}

		}
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
		"	error:1:1-9	$\n" + // TODO(sbarzowski) if seems we have off-by-one in location
		"	During evaluation	\n" +
		""},
	{"error_in_func", `local x(n) = if n == 0 then error "x" else x(n - 1); x(3)`, "RUNTIME ERROR: x\n" +
		"	error_in_func:1:29-37	function <x>\n" +
		"	error_in_func:1:44-52	function <x>\n" +
		"	error_in_func:1:44-52	function <x>\n" +
		"	error_in_func:1:44-52	function <x>\n" +
		"	error_in_func:1:54-58	$\n" +
		"	During evaluation	\n" +
		""},
	{"error_in_error", `error (error "x")`, "RUNTIME ERROR: x\n" +
		"	error_in_error:1:8-16	$\n" +
		"	During evaluation	\n" +
		""},
}

func TestMinimalError(t *testing.T) {
	formatter := ErrorFormatter{MaxStackTraceSize: 20}
	genericTestErrorMessage(t, minimalErrorTests, func(r RuntimeError) string {
		return formatter.format(r)
	})
}

// TODO(sbarzowski) test pretty errors once they are stable-ish
// probably "golden" pattern is the right one for that

func removeExcessiveWhitespace(s string) string {
	var buf bytes.Buffer
	separated := true
	for i, w := 0, 0; i < len(s); i += w {
		runeValue, width := utf8.DecodeRuneInString(s[i:])
		if runeValue == '\n' || runeValue == ' ' {
			if !separated {
				buf.WriteString(" ")
				separated = true
			}
		} else {
			buf.WriteRune(runeValue)
			separated = false
		}
		w = width
	}
	return buf.String()
}

func TestCustomImporter(t *testing.T) {
	vm := MakeVM()
	vm.Importer(&MemoryImporter{
		map[string]string{
			"a.jsonnet": "2 + 2",
			"b.jsonnet": "3 + 3",
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
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
}
