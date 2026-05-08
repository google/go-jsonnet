// Regression test for https://github.com/google/go-jsonnet/issues/785
//
// The pattern below builds an accumulator where each iteration's `+:`
// causes the resulting field value to reference itself, producing an
// extendedObject DAG (left and right of the `+` end up pointing at the
// same uncachedObject). Walking the DAG as a tree leads to exponential
// behaviour in manifestJSON / checkAssertions; we make sure here that
// it now scales linearly and produces the correct value.
local ouch(values) =
  std.foldl(
    function(acc, value)
      local lookup = std.get(acc, "a", {});
      acc { ["a"]+: lookup },
    values,
    {}
  );

local values = [null for x in std.range(1, 50)];
ouch(values)
