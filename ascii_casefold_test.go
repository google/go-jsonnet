package jsonnet

import (
	"strings"
	"testing"
)

func TestEqualsIgnoreCaseASCIIParity(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{`std.equalsIgnoreCase("Hello", "hELLO")`, "true"},
		{`std.equalsIgnoreCase("", "")`, "true"},
		{`std.equalsIgnoreCase("a", "ab")`, "false"},
		{`std.equalsIgnoreCase("\u00e9", "\u00c9")`, "false"},
		{`std.equalsIgnoreCase("\u212a", "k")`, "false"},
		{`std.equalsIgnoreCase("\u017f", "s")`, "false"},
		{`std.equalsIgnoreCase("\u00e9A", "\u00e9a")`, "true"},
		{`std.equalsIgnoreCase("\u00e9A", "\u00c9a")`, "false"},
	} {
		t.Run(test.input, func(t *testing.T) {
			output, err := MakeVM().EvaluateAnonymousSnippet("casefold.jsonnet", test.input)
			if err != nil || strings.TrimSpace(output) != test.want {
				t.Fatalf("got %q, %v; want %s", output, err, test.want)
			}
		})
	}
}
