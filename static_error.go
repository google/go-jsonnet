package jsonnet

import (
	"fmt"
)

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
func makeLocationRangeMessage(msg string) LocationRange {
	return LocationRange{FileName: msg}
}

func makeLocationRange(fn string, begin Location, end Location) LocationRange {
	return LocationRange{FileName: fn, Begin: begin, End: end}
}

//////////////////////////////////////////////////////////////////////////////
// StaticError

// StaticError represents an error during parsing/lexing some jsonnet.
type StaticError struct {
	Loc LocationRange
	Msg string
}

func makeStaticErrorMsg(msg string) StaticError {
	return StaticError{Msg: msg}
}

func makeStaticErrorPoint(msg string, fn string, l Location) StaticError {
	return StaticError{Msg: msg, Loc: makeLocationRange(fn, l, l)}
}

func makeStaticError(msg string, lr LocationRange) StaticError {
	return StaticError{Msg: msg, Loc: lr}
}

func (err StaticError) Error() string {
	loc := ""
	if err.Loc.IsSet() {
		loc = err.Loc.String()
	}
	return fmt.Sprintf("%v %v", loc, err.Msg)
}
