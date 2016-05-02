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

// Note: There are no garbage collection params because we're using the native
// Go garbage collector.

// VM is the core interpreter and is the touchpoint used to parse and execute
// Jsonnet.
type VM struct {
	MaxStack int
	MaxTrace int // The number of lines of stack trace to display (0 for all of them).
	ext      vmExtMap
}

// MakeVM creates a new VM with default parameters.
func MakeVM() *VM {
	return &VM{
		MaxStack: 500,
		MaxTrace: 20,
	}
}

// ExtVar binds a Jsonnet external var to the given value.
func (vm *VM) ExtVar(key string, val string) {
	vm.ext[key] = vmExt{value: val, isCode: false}
}

// ExtCode binds a Jsonnet external code var to the given value.
func (vm *VM) ExtCode(key string, val string) {
	vm.ext[key] = vmExt{value: val, isCode: true}
}

// EvaluateSnippet evaluates a string containing Jsonnet code, return a JSON
// string.
//
// The filename parameter is only used for error messages.
func (vm *VM) EvaluateSnippet(filename string, snippet string) (string, error) {
	tokens, err := lex(filename, snippet)
	if err != nil {
		return "", err
	}
	ast, err := parse(tokens)
	if err != nil {
		return "", err
	}
	err = desugarFile(&ast)
	if err != nil {
		return "", err
	}
	output, err := evaluate(ast, vm.ext, vm.MaxStack)
	if err != nil {
		return "", err
	}
	return output, nil
}
