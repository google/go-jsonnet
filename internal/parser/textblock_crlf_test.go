/*
Copyright 2026 Google Inc. All rights reserved.

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

package parser

import "testing"

// Text blocks with CRLF line endings (common on Windows) previously failed
// with "Text block not terminated with |||" because blank-line detection only
// looked for '\n' and left a leading '\r' that broke the indent check (#786).
func TestLexTextBlockCRLF(t *testing.T) {
	input := "{\r\n   |||\r\n      Hello\r\n\r\n      World\r\n   |||,\r\n}\r\n"
	tokens, err := Lex("test.jsonnet", "", input)
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	var block *token
	for i := range tokens {
		if tokens[i].kind == tokenStringBlock {
			block = &tokens[i]
			break
		}
	}
	if block == nil {
		t.Fatalf("expected tokenStringBlock, got %#v", tokens)
	}
	want := "Hello\n\nWorld\n"
	if block.data != want {
		t.Fatalf("string block data = %q, want %q", block.data, want)
	}
}

func TestLexTextBlockLFUnchanged(t *testing.T) {
	input := "{\n   |||\n      Hello\n\n      World\n   |||,\n}\n"
	tokens, err := Lex("test.jsonnet", "", input)
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	var block *token
	for i := range tokens {
		if tokens[i].kind == tokenStringBlock {
			block = &tokens[i]
			break
		}
	}
	if block == nil {
		t.Fatalf("expected tokenStringBlock")
	}
	if block.data != "Hello\n\nWorld\n" {
		t.Fatalf("string block data = %q", block.data)
	}
}

func TestLexTextBlockCRLFChomp(t *testing.T) {
	input := "{x: |||\r\n  hi\r\n|||}\r\n"
	tokens, err := Lex("test.jsonnet", "", input)
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	var block *token
	for i := range tokens {
		if tokens[i].kind == tokenStringBlock {
			block = &tokens[i]
			break
		}
	}
	if block == nil {
		t.Fatalf("expected tokenStringBlock")
	}
	if block.data != "hi\n" {
		t.Fatalf("string block data = %q, want %q", block.data, "hi\n")
	}
}
