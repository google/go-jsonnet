// Package testutils provides general testing utilities.
package testutils

import (
	"bytes"
	"os"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Diff produces a pretty diff of two files
func Diff(a, b string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a, b, false)
	return dmp.DiffPrettyText(diffs)
}

// CompareWithGolden check if a file is the same as golden file.
// If it is not it produces a pretty diff.
func CompareWithGolden(result string, golden []byte) (string, bool) {
	if !bytes.Equal(golden, []byte(result)) {
		// TODO(sbarzowski) better reporting of differences in whitespace
		// missing newline issues can be very subtle now
		return Diff(result, string(golden)), true
	}
	return "", false
}

// UpdateGoldenFile updates a golden file with new contents if the new contents
// are actually different from what is already there. It returns whether or not
// the overwrite was performed (i.e. the desired content was different than actual).
func UpdateGoldenFile(path string, content []byte, mode os.FileMode) (changed bool, err error) {
	old, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	// If it exists and already has the right content, do nothing,
	if bytes.Equal(old, content) && !os.IsNotExist(err) {
		return false, nil
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return false, err
	}
	return true, nil
}
