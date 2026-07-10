module example/go-jsonnet-using-bazel

go 1.25.0

require github.com/google/go-jsonnet v0.22.0

require (
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/google/go-jsonnet => ../../
