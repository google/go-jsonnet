module example/go-jsonnet-using-bazel

go 1.23.7

require github.com/google/go-jsonnet v0.21.0-rc2

require (
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/google/go-jsonnet => ../../
