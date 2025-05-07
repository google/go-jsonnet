package main

import (
	"fmt"
	"os"
	"path"
	"unsafe"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/formatter"

	// #cgo CXXFLAGS: -std=c++11 -Wall
	// #include "internal.h"
	"C"
)
import (
	"errors"
	"sort"
	"strings"
)

type vm struct {
	*jsonnet.VM
	importer      *jsonnet.FileImporter
	formatOptions formatter.Options
}

type jsonValue struct {
	val interface{}
	// these objects are exclusively owned by this jsonValue
	owned []*C.struct_JsonnetJsonValue
}

type importer struct {
	cb  *C.JsonnetImportCallback
	ctx unsafe.Pointer

	// An additional level of cache which allows
	// using this API as a drop-in replacement for
	// the C++ version.
	//
	// Importer contract requires returning the same
	// contents every time. This enforces that the same
	// imported file will always have the same contents.
	// This caching is not performed on the Go side normally,
	// because in many cases the proper caching can only
	// be performed within the importer. In particular,
	// when multiple paths are tried, the presence and contents
	// of each should be cached.
	//
	// In this case, the API requires us to take ownership
	// of the provided string, so we the implementation needs
	// to provide a new one each time, so we need to fake
	// good behavior.
	contentCache map[string]jsonnet.Contents
}

// Import fetches data from a given path by using c.JsonnetImportCallback
func (i *importer) Import(importedFrom, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	var (
		buflen     = C.size_t(0)
		msgC       *C.char
		bufPtr       unsafe.Pointer
		dir, _     = path.Split(importedFrom)
		foundHereC *C.char
	)

	// TODO(sbarzowski) Consider supporting returning null for paths,
	// which are already resolved. We cannot expect cross-language interface
	// to let us easily return the same Go Contents. Instead, we can allow
	// returning nothing (NULL), if they know that we have the contents
	// cached anyway.

	success := C.jsonnet_internal_execute_import(i.cb, i.ctx, C.CString(dir), C.CString(importedPath), &foundHereC, &msgC, &bufPtr, &buflen)
	if success != 0 {
		// Failure
		msg := C.GoString(msgC)
		C.jsonnet_internal_free_string(msgC)
		return jsonnet.Contents{}, "", errors.New("importer error: " + msg)
	}
	result := C.GoBytes(bufPtr, C.int(buflen))
	C.jsonnet_internal_free_pointer(bufPtr)

	foundHere := C.GoString(foundHereC)
	C.jsonnet_internal_free_string(foundHereC)


	if _, isCached := i.contentCache[foundHere]; !isCached {
		i.contentCache[foundHere] = jsonnet.MakeContentsRaw(result)
	}
	return i.contentCache[foundHere], foundHere, nil
}

var handles = newHandlesTable()
var versionString *C.char

//export jsonnet_version
func jsonnet_version() *C.char {
	if versionString == nil {
		version := jsonnet.Version() + " (go-jsonnet)"
		versionString = C.CString(version)
	}
	return versionString
}

//export jsonnet_make
func jsonnet_make() *C.struct_JsonnetVm {
	newVM := &vm{jsonnet.MakeVM(), &jsonnet.FileImporter{}, formatter.DefaultOptions()}
	newVM.Importer(newVM.importer)

	id, err := handles.make(newVM)

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	return C.jsonnet_internal_make_vm_with_id(C.uintptr_t(id))
}

//export jsonnet_destroy
func jsonnet_destroy(vmRef *C.struct_JsonnetVm) {
	if err := handles.free(uintptr(vmRef.id)); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	C.jsonnet_internal_free_vm(vmRef)
}

func getVM(vmRef *C.struct_JsonnetVm) *vm {
	ref, err := handles.get(uintptr(vmRef.id))

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	v, ok := ref.(*vm)

	if !ok {
		fmt.Fprintln(os.Stderr, "provided handle has a different type")
		os.Exit(1)
	}

	return v
}

func evaluateSnippet(vmRef *C.struct_JsonnetVm, filename string, code string, e *C.int) *C.char {
	vm := getVM(vmRef)
	// We still use a deprecated function to keep backwards compatible behavior.
	//nolint:staticcheck
	out, err := vm.EvaluateSnippet(filename, code)
	var result *C.char
	if err != nil {
		*e = 1
		result = C.CString(err.Error())
	} else {
		*e = 0
		result = C.CString(out)
	}
	return result
}

func evaluateSnippetStream(vmRef *C.struct_JsonnetVm, filename string, code string, e *C.int) *C.char {
	vm := getVM(vmRef)
	// We still use a deprecated function to keep backwards compatible behavior.
	//nolint:staticcheck
	out, err := vm.EvaluateSnippetStream(filename, code)
	var result *C.char
	if err != nil {
		*e = 1
		result = C.CString(err.Error())
	} else {
		*e = 0
		var buf strings.Builder
		for _, s := range out {
			buf.WriteString(s)
			buf.WriteByte(0)
		}
		result = C.CString(buf.String())
	}
	return result
}

func evaluateSnippetMulti(vmRef *C.struct_JsonnetVm, filename string, code string, e *C.int) *C.char {
	vm := getVM(vmRef)
	// We still use a deprecated function to keep backwards compatible behavior.
	//nolint:staticcheck
	out, err := vm.EvaluateSnippetMulti(filename, code)
	var result *C.char
	if err != nil {
		*e = 1
		result = C.CString(err.Error())
	} else {
		*e = 0
		// We go through filenames in sorted order to get deterministic output.
		filenames := make([]string, 0, len(out))
		for filename := range out {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)
		var buf strings.Builder
		for _, filename := range filenames {
			buf.WriteString(filename)
			buf.WriteByte(0)
			buf.WriteString(out[filename])
			buf.WriteByte(0)
		}
		result = C.CString(buf.String())
	}
	return result
}

//export jsonnet_evaluate_snippet
func jsonnet_evaluate_snippet(vmRef *C.struct_JsonnetVm, filename *C.char, code *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	s := C.GoString(code)
	return evaluateSnippet(vmRef, f, s, e)
}

//export jsonnet_evaluate_snippet_stream
func jsonnet_evaluate_snippet_stream(vmRef *C.struct_JsonnetVm, filename *C.char, code *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	s := C.GoString(code)
	return evaluateSnippetStream(vmRef, f, s, e)
}

//export jsonnet_evaluate_snippet_multi
func jsonnet_evaluate_snippet_multi(vmRef *C.struct_JsonnetVm, filename *C.char, code *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	s := C.GoString(code)
	return evaluateSnippetMulti(vmRef, f, s, e)
}

//export jsonnet_evaluate_file
func jsonnet_evaluate_file(vmRef *C.struct_JsonnetVm, filename *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	data, err := os.ReadFile(f)
	if err != nil {
		*e = 1
		return C.CString(fmt.Sprintf("Failed to read input file: %s: %s", f, err.Error()))
	}
	return evaluateSnippet(vmRef, f, string(data), e)
}

//export jsonnet_evaluate_file_stream
func jsonnet_evaluate_file_stream(vmRef *C.struct_JsonnetVm, filename *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	data, err := os.ReadFile(f)
	if err != nil {
		*e = 1
		return C.CString(fmt.Sprintf("Failed to read input file: %s: %s", f, err.Error()))
	}
	return evaluateSnippetStream(vmRef, f, string(data), e)
}

//export jsonnet_evaluate_file_multi
func jsonnet_evaluate_file_multi(vmRef *C.struct_JsonnetVm, filename *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	data, err := os.ReadFile(f)
	if err != nil {
		*e = 1
		return C.CString(fmt.Sprintf("Failed to read input file: %s: %s", f, err.Error()))
	}
	return evaluateSnippetMulti(vmRef, f, string(data), e)
}

//export jsonnet_max_stack
func jsonnet_max_stack(vmRef *C.struct_JsonnetVm, v C.uint) {
	vm := getVM(vmRef)
	vm.MaxStack = int(v) // potentially dangerous conversion
}

//export jsonnet_string_output
func jsonnet_string_output(vmRef *C.struct_JsonnetVm, v C.int) {
	vm := getVM(vmRef)
	vm.StringOutput = v != 0
}

//export jsonnet_max_trace
func jsonnet_max_trace(vmRef *C.struct_JsonnetVm, v C.uint) {
	vm := getVM(vmRef)
	vm.ErrorFormatter.SetMaxStackTraceSize(int(v)) // potentially dangerous conversion
}

//export jsonnet_jpath_add
func jsonnet_jpath_add(vmRef *C.struct_JsonnetVm, path *C.char) {
	vm := getVM(vmRef)
	vm.importer.JPaths = append(vm.importer.JPaths, C.GoString(path))
}

//export jsonnet_ext_var
func jsonnet_ext_var(vmRef *C.struct_JsonnetVm, key, value *C.char) {
	vm := getVM(vmRef)
	vm.ExtVar(C.GoString(key), C.GoString(value))
}

//export jsonnet_ext_code
func jsonnet_ext_code(vmRef *C.struct_JsonnetVm, key, value *C.char) {
	vm := getVM(vmRef)
	vm.ExtCode(C.GoString(key), C.GoString(value))
}

//export jsonnet_tla_var
func jsonnet_tla_var(vmRef *C.struct_JsonnetVm, key, value *C.char) {
	vm := getVM(vmRef)
	vm.TLAVar(C.GoString(key), C.GoString(value))
}

//export jsonnet_tla_code
func jsonnet_tla_code(vmRef *C.struct_JsonnetVm, key, value *C.char) {
	vm := getVM(vmRef)
	vm.TLACode(C.GoString(key), C.GoString(value))
}

//export jsonnet_native_callback
func jsonnet_native_callback(vmRef *C.struct_JsonnetVm, name *C.char, cb *C.JsonnetNativeCallback, ctx unsafe.Pointer, params **C.char) {
	vm := getVM(vmRef)
	p := unsafe.Pointer(params)
	sz := unsafe.Sizeof(*params)

	var paramNames ast.Identifiers

	for i := 0; ; i++ {
		param := (**C.char)(unsafe.Pointer(uintptr(p) + uintptr(i)*sz))

		if *param == nil {
			break
		}

		paramNames = append(paramNames, ast.Identifier(C.GoString(*param)))
	}

	f := &jsonnet.NativeFunction{
		Name:   C.GoString(name),
		Params: paramNames,
		Func: func(x []interface{}) (interface{}, error) {
			var (
				arr     []*C.struct_JsonnetJsonValue
				argv    **C.struct_JsonnetJsonValue
				success = C.int(0)
			)

			if len(x) > 0 {
				arr = make([]*C.struct_JsonnetJsonValue, 0, len(x))

				for _, json := range x {
					arr = append(arr, createJSONValue(vmRef, json))
				}

				argv = &(arr[0])
			}

			result := C.jsonnet_internal_execute_native(cb, ctx, argv, &success)
			v := getJSONValue(result)

			for _, val := range arr {
				jsonnet_json_destroy(vmRef, val)
			}

			jsonnet_json_destroy(vmRef, result)

			if success != 1 {
				return nil, fmt.Errorf("failed to execute native callback, code: %d", success)
			}

			return v.val, nil
		},
	}

	vm.NativeFunction(f)
}

//export jsonnet_import_callback
func jsonnet_import_callback(vmRef *C.struct_JsonnetVm, cb *C.JsonnetImportCallback, ctx unsafe.Pointer) {
	vm := getVM(vmRef)

	vm.Importer(&importer{
		ctx:          ctx,
		cb:           cb,
		contentCache: make(map[string]jsonnet.Contents),
	})
}

//export jsonnet_json_extract_string
func jsonnet_json_extract_string(vmRef *C.struct_JsonnetVm, json *C.struct_JsonnetJsonValue) *C.char {
	v := getJSONValue(json)
	str, ok := v.val.(string)

	if !ok {
		return nil
	}

	return C.CString(str)
}

//export jsonnet_json_extract_number
func jsonnet_json_extract_number(vmRef *C.struct_JsonnetVm, json *C.struct_JsonnetJsonValue, out *C.double) C.int {
	v := getJSONValue(json)

	switch f := v.val.(type) {
	case float64:
		*out = C.double(f)
		return 1
	case int:
		*out = C.double(f)
		return 1
	}

	return 0
}

//export jsonnet_json_extract_bool
func jsonnet_json_extract_bool(vmRef *C.struct_JsonnetVm, json *C.struct_JsonnetJsonValue) C.int {
	v := getJSONValue(json)
	b, ok := v.val.(bool)

	if !ok {
		return 2
	}

	if b {
		return 1
	}

	return 0
}

//export jsonnet_json_extract_null
func jsonnet_json_extract_null(vmRef *C.struct_JsonnetVm, json *C.struct_JsonnetJsonValue) C.int {
	v := getJSONValue(json)

	if v.val == nil {
		return 1
	}

	return 0
}

//export jsonnet_json_make_string
func jsonnet_json_make_string(vmRef *C.struct_JsonnetVm, v *C.char) *C.struct_JsonnetJsonValue {
	return createJSONValue(vmRef, C.GoString(v))
}

//export jsonnet_json_make_number
func jsonnet_json_make_number(vmRef *C.struct_JsonnetVm, v C.double) *C.struct_JsonnetJsonValue {
	return createJSONValue(vmRef, float64(v))
}

//export jsonnet_json_make_bool
func jsonnet_json_make_bool(vmRef *C.struct_JsonnetVm, v C.int) *C.struct_JsonnetJsonValue {
	return createJSONValue(vmRef, v != 0)
}

//export jsonnet_json_make_null
func jsonnet_json_make_null(vmRef *C.struct_JsonnetVm) *C.struct_JsonnetJsonValue {
	return createJSONValue(vmRef, nil)
}

//export jsonnet_json_make_array
func jsonnet_json_make_array(vmRef *C.struct_JsonnetVm) *C.struct_JsonnetJsonValue {
	return createJSONValue(vmRef, []interface{}{})
}

//export jsonnet_json_array_append
func jsonnet_json_array_append(vmRef *C.struct_JsonnetVm, arr *C.struct_JsonnetJsonValue, v *C.struct_JsonnetJsonValue) {
	json := getJSONValue(arr)
	slice, ok := json.val.([]interface{})

	if !ok {
		fmt.Fprintf(os.Stderr, "array should be provided")
		os.Exit(1)
	}

	json.val = append(slice, getJSONValue(v).val)
	json.owned = append(json.owned, v)
}

//export jsonnet_json_make_object
func jsonnet_json_make_object(vmRef *C.struct_JsonnetVm) *C.struct_JsonnetJsonValue {
	return createJSONValue(vmRef, make(map[string]interface{}))
}

//export jsonnet_json_object_append
func jsonnet_json_object_append(
	vmRef *C.struct_JsonnetVm,
	obj *C.struct_JsonnetJsonValue,
	f *C.char,
	v *C.struct_JsonnetJsonValue,
) {
	d := getJSONValue(obj)
	table, ok := d.val.(map[string]interface{})

	if !ok {
		fmt.Fprintf(os.Stderr, "object should be provided")
		os.Exit(1)
	}

	table[C.GoString(f)] = getJSONValue(v).val
	d.owned = append(d.owned, v)
}

//export jsonnet_json_destroy
func jsonnet_json_destroy(vmRef *C.struct_JsonnetVm, v *C.struct_JsonnetJsonValue) {
	for _, child := range getJSONValue(v).owned {
		jsonnet_json_destroy(vmRef, child)
	}

	if err := handles.free(uintptr(v.id)); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	C.jsonnet_internal_free_json(v)
}

func createJSONValue(vmRef *C.struct_JsonnetVm, val interface{}) *C.struct_JsonnetJsonValue {
	id, err := handles.make(&jsonValue{val: val})

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	return C.jsonnet_internal_make_json_with_id(C.uintptr_t(id))
}

func getJSONValue(jsonRef *C.struct_JsonnetJsonValue) *jsonValue {
	ref, err := handles.get(uintptr(jsonRef.id))

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	v, ok := ref.(*jsonValue)

	if !ok {
		fmt.Fprintf(os.Stderr, "provided handle has a different type")
		os.Exit(1)
	}

	return v
}

//export jsonnet_fmt_indent
func jsonnet_fmt_indent(vmRef *C.struct_JsonnetVm, n C.int) {
	vm := getVM(vmRef)
	vm.formatOptions.Indent = int(n)
}

//export jsonnet_fmt_max_blank_lines
func jsonnet_fmt_max_blank_lines(vmRef *C.struct_JsonnetVm, n C.int) {
	vm := getVM(vmRef)
	vm.formatOptions.MaxBlankLines = int(n)
}

//export jsonnet_fmt_string
func jsonnet_fmt_string(vmRef *C.struct_JsonnetVm, c C.int) {
	vm := getVM(vmRef)
	switch c {
	case 'd':
		vm.formatOptions.StringStyle = formatter.StringStyleDouble
	case 's':
		vm.formatOptions.StringStyle = formatter.StringStyleSingle
	case 'l':
		vm.formatOptions.StringStyle = formatter.StringStyleLeave
	default:
		vm.formatOptions.StringStyle = formatter.StringStyleLeave
	}
}

//export jsonnet_fmt_comment
func jsonnet_fmt_comment(vmRef *C.struct_JsonnetVm, c C.int) {
	vm := getVM(vmRef)
	switch c {
	case 'h':
		vm.formatOptions.CommentStyle = formatter.CommentStyleHash
	case 's':
		vm.formatOptions.CommentStyle = formatter.CommentStyleSlash
	case 'l':
		vm.formatOptions.CommentStyle = formatter.CommentStyleLeave
	default:
		vm.formatOptions.CommentStyle = formatter.CommentStyleLeave
	}
}

//export jsonnet_fmt_pad_arrays
func jsonnet_fmt_pad_arrays(vmRef *C.struct_JsonnetVm, v C.int) {
	vm := getVM(vmRef)
	vm.formatOptions.PadArrays = v != 0
}

//export jsonnet_fmt_pad_objects
func jsonnet_fmt_pad_objects(vmRef *C.struct_JsonnetVm, v C.int) {
	vm := getVM(vmRef)
	vm.formatOptions.PadObjects = v != 0
}

//export jsonnet_fmt_pretty_field_names
func jsonnet_fmt_pretty_field_names(vmRef *C.struct_JsonnetVm, v C.int) {
	vm := getVM(vmRef)
	vm.formatOptions.PrettyFieldNames = v != 0
}

//export jsonnet_fmt_sort_imports
func jsonnet_fmt_sort_imports(vmRef *C.struct_JsonnetVm, v C.int) {
	vm := getVM(vmRef)
	vm.formatOptions.SortImports = v != 0
}

//export jsonnet_fmt_snippet
func jsonnet_fmt_snippet(vmRef *C.struct_JsonnetVm, filename *C.char, code *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	s := C.GoString(code)
	return formatSnippet(vmRef, f, s, e)
}

//export jsonnet_fmt_file
func jsonnet_fmt_file(vmRef *C.struct_JsonnetVm, filename *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	data, err := os.ReadFile(f)
	if err != nil {
		*e = 1
		return C.CString(fmt.Sprintf("Failed to read input file: %s: %s", f, err.Error()))
	}
	return formatSnippet(vmRef, f, string(data), e)
}

func formatSnippet(vmRef *C.struct_JsonnetVm, filename string, code string, e *C.int) *C.char {
	vm := getVM(vmRef)
	out, err := formatter.Format(filename, code, vm.formatOptions)
	var result *C.char
	if err != nil {
		*e = 1
		result = C.CString(err.Error())
	} else {
		*e = 0
		result = C.CString(out)
	}
	return result
}

type traceOut struct {
	cb *C.JsonnetIoWriterCallback
}

func (o *traceOut) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	success := C.int(0)
	var n C.int = C.jsonnet_internal_execute_writer(o.cb, unsafe.Pointer(&p[0]),
				                                    C.size_t(len(p)), &success)
	if success != 1 {
		return int(n), errors.New("std.trace() failed to write to output stream")
	}
	return int(n), nil
}

//export jsonnet_set_trace_out_callback
func jsonnet_set_trace_out_callback(vmRef *C.struct_JsonnetVm, cb *C.JsonnetIoWriterCallback) {
	vm := getVM(vmRef)
	vm.SetTraceOut(&traceOut{cb})
}

//export jsonnet_realloc
func jsonnet_realloc(vmRef *C.struct_JsonnetVm, buf *C.char, sz C.size_t) *C.char {
	return C.jsonnet_internal_realloc(vmRef, buf, sz)
}


func main() {
}
