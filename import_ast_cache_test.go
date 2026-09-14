/*
Copyright 2017 Google Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jsonnet

import (
	"sync"
	"testing"
)

// The cache's core promise: a second getOrParse for the same foundAt returns
// the same *parsed* result instead of parsing again. SnippetToAST allocates a
// fresh ast.Node on every call, so two independent parses can never produce
// the same node pointer; observing the same pointer twice is only possible
// if the second call actually hit the cache instead of reparsing.
func TestImportASTCacheParsesOnce(t *testing.T) {
	cache := NewImportASTCache()
	contents := MakeContents("2 + 2")

	node1, err := cache.getOrParse("shared.jsonnet", contents)
	if err != nil {
		t.Fatalf("unexpected error on first parse: %v", err)
	}
	node2, err := cache.getOrParse("shared.jsonnet", contents)
	if err != nil {
		t.Fatalf("unexpected error on second parse: %v", err)
	}
	if node1 != node2 {
		t.Errorf("expected getOrParse to return the cached node on the second call, got a distinct node -- it reparsed instead of reusing the cache")
	}
}

// A parse error is cached too, and stays consistent.
func TestImportASTCacheCachesParseErrors(t *testing.T) {
	cache := NewImportASTCache()
	contents := MakeContents("{ this is not valid jsonnet")

	_, err1 := cache.getOrParse("broken.jsonnet", contents)
	if err1 == nil {
		t.Fatalf("expected a parse error, got none")
	}
	_, err2 := cache.getOrParse("broken.jsonnet", contents)
	if err2 == nil {
		t.Fatalf("expected a parse error on the cached path too, got none")
	}
	if err1.Error() != err2.Error() {
		t.Errorf("expected the same cached error both times, got %q then %q", err1, err2)
	}
}

// Distinct files never collide in the cache.
func TestImportASTCacheKeysByFoundAt(t *testing.T) {
	cache := NewImportASTCache()

	nodeA, err := cache.getOrParse("a.jsonnet", MakeContents("1 + 1"))
	if err != nil {
		t.Fatalf("unexpected error parsing a.jsonnet: %v", err)
	}
	nodeB, err := cache.getOrParse("b.jsonnet", MakeContents("2 + 2"))
	if err != nil {
		t.Fatalf("unexpected error parsing b.jsonnet: %v", err)
	}
	if nodeA == nodeB {
		t.Errorf("expected distinct foundAt keys to produce distinct cached nodes")
	}
}

// The real motivating use (Tanka's tk export --parallel) races many
// goroutines against one ImportASTCache on the same not-yet-cached file.
// Parsing itself deliberately happens outside the lock (see getOrParse), so
// on a genuine simultaneous first touch more than one goroutine can parse
// independently before the cache is populated -- this only asserts the
// guarantee that design actually makes: -race finds nothing, nobody errors
// or panics, and once the race is over the cache has converged on one
// entry that every later caller gets.
func TestImportASTCacheConcurrentAccess(t *testing.T) {
	cache := NewImportASTCache()
	contents := MakeContents("40 + 2")

	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make([]error, goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, err := cache.getOrParse("racy.jsonnet", contents)
			errs[i] = err
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, err)
		}
	}

	// The race is settled now: every subsequent call must agree.
	settled, err := cache.getOrParse("racy.jsonnet", contents)
	if err != nil {
		t.Fatalf("unexpected error after the race settled: %v", err)
	}
	again, err := cache.getOrParse("racy.jsonnet", contents)
	if err != nil {
		t.Fatalf("unexpected error on repeat call: %v", err)
	}
	if settled != again {
		t.Errorf("expected the cache to have converged on one entry once no goroutines are racing it")
	}
}

// The property SetImportASTCache exists for: two VMs configured with
// different ExtCode share one ImportASTCache (so the shared library they
// both import is parsed once, not twice) and still each evaluate to their
// own, correct, non-cross-contaminated result. This is the regression test
// for the hazard the ImportASTCache docstring calls out: only the parse
// step is shared, evaluation (which depends on each VM's own ExtCode) is
// not.
func TestSharedImportASTCacheAcrossVMsWithDifferentExtCode(t *testing.T) {
	shared := NewImportASTCache()

	makeSharedVM := func(clusterName string) *VM {
		vm := MakeVM()
		vm.Importer(&MemoryImporter{
			Data: map[string]Contents{
				// The same library, imported by both VMs. If SetImportASTCache
				// mistakenly shared evaluated values instead of only the parsed
				// AST, one VM's ExtCode would leak into the other's result below.
				"lib.jsonnet": MakeContents(`{ cluster: std.extVar('cluster') }`),
			},
		})
		vm.ExtVar("cluster", clusterName)
		vm.SetImportASTCache(shared)
		return vm
	}

	vmA := makeSharedVM("cluster-a")
	vmB := makeSharedVM("cluster-b")

	outA, err := vmA.EvaluateAnonymousSnippet("root-a.jsonnet", `(import "lib.jsonnet").cluster`)
	if err != nil {
		t.Fatalf("vmA: unexpected error: %v", err)
	}
	outB, err := vmB.EvaluateAnonymousSnippet("root-b.jsonnet", `(import "lib.jsonnet").cluster`)
	if err != nil {
		t.Fatalf("vmB: unexpected error: %v", err)
	}

	outA = removeExcessiveWhitespace(outA)
	outB = removeExcessiveWhitespace(outB)

	if outA != `"cluster-a"` {
		t.Errorf(`vmA: expected "cluster-a", got %s`, outA)
	}
	if outB != `"cluster-b"` {
		t.Errorf(`vmB: expected "cluster-b", got %s`, outB)
	}

	// And the parse itself really was shared: the same foundAt key was
	// populated once, not once per VM.
	if len(shared.cache) != 1 {
		t.Errorf("expected exactly one cached AST entry (lib.jsonnet, parsed once), got %d", len(shared.cache))
	}
}

// Without SetImportASTCache, behavior is exactly as before this type
// existed: nothing is shared, and two VMs importing "the same" logical file
// through independent Importers don't have to agree on anything.
func TestNoSharedImportASTCacheByDefault(t *testing.T) {
	vm := MakeVM()
	vm.Importer(&MemoryImporter{
		Data: map[string]Contents{"a.jsonnet": MakeContents("2 + 2")},
	})
	out, err := vm.EvaluateAnonymousSnippet("root.jsonnet", `import "a.jsonnet"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removeExcessiveWhitespace(out) != "4" {
		t.Errorf("expected 4, got %s", out)
	}
	if vm.importCache.sharedASTCache != nil {
		t.Errorf("expected no shared AST cache by default")
	}
}
