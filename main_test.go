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
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
	"github.com/google/go-jsonnet/internal/testutils"
)

var update = flag.Bool("update", false, "update .golden files")
var jsonnetCmd = flag.String("cmd", "", "path to jsonnet command (if not specified or empty, internal implementation is used)")

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
	"testdata/extvar_code":               {extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_error":              {extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_hermetic":           {extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_mutually_recursive": {extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_self_recursive":     {extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_static_error":       {extVars: standardExtVars, extCode: standardExtCode},
	"testdata/extvar_string":             {extVars: standardExtVars, extCode: standardExtCode},
}

type mainTest struct {
	name   string
	input  string
	golden string
	meta   *testMetadata
}

var jsonToString = &NativeFunction{
	Name:   "jsonToString",
	Params: ast.Identifiers{"x"},
	Func: func(x []interface{}) (interface{}, error) {
		bytes, err := json.Marshal(x[0])
		if err != nil {
			return nil, err
		}
		return string(bytes), nil
	},
}

var nativeError = &NativeFunction{
	Name:   "nativeError",
	Params: ast.Identifiers{},
	Func: func(x []interface{}) (interface{}, error) {
		return nil, errors.New("native function error")
	},
}

var nativePanic = &NativeFunction{
	Name:   "nativePanic",
	Params: ast.Identifiers{},
	Func: func(x []interface{}) (interface{}, error) {
		panic("native function panic")
	},
}

type jsonnetInput struct {
	name             string
	input            []byte
	eKind            evalKind
	stringOutputMode bool
	extVars          map[string]string
	extCode          map[string]string
}

type jsonnetResult struct {
	// One of output or outputMulti is populated.
	// If isError is set, the error is stored in output.
	output      string
	outputMulti map[string]string

	isError bool
}

func testChildren(node ast.Node) {
	// Test that Children works on every node in the tree
	for _, child := range parser.Children(node) {
		testChildren(child)
	}
	// TODO(sbarzowski) it would be great to check somehow that all nodes were reached
}

func runInternalJsonnet(i jsonnetInput) jsonnetResult {
	vm := MakeVM()
	errFormatter := termErrorFormatter{pretty: true, maxStackTraceSize: 9}

	vm.StringOutput = i.stringOutputMode
	for name, value := range i.extVars {
		vm.ExtVar(name, value)
	}
	for name, value := range i.extCode {
		vm.ExtCode(name, value)
	}

	vm.NativeFunction(jsonToString)
	vm.NativeFunction(nativeError)
	vm.NativeFunction(nativePanic)

	rawAST, _, staticErr := parser.SnippetToRawAST(ast.DiagnosticFileName(i.name), "", string(i.input))
	if staticErr != nil {
		return jsonnetResult{
			output:  errFormatter.Format(staticErr) + "\n",
			isError: true,
		}
	}
	testChildren(rawAST)

	desugaredAST, err := SnippetToAST(i.name, string(i.input))
	if err != nil {
		return jsonnetResult{
			output:  errFormatter.Format(err) + "\n",
			isError: true,
		}
	}
	testChildren(desugaredAST)

	// TODO(sbarzowski) We should treat the tests as anonymous snippets or import them with an importer.
	rawOutput, err := vm.evaluateSnippet(ast.DiagnosticFileName(i.name), i.name, string(i.input), i.eKind)
	switch {
	case err != nil:
		// TODO(sbarzowski) perhaps somehow mark that we are processing
		// an error. But for now we can treat them the same.
		return jsonnetResult{
			output:  errFormatter.Format(err) + "\n",
			isError: true,
		}
	case i.eKind == evalKindMulti:
		return jsonnetResult{
			outputMulti: rawOutput.(map[string]string),
		}
	default:
		return jsonnetResult{
			output: rawOutput.(string),
		}
	}
}

// TODO(lukegb) CLI test support is presently completely broken: fix?
func runJsonnetCommand(i jsonnetInput) jsonnetResult {
	// TODO(sbarzowski) Special handling of errors (which may differ between versions)
	if i.eKind != evalKindRegular {
		panic(fmt.Sprintf("eKind must be evalKindRegular for jsonnet CLI testing; was %v", i.eKind))
	}
	input := bytes.NewBuffer(i.input)
	var output bytes.Buffer
	isError := false
	cmd := exec.Cmd{
		Path:   *jsonnetCmd,
		Stdin:  input,
		Stdout: &output,
		Stderr: &output,
		Args:   []string{"jsonnet", "-"},
	}
	err := cmd.Run()
	if err != nil {
		switch err := err.(type) {
		case *exec.ExitError:
			// It finished with non-zero exit code
			isError = true
		default:
			// We weren't able to run it
			panic(err)
		}
	}
	return jsonnetResult{
		output:  output.String(),
		isError: isError,
	}
}

func runJsonnet(i jsonnetInput) jsonnetResult {
	if jsonnetCmd != nil && *jsonnetCmd != "" {
		return runJsonnetCommand(i)
	}
	return runInternalJsonnet(i)
}

func compareSingleGolden(path string, result jsonnetResult) []error {
	if result.outputMulti != nil {
		return []error{fmt.Errorf("outputMulti is populated in a single-file test for %v", path)}
	}
	golden, err := os.ReadFile(path)
	if err != nil {
		return []error{fmt.Errorf("reading file %s: %v", path, err)}
	}
	if diff, hasDiff := testutils.CompareWithGolden(result.output, golden); hasDiff {
		return []error{fmt.Errorf("golden file %v has diff:\n%v", path, diff)}
	}
	return nil
}

func updateSingleGolden(path string, result jsonnetResult) (updated []string, err error) {
	if result.outputMulti != nil {
		return nil, fmt.Errorf("outputMulti is populated in a single-file test for %v", path)
	}
	changed, err := testutils.UpdateGoldenFile(path, []byte(result.output), 0666)
	if err != nil {
		return nil, fmt.Errorf("updating golden file %v: %v", path, err)
	}
	if changed {
		return []string{path}, nil
	}
	return nil, nil
}

func compareMultifileGolden(path string, result jsonnetResult) []error {
	expectFiles, err := os.ReadDir(path)
	if err != nil {
		return []error{fmt.Errorf("reading golden dir %v: %v", path, err)}
	}
	goldenContent := map[string][]byte{}
	var errs []error
	for _, f := range expectFiles {
		golden, err := os.ReadFile(filepath.Join(path, f.Name()))
		if err != nil {
			return []error{fmt.Errorf("reading file %s: %v", f.Name(), err)}
		}
		if _, ok := result.outputMulti[f.Name()]; !ok {
			errs = append(errs, fmt.Errorf("jsonnet did not output expected file %v", f.Name()))
			continue
		}
		goldenContent[f.Name()] = golden
	}
	for fn, content := range result.outputMulti {
		if _, ok := goldenContent[fn]; !ok {
			errs = append(errs, fmt.Errorf("jsonnet outputted file %v which does not exist in goldens", fn))
			continue
		}
		if diff, hasDiff := testutils.CompareWithGolden(content, goldenContent[fn]); hasDiff {
			errs = append(errs, fmt.Errorf("golden file %v has diff:\n%v", fn, diff))
		}
	}
	return errs
}

func updateMultifileGolden(path string, result jsonnetResult) ([]string, error) {
	expectFiles, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading golden directory %v: %v", path, err)
	}
	var updatedFiles []string
	for fn, content := range result.outputMulti {
		updated, err := testutils.UpdateGoldenFile(filepath.Join(path, fn), []byte(content), 0666)
		if err != nil {
			return nil, fmt.Errorf("updating golden file %v: %v", fn, err)
		}
		if updated {
			updatedFiles = append(updatedFiles, filepath.Join(path, fn))
		}
	}
	// Delete excess files
	for _, f := range expectFiles {
		if _, ok := result.outputMulti[f.Name()]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(path, f.Name())); err != nil {
			return nil, fmt.Errorf("removing golden file %v: %v", f.Name(), err)
		}
	}
	return updatedFiles, nil
}

func runTest(t *testing.T, test *mainTest) {
	read := func(file string) []byte {
		bytz, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading file: %s: %v", file, err)
		}
		return bytz
	}

	input := read(test.input)

	eKind := evalKindRegular
	compareFunc := compareSingleGolden
	updateFunc := updateSingleGolden

	// If the golden path is a directory, this is a multi-test.
	if info, err := os.Stat(test.golden); err == nil && info.IsDir() {
		eKind = evalKindMulti
		compareFunc = compareMultifileGolden
		updateFunc = updateMultifileGolden
	}

	result := runJsonnet(jsonnetInput{
		name:             test.name,
		input:            input,
		eKind:            eKind,
		stringOutputMode: strings.HasSuffix(test.golden, "_string_output.golden"),
		extVars:          test.meta.extVars,
		extCode:          test.meta.extCode,
	})

	if eKind == evalKindMulti && result.isError {
		// If it's an error, then result.output is populated instead.
		// Since we use the golden file being a directory to determine if we
		// should run in multi-file mode, we put the output into an "error" file instead.
		result.outputMulti = map[string]string{"error": result.output}
		result.output = ""
	}

	if *update {
		updated, err := updateFunc(test.golden, result)
		if err != nil {
			t.Error(err)
		}
		for _, updatedFile := range updated {
			fmt.Fprintf(os.Stderr, "updated golden %v\n", updatedFile)
		}
		return
	}
	for _, err := range compareFunc(test.golden, result) {
		t.Error(err)
	}
}

func TestEval(t *testing.T) {
	files, err := filepath.Glob("testdata/*.jsonnet")
	if err != nil {
		t.Fatal(err)
	}
	tests := make([]mainTest, 0, len(files))
	for _, input := range files {
		name := strings.TrimSuffix(input, ".jsonnet")
		var meta testMetadata
		if val, exists := metadataForTests[name]; exists {
			meta = val
		}
		tests = append(tests, mainTest{name: name, input: input, golden: name + ".golden", meta: &meta})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runTest(t, &test)
		})
	}
}

func withinWorkingDirectory(t *testing.T, dir string) func() {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return func() {
		err := os.Chdir(cwd)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestEvalUnusualFilenames(t *testing.T) {
	// Escaped filenames exist because their unescaped forms are invalid on Windows. We have no
	// choice but to skip these in testing.
	if runtime.GOOS == "windows" {
		return
	}

	// Are we running within "bazel test"?
	dir := os.Getenv("TEST_TMPDIR")
	if len(dir) == 0 {
		var err error
		if dir, err = os.MkdirTemp("", "jsonnet"); err != nil {
			t.Fatal(err)
		}
		defer func() {
			err := os.RemoveAll(dir)
			if err != nil {
				panic(err)
			}
		}()
	}

	copySmallFile := func(t *testing.T, dst, src string) {
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, b, 0444); err != nil {
			t.Fatal(err)
		}
	}
	// These files are imported by files below, but we don't need to exercise their round-trip
	// behavior here, as they're covered by TestEval above.
	for _, f := range []string{"true"} {
		for _, ext := range []string{".jsonnet"} {
			name := f + ext
			copySmallFile(t, filepath.Join(dir, name), filepath.Join("testdata", name))
		}
	}

	// Temporarily switch into our scratch directory.
	defer withinWorkingDirectory(t, dir)()

	emptyMetadata := &testMetadata{}
	for _, f := range []struct {
		name    string
		content []byte
		golden  []byte
	}{
		{
			`"`,
			[]byte(`// This file is there only for its filename: to test escaping in imports
{}
`),
			[]byte(`{ }
`),
		},
		{
			`'`,
			[]byte(`// This file is there only for its filename: to test escaping in imports
{}
`),
			[]byte(`{ }
`),
		},
		{
			"import_various_literals_escaped",
			[]byte(`[
	import "\u0074rue.jsonnet",
	import '\u0074rue.jsonnet',
	importstr @""".jsonnet",
	importstr @'''.jsonnet',
]
`),
			[]byte(`[
   true,
   true,
   "// This file is there only for its filename: to test escaping in imports\n{}\n",
   "// This file is there only for its filename: to test escaping in imports\n{}\n"
]
`),
		},
		{
			"importstr_various_literals_escaped",
			[]byte(`[
	importstr "\u0074rue.jsonnet",
	importstr '\u0074rue.jsonnet',
	importstr @""".jsonnet",
	importstr @'''.jsonnet',
]
`),
			[]byte(`[
   "true\n",
   "true\n",
   "// This file is there only for its filename: to test escaping in imports\n{}\n",
   "// This file is there only for its filename: to test escaping in imports\n{}\n"
]
`),
		},
	} {
		if err := os.WriteFile(f.name+".jsonnet", f.content, 0444); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f.name+".golden", f.golden, 0444); err != nil {
			t.Fatal(err)
		}
		t.Run(f.name, func(t *testing.T) {
			runTest(t, &mainTest{
				name:   f.name,
				input:  f.name + ".jsonnet",
				golden: f.name + ".golden",
				meta:   emptyMetadata,
			})
		})
	}
}
