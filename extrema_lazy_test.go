package jsonnet

import (
	"strings"
	"testing"
)

func TestExtremaUnusedArguments(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{`std.minArray([2, 1], onEmpty=error "unused")`, "1"},
		{`std.maxArray([2, 1], onEmpty=error "unused")`, "2"},
		{`std.minArray([], keyF=error "unused", onEmpty=7)`, "7"},
		{`std.maxArray([], keyF=error "unused", onEmpty=7)`, "7"},
		{`std.minArray([3], keyF=error "unused")`, "3"},
		{`std.maxArray([3], keyF=error "unused")`, "3"},
		{`std.minArray([3], keyF=function(x) error "unused")`, "3"},
		{`std.maxArray([3], keyF=function(x) error "unused")`, "3"},
		{`std.minArray([{x: 2}, {x: 1}], keyF=function(x) x.x).x`, "1"},
		{`std.maxArray([{x: 2}, {x: 1}], keyF=function(x) x.x).x`, "2"},
	} {
		t.Run(test.input, func(t *testing.T) {
			output, err := MakeVM().EvaluateAnonymousSnippet("extrema.jsonnet", test.input)
			if err != nil || strings.TrimSpace(output) != test.want {
				t.Fatalf("got %q, %v; want %s", output, err, test.want)
			}
		})
	}
}

func TestExtremaRequiredArgumentErrors(t *testing.T) {
	for _, test := range []struct{ input, message string }{
		{`std.minArray([])`, "Expected at least one element"},
		{`std.maxArray([])`, "Expected at least one element"},
		{`std.minArray([], onEmpty=error "empty fallback")`, "empty fallback"},
		{`std.maxArray([], onEmpty=error "empty fallback")`, "empty fallback"},
		{`std.minArray([1], onEmpty=error "strict fallback") tailstrict`, "strict fallback"},
		{`std.maxArray([1], onEmpty=error "strict fallback") tailstrict`, "strict fallback"},
		{`std.minArray([], keyF=error "strict key", onEmpty=7) tailstrict`, "strict key"},
		{`std.maxArray([], keyF=error "strict key", onEmpty=7) tailstrict`, "strict key"},
	} {
		t.Run(test.input, func(t *testing.T) {
			_, err := MakeVM().EvaluateAnonymousSnippet("extrema.jsonnet", test.input)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("got %v; want error containing %q", err, test.message)
			}
		})
	}
}
