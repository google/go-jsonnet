// Fields are declared in z, a, m order; output should reflect that
// when --preserve-field-order is set, or be sorted alphabetically otherwise.
local obj = { z: 1, a: 2, m: 3 };
local nested = { z: { y: 10, b: 20 }, a: { q: 30, c: 40 } };
local extended = { z: 1, a: 2 } + { m: 3, b: 4 };
{
  plain: obj,
  nested: nested,
  extended: extended,
}
