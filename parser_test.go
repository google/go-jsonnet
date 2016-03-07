/*
Copyright 2016 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package jsonnet

import (
	"fmt"
	"testing"

	"github.com/kr/pretty"
)

var tests = []string{
	`true`,
	`1`,
	`1.2e3`,
	`!true`,
	`null`,

	`$.foo.bar`,
	`self.foo.bar`,
	`super.foo.bar`,
	`super[1]`,
	`error "Error!"`,

	`"world"`,
	`'world'`,
	`|||
   world
|||`,

	`foo(bar)`,
	`foo.bar`,
	`foo[bar]`,

	`true || false`,
	`0 && 1 || 0`,
	`0 && (1 || 0)`,

	`local foo = "bar"; foo`,
	`local foo(bar) = bar; foo(1)`,
	`{ local foo = "bar", baz: 1}`,
	`{ local foo(bar) = bar, baz: foo(1)}`,

	`{ foo(bar, baz): bar+baz }`,

	`{ ["foo" + "bar"]: 3 }`,
	`{ ["field" + x]: x for x in [1, 2, 3] }`,
	`{ ["field" + x]: x for x in [1, 2, 3] if x <= 2 }`,
	`{ ["field" + x + y]: x + y for x in [1, 2, 3] if x <= 2 for y in [4, 5, 6]}`,

	`[]`,
	`[a, b, c]`,
	`[x for x in [1,2,3] ]`,
	`[x for x in [1,2,3] if x <= 2]`,
	`[x+y for x in [1,2,3] if x <= 2 for y in [4, 5, 6]]`,

	`{}`,
	`{ hello: "world" }`,
	`{ hello +: "world" }`,
	`{
  hello: "world",
	"name":: joe,
	'mood'::: "happy",
	|||
	  key type
|||: "block",
}`,

	`assert true: 'woah!'; true`,
	`{ assert true: 'woah!', foo: bar }`,

	`if n > 1 then 'foos' else 'foo'`,

	`local foo = function(x) x + 1; true`,

	`import 'foo.jsonnet'`,
	`importstr 'foo.text'`,

	`{a: b} + {c: d}`,
	`{a: b}{c: d}`,
}

func TestParser(t *testing.T) {
	for _, s := range tests {
		tokens, err := lex("test", s)
		if err != nil {
			t.Errorf("Unexpected lex error\n  input: %v\n  error: %v", s, err)
			continue
		}
		ast, err := parse(tokens)
		if err != nil {
			t.Errorf("Unexpected parse error\n  input: %v\n  error: %v", s, err)
		}
		if false {
			fmt.Printf("input: %v\nast: %# v\n\n", s, pretty.Formatter(ast))
		}
	}
}
