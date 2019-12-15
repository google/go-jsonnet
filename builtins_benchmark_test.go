package jsonnet

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

var jsonnetPath string
var outputPassthru bool

func init() {
	flag.StringVar(&jsonnetPath, "jsonnetPath", "./jsonnet", "Path to jsonnet binary")
	flag.BoolVar(&outputPassthru, "outputPassthru", false, "Pass stdout/err from jsonnet")
}

func RunBenchmark(b *testing.B, name string) {
	for n := 0; n < b.N; n++ {
		cmd := exec.Command(jsonnetPath, fmt.Sprintf("./builtin-benchmarks/%s.jsonnet", name))
		if outputPassthru {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		err := cmd.Run()
		if err != nil {
			b.Fail()
		}
	}
}

func Benchmark_Builtin_substr(b *testing.B) {
	RunBenchmark(b, "substr")
}

func Benchmark_Builtin_reverse(b *testing.B) {
	RunBenchmark(b, "reverse")
}
