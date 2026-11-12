package jsonnet

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRangeMaximumInt32Endpoint(t *testing.T) {
	for _, test := range []struct {
		input string
		want  []int64
	}{
		{`std.range(2147483647, 2147483647)`, []int64{2147483647}},
		{`std.range(2147483646, 2147483647)`, []int64{2147483646, 2147483647}},
		{`std.range(-2, 1)`, []int64{-2, -1, 0, 1}},
		{`std.range(2, 1)`, []int64{}},
	} {
		t.Run(test.input, func(t *testing.T) {
			output, err := MakeVM().EvaluateAnonymousSnippet("range.jsonnet", test.input)
			if err != nil {
				t.Fatal(err)
			}
			var values []int64
			if err := json.Unmarshal([]byte(output), &values); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(values, test.want) {
				t.Fatalf("got %v, want %v", values, test.want)
			}
		})
	}
}
