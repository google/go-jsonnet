package utils

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/errors"
)

// ErrCollector is a struct for accumulating warnings / errors from the linter.
// It is slightly more convenient and more clear than passing pointers to slices around.
type ErrCollector struct {
	Errs []errors.StaticError
}

// Collect adds an error to the list
func (ec *ErrCollector) Collect(err errors.StaticError) {
	ec.Errs = append(ec.Errs, err)
}

// StaticErr constructs a static error from msg and loc and adds it to the list.
func (ec *ErrCollector) StaticErr(msg string, loc *ast.LocationRange) {
	ec.Collect(errors.MakeStaticError(msg, *loc))
}
