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

// Note: There are no garbage collection params because we're using the native Go garbage collector.
type Vm struct {
	maxStack int
	maxTrace int
	ext      vmExtMap
}

func MakeVm() *Vm {
	return &Vm{
		maxStack: 500,
		maxTrace: 20,
	}
}

func (vm *Vm) ExtVar(key string, val string) {
	vm.ext[key] = vmExt{value: val, isCode: false}
}

func (vm *Vm) ExtCode(key string, val string) {
	vm.ext[key] = vmExt{value: val, isCode: true}
}

func (vm *Vm) EvaluateSnippet(filename string, snippet string) (string, error) {
	tokens, err := lex(filename, snippet)
	if err != nil {
		return "", err
	}
	ast, err := parse(tokens)
	if err != nil {
		return "", err
	}
	ast, err = desugarFile(ast)
	output, err := execute(ast, vm.ext, vm.maxStack)
	if err != nil {
		return "", err
	}
	return output, nil
}
