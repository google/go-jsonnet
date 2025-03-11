load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
)

# NB: update_cpp_jsonnet.sh looks for these.
CPP_JSONNET_SHA256 = "f104659d3feb42c871d40c5142577dff2cd3b2eda33ac9534f5f12b14643748d"
CPP_JSONNET_GITHASH = "2c3b51491a67ab7b24aecfc23fa3f73f68135129"
CPP_JSONNET_RELEASE_VERSION = "v0.21.0-rc1"

CPP_JSONNET_STRIP_PREFIX = (
    "jsonnet-" + (
        CPP_JSONNET_RELEASE_VERSION if CPP_JSONNET_RELEASE_VERSION else CPP_JSONNET_GITHASH
    )
)
CPP_JSONNET_URL = (
    "https://github.com/google/jsonnet/releases/download/%s/jsonnet-%s.tar.gz" % (
        CPP_JSONNET_RELEASE_VERSION,
        CPP_JSONNET_RELEASE_VERSION,
    ) if CPP_JSONNET_RELEASE_VERSION else "https://github.com/google/jsonnet/archive/%s.tar.gz" % CPP_JSONNET_GITHASH
)

def jsonnet_go_repositories():
    http_archive(
        name = "io_bazel_rules_go",
        sha256 = "b78f77458e77162f45b4564d6b20b6f92f56431ed59eaaab09e7819d1d850313",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.53.0/rules_go-v0.53.0.zip",
            "https://github.com/bazelbuild/rules_go/releases/download/v0.53.0/rules_go-v0.53.0.zip",
        ],
    )

    http_archive(
        name = "bazel_gazelle",
        sha256 = "5d80e62a70314f39cc764c1c3eaa800c5936c9f1ea91625006227ce4d20cd086",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.42.0/bazel-gazelle-v0.42.0.tar.gz",
            "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.42.0/bazel-gazelle-v0.42.0.tar.gz",
        ],
    )
    http_archive(
        name = "cpp_jsonnet",
        sha256 = CPP_JSONNET_SHA256,
        strip_prefix = CPP_JSONNET_STRIP_PREFIX,
        urls = [CPP_JSONNET_URL],
    )
