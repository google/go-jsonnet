load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
)

# NB: update_cpp_jsonnet.sh looks for these.
CPP_JSONNET_SHA256 = "e7d14d8ad65dc334b9a9f3bd1c542a82b3b80029860a8d78fd829b23d0e9845b"
CPP_JSONNET_GITHASH = "5a4e8e34cc1fe841bdebb983646b9b9ae8fa8ca4"

def jsonnet_go_repositories():
    http_archive(
        name = "io_bazel_rules_go",
        sha256 = "33acc4ae0f70502db4b893c9fc1dd7a9bf998c23e7ff2c4517741d4049a976f8",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.48.0/rules_go-v0.48.0.zip",
            "https://github.com/bazelbuild/rules_go/releases/download/v0.48.0/rules_go-v0.48.0.zip",
        ],
    )

    http_archive(
        name = "bazel_gazelle",
        sha256 = "d76bf7a60fd8b050444090dfa2837a4eaf9829e1165618ee35dceca5cbdf58d5",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.37.0/bazel-gazelle-v0.37.0.tar.gz",
            "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.37.0/bazel-gazelle-v0.37.0.tar.gz",
        ],
    )
    http_archive(
        name = "cpp_jsonnet",
        sha256 = CPP_JSONNET_SHA256,
        strip_prefix = "jsonnet-%s" % CPP_JSONNET_GITHASH,
        urls = ["https://github.com/google/jsonnet/archive/%s.tar.gz" % CPP_JSONNET_GITHASH],
    )
