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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-jsonnet/ast"
	"github.com/sergi/go-diff/diffmatchpatch"
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

type jsonnetInput struct {
	name    string
	input   []byte
	eKind   evalKind
	extVars map[string]string
	extCode map[string]string
}

type jsonnetResult struct {
	output      string
	outputMulti map[string]string
	isError     bool
}

func runInternalJsonnet(i jsonnetInput) jsonnetResult {
	vm := MakeVM()
	errFormatter := termErrorFormatter{pretty: true, maxStackTraceSize: 9}

	for name, value := range i.extVars {
		vm.ExtVar(name, value)
	}
	for name, value := range i.extCode {
		vm.ExtCode(name, value)
	}

	vm.NativeFunction(jsonToString)
	vm.NativeFunction(nativeError)

	rawOutput, err := vm.evaluateSnippet(i.name, string(i.input), i.eKind)
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

func runTest(t *testing.T, test *mainTest) {

	read := func(file string) []byte {
		bytz, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatalf("reading file: %s: %v", file, err)
		}
		return bytz
	}

	input := read(test.input)

	eKind := evalKindRegular
	// If the golden path is to a directory, this is a multi-test.
	if info, err := os.Stat(test.golden); err == nil && info.IsDir() {
		eKind = evalKindMulti
	}

	result := runJsonnet(jsonnetInput{
		name:    test.name,
		input:   input,
		eKind:   eKind,
		extVars: test.meta.extVars,
		extCode: test.meta.extCode,
	})

	// TODO(sbarzowski) report which files were updated
	if *update {
		err := ioutil.WriteFile(test.golden, []byte(result.output), 0666)
		if err != nil {
			t.Errorf("error updating golden files: %v", err)
		}
		return
	}
	compareGolden := func(inputPath, goldenPath, result string, golden []byte) {
		if bytes.Compare(golden, []byte(result)) != 0 {
			// TODO(sbarzowski) better reporting of differences in whitespace
			// missing newline issues can be very subtle now
			t.Fail()
			t.Errorf("Mismatch when running %s. Golden: %s\n", inputPath, goldenPath)
			data := diff(result, string(golden))
			t.Errorf("diff %s jsonnet %s\n", goldenPath, inputPath)
			t.Errorf(string(data))
		}
	}

	switch eKind {
	default:
		golden := read(test.golden)
		compareGolden(test.input, test.golden, result.output, golden)
	case evalKindMulti:
		expectFiles, err := ioutil.ReadDir(test.golden)
		if err != nil {
			t.Fatalf("ReadDir(%q): %v", test.golden, err)
		}
		goldenContent := map[string][]byte{}
		for _, f := range expectFiles {
			goldenContent[f.Name()] = read(filepath.Join(test.golden, f.Name()))
		}
		for fn, content := range result.outputMulti {
			compareGolden(test.input, fn, content, goldenContent[fn])
		}
	}
}

func expandEscapedFilenames(t *testing.T) error {
	// Escaped filenames exist because their unescaped forms are invalid on
	// Windows. We have no choice but to skip these in testing.
	if runtime.GOOS == "windows" {
		return nil
	}

	match, err := filepath.Glob("testdata/*%*")
	if err != nil {
		return err
	}

	filenameEscapeRE := regexp.MustCompile(`%[0-9A-Fa-f]{2}`)

	for _, input := range match {
		file := filenameEscapeRE.ReplaceAllStringFunc(input, func(s string) string {
			code, err := strconv.ParseUint(s[1:], 16, 8)
			if err != nil {
				panic(err)
			}
			return string([]byte{byte(code)})
		})
		if file != input {
			if err := os.Link(input, file); err != nil {
				if !os.IsExist(err) {
					return fmt.Errorf("linking %s -> %s: %v", file, input, err)
				}
			}
			t.Logf("Linked %s -> %s", input, file)
		}
	}

	return nil
}

func TestMain(t *testing.T) {
	flag.Parse()

	if err := expandEscapedFilenames(t); err != nil {
		t.Fatal(err)
	}

	var mainTests []mainTest
	match, err := filepath.Glob("testdata/*.jsonnet")
	if err != nil {
		t.Fatal(err)
	}

	jsonnetExtRE := regexp.MustCompile(`\.jsonnet$`)

	for _, input := range match {
		// Skip escaped filenames.
		if strings.ContainsRune(input, '%') {
			continue
		}
		name := jsonnetExtRE.ReplaceAllString(input, "")
		golden := jsonnetExtRE.ReplaceAllString(input, ".golden")
		var meta testMetadata
		if val, exists := metadataForTests[name]; exists {
			meta = val
		}
		mainTests = append(mainTests, mainTest{name: name, input: input, golden: golden, meta: &meta})
	}
	for _, test := range mainTests {
		t.Run(test.name, func(t *testing.T) {
			runTest(t, &test)
		})
	}
}

func diff(a, b string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a, b, false)
	return dmp.DiffPrettyText(diffs)
}
