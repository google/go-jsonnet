local array = [
  'bar',
  1,
  1.42,
  -1,
  false,
  true,
];

{
  array: std.manifestTomlEx(array, '  '),
}
