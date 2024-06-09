local object = {
  foo: 'baz',
  abc: 'def',
  bar: self.foo,
  baz: 1,
  bazel: 1.42,
  boom: -1,
  bim: false,
  bam: true,
  blamo: {
    cereal: [
      '<>& fizbuzz',
      ['a', ['b']],
    ],

    treats: [
      {
        name: 'chocolate',
      },
    ],
  },
};

local object2 = {
  key: 'value',
  simple: { t: 5 },
  section: {
    a: 1,
    nested: { b: 2 },
    'e$caped': { q: 't' },
    array: [
      { c: 3 },
      { d: 4 },
    ],
    nestedArray: [{
      k: 'v',
      nested: { e: 5 },
    }],
  },
  arraySection: [
    { q: 1 },
    { w: 2 },
  ],
  'escaped"Section': { z: 'q' },
  emptySection: {},
  emptyArraySection: [{}],
  bool: true,
  notBool: false,
  number: 7,
  array: ['s', 1, [2, 3], { r: 6, a: ['0', 'z'] }],
  emptyArray: [],
  '"': 4,
};

local object3 = {
    '0X_0a_74_ae': 'BARE_KEY',
  '__-0X_0a_74_ae': 'BARE_KEY',
  '-0B1010_0111_0100_1010_1110': 'string
                                    with some
                                    newlines
        ',
  '__-0B1010_0111_0100_1010_1110': 'a new line
  ',
  x: 'BARE_KEY',
  b: {
    y: 'boolean true',
    yes: 'boolean true',
    Yes: 'boolean true',
    True: 'boolean true',
    'true': 'boolean true',
    on: 'boolean true',
    On: 'boolean true',
    NO: 'boolean false',
    n: 'boolean false',
    N: 'boolean false',
    off: 'boolean false',
    OFF: 'boolean false',
    'null': 'null word',
    NULL: 'null word capital',
    Null: 'null word',
  },
  just_letters_underscores: 142321,
  'just-letters-dashes': '+1101_1111',
  'jsonnet.org/k8s-label-like': '0600',
  '192.168.0.1': [{a: 2, b: 'str'}, {c : []}],
  '1-234-567-8901': null,
};

{
  object: std.manifestYamlDoc(object),
  object2: std.manifestYamlDoc(object2),
  object3: std.manifestYamlDoc(object3),

  object_indent: std.manifestYamlDoc(object, indent_array_in_object=true),
  object2_indent: std.manifestYamlDoc(object2, indent_array_in_object=true),
  object3_indent: std.manifestYamlDoc(object3, indent_array_in_object=true),

  object_unquoted: std.manifestYamlDoc(object, quote_keys=false),
  object2_unquoted: std.manifestYamlDoc(object2, quote_keys=false),
  object3_unquoted: std.manifestYamlDoc(object3, quote_keys=false),

  object_indent_unquoted: std.manifestYamlDoc(object, indent_array_in_object=true, quote_keys=false),
  object2_indent_unquoted: std.manifestYamlDoc(object2, indent_array_in_object=true, quote_keys=false),
  object3_indent_unquoted: std.manifestYamlDoc(object3, indent_array_in_object=true, quote_keys=false),
}
