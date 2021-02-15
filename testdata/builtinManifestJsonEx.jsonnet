local a = {
  foo: 'bar',
  bar: self.foo,
  baz: 1,
  bazel: 1.42,
  boom: -1,
  bim: false,
  bam: true,
  blamo: {
    cereal: [
      '<>& fizbuzz',
    ],

    treats: [
      {
        name: 'chocolate',
      },
    ],
  },
};

local b = [
  'bar',
  a.foo,
  1,
  1.42,
  -1,
  false,
  true,
  {
    cereal: [
      '<>& fizbuzz',
    ],

    treats: [
      {
        name: 'chocolate',
      },
    ],
  },
];

{
  a: std.manifestJsonEx(a, '  '),
  b: std.manifestJsonEx(b, '  '),
}
