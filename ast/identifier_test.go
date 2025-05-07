package ast

import (
	"fmt"
	"testing"
)

func testIdentifiers(n int) Identifiers {
	var is []Identifier
	for i := 0; i < n; i++ {
		is = append(is, Identifier(fmt.Sprintf("id-%06d", i)))
	}
	return Identifiers(is)
}

var results []Identifier

func BenchmarkToOrderedSlice(b *testing.B) {
	tests := []Identifiers{
		testIdentifiers(1),
		testIdentifiers(10),
		testIdentifiers(100),
		testIdentifiers(1000),
		testIdentifiers(10000),
		testIdentifiers(100000),
	}

	for _, t := range tests {
		is := IdentifierSet{}
		is.AddIdentifiers(t)

		b.Run(fmt.Sprintf("%d unique identifiers", len(t)), func(b *testing.B) {
			var r []Identifier
			for i := 0; i < b.N; i++ {
				r = is.ToOrderedSlice()
			}
			results = r
		})
	}
}

func BenchmarkToSlice(b *testing.B) {
	tests := []Identifiers{
		testIdentifiers(1),
		testIdentifiers(10),
		testIdentifiers(100),
		testIdentifiers(1000),
		testIdentifiers(10000),
		testIdentifiers(100000),
	}

	for _, t := range tests {
		is := IdentifierSet{}
		is.AddIdentifiers(t)

		b.Run(fmt.Sprintf("%d unique identifiers", len(t)), func(b *testing.B) {
			var r []Identifier
			for i := 0; i < b.N; i++ {
				r = is.ToSlice()
			}
			results = r
		})
	}
}
