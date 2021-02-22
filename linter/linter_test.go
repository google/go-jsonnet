package linter

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/internal/testutils"
)

var update = flag.Bool("update", false, "update .golden files")

type linterTest struct {
	name   string
	input  string
	output string
}

type ChangedGoldensList struct {
	changedGoldens []string
}

func runTest(t *testing.T, test *linterTest, changedGoldensList *ChangedGoldensList) {
	read := func(file string) []byte {
		bytz, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatalf("reading file: %s: %v", file, err)
		}
		return bytz
	}

	input := read(test.input)

	vm := jsonnet.MakeVM()

	var outBuilder strings.Builder

	errorsFound := LintSnippet(vm, &outBuilder, test.name, string(input))

	outData := outBuilder.String()

	if outData == "" && errorsFound {
		t.Error(fmt.Errorf("return value indicates problems present, but no output was produced"))
	}

	if outData != "" && !errorsFound {
		t.Error(fmt.Errorf("return value indicates no problems, but output is not empty:\n%v", outData))
	}

	if *update {
		changed, err := testutils.UpdateGoldenFile(test.output, []byte(outData), 0666)
		if err != nil {
			t.Error(err)
		}
		if changed {
			changedGoldensList.changedGoldens = append(changedGoldensList.changedGoldens, test.output)
		}
	} else {
		golden, err := ioutil.ReadFile(test.output)
		if err != nil {
			t.Error(err)
			return
		}
		if diff, hasDiff := testutils.CompareWithGolden(outData, golden); hasDiff {
			t.Error(fmt.Errorf("golden file %v has diff:\n%v", test.input, diff))
		}
	}
}

func TestLinter(t *testing.T) {
	flag.Parse()

	var tests []*linterTest

	match, err := filepath.Glob("testdata/*.jsonnet")
	if err != nil {
		t.Fatal(err)
	}

	matchRegular, err := filepath.Glob("../testdata/*.jsonnet")
	if err != nil {
		t.Fatal(err)
	}
	match = append(match, matchRegular...)

	jsonnetExtRE := regexp.MustCompile(`\.jsonnet$`)

	for _, input := range match {
		// Skip escaped filenames.
		if strings.ContainsRune(input, '%') {
			continue
		}
		name := jsonnetExtRE.ReplaceAllString(input, "")
		golden := jsonnetExtRE.ReplaceAllString(input, ".linter.golden")
		tests = append(tests, &linterTest{
			name:   name,
			input:  input,
			output: golden,
		})
	}

	changedGoldensList := ChangedGoldensList{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runTest(t, test, &changedGoldensList)
		})
	}

	if *update {
		// Little hack: a failed test which pritns update stats.
		t.Run("Goldens Updated", func(t *testing.T) {
			t.Logf("Expected failure, for printing update stats. Does not appear without `-update`.")
			t.Logf("%d linter goldens updated:\n", len(changedGoldensList.changedGoldens))
			for _, golden := range changedGoldensList.changedGoldens {
				t.Log(golden)
			}
			t.Fail()
		})
	}
}
