#pragma once

#include <stdint.h>

struct JsonnetVm {
    uintptr_t id;
};

struct JsonnetVm *jsonnet_internal_make_vm_with_id(uintptr_t id);
void jsonnet_internal_free_vm(struct JsonnetVm *x);

struct JsonnetJsonValue {
    uintptr_t id;
};

struct JsonnetJsonValue *jsonnet_internal_make_json_with_id(uintptr_t id);
void jsonnet_internal_free_json(struct JsonnetJsonValue *x);

typedef struct JsonnetJsonValue *JsonnetNativeCallback(void *ctx,
                                                       const struct JsonnetJsonValue *const *argv,
                                                       int *success);

struct JsonnetJsonValue* jsonnet_internal_execute_native(JsonnetNativeCallback *cb,
                                                  void *ctx,
                                                  const struct JsonnetJsonValue *const *argv,
                                                  int *success);

typedef int JsonnetImportCallback(void *ctx, const char *base, const char *rel,
                                  char **found_here, char **buf, size_t *buflen);

int jsonnet_internal_execute_import(JsonnetImportCallback *cb,
                                    void *ctx,
                                    const char *base,
                                    const char *rel,
                                    char **found_here,
                                    char **msg,
                                    void **buf, size_t *buflen);

typedef int JsonnetIoWriterCallback(const void *buf, size_t nbytes, int *success);

int jsonnet_internal_execute_writer(JsonnetIoWriterCallback *cb,
                                    const void *buf,
                                    size_t nbytes,
                                    int *success);

void jsonnet_internal_free_string(char *str);
void jsonnet_internal_free_pointer(void *ptr);

char* jsonnet_internal_realloc(struct JsonnetVm *vm, char *str, size_t sz);
