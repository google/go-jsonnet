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

{
  object: std.manifestTomlEx(object, '  '),
  object2: std.manifestTomlEx(object2, '  '),
}