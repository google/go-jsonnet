load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
)

# NB: update_cpp_jsonnet.sh looks for these.
CPP_JSONNET_SHA256 = "21ebdb2d9e3ac83f5ee80a94ef37112b412440407e2f3db8e8147544a64b8ae1"
CPP_JSONNET_GITHASH = "ca2d672ffe4c243570671ee0cd62d887f123372e"

def jsonnet_go_repositories():
    http_archive(
        name = "io_bazel_rules_go",
        sha256 = "16e9fca53ed6bd4ff4ad76facc9b7b651a89db1689a2877d6fd7b82aa824e366",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.34.0/rules_go-v0.34.0.zip",
            "https://github.com/bazelbuild/rules_go/releases/download/v0.34.0/rules_go-v0.34.0.zip",
        ],
    )

    http_archive(
        name = "bazel_gazelle",
        sha256 = "501deb3d5695ab658e82f6f6f549ba681ea3ca2a5fb7911154b5aa45596183fa",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.26.0/bazel-gazelle-v0.26.0.tar.gz",
            "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.26.0/bazel-gazelle-v0.26.0.tar.gz",
        ],
    )
    http_archive(
        name = "cpp_jsonnet",
        sha256 = CPP_JSONNET_SHA256,
        strip_prefix = "jsonnet-%s" % CPP_JSONNET_GITHASH,
        urls = ["https://github.com/google/jsonnet/archive/%s.tar.gz" % CPP_JSONNET_GITHASH],
    )
