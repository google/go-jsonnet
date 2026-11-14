package jsonnet

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestParseIntFiniteDecimalRange(t *testing.T) {
	for _, test := range []struct {
		input string
		want  float64
	}{
		{"0", 0},
		{"-0", math.Copysign(0, -1)},
		{"00012", 12},
		{"-00012", -12},
		{"100000000000000000000", 1e20},
		{"-100000000000000000000", -1e20},
	} {
		t.Run(test.input, func(t *testing.T) {
			output, err := MakeVM().EvaluateAnonymousSnippet("parse.jsonnet", fmt.Sprintf("std.parseInt(%q)", test.input))
			if err != nil {
				t.Fatal(err)
			}
			var got float64
			if err := json.Unmarshal([]byte(output), &got); err != nil {
				t.Fatal(err)
			}
			if got != test.want || math.Signbit(got) != math.Signbit(test.want) {
				t.Fatalf("got %v (signbit %t); want %v", got, math.Signbit(got), test.want)
			}
		})
	}
}

func TestParseIntCanonicalDigitFold(t *testing.T) {
	for _, input := range []string{
		"9223372036854775808",
		"-9223372036854775809",
		strings.Repeat("1234567890", 20),
		"-" + strings.Repeat("9876543210", 20),
	} {
		source := fmt.Sprintf(`
local parseNat(s) = std.foldl(
  function(aggregate, char) 10 * aggregate + (std.codepoint(char) - 48),
  std.stringChars(s), 0);
local reference(s) = if s[0] == '-' then -parseNat(s[1:]) else parseNat(s);
std.parseInt(%q) == reference(%q)
`, input, input)
		output, err := MakeVM().EvaluateAnonymousSnippet("parse.jsonnet", source)
		if err != nil || strings.TrimSpace(output) != "true" {
			t.Errorf("%s: got %q, %v; want canonical digit-fold result", input, output, err)
		}
	}
}

func TestParseIntRejectsInvalidDecimalSyntax(t *testing.T) {
	for _, input := range []string{"", "-", "+1", "+0", "--1", " 1", "1 ", "1.0", "1e3", "1_000", "0x10", "\u0661"} {
		_, err := MakeVM().EvaluateAnonymousSnippet("parse.jsonnet", fmt.Sprintf("std.parseInt(%q)", input))
		if err == nil || !strings.Contains(err.Error(), "is not a base 10 integer") {
			t.Errorf("%q: expected decimal syntax error, got %v", input, err)
		}
	}
}

func TestParseIntRejectsFloatOverflow(t *testing.T) {
	for _, sign := range []string{"", "-"} {
		input := sign + "1" + strings.Repeat("0", 309)
		_, err := MakeVM().EvaluateAnonymousSnippet("parse.jsonnet", fmt.Sprintf("std.parseInt(%q)", input))
		if err == nil || !strings.Contains(err.Error(), "RUNTIME ERROR: Overflow") {
			t.Errorf("expected floating-point overflow, got %v", err)
		}
	}
}
