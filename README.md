# go-jsonnet

Forked from https://github.com/google/go-jsonnet

We use it in https://github.com/keboola/keboola-as-code

## Changes

- Added the ability to find out where value generated from native function was used (https://github.com/keboola/go-jsonnet/commit/e94cde1292db53707112a21a872c54865e058c81);
- Added method `VM.Bind(ast.Identifier, ast.Node)` to bind a global value (https://github.com/keboola/go-jsonnet/commit/33be8e4f383657d76f5c72bdaa55bd9afc4bdd0b).
- Added function `formatter.FormatAst` (https://github.com/keboola/go-jsonnet/commit/9ad4d733a48d4d9ace408df008043c5d8865f329).
- Added public `parser` package (https://github.com/keboola/go-jsonnet/commit/2ec811651f6c1cbf8e8ea0f9ae0e4ecc85b5d36c).
- Add public `pass` package (https://github.com/keboola/go-jsonnet/commit/d84d404673c55bf44ed99ca26356f4df3f6a0238).
