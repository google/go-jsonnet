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

	"github.com/google/go-jsonnet"
)

func usage() {
	fmt.Println("usage: jsonnet <filename>")
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
	vm := jsonnet.MakeVM()
	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}
	filename := os.Args[1]
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
