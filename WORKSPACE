workspace(name = "google_jsonnet_go")

load(
    "@google_jsonnet_go//bazel:repositories.bzl",
    "jsonnet_go_repositories",
)

jsonnet_go_repositories()

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains")

go_register_toolchains(version = "host")

load(
    "@google_jsonnet_go//bazel:deps.bzl",
    "jsonnet_go_dependencies",
)

jsonnet_go_dependencies()

#gazelle:repository_macro bazel/deps.bzl%jsonnet_go_dependencies
