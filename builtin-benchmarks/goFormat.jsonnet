[
  std.goFormat('test-%v-%v', [i, y])
  for i in std.range(0, 100)
  for y in std.range(0, 100)
]
