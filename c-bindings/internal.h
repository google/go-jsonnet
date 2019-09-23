#pragma once

#include <stdint.h>

struct JsonnetVm {
    uint32_t id;
};

struct JsonnetVm *jsonnet_internal_make_vm_with_id(uint32_t id);
void jsonnet_internal_free_vm(struct JsonnetVm *x);

struct JsonnetJsonValue {
    uint32_t id;
};

struct JsonnetJsonValue *jsonnet_internal_make_json_with_id(uint32_t id);
void jsonnet_internal_free_json(struct JsonnetJsonValue *x);

typedef struct JsonnetJsonValue *JsonnetNativeCallback(void *ctx,
                                                       const struct JsonnetJsonValue *const *argv,
                                                       int *success);

struct JsonnetJsonValue* jsonnet_internal_execute_native(JsonnetNativeCallback *cb,
                                                  void *ctx,
                                                  const struct JsonnetJsonValue *const *argv,
                                                  int *success);