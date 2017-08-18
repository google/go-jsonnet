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

package ast

import "fmt"

//////////////////////////////////////////////////////////////////////////////
// Location

// Location represents a single location in an (unspecified) file.
type Location struct {
	Line   int
	Column int
}

// IsSet returns if this Location has been set.
func (l *Location) IsSet() bool {
	return l.Line != 0
}

func (l *Location) String() string {
	return fmt.Sprintf("%v:%v", l.Line, l.Column)
}

//////////////////////////////////////////////////////////////////////////////
// LocationRange

// LocationRange represents a range of a source file.
type LocationRange struct {
	FileName string
	Begin    Location
	End      Location
}

// IsSet returns if this LocationRange has been set.
func (lr *LocationRange) IsSet() bool {
	return lr.Begin.IsSet()
}

func (lr *LocationRange) String() string {
	if !lr.IsSet() {
		return lr.FileName
	}

	var filePrefix string
	if len(lr.FileName) > 0 {
		filePrefix = lr.FileName + ":"
	}
	if lr.Begin.Line == lr.End.Line {
		if lr.Begin.Column == lr.End.Column {
			return fmt.Sprintf("%s%v", filePrefix, lr.Begin.String())
		}
		return fmt.Sprintf("%s%v-%v", filePrefix, lr.Begin.String(), lr.End.Column)
	}

	return fmt.Sprintf("%s(%v)-(%v)", filePrefix, lr.Begin.String(), lr.End.String())
}

// This is useful for special locations, e.g. manifestation entry point.
func MakeLocationRangeMessage(msg string) LocationRange {
	return LocationRange{FileName: msg}
}

func MakeLocationRange(fn string, begin Location, end Location) LocationRange {
	return LocationRange{FileName: fn, Begin: begin, End: end}
}
