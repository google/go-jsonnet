package jsonnet

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

var (
	jsonnetPath    = flag.String("jsonnetPath", "./jsonnet", "Path to jsonnet binary")
	outputPassthru = flag.Bool("outputPassthru", false, "Pass stdout/err from jsonnet")
)

func RunBenchmark(b *testing.B, name string) {
	for n := 0; n < b.N; n++ {
		cmd := exec.Command(*jsonnetPath, fmt.Sprintf("./builtin-benchmarks/%s.jsonnet", name))
		if *outputPassthru {
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

func Benchmark_Builtin_base64Decode(b *testing.B) {
	RunBenchmark(b, "base64Decode")
}

func Benchmark_Builtin_base64DecodeBytes(b *testing.B) {
	RunBenchmark(b, "base64DecodeBytes")
}

func Benchmark_Builtin_base64(b *testing.B) {
	RunBenchmark(b, "base64")
}

func Benchmark_Builtin_base64_byte_array(b *testing.B) {
	RunBenchmark(b, "base64_byte_array")
}
