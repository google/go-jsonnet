#include <stdio.h>
#include <stdlib.h>

extern "C" {
    #include "libjsonnet.h"
    #include "internal.h"
}

#include "json.h"

struct JsonnetVm *jsonnet_internal_make_vm_with_id(uint32_t id) {
    JsonnetVm *vm = new JsonnetVm();
    vm->id = id;
    return vm;
}

void jsonnet_internal_free_vm(struct JsonnetVm *x) {
    delete(x);
}

inline static void todo() {
    fputs("TODO, NOT IMPLEMENTED YET\n", stderr);
    abort();
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

void jsonnet_native_callback(struct JsonnetVm *vm, const char *name, JsonnetNativeCallback *cb,
    void *ctx, const char *const *params)
{
    todo();
}

char *jsonnet_evaluate_file_multi(JsonnetVm *vm, const char *filename, int *error)
{
    todo();
    return nullptr;
}

char *jsonnet_evaluate_file_stream(JsonnetVm *vm, const char *filename, int *error)
{
    todo();
    return nullptr;
}

char *jsonnet_evaluate_snippet_multi(JsonnetVm *vm, const char *filename, const char *snippet,
                                     int *error)
{
    todo();
    return nullptr;
}

char *jsonnet_evaluate_snippet_stream(JsonnetVm *vm, const char *filename, const char *snippet,
                                      int *error)
{
    todo();
    return nullptr;
}
