load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
)

def _maybe(repo_rule, name, **kwargs):
    """Executes the given repository rule if it hasn't been executed already.
    Args:
      repo_rule: The repository rule to be executed (e.g.,
          `native.git_repository`.)
      name: The name of the repository to be defined by the rule.
      **kwargs: Additional arguments passed directly to the repository rule.
    """
    if not native.existing_rule(name):
        repo_rule(name = name, **kwargs)

def jsonnet_go_repositories():
    _maybe(
        http_archive,
        name = "io_bazel_rules_go",
        sha256 = "e6f8cb2da438cc4899809b66ba96d57397ed871640fe5c848ca9c56190b7c8ba",
        strip_prefix = "rules_go-8ea79bbd5e6ea09dc611c245d1dc09ef7ab7118a",
        urls = ["https://github.com/bazelbuild/rules_go/archive/8ea79bbd5e6ea09dc611c245d1dc09ef7ab7118a.zip"],
    )
    _maybe(
        http_archive,
        name = "bazel_gazelle",
        sha256 = "c5faf839dd1da0065ed7d44ac248b01ab5ffcd0db46e7193439906df68867c39",
        strip_prefix = "bazel-gazelle-38bd65ead186af23549480d6189b89c7c53c023e",
        urls = ["https://github.com/bazelbuild/bazel-gazelle/archive/38bd65ead186af23549480d6189b89c7c53c023e.zip"],
    )
    _maybe(
        http_archive,
        name = "cpp_jsonnet",
        sha256 = "82d3cd35de8ef230d094b60a30e7659f415c350b0aa2bd62162cf2afdf163959",
        strip_prefix = "jsonnet-90cad75dcc2eafdcf059c901169d36539dc8a699",
        urls = ["https://github.com/google/jsonnet/archive/90cad75dcc2eafdcf059c901169d36539dc8a699.tar.gz"],
    )
