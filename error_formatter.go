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

	"github.com/fatih/color"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/parser"
)

type ErrorFormatter struct {
	// MaxStackTraceSize  is the maximum length of stack trace before cropping
	MaxStackTraceSize int

	// Examples of current state of the art.
	// http://elm-lang.org/blog/compiler-errors-for-humans
	// https://clang.llvm.org/diagnostics.html
	pretty   bool
	colorful bool
	SP       *ast.SourceProvider
}

func (ef *ErrorFormatter) format(err error) string {
	switch err := err.(type) {
	case RuntimeError:
		return ef.formatRuntime(&err)
	case parser.StaticError:
		return ef.formatStatic(&err)
	default:
		return ef.formatInternal(err)
	}
}

func (ef *ErrorFormatter) formatRuntime(err *RuntimeError) string {
	return err.Error() + "\n" + ef.buildStackTrace(err.StackTrace)
}

func (ef *ErrorFormatter) formatStatic(err *parser.StaticError) string {
	var buf bytes.Buffer
	buf.WriteString(err.Error() + "\n")
	ef.showCode(&buf, err.Loc)
	return buf.String()
}

const bugURL = "https://github.com/google/go-jsonnet/issues"

func (ef *ErrorFormatter) formatInternal(err error) string {
	return "INTERNAL ERROR: " + err.Error() + "\n" +
		"Please report a bug here: " + bugURL + "\n"
}

func (ef *ErrorFormatter) showCode(buf *bytes.Buffer, loc ast.LocationRange) {
	errFprintf := fmt.Fprintf
	if ef.colorful {
		errFprintf = color.New(color.FgRed).Fprintf
	}
	if loc.WithCode() {
		// TODO(sbarzowski) include line numbers
		// TODO(sbarzowski) underline errors instead of depending only on color
		fmt.Fprintf(buf, "\n")
		beginning := ast.LineBeginning(&loc)
		ending := ast.LineEnding(&loc)
		fmt.Fprintf(buf, "%v", ef.SP.GetSnippet(beginning))
		errFprintf(buf, "%v", ef.SP.GetSnippet(loc))
		fmt.Fprintf(buf, "%v", ef.SP.GetSnippet(ending))
		buf.WriteByte('\n')
	}
	fmt.Fprintf(buf, "\n")
}

func (ef *ErrorFormatter) frame(frame *TraceFrame, buf *bytes.Buffer) {
	// TODO(sbarzowski) tabs are probably a bad idea
	fmt.Fprintf(buf, "\t%v\t%v\n", frame.Loc.String(), frame.Name)
	if ef.pretty {
		ef.showCode(buf, frame.Loc)
	}
}

func (ef *ErrorFormatter) buildStackTrace(frames []TraceFrame) string {
	// https://github.com/google/jsonnet/blob/master/core/libjsonnet.cpp#L594
	maxAbove := ef.MaxStackTraceSize / 2
	maxBelow := ef.MaxStackTraceSize - maxAbove
	var buf bytes.Buffer
	sz := len(frames)
	for i := 0; i < sz; i++ {
		// TODO(sbarzowski) make pretty format more readable (it's already useful)
		if ef.pretty {
			fmt.Fprintf(&buf, "-------------------------------------------------\n")
		}
		if i >= maxAbove && i < sz-maxBelow {
			if ef.pretty {
				fmt.Fprintf(&buf, "\t... (skipped %v frames)\n", sz-maxAbove-maxBelow)
			} else {
				buf.WriteString("\t...\n")
			}

			i = sz - maxBelow - 1
		} else {
			ef.frame(&frames[sz-i-1], &buf)
		}
	}
	return buf.String()
}
