/*
Copyright 2019 Google Inc. All rights reserved.

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
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

type stack struct {
	elements []interface{}
}

func (s *stack) Push(e interface{}) {
	s.elements = append(s.elements, e)
}

func (s *stack) Pop() (interface{}, error) {
	if len(s.elements) == 0 {
		return nil, errors.New("cannot pop from empty stack")
	}
	l := len(s.elements)
	e := s.elements[l-1]
	s.elements = s.elements[:l-1]
	return e, nil
}

func (s *stack) Size() int {
	return len(s.elements)
}

type jsonMLBuilder struct {
	stack              *stack
	preserveWhitespace bool
	currDepth          int
}

// BuildJsonmlFromString returns a jsomML form of given xml string.
func BuildJsonmlFromString(s string, preserveWhitespace bool) ([]interface{}, error) {
	b := newBuilder(preserveWhitespace)
	d := xml.NewDecoder(strings.NewReader(s))

	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if err := b.addToken(token); err != nil {
			return nil, err
		}
	}

	if b.stack.Size() == 0 {
		// No nodes has been identified
		return nil, fmt.Errorf("%s is not a valid XML", s)
	}

	return b.build(), nil
}

func newBuilder(preserveWhitespace bool) *jsonMLBuilder {
	return &jsonMLBuilder{
		stack:              &stack{},
		preserveWhitespace: preserveWhitespace,
	}
}

func (b *jsonMLBuilder) addToken(token xml.Token) error {
	switch token.(type) {
	case xml.StartElement:
		// check for multiple roots
		if b.currDepth == 0 && b.stack.Size() > 0 {
			// There are multiple root elements
			return errors.New("XML cannot have multiple root elements")
		}

		t := token.(xml.StartElement)
		node := []interface{}{t.Name.Local}
		// Add Attributes
		if len(t.Attr) > 0 {
			attr := make(map[string]interface{})
			for _, a := range t.Attr {
				attr[a.Name.Local] = a.Value
			}
			node = append(node, attr)
		}
		b.stack.Push(node)
		b.currDepth++
	case xml.CharData:
		t := token.(xml.CharData)
		s := string(t)
		if !b.preserveWhitespace {
			s = strings.TrimSpace(s)
		}
		if len(s) > 0 { // Skip empty strings
			b.appendToLastNode(string(t))
		}
	case xml.EndElement:
		b.squashLastNode()
		b.currDepth--
	}

	return nil
}

func (b *jsonMLBuilder) build() []interface{} {
	root, _ := b.stack.Pop()
	return root.([]interface{})
}

func (b *jsonMLBuilder) appendToLastNode(e interface{}) {
	if b.stack.Size() == 0 {
		return
	}
	node, _ := b.stack.Pop()
	n := node.([]interface{})
	n = append(n, e)
	b.stack.Push(n)
}

func (b *jsonMLBuilder) squashLastNode() {
	if b.stack.Size() < 2 {
		return
	}
	n, _ := b.stack.Pop()
	b.appendToLastNode(n)
}
