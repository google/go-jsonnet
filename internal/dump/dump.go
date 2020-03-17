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

// Package dump can dump a Go data structure to Go source file, so that it can
// be statically embedded into other code.
package dump

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
)

var packageNameStripperRegexp = regexp.MustCompile(`\b[a-zA-Z_]+[a-zA-Z_0-9]+\.`)

// Options represents configuration option
type Options struct {
	StripPackageNames   bool
	HidePrivateFields   bool
	HomePackage         string
	VariableName        string
	VariableDescription string
}

// Config is the default config used when calling Dump
var Config = Options{
	StripPackageNames:   false,
	HidePrivateFields:   true,
	VariableName:        "Obj",
	VariableDescription: "",
}

type dumpState struct {
	w      io.Writer
	depth  int
	config *Options

	allPointers     []uintptr
	visitedPointers []uintptr

	reusedPointers    []uintptr
	primitivePointers []uintptr

	homePackageRegexp *regexp.Regexp
}

func mustWrite(w io.Writer, data []byte) {
	_, err := w.Write(data)
	if err != nil {
		panic(err)
	}
}

func (s *dumpState) indent() {
	mustWrite(s.w, bytes.Repeat([]byte("\t"), s.depth))
}

func (s *dumpState) newline() {
	mustWrite(s.w, []byte("\n"))
}

func (s *dumpState) dumpType(v reflect.Value) {
	typeName := v.Type().String()
	if s.config.StripPackageNames {
		typeName = packageNameStripperRegexp.ReplaceAllLiteralString(typeName, "")
	} else if s.homePackageRegexp != nil {
		typeName = s.homePackageRegexp.ReplaceAllLiteralString(typeName, "")
	}
	mustWrite(s.w, []byte(typeName))
}

func (s *dumpState) dumpSlice(v reflect.Value) {
	s.dumpType(v)
	numEntries := v.Len()
	if numEntries == 0 {
		mustWrite(s.w, []byte("{}"))
		return
	}
	mustWrite(s.w, []byte("{"))
	s.newline()
	s.depth++
	for i := 0; i < numEntries; i++ {
		s.indent()
		s.dumpVal(v.Index(i))
		mustWrite(s.w, []byte(","))
		s.newline()
	}
	s.depth--
	s.indent()
	mustWrite(s.w, []byte("}"))
}

func (s *dumpState) dumpStruct(v reflect.Value) {
	dumpPreamble := func() {
		s.dumpType(v)
		mustWrite(s.w, []byte("{"))
		s.newline()
		s.depth++
	}
	preambleDumped := false
	vt := v.Type()
	numFields := v.NumField()
	for i := 0; i < numFields; i++ {
		vtf := vt.Field(i)
		if s.config.HidePrivateFields && vtf.PkgPath != "" {
			continue
		}
		if !preambleDumped {
			dumpPreamble()
			preambleDumped = true
		}
		s.indent()
		mustWrite(s.w, []byte(vtf.Name))
		mustWrite(s.w, []byte(": "))
		s.dumpVal(v.Field(i))
		mustWrite(s.w, []byte(","))
		s.newline()
	}
	if preambleDumped {
		s.depth--
		s.indent()
		mustWrite(s.w, []byte("}"))
	} else {
		// There were no fields dumped
		s.dumpType(v)
		mustWrite(s.w, []byte("{}"))
	}
}

func (s *dumpState) dumpMap(v reflect.Value) {
	s.dumpType(v)
	mustWrite(s.w, []byte("{"))
	s.newline()
	s.depth++
	keys := v.MapKeys()
	sort.Sort(mapKeySorter{keys})
	for _, key := range keys {
		s.indent()
		s.dumpVal(key)
		mustWrite(s.w, []byte(": "))
		s.dumpVal(v.MapIndex(key))
		mustWrite(s.w, []byte(","))
		s.newline()
	}
	s.depth--
	s.indent()
	mustWrite(s.w, []byte("}"))
}

func (s *dumpState) dump(value interface{}) {
	writeVar := func() {
		if s.config.VariableDescription != "" {
			fmt.Fprintf(s.w, "\n// %s\n", s.config.VariableDescription)
		}
		mustWrite(s.w, []byte("var "+s.config.VariableName+" = "))
	}

	if value == nil {
		writeVar()
		printNil(s.w)
		s.newline()
		return
	}
	v := reflect.ValueOf(value)

	s.dumpPrimitivePointerVal(v)
	s.visitedPointers = []uintptr{}

	s.dumpReusedPointerVal(v)

	writeVar()
	if v.Kind() == reflect.Ptr && v.IsNil() {
		printNil(s.w)
	} else {
		// Replace the writer with a temporary buffer so that
		// we can see how big the output is. If it's too big,
		// we use an intermediate value so that godoc output
		// isn't swamped by the literal.
		oldWriter := s.w
		buf := new(bytes.Buffer)
		s.w = buf
		s.dumpVal(v)
		s.w = oldWriter
		if buf.Len() > 100 {
			mustWrite(s.w, []byte("_"+s.config.VariableName))
			s.newline()
			mustWrite(s.w, []byte("var _"+s.config.VariableName+" = "))
		}
		mustWrite(s.w, buf.Bytes())
	}
	s.newline()
}

func (s *dumpState) printPrimitivePointer(value reflect.Value, pointerName string) {
	v := deInterface(value)

	mustWrite(s.w, []byte("var "+pointerName+"Var"+" = "))
	switch v.Kind() {
	case reflect.Bool:
		printBool(s.w, v.Bool())

	case reflect.Int:
		printInt(s.w, v, s.config.StripPackageNames)
	case reflect.Int8:
		printInt(s.w, v, s.config.StripPackageNames)
	case reflect.Int16:
		printInt(s.w, v, s.config.StripPackageNames)
	case reflect.Int32:
		printInt(s.w, v, s.config.StripPackageNames)
	case reflect.Int64:
		printInt(s.w, v, s.config.StripPackageNames)

	case reflect.Uint:
		printUint(s.w, v)
	case reflect.Uint8:
		printUint(s.w, v)
	case reflect.Uint16:
		printUint(s.w, v)
	case reflect.Uint32:
		printUint(s.w, v)
	case reflect.Uint64:
		printUint(s.w, v)

	case reflect.Float32:
		printFloat(s.w, v.Float(), 32, "float32")
	case reflect.Float64:
		printFloat(s.w, v.Float(), 64, "float64")

	case reflect.Complex64:
		printComplex(s.w, v.Complex(), 32)
	case reflect.Complex128:
		printComplex(s.w, v.Complex(), 64)

	case reflect.String:
		mustWrite(s.w, []byte(strconv.Quote(v.String())))
	}

	s.newline()
	mustWrite(s.w, []byte("var "+pointerName+" = &"+pointerName+"Var"))
	s.newline()
}

func (s *dumpState) dumpPrimitivePointerVal(value reflect.Value) {
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return
	}
	v := deInterface(value)
	kind := v.Kind()

	switch kind {
	case reflect.Invalid:
		// Do nothing.  We should never get here since invalid has already
		// been handled above.
		mustWrite(s.w, []byte("<invalid>"))

	case reflect.Slice:
		if v.IsNil() {
			break
		}
		fallthrough

	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			s.dumpPrimitivePointerVal(v.Index(i))
		}

	case reflect.Map:
		keys := v.MapKeys()
		sort.Sort(mapKeySorter{keys})
		for _, key := range keys {
			s.dumpPrimitivePointerVal(v.MapIndex(key))
		}

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			s.dumpPrimitivePointerVal(v.Field(i))
		}

	case reflect.Ptr:
		pointerName, isPrimitive, isFirstVisit := s.nameForPointer(v), s.isPrimitivePointer(v), s.visitPointerAndCheckIfFirstTime(v)
		if isPrimitive && isFirstVisit {
			s.printPrimitivePointer(v.Elem(), pointerName)
		} else {
			s.dumpPrimitivePointerVal(v.Elem())
		}
	}
}

func (s *dumpState) dumpReusedPointerVal(value reflect.Value) {
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return
	}

	v := deInterface(value)
	kind := v.Kind()

	switch kind {
	case reflect.Invalid:
		// Do nothing.  We should never get here since invalid has already
		// been handled above.
		mustWrite(s.w, []byte("<invalid>"))

	case reflect.Slice:
		if v.IsNil() {
			break
		}
		fallthrough

	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			s.dumpReusedPointerVal(v.Index(i))
		}

	case reflect.Map:
		keys := v.MapKeys()
		sort.Sort(mapKeySorter{keys})
		for _, key := range keys {
			s.dumpReusedPointerVal(v.MapIndex(key))
		}

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			s.dumpReusedPointerVal(v.Field(i))
		}

	case reflect.Ptr:
		pointerName, isReused, isPrimitive, isFirstVisit := s.nameForPointer(v), s.isReusedPointer(v), s.isPrimitivePointer(v), s.visitPointerAndCheckIfFirstTime(v)
		if isReused && !isPrimitive && isFirstVisit {
			mustWrite(s.w, []byte("var "+pointerName+" = &"))
			s.dumpVal(v.Elem())
			s.newline()
		} else {
			s.dumpReusedPointerVal(v.Elem())
		}
	}
}

func (s *dumpState) dumpVal(value reflect.Value) {
	if value.Kind() == reflect.Ptr && value.IsNil() {
		mustWrite(s.w, []byte("nil"))
		return
	}

	v := deInterface(value)
	kind := v.Kind()
	switch kind {
	case reflect.Invalid:
		// Do nothing.  We should never get here since invalid has already
		// been handled above.
		mustWrite(s.w, []byte("<invalid>"))

	case reflect.Bool:
		printBool(s.w, v.Bool())

	case reflect.Int:
		printInt(s.w, v, s.config.StripPackageNames)
	case reflect.Int8:
		printInt(s.w, v, s.config.StripPackageNames)
	case reflect.Int16:
		printInt(s.w, v, s.config.StripPackageNames)
	case reflect.Int32:
		printInt(s.w, v, s.config.StripPackageNames)
	case reflect.Int64:
		printInt(s.w, v, s.config.StripPackageNames)

	case reflect.Uint:
		printUint(s.w, v)
	case reflect.Uint8:
		printUint(s.w, v)
	case reflect.Uint16:
		printUint(s.w, v)
	case reflect.Uint32:
		printUint(s.w, v)
	case reflect.Uint64:
		printUint(s.w, v)

	case reflect.Float32:
		printFloat(s.w, v.Float(), 32, "float32")
	case reflect.Float64:
		printFloat(s.w, v.Float(), 64, "float64")

	case reflect.Complex64:
		printComplex(s.w, v.Complex(), 32)
	case reflect.Complex128:
		printComplex(s.w, v.Complex(), 64)

	case reflect.String:
		mustWrite(s.w, []byte(strconv.Quote(v.String())))

	case reflect.Slice:
		if v.IsNil() {
			printNil(s.w)
			break
		}
		fallthrough

	case reflect.Array:
		s.dumpSlice(v)

	case reflect.Interface:
		// The only time we should get here is for nil interfaces due to
		// unpackValue calls.
		if v.IsNil() {
			printNil(s.w)
		}

	case reflect.Ptr:
		pointerName, isPrimitive, isReused := s.nameForPointer(v), s.isPrimitivePointer(v), s.isReusedPointer(v)
		if isPrimitive || isReused {
			mustWrite(s.w, []byte(pointerName))
		} else {
			mustWrite(s.w, []byte("&"))
			s.dumpVal(v.Elem())
		}

	case reflect.Map:
		s.dumpMap(v)

	case reflect.Struct:
		s.dumpStruct(v)

	default:
		if v.CanInterface() {
			fmt.Fprintf(s.w, "%v", v.Interface())
		} else {
			fmt.Fprintf(s.w, "%v", v.String())
		}
	}
}

func (s *dumpState) nameForPointer(v reflect.Value) string {
	if isPointerValue(v) {
		ptr := v.Pointer()
		for i, addr := range s.allPointers {
			if ptr == addr {
				return fmt.Sprintf("p%d", i)
			}
		}
	}
	return ""
}

func (s *dumpState) isPrimitivePointer(v reflect.Value) bool {
	if isPointerValue(v) {
		ptr := v.Pointer()
		for _, addr := range s.primitivePointers {
			if addr == ptr {
				return true
			}
		}
	}
	return false
}

func (s *dumpState) isReusedPointer(v reflect.Value) bool {
	if isPointerValue(v) {
		ptr := v.Pointer()
		for _, addr := range s.reusedPointers {
			if addr == ptr {
				return true
			}
		}
	}
	return false
}

func (s *dumpState) visitPointerAndCheckIfFirstTime(v reflect.Value) bool {
	if isPointerValue(v) {
		ptr := v.Pointer()
		for _, addr := range s.visitedPointers {
			if addr == ptr {
				return false
			}
		}
		s.visitedPointers = append(s.visitedPointers, ptr)
	}

	return true
}

// prepares a new state object for dumping the provided value
func newDumpState(value interface{}, options *Options) *dumpState {
	result := &dumpState{
		config: options,
	}

	result.allPointers, result.reusedPointers, result.primitivePointers = GetPointers(reflect.ValueOf(value))

	if options.HomePackage != "" {
		result.homePackageRegexp = regexp.MustCompile(fmt.Sprintf("\\b%s\\.", options.HomePackage))
	}

	return result
}

// Dump a value to stdout
func Dump(value interface{}) {
	(&Config).Dump(value)
}

// Sdump dumps a value to a string
func Sdump(value interface{}) string {
	return (&Config).Sdump(value)
}

// Dump a value to stdout according to the options
func (o Options) Dump(value interface{}) {

	state := newDumpState(value, &o)
	state.w = os.Stdout
	state.dump(value)
}

// Sdump dumps a value to a string according to the options
func (o Options) Sdump(value interface{}) string {
	buf := new(bytes.Buffer)

	state := newDumpState(value, &o)
	state.w = buf
	state.dump(value)

	return buf.String()
}

type mapKeySorter struct {
	keys []reflect.Value
}

func (s mapKeySorter) Len() int {
	return len(s.keys)
}

func (s mapKeySorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

func (s mapKeySorter) Less(i, j int) bool {
	return fmt.Sprintf("%s", s.keys[i].Interface()) < fmt.Sprintf("%s", s.keys[j].Interface())
}
