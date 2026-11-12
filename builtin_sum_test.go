package jsonnet

import (
	"strings"
	"testing"
)

func TestSumRejectsOverflow(t *testing.T) {
	for _, input := range []string{
		`std.sum([1e308, 1e308])`,
		`std.sum([-1e308, -1e308])`,
		`std.avg([1e308, 1e308])`,
		`std.sum([1e308, 1e308, -1e308])`,
		`std.sum([1e308, 1e308]) > 0`,
		`std.sum([1e308, 1e308, error "must not be evaluated"])`,
	} {
		t.Run(input, func(t *testing.T) {
			_, err := MakeVM().EvaluateAnonymousSnippet("sum.jsonnet", input)
			if err == nil || !strings.Contains(err.Error(), "RUNTIME ERROR: Overflow") {
				t.Fatalf("expected overflow runtime error, got %v", err)
			}
		})
	}
}

func TestSumFinite(t *testing.T) {
	for _, input := range []string{
		`std.sum([]) == 0`,
		`std.sum([1, 2, 3]) == 6`,
		`std.sum([1e308, -1e308]) == 0`,
		`std.avg([1, 2, 3]) == 2`,
	} {
		output, err := MakeVM().EvaluateAnonymousSnippet("sum.jsonnet", input)
		if err != nil || strings.TrimSpace(output) != "true" {
			t.Errorf("%s: got %q, %v", input, output, err)
		}
	}
}
