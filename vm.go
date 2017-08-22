/*
Copyright 2017 Google Inc. All rights reserved.

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
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/parser"
)

// Note: There are no garbage collection params because we're using the native
// Go garbage collector.

// TODO(sbarzowski) prepare API that maps 1-1 to libjsonnet api

// VM is the core interpreter and is the touchpoint used to parse and execute
// Jsonnet.
type VM struct {
	MaxStack int
	MaxTrace int // The number of lines of stack trace to display (0 for all of them).
	ext      vmExtMap
	importer Importer
	ef       ErrorFormatter
}

// TODO(sbarzowski) actually support these
// External variable (or code) provided before execution
type vmExt struct {
	value  string // what is it?
	isCode bool   // what is it?
}

type vmExtMap map[string]vmExt

// MakeVM creates a new VM with default parameters.
func MakeVM() *VM {
	return &VM{
		MaxStack: 500,
		MaxTrace: 20,
		ext:      make(vmExtMap),
		ef:       ErrorFormatter{},
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

func (vm *VM) evaluateSnippet(filename string, snippet string) (string, error) {
	node, err := snippetToAST(filename, snippet)
	if err != nil {
		return "", err
	}
	output, err := evaluate(node, vm.ext, vm.MaxStack, &FileImporter{})
	if err != nil {
		return "", err
	}
	return output, nil
}

// EvaluateSnippet evaluates a string containing Jsonnet code, return a JSON
// string.
//
// The filename parameter is only used for error messages.
func (vm *VM) EvaluateSnippet(filename string, snippet string) (json string, formattedErr error) {
	defer func() {
		if r := recover(); r != nil {
			formattedErr = errors.New(vm.ef.format(fmt.Errorf("(CRASH) %v\n%s", r, debug.Stack())))
		}
	}()
	json, err := vm.evaluateSnippet(filename, snippet)
	if err != nil {
		return "", errors.New(vm.ef.format(err))
	}
	return json, nil
}

func snippetToAST(filename string, snippet string) (ast.Node, error) {
	tokens, err := parser.Lex(filename, snippet)
	if err != nil {
		return nil, err
	}
	node, err := parser.Parse(tokens)
	if err != nil {
		return nil, err
	}
	// fmt.Println(ast.(dumpable).dump())
	err = desugarFile(&node)
	if err != nil {
		return nil, err
	}
	err = analyze(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}
