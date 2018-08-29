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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"strings"

	"github.com/google/go-jsonnet/ast"
)

func execCommand(cmd *exec.Cmd) (*bytes.Buffer, error) {
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		cmdStr := cmd.Path + " " + strings.Join(cmd.Args, " ")
		if stderr.Len() == 0 {
			return stdout, fmt.Errorf("%s: %v", cmdStr, err)
		}
		return stdout, fmt.Errorf("%s %v: %s", cmdStr, err, stderr.Bytes())
	}
	return stdout, nil
}

var nativeFunctionExec = &NativeFunction{
	Name:   "exec",
	Params: ast.Identifiers{"cmd", "positional_args"},
	Func: func(input []interface{}) (interface{}, error) {
		cmdStr, ok := input[0].(string)

		if !ok {
			return nil, errors.New("cmd must be a string")
		}

		params, ok := input[1].([]interface{})
		if !ok {
			return nil, fmt.Errorf("positional_args to exec must be an array of strings got %v", reflect.TypeOf(input[1]))
		}

		args := make([]string, len(params))
		for idx, val := range params {
			args[idx] = fmt.Sprint(val)
		}

		cmd := exec.Command(cmdStr, args...)

		stdout, err := execCommand(cmd)

		if err != nil {
			return nil, err
		}

		var jsonResp map[string]interface{}
		if err := json.Unmarshal(stdout.Bytes(), &jsonResp); err != nil {
			return nil, fmt.Errorf(`cmd: "%s %s" did not return valid json. Got "%v"`, cmd.Path, strings.Join(args, ""), stdout.String())
		}

		return jsonResp, nil
	},
}
