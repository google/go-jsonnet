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
	"flag"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var update = flag.Bool("update", false, "update .golden files")

// Just a few simple sanity tests for now.  Eventually we'll share end-to-end tests with the C++
// implementation but unsure if that should be done here or via some external framework.

type mainTest struct {
	name   string
	input  string
	golden string
}

func TestMain(t *testing.T) {
	flag.Parse()
	var mainTests []mainTest
	match, err := filepath.Glob("testdata/*.input")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range match {
		golden := input
		name := input
		if strings.HasSuffix(input, ".input") {
			name = input[:len(input)-len(".input")]
			golden = name + ".golden"
		}
		mainTests = append(mainTests, mainTest{name: name, input: input, golden: golden})
	}
	for _, test := range mainTests {
		t.Run(test.name, func(t *testing.T) {
			vm := MakeVM()
			read := func(file string) []byte {
				bytz, err := ioutil.ReadFile(file)
				if err != nil {
					t.Fatalf("reading file: %s: %v", file, err)
				}
				return bytz
			}

			input := read(test.input)
			output, err := vm.EvaluateSnippet(test.name, string(input))
			if err != nil {
				t.Fail()
				t.Errorf("evaluate snippet: %v", err)
			}
			if *update {
				err := ioutil.WriteFile(test.golden, []byte(output), 0666)
				if err != nil {
					t.Errorf("error updating golden files: %v", err)
				}
				return
			}
			golden := read(test.golden)
			if bytes.Compare(golden, []byte(output)) != 0 {
				t.Fail()
				t.Errorf("%s.input != %s\n", test.name, test.golden)
				data := diff( output, string(golden),)
				if err != nil {
					t.Errorf("computing diff: %s", err)
				}
				t.Errorf("diff %s jsonnet %s.input\n", test.golden, test.name)
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
