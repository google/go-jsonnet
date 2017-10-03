/*
Copyright 2017 Google Inc. All rights reserved.

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

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/google/go-jsonnet"
)

func usage() {
	fmt.Println("usage: jsonnet <filename>")
}

func getVar(s string) (string, string, error) {
	parts := strings.SplitN(s, "=", 2)
	name := parts[0]
	if len(parts) == 1 {
		content, exists := os.LookupEnv(name)
		if exists {
			return name, content, nil
		}
		return "", "", fmt.Errorf("ERROR: Environment variable %v was undefined.", name)
	} else {
		return name, parts[1], nil
	}
}

func main() {
	// https://blog.golang.org/profiling-go-programs
	var cpuprofile = os.Getenv("JSONNET_CPU_PROFILE")
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// TODO(sbarzowski) Be consistent about error codes with C++ maybe
	var filename string

	vm := jsonnet.MakeVM()
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--tla-str":
			i++
			name, content, err := getVar(os.Args[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			vm.TLAVar(name, content)
		case "--tla-code":
			i++
			name, content, err := getVar(os.Args[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			vm.TLACode(name, content)
		case "--ext-code":
			i++
			name, content, err := getVar(os.Args[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			vm.ExtCode(name, content)
		case "--ext-str":
			i++
			name, content, err := getVar(os.Args[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			vm.ExtVar(name, content)
		default:
			if filename != "" {
				usage()
				os.Exit(1)
			}
			filename = arg
		}
	}
	snippet, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err.Error())
		os.Exit(1)
	}
	json, err := vm.EvaluateSnippet(filename, string(snippet))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err.Error())
		os.Exit(2)
	}
	fmt.Println(json)
}
