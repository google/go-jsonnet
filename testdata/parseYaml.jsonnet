[
  std.parseYaml(text)
  for text in [
    // various node types
    |||
      foo: bar
      aaa: {}
      ąę: ćż
      xxx:
      - 42
      - asdf
      - {}
    |||,

    // Returns an array of documents when there is an explicit document(s), i.e.
    // the text is a "multi-doc" stream
    |||
      ---
      a: 1
      ---
      a: 2
    |||,

    // The first document in a "multi-doc" stream can be an implicit document.
    |||
      a: 1
      ---
      a: 2
    |||,

    // Whitespaces are allowed after the document start marker
    |||
      ---
      a: 1
      ---
      a: 2
    |||,

    // Document start marker needs a following line break or a whitespace.
    |||
      a: 1
      ---a: 2
      a---: 3
    |||,

    // Scalar documents can start on the same line as the document-start marker
    |||
      a: 1
      --- >
        hello
        world
      --- 3
    |||,

    // Documents can be empty; this is interpreted as null
    |||
      a: 1
      ---
      --- 2
    |||,

    "---",

    // A leading comment before the first document-start marker must not
    // produce a spurious null document (issue #660).
    |||
      # Test
      ---
      foo: bar
      ---
      baz: cuux
    |||,

    // Multiple leading comments and blank lines before the first marker.
    |||
      # Comment 1
      # Comment 2

      ---
      a: 1
    |||,
  ]
]
