package jsonnet

import (
	"encoding/json"
	"math"
	"testing"
)

func TestRoundHalfTowardsPositiveInfinity(t *testing.T) {
	for _, test := range []struct {
		input string
		want  float64
	}{
		{"-2.5", -2},
		{"-1.5", -1},
		{"-0.5", 0},
		{"-0.0", 0},
		{"-0.49999999999999994", 0},
		{"-0.5000000000000001", -1},
		{"0.49999999999999994", 0},
		{"0.5", 1},
		{"1.5", 2},
		{"-2.49", -2},
		{"-2.51", -3},
		{"4503599627370497", 4503599627370497},
		{"-4503599627370497", -4503599627370497},
	} {
		t.Run(test.input, func(t *testing.T) {
			output, err := MakeVM().EvaluateAnonymousSnippet("round.jsonnet", "std.round("+test.input+")")
			if err != nil {
				t.Fatal(err)
			}
			var got float64
			if err := json.Unmarshal([]byte(output), &got); err != nil {
				t.Fatal(err)
			}
			if got != test.want || (got == 0 && math.Signbit(got)) {
				t.Fatalf("got %v (signbit %t); want %v", got, math.Signbit(got), test.want)
			}
		})
	}
}
