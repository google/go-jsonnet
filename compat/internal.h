#pragma once

#include <stdint.h>

struct JsonnetVm {
    uint32_t id;
};

struct JsonnetVm *jsonnet_internal_make_vm_with_id(uint32_t id);
void jsonnet_internal_free(struct JsonnetVm *x);