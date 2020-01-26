package jsonnet

import (
	"flag"
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

func Benchmark_Builtin_substr(b *testing.B) {
	for n := 0; n < b.N; n++ {
		cmd := exec.Command(jsonnetPath, "./builtin-benchmarks/substr.jsonnet")
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
