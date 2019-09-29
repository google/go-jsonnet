#!/usr/bin/env python3
import unittest
import ctypes
import re


lib = ctypes.CDLL('../c-bindings/libgojsonnet.so')

# jsonnet declaration

lib.jsonnet_evaluate_snippet.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.c_char_p,
    ctypes.POINTER(ctypes.c_int),
]
lib.jsonnet_evaluate_snippet.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_make.argtypes = []
lib.jsonnet_make.restype = ctypes.c_void_p

lib.jsonnet_string_output.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_string_output.restype = None

t =  [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.c_char_p,
]
lib.jsonnet_ext_var.argtypes = t
lib.jsonnet_ext_code.argtypes = t
lib.jsonnet_tla_var.argtypes = t
lib.jsonnet_tla_code.argtypes = t

lib.jsonnet_jpath_add.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
]
lib.jsonnet_jpath_add.restype = None
        
lib.jsonnet_max_trace.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_max_trace.restype = None

lib.jsonnet_evaluate_file.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.POINTER(ctypes.c_int),
]
lib.jsonnet_evaluate_file.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_destroy.argtypes = [
    ctypes.c_void_p
]
lib.jsonnet_destroy.restype = None

lib.jsonnet_realloc.argtypes = [
    ctypes.c_void_p,
    ctypes.POINTER(ctypes.c_char),
    ctypes.c_ulong,
]
lib.jsonnet_realloc.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_version.argtypes = []
lib.jsonnet_version.restype = ctypes.POINTER(ctypes.c_char)

NATIVE_CALLBACK = ctypes.CFUNCTYPE(ctypes.c_void_p, ctypes.c_void_p, ctypes.POINTER(ctypes.c_void_p), ctypes.POINTER(ctypes.c_int))

lib.jsonnet_native_callback.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    NATIVE_CALLBACK,
    ctypes.c_void_p,
    ctypes.POINTER(ctypes.c_char_p),
]
lib.jsonnet_native_callback.restype = None

# json declaration

lib.jsonnet_json_make_string.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
]
lib.jsonnet_json_make_string.restype = ctypes.c_void_p

lib.jsonnet_json_extract_string.argtypes = [
    ctypes.c_void_p,
    ctypes.c_void_p,
]
lib.jsonnet_json_extract_string.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_json_make_number.argtypes = [
    ctypes.c_void_p,
    ctypes.c_double,
]
lib.jsonnet_json_make_number.restype = ctypes.c_void_p

lib.jsonnet_json_extract_number.argtypes = [
    ctypes.c_void_p,
    ctypes.c_void_p,
    ctypes.POINTER(ctypes.c_double)
]
lib.jsonnet_json_extract_number.restype = ctypes.c_int

lib.jsonnet_json_make_bool.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_json_make_bool.restype = ctypes.c_void_p

lib.jsonnet_json_extract_bool.argtypes = [
    ctypes.c_void_p,
    ctypes.c_void_p,
]
lib.jsonnet_json_extract_bool.restype = ctypes.c_int

lib.jsonnet_json_make_null.argtypes = [
    ctypes.c_void_p,
]
lib.jsonnet_json_make_null.restype = ctypes.c_void_p

lib.jsonnet_json_extract_null.argtypes = [
    ctypes.c_void_p,
    ctypes.c_void_p,
]
lib.jsonnet_json_extract_null.restype = ctypes.c_int

lib.jsonnet_json_make_array.argtypes = [
    ctypes.c_void_p,
]
lib.jsonnet_json_make_array.restype = ctypes.c_void_p

lib.jsonnet_json_array_append.argtypes = [
    ctypes.c_void_p,
    ctypes.c_void_p,
    ctypes.c_void_p,
]
lib.jsonnet_json_array_append.restype = None

lib.jsonnet_json_make_object.argtypes = [
    ctypes.c_void_p,
]
lib.jsonnet_json_make_object.restype = ctypes.c_void_p

lib.jsonnet_json_object_append.argtypes = [
    ctypes.c_void_p,
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.c_void_p,
]
lib.jsonnet_json_object_append.restype = None

lib.jsonnet_json_destroy.argtypes = [
    ctypes.c_void_p,
    ctypes.c_void_p,
]
lib.jsonnet_json_destroy.restype = None

# utils

def free_buffer(vm, buf):
    assert not lib.jsonnet_realloc(vm, buf, 0)

def to_bytes(buf):
    return ctypes.cast(buf, ctypes.c_char_p).value

@NATIVE_CALLBACK
def square_native(ctx, argv, success):
    a = ctypes.c_double(0)
    res = lib.jsonnet_json_extract_number(ctx, argv[0], ctypes.byref(a))

    if res == 0:
        success[0] = ctypes.c_int(0)
        return lib.jsonnet_json_make_string(ctx, b"Bad param 'a'.")

    success[0] = ctypes.c_int(1)
    return lib.jsonnet_json_make_number(ctx, a.value**2)

@NATIVE_CALLBACK
def concat_native(ctx, argv, success):
    a = lib.jsonnet_json_extract_string(ctx, argv[0]);
    b = lib.jsonnet_json_extract_string(ctx, argv[1]);

    if a == None or b == None:
        success[0] = ctypes.c_int(0)
        return lib.jsonnet_json_make_string(ctx, "Bad params.");

    res = lib.jsonnet_json_make_string(ctx, to_bytes(a) + to_bytes(b))
    success[0] = ctypes.c_int(1)
    return res

@NATIVE_CALLBACK
def build_native(ctx, argv, success):
    m = lib.jsonnet_json_make_object(ctx)
    lib.jsonnet_json_object_append(ctx, m, b"a", lib.jsonnet_json_make_string(ctx, b"hello"))
    lib.jsonnet_json_object_append(ctx, m, b"b", lib.jsonnet_json_make_string(ctx, b"world"))

    res = lib.jsonnet_json_make_array(ctx)
    lib.jsonnet_json_array_append(ctx, res, m)

    success[0] = ctypes.c_int(1)
    return res

class TestJsonnetEvaluateBindings(unittest.TestCase):
    def setUp(self):
        self.err = ctypes.c_int()
        self.err_ref =  ctypes.byref(self.err)
        self.vm = lib.jsonnet_make()

    def test_add_strings(self):
        res = lib.jsonnet_evaluate_snippet(self.vm, b"vm1", b"'xxx' + 'yyy'", self.err_ref)
        self.assertEqual(b'"xxxyyy"\n', to_bytes(res))
        free_buffer(self.vm, res)

    def test_string_output(self):
        lib.jsonnet_string_output(self.vm, 1)
        res = lib.jsonnet_evaluate_snippet(self.vm, b"vm2", b"'xxx' + 'yyy'", self.err_ref)
        self.assertEqual(b'xxxyyy\n', to_bytes(res))
        free_buffer(self.vm, res)


    def test_params(self):
        lib.jsonnet_ext_var(self.vm, b"e1", b"a")
        lib.jsonnet_ext_code(self.vm, b"e2", b"'b'")
        lib.jsonnet_tla_var(self.vm, b"t1", b"c")
        lib.jsonnet_tla_code(self.vm, b"t2", b"'d'")

        res = lib.jsonnet_evaluate_snippet(self.vm, b"ext_and_tla", b"""function(t1, t2) std.extVar("e1") + std.extVar("e2") + t1 + t2""", self.err_ref)
        self.assertEqual(b'"abcd"\n', to_bytes(res))

        free_buffer(self.vm, res)


    def test_jpath(self):
        lib.jsonnet_jpath_add(self.vm, b"jsonnet_import_test/")
        res = lib.jsonnet_evaluate_snippet(self.vm, b"jpath", b"""import 'foo.jsonnet'""", self.err_ref)
        self.assertEqual(b"42\n", to_bytes(res))
        free_buffer(self.vm, res)


    def test_max_trace(self):
        lib.jsonnet_max_trace(self.vm, 4)
        res = lib.jsonnet_evaluate_snippet(self.vm, b"max_trace", b"""local f(x) = if x == 0 then error 'expected' else f(x - 1); f(10)""", self.err_ref)
        expectedTrace = b'RUNTIME ERROR: expected\n\tmax_trace:1:29-45\tfunction <f>\n\tmax_trace:1:51-59\tfunction <f>\n\t...\n\tmax_trace:1:61-66\t$\n\tDuring evaluation\t\n'
        self.assertEqual(expectedTrace, to_bytes(res))
        free_buffer(self.vm, res)

    def test_evaluate_file(self):
        res = lib.jsonnet_evaluate_file(self.vm, b"jsonnet_import_test/foo.jsonnet", self.err_ref)
        self.assertEqual(b"42\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_jsonnet_version(self):
        res = lib.jsonnet_version()
        match = re.match(r'^v[0-9]+[.][0-9]+[.][0-9]+ [(]go-jsonnet[)]$', to_bytes(res).decode('utf-8'))
        self.assertIsNotNone(match)

    def test_jsonnet_native_callback_square(self):
        arr = (ctypes.c_char_p * 2)()
        arr[0] = b"a"
        arr[1] = ctypes.c_char_p()

        lib.jsonnet_native_callback(self.vm, b"square", square_native, self.vm, arr)
        res = lib.jsonnet_evaluate_snippet(self.vm, b"native_callback", b"std.native('square')(6+3)", self.err_ref)
        self.assertEqual(b'81\n', to_bytes(res))
        free_buffer(self.vm, res)

    def test_jsonnet_native_callback_concat(self):
        arr = (ctypes.c_char_p * 3)()
        arr[0] = b"a"
        arr[1] = b"b"
        arr[2] = ctypes.c_char_p()

        lib.jsonnet_native_callback(self.vm, b"concat", concat_native, self.vm, arr)
        res = lib.jsonnet_evaluate_snippet(self.vm, b"concat_callback", b"std.native('concat')('hello', 'ween')", self.err_ref)
        self.assertEqual(b'"helloween"\n', to_bytes(res))
        free_buffer(self.vm, res)

    def test_jsonnet_native_callback_build(self):
        arr = (ctypes.c_char_p * 1)()
        arr[0] = ctypes.c_char_p()

        lib.jsonnet_native_callback(self.vm, b"build", build_native, self.vm, arr)
        res = lib.jsonnet_evaluate_snippet(self.vm, b"build_callback", b"std.native('build')()", self.err_ref)
        self.assertEqual(b'[\n   {\n      "a": "hello",\n      "b": "world"\n   }\n]\n', to_bytes(res))

        free_buffer(self.vm, res)

    def tearDown(self):
        lib.jsonnet_destroy(self.vm)


class TestJsonnetJsonValueBindings(unittest.TestCase):
    def setUp(self):
        self.vm = lib.jsonnet_make()

    def test_jsonnet_string(self):
        h = lib.jsonnet_json_make_string(self.vm, b"test")
        res = lib.jsonnet_json_extract_string(self.vm, h)

        self.assertEqual(b"test", to_bytes(res))
        lib.jsonnet_json_destroy(self.vm, h)

    def test_jsonnet_number(self):
        h = lib.jsonnet_json_make_number(self.vm, 9.9991)
        actual = ctypes.c_double(0)
        res = lib.jsonnet_json_extract_number(self.vm, h, ctypes.byref(actual))

        self.assertEqual(1, res)
        self.assertEqual(9.9991, actual.value)
        lib.jsonnet_json_destroy(self.vm, h)

    def test_jsonnet_bool(self):
        h = lib.jsonnet_json_make_bool(self.vm, 3)
        res = lib.jsonnet_json_extract_bool(self.vm, h)

        self.assertEqual(1, res)
        lib.jsonnet_json_destroy(self.vm, h)

    def test_jsonnet_null(self):
        h = lib.jsonnet_json_make_null(self.vm)
        res = lib.jsonnet_json_extract_null(self.vm, h)

        self.assertEqual(1, res)
        lib.jsonnet_json_destroy(self.vm, h)

    def test_jsonnet_array(self):
        h = lib.jsonnet_json_make_array(self.vm)
        
        lib.jsonnet_json_array_append(self.vm, h, lib.jsonnet_json_make_string(self.vm, b"Test 1.1"))
        lib.jsonnet_json_array_append(self.vm, h, lib.jsonnet_json_make_string(self.vm, b"Test 1.2"))
        lib.jsonnet_json_array_append(self.vm, h, lib.jsonnet_json_make_string(self.vm, b"Test 1.3"))
        lib.jsonnet_json_array_append(self.vm, h, lib.jsonnet_json_make_bool(self.vm, 1))
        lib.jsonnet_json_array_append(self.vm, h, lib.jsonnet_json_make_number(self.vm, 42))
        lib.jsonnet_json_array_append(self.vm, h, lib.jsonnet_json_make_null(self.vm))
        lib.jsonnet_json_array_append(self.vm, h, lib.jsonnet_json_make_object(self.vm))

        lib.jsonnet_json_destroy(self.vm, h)

    def test_jsonnet_object(self):
        h = lib.jsonnet_json_make_object(self.vm)
        
        lib.jsonnet_json_object_append(self.vm, h, b"arg1", lib.jsonnet_json_make_string(self.vm, b"Test 1.1"))
        lib.jsonnet_json_object_append(self.vm, h, b"arg2", lib.jsonnet_json_make_string(self.vm, b"Test 1.2"))
        lib.jsonnet_json_object_append(self.vm, h, b"arg3", lib.jsonnet_json_make_string(self.vm, b"Test 1.3"))
        lib.jsonnet_json_object_append(self.vm, h, b"arg4", lib.jsonnet_json_make_bool(self.vm, 1))
        lib.jsonnet_json_object_append(self.vm, h, b"arg5", lib.jsonnet_json_make_number(self.vm, 42))
        lib.jsonnet_json_object_append(self.vm, h, b"arg6", lib.jsonnet_json_make_null(self.vm))
        lib.jsonnet_json_object_append(self.vm, h, b"arg7", lib.jsonnet_json_make_object(self.vm))

        lib.jsonnet_json_destroy(self.vm, h)


    def tearDown(self):
        lib.jsonnet_destroy(self.vm)

if __name__ == '__main__':
    unittest.main()