# go-jsonnet

[![GoDoc Widget]][GoDoc] [![Coverage Status Widget]][Coverage Status]

[GoDoc]: https://godoc.org/github.com/google/go-jsonnet
[GoDoc Widget]: https://godoc.org/github.com/google/go-jsonnet?status.png
[Coverage Status Widget]: https://coveralls.io/repos/github/google/go-jsonnet/badge.svg?branch=master
[Coverage Status]: https://coveralls.io/github/google/go-jsonnet?branch=master

This an implementation of [Jsonnet](http://jsonnet.org/) in pure Go. It is a feature complete, production-ready implementation. Bindings to C and Python are available (but not battle-tested).

This code is known to work on Go 1.24 and above. We recommend always using the newest stable release of Go.

Jsonnet can be used on the command line to evaluate Jsonnet input files and produce JSON, Yaml, or other outputs, or it can be used as a library.

**Security note:** If you are running a system that evaluates untrusted Jsonnet
code, extra care should be used to defend against data exfiltration risks. By
default, the `import`, `importstr` and `importbin` language constructs can read
from any path accessible to the Jsonnet process. The Jsonnet library allows you
to provide your own logic for processing imports, which is one way to restrict
imports to known safe sources. You could also consider running Jsonnet inside a
carefully configured gVisor or Firecracker container or other secure container
system to provide more general isolation if evaluating untrusted code.

## Installing the command line tool

```shell
# Using `go get` to install binaries is deprecated.
# The version suffix is mandatory.
go install github.com/google/go-jsonnet/cmd/jsonnet@latest

# Or other tools in the 'cmd' directory
go install github.com/google/go-jsonnet/cmd/jsonnet-lint@latest
```

It's also available on Homebrew:

```
brew install go-jsonnet
```

## Installing tools for working with Jsonnet code (formatter, linter)

`jsonnetfmt` and `jsonnet-lint` are also available as [pre-commit](https://github.com/pre-commit/pre-commit) hooks. Example `.pre-commit-config.yaml`:
```yaml
- repo: https://github.com/google/go-jsonnet
  rev: # ref you want to point at, e.g. v0.17.0
  hooks:
    - id: jsonnet-format
    - id: jsonnet-lint
```

## Using the library

It can also be embedded in your own Go programs as a library:

```go
package main

import (
	"fmt"
	"log"

	"github.com/google/go-jsonnet"
)

func main() {
	vm := jsonnet.MakeVM()

	snippet := `{
		person1: {
		    name: "Alice",
		    welcome: "Hello " + self.name + "!",
		},
		person2: self.person1 { name: "Bob" },
	}`

	jsonStr, err := vm.EvaluateAnonymousSnippet("example1.jsonnet", snippet)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(jsonStr)
	/*
	   {
	     "person1": {
	         "name": "Alice",
	         "welcome": "Hello Alice!"
	     },
	     "person2": {
	         "name": "Bob",
	         "welcome": "Hello Bob!"
	     }
	   }
	*/
}
```

## Build instructions (go 1.24+)

```bash
git clone git@github.com:google/go-jsonnet.git
cd go-jsonnet
go build ./cmd/jsonnet
go build ./cmd/jsonnetfmt
go build ./cmd/jsonnet-deps
```
To build with [Bazel](https://bazel.build/) instead:
```bash
git clone git@github.com:google/go-jsonnet.git
cd go-jsonnet
git submodule init
git submodule update
bazel build //cmd/jsonnet
bazel build //cmd/jsonnetfmt
bazel build //cmd/jsonnet-deps
```
The resulting _jsonnet_ program will then be available at a platform-specific path, such as _bazel-bin/cmd/jsonnet/darwin_amd64_stripped/jsonnet_ for macOS.

Bazel also accommodates cross-compiling the program. To build the _jsonnet_ program for various popular platforms, run the following commands:

Target platform | Build command
--------------- | -------------------------------------------------------------------------------------
Current host    | _bazel build //cmd/jsonnet_
Linux           | _bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //cmd/jsonnet_
macOS           | _bazel build --platforms=@io_bazel_rules_go//go/toolchain:darwin_amd64 //cmd/jsonnet_
Windows         | _bazel build --platforms=@io_bazel_rules_go//go/toolchain:windows_amd64 //cmd/jsonnet_

For additional target platform names, see the per-Go release definitions [here](https://github.com/bazelbuild/rules_go/blob/master/go/private/sdk_list.bzl#L21-L31) in the _rules_go_ Bazel package.

Additionally if any files were moved around, see the section [Keeping the Bazel files up to date](#keeping-the-bazel-files-up-to-date).

## Building libjsonnet.wasm

The [WASM](https://webassembly.org/) build can be used to embed go-jsonnet for use (client side) in the web browser, or in other WASM execution environments. For example, this is used to evaluate the live code snippets on https://jsonnet.org/

```bash
GOOS=js GOARCH=wasm go build -o libjsonnet.wasm ./cmd/wasm
```

Or if using bazel:

```
bazel build //cmd/wasm:libjsonnet.wasm
```

## Running tests

```bash
./tests.sh  # Also runs `go test ./...`
```

### Adding and updating golden tests

Most language behavior tests live in `testdata/` as pairs of Jsonnet input
files and `.golden` expected-output files. To add a new core language test, add
`testdata/<name>.jsonnet`, then regenerate and review the matching golden file:

```bash
go test . -run TestEval -update
```

Formatter and linter golden tests use the same update flag in their own
packages:

```bash
go test ./formatter -run TestFormatter -update
go test ./linter -run TestLinter -update
```

The formatter and linter update runs intentionally report a `Goldens_Updated`
subtest failure so the list of changed files is visible in the test output.
Always review the generated `.golden`, `.fmt.golden`, or `.linter.golden`
changes before committing them, then run `go test ./...` or `./tests.sh`.

## Running Benchmarks

### Method 1

```bash
go get golang.org/x/tools/cmd/benchcmp
```

1. Make sure you build a jsonnet binary _prior_ to making changes.

```bash
go build -o jsonnet-old ./cmd/jsonnet
```

2. Make changes (iterate as needed), and rebuild new binary

```bash
go build ./cmd/jsonnet
```

3. Run benchmark:

```bash
# e.g. ./benchmark.sh Builtin
./benchmark.sh <TestNameFilter>
```

### Method 2

1. get `benchcmp`

```bash
go get golang.org/x/tools/cmd/benchcmp
```

2. Make sure you build a jsonnet binary _prior_ to making changes.

```bash
make build-old
```

3. iterate with (which will also automatically rebuild the new binary `./jsonnet`)

_replace the FILTER with the name of the test you are working on_

```bash
FILTER=Builtin_manifestJsonEx make benchmark
```

## Update cpp-jsonnet sub-repo

This repo depends on [the original Jsonnet repo](https://github.com/google/jsonnet). Shared parts include the standard library, headers files for C API and some tests.

You can update the submodule and regenerate dependent files with one command:
```
./update_cpp_jsonnet.sh
```

Note: It needs to be run from repo root.

## Updating and modifying the standard library

Standard library source code is kept in `cpp-jsonnet` submodule, because it is shared with [Jsonnet C++
implementation](https://github.com/google/jsonnet).

For performance reasons we perform preprocessing on the standard library, so for the changes to be visible, regeneration is necessary:

```bash
go run cmd/dumpstdlibast/dumpstdlibast.go cpp-jsonnet/stdlib/std.jsonnet > astgen/stdast.go
```

The above command creates the _astgen/stdast.go_ file which puts the desugared standard library into the right data structures, which lets us avoid the parsing overhead during execution. Note that this step is not necessary to perform manually when building with Bazel; the Bazel target regenerates the _astgen/stdast.go_ (writing it into Bazel's build sandbox directory tree) file when necessary.

## Keeping the Bazel files up to date
Note that we maintain the Go-related Bazel targets with [the Gazelle tool](https://github.com/bazelbuild/bazel-gazelle). The Go module (_go.mod_ in the root directory) remains the primary source of truth. Gazelle analyzes both that file and the rest of the Go files in the repository to create and adjust appropriate Bazel targets for building Go packages and executable programs.

After changing any dependencies within the files covered by this Go module, it is helpful to run _go mod tidy_ to ensure that the module declarations match the state of the Go source code. In order to synchronize the Bazel rules with material changes to the Go module, run the gazelle command:
```bash
bazel run //:gazelle
```
