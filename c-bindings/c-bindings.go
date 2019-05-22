package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/go-jsonnet"

	// #cgo CXXFLAGS: -std=c++11 -Wall -I../cpp-jsonnet/include
	// #include "internal.h"
	"C"
)

const maxID = 1000

type vm struct {
	*jsonnet.VM
	importer *jsonnet.FileImporter
}

// Because of Go GC, there are restrictions on keeping Go pointers in C.
// We cannot just pass *jsonnet.VM to C. So instead we use "handle" structs in C
// which refer to JsonnetVM by a numeric id.
// The way it is implemented below is simple and has low overhead, but requires us to keep
// a list of used IDs. This results in a permanent "leak". I don't expect it to ever
// become a problem.
// The VM IDs start with 1, so 0 is never a valid ID and the VM's index in the array is (ID - 1).
var VMs = []*vm{}
var freedIDs = []uint32{}

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
	var id uint32

	newVM := &vm{jsonnet.MakeVM(), &jsonnet.FileImporter{}}
	newVM.Importer(newVM.importer)

	if len(freedIDs) > 0 {
		id, freedIDs = freedIDs[len(freedIDs)-1], freedIDs[:len(freedIDs)-1]
		VMs[id-1] = newVM
	} else {
		id = uint32(len(VMs) + 1)
		if id > maxID {
			fmt.Fprintf(os.Stderr, "Maximum number of constructed Jsonnet VMs exceeded (%d)\n", maxID)
			os.Exit(1)
		}
		VMs = append(VMs, newVM)
	}
	addr := C.jsonnet_internal_make_vm_with_id(C.uint32_t(id))
	return addr
}

//export jsonnet_destroy
func jsonnet_destroy(vmRef *C.struct_JsonnetVm) {
	VMs[vmRef.id-1] = nil
	freedIDs = append(freedIDs, uint32(vmRef.id))
	C.jsonnet_internal_free_vm(vmRef)
}

func getVM(vmRef *C.struct_JsonnetVm) *vm {
	return VMs[vmRef.id-1]
}

func evaluateSnippet(vmRef *C.struct_JsonnetVm, filename string, code string, e *C.int) *C.char {
	vm := getVM(vmRef)
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

//export jsonnet_evaluate_snippet
func jsonnet_evaluate_snippet(vmRef *C.struct_JsonnetVm, filename *C.char, code *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	s := C.GoString(code)
	return evaluateSnippet(vmRef, f, s, e)
}

//export jsonnet_evaluate_file
func jsonnet_evaluate_file(vmRef *C.struct_JsonnetVm, filename *C.char, e *C.int) *C.char {
	f := C.GoString(filename)
	data, err := ioutil.ReadFile(f)
	if err != nil {
		*e = 1
		// TODO(sbarzowski) make sure that it's ok allocation-wise
		return C.CString(fmt.Sprintf("Failed to read input file: %s: %s", f, err.Error()))
	} else {
		return evaluateSnippet(vmRef, f, string(data), e)
	}
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

func main() {

}
