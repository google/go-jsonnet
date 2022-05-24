//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"path/filepath"
	"syscall/js"

	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/internal/formatter"
)

// JavascriptImporter allows importing files from a pre-defined map of absolute
// paths.
type JavascriptImporter struct {
	files map[string]string
}

// Import looks up files in JavascriptImporter
func (importer *JavascriptImporter) Import(importedFrom, importedPath string) (jsonnet.Contents, string, error) {
	fileRootPath := filepath.Dir(importedFrom)
	fullFilePath := filepath.Clean(fmt.Sprintf("%s/%s", fileRootPath, importedPath))

	fileContent, exists := importer.files[fullFilePath]
	if exists {
		return jsonnet.MakeContents(fileContent), importedPath, nil
	} else {
		return jsonnet.Contents{}, "", fmt.Errorf("File not found %v", fullFilePath)
	}
}

func processObjectParam(name string, value js.Value) (map[string]string, error) {
	if value.Type() != js.TypeObject {
		return nil, fmt.Errorf("'%s' was not an object: %v", name, value)
	}
	jsKeysArray := js.Global().Get("Object").Get("keys").Invoke(value)
	result := make(map[string]string)
	for i := 0; i < jsKeysArray.Length(); i++ {
		filename := jsKeysArray.Index(i).String()
		keyValue := value.Get(filename)
		if keyValue.Type() != js.TypeString {
			return nil, fmt.Errorf("'%s' key '%s' was not bound to a string: %v", name, filename, keyValue)
		}
		result[filename] = keyValue.String()
	}
	return result, nil
}

func jsonnetEvaluateSnippet(this js.Value, p []js.Value) (interface{}, error) {
	if len(p) != 7 {
		return "", fmt.Errorf("wrong number of parameters: %d", len(p))
	}
	if p[0].Type() != js.TypeString {
		return "", fmt.Errorf("filename was not a string: %v", p[0])
	}
	if p[1].Type() != js.TypeString {
		return "", fmt.Errorf("code was not a string: %v", p[0])
	}
	filename := p[0].String()
	code := p[1].String()
	files, err := processObjectParam("files", p[2])
	if err != nil {
		return "", err
	}
	extStrs, err := processObjectParam("extStrs", p[3])
	if err != nil {
		return "", err
	}
	extCodes, err := processObjectParam("extCodes", p[4])
	if err != nil {
		return "", err
	}
	tlaStrs, err := processObjectParam("tlaStrs", p[5])
	if err != nil {
		return "", err
	}
	tlaCodes, err := processObjectParam("tlaCodes", p[6])
	if err != nil {
		return "", err
	}

	vm := jsonnet.MakeVM()
	vm.Importer(&JavascriptImporter{files: files})
	for key, val := range extStrs {
		vm.ExtVar(key, val)
	}
	for key, val := range extCodes {
		vm.ExtCode(key, val)
	}
	for key, val := range tlaStrs {
		vm.TLAVar(key, val)
	}
	for key, val := range tlaCodes {
		vm.TLACode(key, val)
	}

	return vm.EvaluateAnonymousSnippet(filename, code)
}

func jsonnetFmtSnippet(this js.Value, p []js.Value) (interface{}, error) {
	if len(p) != 2 {
		return "", fmt.Errorf("wrong number of parameters: %d", len(p))
	}
	if p[0].Type() != js.TypeString {
		return "", fmt.Errorf("filename was not a string: %v", p[0])
	}
	if p[1].Type() != js.TypeString {
		return "", fmt.Errorf("code was not a string: %v", p[0])
	}
	filename := p[0].String()
	code := p[1].String()

	return formatter.Format(filename, code, formatter.DefaultOptions())
}

// promiseFuncOf is like js.FuncOf but returns a promise.
// The promise is able to propagate errors naturally across the wasm /
// javascript bridge.
func promiseFuncOf(jsFunc func(this js.Value, p []js.Value) (interface{}, error)) js.Func {
	return js.FuncOf(func(this js.Value, p []js.Value) interface{} {
		return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]
			go func() {
				value, err := jsFunc(this, p)
				if err != nil {
					reject.Invoke(js.Global().Get("Error").New(err.Error()))
				} else {
					resolve.Invoke(js.ValueOf(value))
				}
			}()
			return nil
		}))
	})
}

func main() {
	js.Global().Set("jsonnet_evaluate_snippet", promiseFuncOf(jsonnetEvaluateSnippet))
	js.Global().Set("jsonnet_fmt_snippet", promiseFuncOf(jsonnetFmtSnippet))
	<-make(chan bool)
}
