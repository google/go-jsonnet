package jsonnet

import (
	"encoding/json"
	"testing"
)

func TestSubstrLargeBounds(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"large start", `std.substr("abc", 1e100, 1)`, `""`},
		{"large start empty string", `std.substr("", 1e100, 0)`, `""`},
		{"maximum length", `std.substr("abc", 1, 2147483647)`, `"bc"`},
		{"unicode maximum length", `std.substr("a\u00e9\u263a", 1, 2147483647)`, `"\u00e9\u263a"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := MakeVM().EvaluateAnonymousSnippet("substring.jsonnet", test.input)
			if err != nil {
				t.Fatal(err)
			}
			var got, want string
			if err := json.Unmarshal([]byte(output), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.want), &want); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}
