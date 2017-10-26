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

package main

import (
	"os"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/dump"
)

func main() {
	filename := "ast/stdast.go"

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE,0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	node, err := jsonnet.SnippetToAST("<std>", jsonnet.GetStdCode())
	if err != nil {
		panic(err)
	}

	dump.Config.HidePrivateFields = false
	dump.Config.StripPackageNames = true
	dump.Config.VariableName = "StdAst"
	ast := dump.Sdump(node)

	file.WriteString("package ast\n\n")
	file.WriteString(ast)
	file.Sync()
}
