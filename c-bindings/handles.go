package main

import (
	"errors"
	"sync"
	"unsafe"
)

// Because of Go GC, there are restrictions on keeping Go pointers in C.
// We cannot just pass *jsonnet.VM/JsonValue to C. So instead we use "handle" structs in C
// which refer to JsonnetVM/JsonnetJsonValue by a numeric id.
// The way it is implemented below is simple and has low overhead, but requires us to keep
// a list of used IDs. This results in a permanent "leak". I don't expect it to ever
// become a problem.
// The Handle IDs start with 1, so 0 is never a valid ID and the Handle's index in the array is (ID - 1).

// handlesTable is the set of active, valid Jsonnet allocated handles
type handlesTable struct {
	handles map[uintptr]*handle
	mu      sync.Mutex
}

type handle struct {
	ref interface{}
}

// errInvalidHandle tells that there was an attempt to dereference invalid handle ID
var errInvalidHandle = errors.New("invalid handle ID was provided")

func newHandlesTable() handlesTable {
	return handlesTable{
		handles: make(map[uintptr]*handle),
	}
}

// make registers the new object as a handle and returns the corresponding ID
func (h *handlesTable) make(obj interface{}) (uintptr, error) {
	entry := &handle{ref: obj}
	h.mu.Lock()
	defer h.mu.Unlock()
	id := uintptr(unsafe.Pointer(entry))
	h.handles[id] = entry
	return id, nil
}

// free removes an object with the given ID
func (h *handlesTable) free(id uintptr) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if handle := h.handles[id]; handle == nil {
		return errInvalidHandle
	}

	delete(h.handles, id)
	return nil
}

// get returns the corresponding object for the provided ID
func (h *handlesTable) get(id uintptr) (interface{}, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if handle := h.handles[id]; handle != nil {
		return handle.ref, nil
	}

	return nil, errInvalidHandle
}
