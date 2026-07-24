//go:build !cshared

// main_stub is a placeholder so plain `go build`, `go vet`, and `go test ./...`
// work without the cgo c-shared machinery. The real entry point
// (cmd/key-bind/main.go, build tag `cshared`) is compiled only under
// `-tags cshared -buildmode=c-shared`.
package main

func main() {}
