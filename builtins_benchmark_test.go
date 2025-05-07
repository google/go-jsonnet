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

func Benchmark_Builtin_parseInt(b *testing.B) {
	RunBenchmark(b, "parseInt")
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

func Benchmark_Builtin_escapeStringJson(b *testing.B) {
	RunBenchmark(b, "escapeStringJson")
}

func Benchmark_Builtin_manifestJsonEx(b *testing.B) {
	RunBenchmark(b, "manifestJsonEx")
}

func Benchmark_Builtin_manifestTomlEx(b *testing.B) {
	RunBenchmark(b, "manifestTomlEx")
}

func Benchmark_Builtin_manifestYamlDoc(b *testing.B) {
	RunBenchmark(b, "manifestYamlDoc")
}

func Benchmark_Builtin_comparison(b *testing.B) {
	RunBenchmark(b, "comparison")
}

func Benchmark_Builtin_comparison2(b *testing.B) {
	RunBenchmark(b, "comparison2")
}

func Benchmark_Builtin_foldl(b *testing.B) {
	RunBenchmark(b, "foldl")
}

func Benchmark_Builtin_member(b *testing.B) {
	RunBenchmark(b, "member")
}

func Benchmark_Builtin_lstripChars(b *testing.B) {
	RunBenchmark(b, "lstripChars")
}

func Benchmark_Builtin_rstripChars(b *testing.B) {
	RunBenchmark(b, "rstripChars")
}

func Benchmark_Builtin_stripChars(b *testing.B) {
	RunBenchmark(b, "stripChars")
}
