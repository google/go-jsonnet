package formatter

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-jsonnet/internal/testutils"
)

var update = flag.Bool("update", false, "update .golden files")

// ErrorWriter encapsulates a writer and an error state indicating when at least
// one error has been written to the writer.
type ErrorWriter struct {
	ErrorsFound bool
	Writer      io.Writer
}

type formatterTest struct {
	name   string
	input  string
	output string
}

type ChangedGoldensList struct {
	changedGoldens []string
}

func runTest(t *testing.T, test *formatterTest, changedGoldensList *ChangedGoldensList) {
	read := func(file string) []byte {
		bytz, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatalf("reading file: %s: %v", file, err)
		}
		return bytz
	}

	input := read(test.input)
	var outBuilder strings.Builder
	output, err := Format(test.name, string(input), DefaultOptions())
	if err != nil {
		errWriter := ErrorWriter{
			Writer:      &outBuilder,
			ErrorsFound: false,
		}

		_, writeErr := errWriter.Writer.Write([]byte(err.Error()))
		if writeErr != nil {
			panic(writeErr)
		}
	} else {
		outBuilder.Write([]byte(output))
	}

	outData := outBuilder.String()

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

func TestFormatter(t *testing.T) {
	flag.Parse()

	var tests []*formatterTest

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
		golden := jsonnetExtRE.ReplaceAllString(input, ".fmt.golden")
		tests = append(tests, &formatterTest{
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
		// Little hack: a failed test which prints update stats.
		t.Run("Goldens Updated", func(t *testing.T) {
			t.Logf("Expected failure, for printing update stats. Does not appear without `-update`.")
			t.Logf("%d formatter goldens updated:\n", len(changedGoldensList.changedGoldens))
			for _, golden := range changedGoldensList.changedGoldens {
				t.Log(golden)
			}
			t.Fail()
		})
	}
}
