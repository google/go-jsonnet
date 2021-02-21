#include <stdio.h>
#include <stdlib.h>

extern "C" {
    #include "libjsonnet.h"
    #include "internal.h"
}

struct JsonnetVm *jsonnet_internal_make_vm_with_id(uintptr_t id) {
    JsonnetVm *vm = new JsonnetVm();
    vm->id = id;
    return vm;
}

void jsonnet_internal_free_vm(struct JsonnetVm *x) {
    delete(x);
}

struct JsonnetJsonValue *jsonnet_internal_make_json_with_id(uintptr_t id) {
    JsonnetJsonValue *json = new JsonnetJsonValue();
    json->id = id;
    return json;
}

void jsonnet_internal_free_json(struct JsonnetJsonValue *x) {
    delete(x);
}

struct JsonnetJsonValue* jsonnet_internal_execute_native(JsonnetNativeCallback *cb,
                                                         void *ctx,
                                                         const struct JsonnetJsonValue *const *argv,
                                                         int *success)
{
    return (cb)(ctx, argv, success);
}

char* jsonnet_internal_execute_import(JsonnetImportCallback *cb,
                                      void *ctx,
                                      const char *base,
                                      const char *rel,
                                      char **found_here,
                                      int *success)
{
    return (cb)(ctx, base, rel, found_here, success);
}

int jsonnet_internal_execute_writer(JsonnetIoWriterCallback *cb,
                                    char *str,
                                    int *success)
{
    return (cb)(str, success);
}

void jsonnet_internal_free_string(char *str) {
    if (str != nullptr) {
        ::free(str);
    }
}

void jsonnet_gc_min_objects(struct JsonnetVm *vm, unsigned v) {
    // no-op
}

void jsonnet_gc_growth_trigger(struct JsonnetVm *vm, double v) {
    // no-op
}

static void memory_panic(void)
{
    fputs("FATAL ERROR: A memory allocation error occurred.\n", stderr);
    abort();
}

char *jsonnet_realloc(JsonnetVm *vm, char *str, size_t sz)
{
    (void)vm;
    if (str == nullptr) {
        if (sz == 0)
            return nullptr;
        auto *r = static_cast<char *>(::malloc(sz));
        if (r == nullptr)
            memory_panic();
        return r;
    } else {
        if (sz == 0) {
            ::free(str);
            return nullptr;
        } else {
            auto *r = static_cast<char *>(::realloc(str, sz));
            if (r == nullptr)
                memory_panic();
            return r;
        }
    }
}
