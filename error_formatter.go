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
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

type ErrorFormatter struct {
	// TODO(sbarzowski) use this
	// MaxStackTraceSize  is the maximum length of stack trace before cropping
	MaxStackTraceSize int
	// TODO(sbarzowski) use these
	pretty   bool
	colorful bool
	SP       SourceProvider
}

func (ef *ErrorFormatter) format(err error) string {
	switch err := err.(type) {
	case RuntimeError:
		return ef.formatRuntime(&err)
	case StaticError:
		return ef.formatStatic(&err)
	default:
		return ef.formatInternal(err)
	}
}

func (ef *ErrorFormatter) formatRuntime(err *RuntimeError) string {
	return err.Error() + "\n" + ef.buildStackTrace(err.StackTrace)
	// TODO(sbarzowski) pretty stuff
}

func (ef *ErrorFormatter) formatStatic(err *StaticError) string {
	return err.Error() + "\n"
	// TODO(sbarzowski) pretty stuff
}

const bugURL = "https://github.com/google/go-jsonnet/issues"

func (ef *ErrorFormatter) formatInternal(err error) string {
	return "INTERNAL ERROR: " + err.Error() + "\n" +
		"Please report a bug here: " + bugURL + "\n"
}

func (ef *ErrorFormatter) buildStackTrace(frames []TraceFrame) string {
	// https://github.com/google/jsonnet/blob/master/core/libjsonnet.cpp#L594
	var buf bytes.Buffer
	for _, f := range frames {
		fmt.Fprintf(&buf, "\t%v\t%v\n", &f.Loc, f.Name)
		// TODO(sbarzowski) handle max stack trace size
		// TODO(sbarzowski) I think the order of frames is reversed
	}
	return buf.String()
}

type SourceProvider interface {
	// TODO(sbarzowski) problem: locationRange.FileName may not necessarily
	// uniquely identify a file. But this is the interface we want to have here.
	getCode(ast.LocationRange) string
}
