/*
Copyright 2020 Google Inc. All rights reserved.

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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/cmd/internal/cmd"
)

func version(o io.Writer) {
	fmt.Fprintf(o, "Jsonnet static dependency parser %s\n", jsonnet.Version())
}

func usage(o io.Writer) {
	version(o)
	fmt.Fprintln(o)
	fmt.Fprintln(o, "jsonnet-deps {<option>} <filename>...")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "Available options:")
	fmt.Fprintln(o, "  -h / --help                This message")
	fmt.Fprintln(o, "  -J / --jpath <dir>         Specify an additional library search dir")
	fmt.Fprintln(o, "                             (right-most wins)")
	fmt.Fprintln(o, "  -o / --output-file <file>  Write to the output file rather than stdout")
	fmt.Fprintln(o, "  --version                  Print version")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "Environment variables:")
	fmt.Fprintln(o, "  JSONNET_PATH is a colon (semicolon on Windows) separated list of directories")
	fmt.Fprintln(o, "  added in reverse order before the paths specified by --jpath (i.e. left-most")
	fmt.Fprintln(o, "  wins). E.g. these are equivalent:")
	fmt.Fprintln(o, "    JSONNET_PATH=a:b jsonnet -J c -J d")
	fmt.Fprintln(o, "    JSONNET_PATH=d:c:a:b jsonnet")
	fmt.Fprintln(o, "    jsonnet -J b -J a -J c -J d")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "In all cases:")
	fmt.Fprintln(o, "  Multichar options are expanded e.g. -abc becomes -a -b -c.")
	fmt.Fprintln(o, "  The -- option suppresses option processing for subsequent arguments.")
	fmt.Fprintln(o, "  Note that since filenames and jsonnet programs can begin with -, it is")
	fmt.Fprintln(o, "  advised to use -- if the argument is unknown, e.g. jsonnet-deps -- \"$FILENAME\".")
}

type config struct {
	inputFiles []string
	outputFile string
	jPaths     []string
}

type processArgsStatus int

const (
	processArgsStatusContinue     = iota
	processArgsStatusSuccessUsage = iota
	processArgsStatusFailureUsage = iota
	processArgsStatusSuccess      = iota
	processArgsStatusFailure      = iota
)

func processArgs(givenArgs []string, conf *config, vm *jsonnet.VM) (processArgsStatus, error) {
	args := cmd.SimplifyArgs(givenArgs)
	remainingArgs := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" {
			return processArgsStatusSuccessUsage, nil
		} else if arg == "-v" || arg == "--version" {
			version(os.Stdout)
			return processArgsStatusSuccess, nil
		} else if arg == "-o" || arg == "--output-file" {
			outputFile := cmd.NextArg(&i, args)
			if len(outputFile) == 0 {
				return processArgsStatusFailure, fmt.Errorf("-o argument was empty string")
			}
			conf.outputFile = outputFile
		} else if arg == "-J" || arg == "--jpath" {
			dir := cmd.NextArg(&i, args)
			if len(dir) == 0 {
				return processArgsStatusFailure, fmt.Errorf("-J argument was empty string")
			}
			conf.jPaths = append(conf.jPaths, dir)
		} else if arg == "--" {
			// All subsequent args are not options.
			i++
			for ; i < len(args); i++ {
				remainingArgs = append(remainingArgs, args[i])
			}
			break
		} else if len(arg) > 1 && arg[0] == '-' {
			return processArgsStatusFailure, fmt.Errorf("unrecognized argument: %s", arg)
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}

	if len(remainingArgs) == 0 {
		return processArgsStatusFailureUsage, fmt.Errorf("must give filename")
	}
	conf.inputFiles = remainingArgs

	return processArgsStatusContinue, nil
}

func writeDependencies(dependencies []string, outputFile string) (err error) {
	var f *os.File

	if outputFile == "" {
		f = os.Stdout
	} else {
		f, err = os.Create(outputFile)
		if err != nil {
			return err
		}
		defer func() {
			if ferr := f.Close(); ferr != nil {
				err = ferr
			}
		}()
	}

	if len(dependencies) != 0 {
		output := strings.Join(dependencies, "\n") + "\n"
		_, err = f.WriteString(output)
		if err != nil {
			return err
		}
	}

	return
}

func main() {
	cmd.StartCPUProfile()
	defer cmd.StopCPUProfile()

	vm := jsonnet.MakeVM()
	vm.ErrorFormatter.SetColorFormatter(color.New(color.FgRed).Fprintf)

	conf := config{}
	jsonnetPath := filepath.SplitList(os.Getenv("JSONNET_PATH"))
	for i := len(jsonnetPath) - 1; i >= 0; i-- {
		conf.jPaths = append(conf.jPaths, jsonnetPath[i])
	}

	status, err := processArgs(os.Args[1:], &conf, vm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: "+err.Error())
	}
	switch status {
	case processArgsStatusContinue:
		break
	case processArgsStatusSuccessUsage:
		usage(os.Stdout)
		os.Exit(0)
	case processArgsStatusFailureUsage:
		if err != nil {
			fmt.Fprintln(os.Stderr, "")
		}
		usage(os.Stderr)
		os.Exit(1)
	case processArgsStatusSuccess:
		os.Exit(0)
	case processArgsStatusFailure:
		os.Exit(1)
	}

	vm.Importer(&jsonnet.FileImporter{JPaths: conf.jPaths})

	for _, file := range conf.inputFiles {
		if _, err := os.Stat(file); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	dependencies, err := vm.FindDependencies("", conf.inputFiles)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	cmd.MemProfile()

	err = writeDependencies(dependencies, conf.outputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
