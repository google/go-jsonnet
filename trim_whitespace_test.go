package jsonnet

import (
	"encoding/json"
	"testing"
)

func TestTrimWhitespaceParity(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{`std.trim(" \t\n\f\rx \t\n\f\r")`, "x"},
		{`std.trim("\u0085\u00a0x\u0085\u00a0")`, "x"},
		{`std.trim("\u000bx\u000b")`, "\vx\v"},
		{`std.trim("\u1680x\u1680")`, "\u1680x\u1680"},
		{`std.trim("\u2000x\u2000")`, "\u2000x\u2000"},
		{`std.trim("\u2028x\u2028")`, "\u2028x\u2028"},
		{`std.trim("\u3000x\u3000")`, "\u3000x\u3000"},
		{`std.trim(" \u2000x\u2000 ")`, "\u2000x\u2000"},
		{`std.trim("")`, ""},
	} {
		t.Run(test.input, func(t *testing.T) {
			output, err := MakeVM().EvaluateAnonymousSnippet("trim.jsonnet", test.input)
			if err != nil {
				t.Fatal(err)
			}
			var got string
			if err := json.Unmarshal([]byte(output), &got); err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %q; want %q", got, test.want)
			}
		})
	}
}
