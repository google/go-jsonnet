local data = {
  a: 'A',
  b: 'B',
};

local process(input) = {
  local v = input[k],
  [k]: v
  for k in std.objectFields(input)
};

process(data)
