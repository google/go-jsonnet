#!/usr/bin/env python3
import unittest
import ctypes
import re


lib = ctypes.CDLL('../c-bindings/libgojsonnet.so')

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


def free_buffer(vm, buf):
    assert not lib.jsonnet_realloc(vm, buf, 0)


def to_bytes(buf):
    return ctypes.cast(buf, ctypes.c_char_p).value


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

    def tearDown(self):
        lib.jsonnet_destroy(self.vm)


if __name__ == '__main__':
    unittest.main()