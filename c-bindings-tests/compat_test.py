#!/usr/bin/env python3
import ctypes
import io
import os
import re
import unittest


lib = ctypes.CDLL('../c-bindings/libgojsonnet.so')

# jsonnet declaration

lib.jsonnet_evaluate_snippet.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.c_char_p,
    ctypes.POINTER(ctypes.c_int),
]
lib.jsonnet_evaluate_snippet.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_evaluate_snippet_stream.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.c_char_p,
    ctypes.POINTER(ctypes.c_int),
]
lib.jsonnet_evaluate_snippet_stream.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_evaluate_snippet_multi.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.c_char_p,
    ctypes.POINTER(ctypes.c_int),
]
lib.jsonnet_evaluate_snippet_multi.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_evaluate_file_stream.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.POINTER(ctypes.c_int),
]
lib.jsonnet_evaluate_file_stream.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_evaluate_file_multi.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.POINTER(ctypes.c_int),
]
lib.jsonnet_evaluate_file_multi.restype = ctypes.POINTER(ctypes.c_char)

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

IMPORT_CALLBACK = ctypes.CFUNCTYPE(
    ctypes.c_char_p,
    ctypes.c_void_p,
    ctypes.POINTER(ctypes.c_char),
    ctypes.POINTER(ctypes.c_char),
    # we use *int instead of **char to pass the real C allocated pointer, that we have to free
    ctypes.POINTER(ctypes.c_uint64),
    ctypes.POINTER(ctypes.c_int)
)

lib.jsonnet_import_callback.argtypes = [
    ctypes.c_void_p,
    IMPORT_CALLBACK,
    ctypes.c_void_p,
]
lib.jsonnet_import_callback.restype = None

IO_WRITER_CALLBACK = ctypes.CFUNCTYPE(
    ctypes.c_int,
    ctypes.c_void_p,
    ctypes.c_size_t,
    ctypes.POINTER(ctypes.c_int)
)

lib.jsonnet_set_trace_out_callback.argtypes = [
    ctypes.c_void_p,
    IO_WRITER_CALLBACK,
]
lib.jsonnet_set_trace_out_callback.restype = None

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

# fmt declaration

lib.jsonnet_fmt_snippet.argtypes = [
    ctypes.c_void_p,
    ctypes.c_char_p,
    ctypes.c_char_p,
    ctypes.POINTER(ctypes.c_int),
]
lib.jsonnet_fmt_snippet.restype = ctypes.POINTER(ctypes.c_char)

lib.jsonnet_fmt_indent.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_fmt_indent.restype = None

lib.jsonnet_fmt_max_blank_lines.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_fmt_max_blank_lines.restype = None

lib.jsonnet_fmt_string.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_fmt_string.restype = None

lib.jsonnet_fmt_comment.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_fmt_comment.restype = None

lib.jsonnet_fmt_pad_arrays.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_fmt_pad_arrays.restype = None

lib.jsonnet_fmt_pad_objects.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_fmt_pad_objects.restype = None

lib.jsonnet_fmt_pretty_field_names.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_fmt_pretty_field_names.restype = None

lib.jsonnet_fmt_sort_imports.argtypes = [
    ctypes.c_void_p,
    ctypes.c_int,
]
lib.jsonnet_fmt_sort_imports.restype = None

# utils

def free_buffer(vm, buf):
    assert not lib.jsonnet_realloc(vm, buf, 0)

def to_bytes(buf):
    return ctypes.cast(buf, ctypes.c_char_p).value

def to_bytes_list(buf):
    res = []
    raw_ptr = ctypes.cast(buf, ctypes.c_void_p).value
    while True:
        elem = ctypes.cast(raw_ptr, ctypes.c_char_p).value
        if len(elem) == 0:
            break
        res.append(elem)
        raw_ptr = raw_ptr + len(elem) + 1
    return res

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
    a = lib.jsonnet_json_extract_string(ctx, argv[0])
    b = lib.jsonnet_json_extract_string(ctx, argv[1])

    if a == None or b == None:
        success[0] = ctypes.c_int(0)
        return lib.jsonnet_json_make_string(ctx, "Bad params.")

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

@IMPORT_CALLBACK
def import_callback(ctx, dir, rel, found_here, success):
    full_path, content = jsonnet_try_path(b"jsonnet_import_test/", to_bytes(rel))

    bcontent = content.encode()
    dst = lib.jsonnet_realloc(ctx, None, len(bcontent) + 1)
    ctypes.memmove(ctypes.addressof(dst.contents), bcontent, len(bcontent) + 1)

    fdst = lib.jsonnet_realloc(ctx, None, len(full_path) + 1)
    ctypes.memmove(ctypes.addressof(fdst.contents), full_path, len(full_path) + 1)
    found_here[0] = ctypes.addressof(fdst.contents)

    success[0] = ctypes.c_int(1)

    return ctypes.addressof(dst.contents)

io_writer_buf = None

@IO_WRITER_CALLBACK
def io_writer_callback(buf, nbytes, success):
    global io_writer_buf
    io_writer_buf = ctypes.string_at(buf, nbytes)
    success[0] = ctypes.c_int(1)
    return nbytes

#  Returns content if worked, None if file not found, or throws an exception
def jsonnet_try_path(dir, rel):
    if not rel:
        raise RuntimeError('Got invalid filename (empty string).')
    if rel[0] == '/':
        full_path = rel
    else:
        full_path = dir + rel
    if full_path[-1] == '/':
        raise RuntimeError('Attempted to import a directory')

    if not os.path.isfile(full_path):
        return full_path, None
    with open(full_path) as f:
        return full_path, f.read()

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

    def test_jsonnet_evaluate_snippet_stream(self):
        res = lib.jsonnet_evaluate_snippet_stream(
            self.vm,
            b"vm1",
            b"['aaa', 'bbb', {foo: 'bar', bar: 'baz'}]",
            self.err_ref
        )
        self.assertEqual([
                b'"aaa"\n',
                b'"bbb"\n',
                b'{\n   "bar": "baz",\n   "foo": "bar"\n}\n'
            ], to_bytes_list(res))

    def test_jsonnet_evaluate_snippet_multi(self):
        res = lib.jsonnet_evaluate_snippet_multi(
            self.vm,
            b"vm1",
            b"{foo: 'bar', bar: 'baz'}",
            self.err_ref
        )
        self.assertEqual([
                b'bar',
                b'"baz"\n',
                b'foo',
                b'"bar"\n',
            ], to_bytes_list(res))

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

    def test_jsonnet_evaluate_file_stream(self):
        res = lib.jsonnet_evaluate_file_stream(self.vm, b"jsonnet_import_test/stream.jsonnet", self.err_ref)
        self.assertEqual([
                b'"aaa"\n',
                b'"bbb"\n',
                b'{\n   "bar": "baz",\n   "foo": "bar"\n}\n'
            ], to_bytes_list(res))

    def test_jsonnet_evaluate_file_multi(self):
        res = lib.jsonnet_evaluate_file_multi(self.vm, b"jsonnet_import_test/multi.jsonnet", self.err_ref)
        self.assertEqual([
                b'bar',
                b'"baz"\n',
                b'foo',
                b'"bar"\n',
            ], to_bytes_list(res))

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

    def test_jsonnet_import_callback(self):
        lib.jsonnet_import_callback(self.vm, import_callback, self.vm)

        res = lib.jsonnet_evaluate_snippet(self.vm, b"jsonnet_import_callback", b"""import 'foo.jsonnet'""", self.err_ref)
        self.assertEqual(b'42\n', to_bytes(res))
        free_buffer(self.vm, res)

    def test_jsonnet_set_trace_out_callback(self):
        lib.jsonnet_set_trace_out_callback(self.vm, io_writer_callback)
        fname = b"vm1"
        msg = b"test_jsonnet_set_trace_out_callback trace message"
        expected = b"TRACE: " + fname + b":1 " + msg + b"\n"
        snippet = b"std.trace('" + msg + b"', 'rest')"
        lib.jsonnet_evaluate_snippet(self.vm, fname, snippet, self.err_ref)
        self.assertEqual(io_writer_buf,  expected)

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

class TestJsonnetFormatBindings(unittest.TestCase):
    def setUp(self):
        self.err = ctypes.c_int()
        self.err_ref = ctypes.byref(self.err)
        self.vm = lib.jsonnet_make()

    def test_format(self):
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"local a = import 'z.libsonnet';\nlocal b = import 'y.libsonnet';\n{'n':1, s: \"y\", a: [1,2]}", self.err_ref)
        self.assertEqual(b"local b = import 'y.libsonnet';\nlocal a = import 'z.libsonnet';\n{ n: 1, s: 'y', a: [1, 2] }\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_indent(self):
        lib.jsonnet_fmt_indent(self.vm, 8)
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"{\nx:1,\ny:2\n}", self.err_ref)
        self.assertEqual(b"{\n        x: 1,\n        y: 2,\n}\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_max_blank_lines(self):
        lib.jsonnet_fmt_max_blank_lines(self.vm, 2)
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"{\nx:1,\n\n\n\n\ny:2\n}", self.err_ref)
        self.assertEqual(b"{\n  x: 1,\n\n\n  y: 2,\n}\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_string(self):
        lib.jsonnet_fmt_string(self.vm, ord('d'))
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"{x:'x'}", self.err_ref)
        self.assertEqual(b"{ x: \"x\" }\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_comment(self):
        lib.jsonnet_fmt_comment(self.vm, ord('h'))
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"// comment\n{}", self.err_ref)
        self.assertEqual(b"# comment\n{}\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_pad_arrays(self):
        lib.jsonnet_fmt_pad_arrays(self.vm, 1)
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"{x:[1,2,3]}", self.err_ref)
        self.assertEqual(b"{ x: [ 1, 2, 3 ] }\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_pad_objects(self):
        lib.jsonnet_fmt_pad_objects(self.vm, 0)
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"{ x: 1 }", self.err_ref)
        self.assertEqual(b"{x: 1}\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_pretty_field_names(self):
        lib.jsonnet_fmt_pretty_field_names(self.vm, 0)
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"{ 'x': 1 }", self.err_ref)
        self.assertEqual(b"{ 'x': 1 }\n", to_bytes(res))
        free_buffer(self.vm, res)

    def test_sort_imports(self):
        lib.jsonnet_fmt_sort_imports(self.vm, 0)
        res = lib.jsonnet_fmt_snippet(self.vm, b"fmt", b"local a = import 'z.libsonnet';\nlocal b = import 'y.libsonnet';\na+b", self.err_ref)
        self.assertEqual(b"local a = import 'z.libsonnet';\nlocal b = import 'y.libsonnet';\na + b\n", to_bytes(res))
        free_buffer(self.vm, res)


    def tearDown(self):
        lib.jsonnet_destroy(self.vm)

if __name__ == '__main__':
    unittest.main()
