#include <stdio.h>
#include <stdlib.h>

extern "C" {
    void jsonnet_gc_min_objects(struct JsonnetVm *vm, unsigned v);
    void jsonnet_gc_growth_trigger(struct JsonnetVm *vm, double v);
    char *jsonnet_realloc(JsonnetVm *vm, char *str, size_t sz);

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

int jsonnet_internal_execute_import(JsonnetImportCallback *cb,
                                    void *ctx,
                                    const char *base,
                                    const char *rel,
                                    char **found_here,
                                    char **msg,
                                    void **buf, size_t *buflen)
{
    char *char_buf;
    int success = (cb)(ctx, base, rel, found_here, &char_buf, buflen);
    if (success == 0) {
		// Success
        *buf = char_buf;
    } else {
		// Fail
        *msg = char_buf;
    }
    return success;
}

int jsonnet_internal_execute_writer(JsonnetIoWriterCallback *cb,
                                    const void *buf,
                                    size_t nbytes,
                                    int *success)
{
    return (cb)(buf, nbytes, success);
}

void jsonnet_internal_free_string(char *str) {
    ::free(str);
}

void jsonnet_internal_free_pointer(void *ptr) {
    ::free(ptr);
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

char *jsonnet_internal_realloc(JsonnetVm *vm, char *str, size_t sz)
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
