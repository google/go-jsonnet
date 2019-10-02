package main

import (
	"errors"
	"fmt"
)

const maxID = 100000

// Because of Go GC, there are restrictions on keeping Go pointers in C.
// We cannot just pass *jsonnet.VM/JsonValue to C. So instead we use "handle" structs in C
// which refer to JsonnetVM/JsonnetJsonValue by a numeric id.
// The way it is implemented below is simple and has low overhead, but requires us to keep
// a list of used IDs. This results in a permanent "leak". I don't expect it to ever
// become a problem.
// The Handle IDs start with 1, so 0 is never a valid ID and the Handle's index in the array is (ID - 1).

// handlesTable is the set of active, valid Jsonnet allocated handles
type handlesTable struct {
	objects  []interface{}
	freedIDs []uint32
}

// errMaxNumberOfOpenHandles tells that there was an attempt to create more than maxID open handles
var errMaxNumberOfOpenHandles = fmt.Errorf("maximum number of constructed Jsonnet handles exceeded (%d)", maxID)

// errInvalidHandle tells that there was an attempt to dereference invalid handle ID
var errInvalidHandle = errors.New("invalid handle ID was provided")

// make registers the new object as a handle and returns the corresponding ID
func (h *handlesTable) make(obj interface{}) (uint32, error) {
	var id uint32

	if len(h.freedIDs) > 0 {
		id, h.freedIDs = h.freedIDs[len(h.freedIDs)-1], h.freedIDs[:len(h.freedIDs)-1]
		h.objects[id-1] = obj
	} else {
		id = uint32(len(h.objects) + 1)

		if id > maxID {
			return 0, errMaxNumberOfOpenHandles
		}

		h.objects = append(h.objects, obj)
	}

	return id, nil
}

// free marks the given handle ID as unused
func (h *handlesTable) free(id uint32) error {
	if err := h.ensureValidID(id); err != nil {
		return err
	}

	h.objects[id-1] = nil
	h.freedIDs = append(h.freedIDs, id)

	return nil
}

// get returns the corresponding object for the provided ID
func (h *handlesTable) get(id uint32) (interface{}, error) {
	if err := h.ensureValidID(id); err != nil {
		return nil, err
	}

	return h.objects[id-1], nil
}

// ensureValidID returns an error if the given handle ID is invalid, otherwise returns nil
func (h *handlesTable) ensureValidID(id uint32) error {
	if id == 0 || uint64(id) > uint64(len(h.objects)) {
		return errInvalidHandle
	}

	return nil
}
