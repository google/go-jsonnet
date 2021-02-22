package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/google/go-jsonnet/cmd/internal/cmd"
	"github.com/google/go-jsonnet/linter"

	jsonnet "github.com/google/go-jsonnet"
)

func version(o io.Writer) {
	fmt.Fprintf(o, "Jsonnet linter %s\n", jsonnet.Version())
}

func usage(o io.Writer) {
	version(o)
	fmt.Fprintln(o)
	fmt.Fprintln(o, "jsonnet-lint {<option>} { <filename> }")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "Available options:")
	fmt.Fprintln(o, "  -h / --help                This message")
	fmt.Fprintln(o, "  -J / --jpath <dir>         Specify an additional library search dir")
	fmt.Fprintln(o, "                             (right-most wins)")
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
	fmt.Fprintln(o, "  <filename> can be - (stdin)")
	fmt.Fprintln(o, "  Multichar options are expanded e.g. -abc becomes -a -b -c.")
	fmt.Fprintln(o, "  The -- option suppresses option processing for subsequent arguments.")
	fmt.Fprintln(o, "  Note that since filenames and jsonnet programs can begin with -, it is")
	fmt.Fprintln(o, "  advised to use -- if the argument is unknown, e.g. jsonnetfmt -- \"$FILENAME\".")
	fmt.Fprintln(o)
	fmt.Fprintln(o, "Exit code:")
	fmt.Fprintln(o, "  0 – If the file was checked no problems were found.")
	fmt.Fprintln(o, "  1 – If errors occured which prevented checking (e.g. specified file is missing).")
	fmt.Fprintln(o, "  2 – If problems were found.")

}

type config struct {
	// TODO(sbarzowski) Allow multiple root files checked at once for greater efficiency
	inputFile string
	evalJpath []string
}

func makeConfig() config {
	return config{
		evalJpath: []string{},
	}
}

type processArgsStatus int

const (
	processArgsStatusContinue     = iota
	processArgsStatusSuccessUsage = iota
	processArgsStatusFailureUsage = iota
	processArgsStatusSuccess      = iota
	processArgsStatusFailure      = iota
)

func processArgs(givenArgs []string, config *config, vm *jsonnet.VM) (processArgsStatus, error) {
	args := cmd.SimplifyArgs(givenArgs)
	remainingArgs := make([]string, 0, len(args))
	i := 0

	for ; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			// All subsequent args are not options.
			i++
			for ; i < len(args); i++ {
				remainingArgs = append(remainingArgs, args[i])
			}
			break
		} else if arg == "-h" || arg == "--help" {
			return processArgsStatusSuccessUsage, nil
		} else if arg == "-v" || arg == "--version" {
			version(os.Stdout)
			return processArgsStatusSuccess, nil
		} else if arg == "-J" || arg == "--jpath" {
			dir := cmd.NextArg(&i, args)
			if len(dir) == 0 {
				return processArgsStatusFailure, fmt.Errorf("-J argument was empty string")
			}
			if dir[len(dir)-1] != '/' {
				dir += "/"
			}
			config.evalJpath = append(config.evalJpath, dir)
		} else if len(arg) > 1 && arg[0] == '-' {
			return processArgsStatusFailure, fmt.Errorf("unrecognized argument: %s", arg)
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}

	if len(remainingArgs) == 0 {
		return processArgsStatusFailureUsage, fmt.Errorf("file not provided")
	}

	if len(remainingArgs) > 1 {
		return processArgsStatusFailure, fmt.Errorf("only one file is allowed")
	}

	config.inputFile = remainingArgs[0]
	return processArgsStatusContinue, nil
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
	os.Exit(1)
}

func main() {
	cmd.StartCPUProfile()
	defer cmd.StopCPUProfile()

	vm := jsonnet.MakeVM()
	vm.ErrorFormatter.SetColorFormatter(color.New(color.FgRed).Fprintf)

	config := makeConfig()
	jsonnetPath := filepath.SplitList(os.Getenv("JSONNET_PATH"))
	for i := len(jsonnetPath) - 1; i >= 0; i-- {
		config.evalJpath = append(config.evalJpath, jsonnetPath[i])
	}

	status, err := processArgs(os.Args[1:], &config, vm)
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

	vm.Importer(&jsonnet.FileImporter{
		JPaths: config.evalJpath,
	})

	inputFile, err := os.Open(config.inputFile)
	if err != nil {
		die(err)
	}
	data, err := ioutil.ReadAll(inputFile)
	if err != nil {
		die(err)
	}
	err = inputFile.Close()
	if err != nil {
		die(err)
	}

	cmd.MemProfile()

	_, err = jsonnet.SnippetToAST(config.inputFile, string(data))
	if err != nil {
		die(err)
	}

	errorsFound := linter.LintSnippet(vm, os.Stderr, config.inputFile, string(data))
	if errorsFound {
		fmt.Fprintf(os.Stderr, "Problems found!\n")
		os.Exit(2)
	}
}
