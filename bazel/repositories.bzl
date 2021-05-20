load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
)

def jsonnet_go_repositories():
    http_archive(
        name = "io_bazel_rules_go",
        sha256 = "7904dbecbaffd068651916dce77ff3437679f9d20e1a7956bff43826e7645fcc",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.25.1/rules_go-v0.25.1.tar.gz",
            "https://github.com/bazelbuild/rules_go/releases/download/v0.25.1/rules_go-v0.25.1.tar.gz",
        ],
    )

    http_archive(
        name = "bazel_gazelle",
        sha256 = "222e49f034ca7a1d1231422cdb67066b885819885c356673cb1f72f748a3c9d4",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
            "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
        ],
    )
    http_archive(
        name = "cpp_jsonnet",
        sha256 = "82d3cd35de8ef230d094b60a30e7659f415c350b0aa2bd62162cf2afdf163959",
        strip_prefix = "jsonnet-90cad75dcc2eafdcf059c901169d36539dc8a699",
        urls = ["https://github.com/google/jsonnet/archive/90cad75dcc2eafdcf059c901169d36539dc8a699.tar.gz"],
    )
