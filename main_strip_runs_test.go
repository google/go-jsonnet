package jsonnet

import (
	"fmt"
	"strings"
	"testing"
)

func TestStripCharsLongRuns(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{`std.lstripChars(std.repeat(" ", 1000) + "x", " ")`, `"x"`},
		{`std.rstripChars("x" + std.repeat(" ", 1000), " ")`, `"x"`},
		{`std.stripChars("\u00e9\u00e9x\u00e9\u00e9", "\u00e9")`, `"x"`},
		{`std.lstripChars("aaab", ["a"])`, `"b"`},
		{`std.rstripChars("baaa", ["a"])`, `"b"`},
		{`std.stripChars("abc", "")`, `"abc"`},
		{`std.stripChars("", "")`, `""`},
	} {
		t.Run(test.input, func(t *testing.T) {
			output, err := MakeVM().EvaluateAnonymousSnippet("strip.jsonnet", test.input)
			if err != nil || strings.TrimSpace(output) != test.want {
				t.Fatalf("got %q, %v; want %s", output, err, test.want)
			}
		})
	}
}

func BenchmarkStripCharsLongRuns(b *testing.B) {
	for _, size := range []int{1000, 10000} {
		for _, strip := range []struct {
			name string
			fn   func(*interpreter, value, value) (value, error)
		}{
			{"left", builtinLstripChars},
			{"right", builtinRstripChars},
		} {
			b.Run(fmt.Sprintf("%s/%d", strip.name, size), func(b *testing.B) {
				text := strings.Repeat(" ", size) + "x"
				if strip.name == "right" {
					text = "x" + strings.Repeat(" ", size)
				}
				input, chars := makeValueString(text), makeValueString(" ")
				i := &interpreter{}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					result, err := strip.fn(i, input, chars)
					if err != nil || result.(valueString).getGoString() != "x" {
						b.Fatalf("incorrect strip result: %v, %v", result, err)
					}
				}
			})
		}
	}
}
